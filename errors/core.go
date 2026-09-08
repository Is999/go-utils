package errors

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

const (
	defaultStackDepth = 32     // 新建栈错误默认采集的最大帧数。
	maxStackDepth     = 64     // 单次栈采集的帧数上限。
	maxChainDepth     = 1024   // 单次遍历的节点数上限；递归渲染时作为深度上限。
	errorJoinSep      = " -> " // Error() 拼接本层与原因消息的分隔符。
)

var (
	stackDepthLimit   atomic.Int32 // 后续新建栈错误使用的采集深度。
	traceOutputEnable atomic.Bool  // 控制渲染是否包含栈帧，不影响采集。
)

// init 默认采集至多 32 帧并启用栈输出，采集深度与渲染开关分别管理。
func init() {
	stackDepthLimit.Store(defaultStackDepth)
	traceOutputEnable.Store(true)
}

// unwrapper 展开单一原因；nil 表示当前节点已无下层错误。
type unwrapper interface {
	// Unwrap 返回被当前节点包装的错误。
	Unwrap() error
}

// multiUnwrapper 展开 Join 等多分支错误，子节点按切片顺序遍历。
type multiUnwrapper interface {
	// Unwrap 返回原错误持有的子节点，读取方不得修改该切片。
	Unwrap() []error
}

// New 创建带调用栈的错误，栈从调用点开始，深度受 SetStackDepth 控制。
func New(msg string) error {
	return newStackError(msg, nil)
}

// Errorf 按 fmt.Sprintf 格式化消息并采集调用栈，不支持用 %w 包装已有错误。
// 需要保留底层错误时使用 Wrapf。
func Errorf(format string, args ...any) error {
	return newStackError(fmt.Sprintf(format, args...), nil)
}

// StackDepth 返回后续新错误采集的最大栈帧数，默认 32。
func StackDepth() int {
	// 初始化写入默认值，SetStackDepth 保证后续值均在 1–64 之间。
	return int(stackDepthLimit.Load())
}

// SetStackDepth 将采集深度限制在 1–64 并返回实际值，只影响之后创建的错误。
func SetStackDepth(depth int) int {
	depth = min(max(depth, 1), maxStackDepth)
	stackDepthLimit.Store(int32(depth))
	return depth
}

// TraceEnabled 返回栈输出开关；关闭仅省略栈，其他字段沿用各输出格式的规则。
func TraceEnabled() bool {
	return traceOutputEnable.Load()
}

// SetTraceEnabled 设置并返回栈输出开关，不影响创建错误时采集栈。
// Trace 在构造时记录开关，TraceString、TraceJSON 和 fmt 在每次渲染时读取开关。
func SetTraceEnabled(enabled bool) bool {
	traceOutputEnable.Store(enabled)
	return enabled
}

// Wrap 为错误附加消息，并在错误链尚无本包调用栈时采集调用位置。
// err 为 nil 时返回 nil；未传 msg 且已有栈时返回原错误，否则创建包装节点。
// msg 只使用第一项；显式传入空字符串也会创建节点，但不改变 Error() 文本。
func Wrap(err error, msg ...string) error {
	if err == nil {
		return nil
	}

	message := ""
	// 先保存消息，避免自定义 Unwrap 在遍历时修改共享消息切片后改变取值。
	if len(msg) > 0 {
		message = msg[0]
	}
	if HasStack(err) {
		if len(msg) == 0 {
			return err
		}
		return &messageError{msg: message, err: err}
	}

	// 从当前公开入口采集，避免经 Tag 转发后首帧指向 Wrap 本身。
	return newStackError(message, err)
}

// Wrapf 按 fmt.Sprintf 格式化消息后包装错误；nil 原样返回，仅在链中无本包栈时采集。
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(format, args...)
	if HasStack(err) {
		return &messageError{msg: msg, err: err}
	}
	return newStackError(msg, err)
}

// WithMessage 仅附加消息，不检查或采集调用栈；err 为 nil 时返回 nil。
// 已在失败点采集栈的错误向外传播时，可用它省去 Wrap 对错误链的栈检查。
func WithMessage(err error, msg string) error {
	if err == nil {
		return nil
	}
	return &messageError{msg: msg, err: err}
}

// WithMessagef 按 fmt.Sprintf 格式化消息，不检查或采集调用栈；err 为 nil 时返回 nil。
func WithMessagef(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return &messageError{msg: fmt.Sprintf(format, args...), err: err}
}

// Join 沿用标准库语义：忽略 nil，保留输入顺序，全部为 nil 时返回 nil。
// 合并结果提供 Unwrap() []error；包级 Unwrap 只展开单链，不展开 Join。
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// WithContext 将本包上下文中的字符串键值附到错误上，不采集调用栈。
// err、ctx 为空或无可提取键值时返回原错误；不读取任意 context key，也不持有整个 ctx。
func WithContext(ctx context.Context, err error) error {
	if err == nil || ctx == nil {
		return err
	}
	// values 由包内私有 context key 持有且只读，直接复用可避免二次拷贝。
	values, _ := ctx.Value(contextValuesKey{}).([]string)
	if len(values) == 0 {
		return err
	}
	return &contextError{err: err, values: values}
}

// contextValuesKey 隔离本包错误字段，避免与调用方的 context key 冲突。
type contextValuesKey struct{}

// WithContextErr 用一对字符串键值替换本包先前的错误上下文；nil ctx 按 Background 处理。
func WithContextErr(ctx context.Context, key, value string) context.Context {
	return context.WithValue(normalizeContext(ctx), contextValuesKey{}, []string{key, value})
}

// WithContextErrs 存储本层错误上下文，忽略末尾未配对的参数。
// 没有完整键值对时不增加上下文层；有完整键值对时替换先前的错误上下文。
func WithContextErrs(ctx context.Context, kvs ...string) context.Context {
	if len(kvs)%2 != 0 {
		kvs = kvs[:len(kvs)-1]
	}
	if len(kvs) == 0 {
		return normalizeContext(ctx)
	}
	nextCtx, _ := WithContextErrsE(ctx, kvs...)
	return nextCtx
}

// WithContextErrsE 复制完整键值对并替换先前字段，空参数会清空本包错误上下文。
// 参数未配对时返回归一化后的原 ctx 和错误，不丢弃尾项。
func WithContextErrsE(ctx context.Context, kvs ...string) (context.Context, error) {
	parent := normalizeContext(ctx)
	if len(kvs)%2 != 0 {
		return parent, Errorf("WithContextErrsE 参数必须成对出现，当前参数个数=%d", len(kvs))
	}
	kv := append([]string(nil), kvs...)
	return context.WithValue(parent, contextValuesKey{}, kv), nil
}

// normalizeContext 将 nil 父上下文替换为 Background。
func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// contextError 携带只读上下文字段，消息和展开行为由底层错误提供。
type contextError struct {
	err    error    // 被包装的原始错误
	values []string // 只读的 [key, value, ...]，与本包 context 共享底层切片。
}

// Error 返回被包装错误的错误消息。
func (e *contextError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

// Unwrap 返回被包装的原始错误，支持错误链展开。
func (e *contextError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Is 只匹配同一个包装对象；上下文值相同不代表同一次错误。
func (e *contextError) Is(target error) bool {
	te, ok := target.(*contextError)
	return ok && e == te
}

// String 返回错误的文本追踪格式，等同于 TraceString。
func (e *contextError) String() string { return TraceString(e) }

// GoString 返回错误的 JSON 追踪格式，等同于 TraceJSON。
func (e *contextError) GoString() string { return TraceJSON(e) }

// Format 实现 fmt.Formatter 接口，支持格式化动词（%v/%s/%q 等）。
func (e *contextError) Format(s fmt.State, verb rune) {
	formatError(s, verb, e)
}

// MarshalJSON 实现 json.Marshaler 接口，返回 JSON 追踪格式。
func (e *contextError) MarshalJSON() ([]byte, error) {
	return []byte(TraceJSON(e)), nil
}

// MarshalText 实现 encoding.TextMarshaler 接口，返回文本追踪格式。
func (e *contextError) MarshalText() ([]byte, error) {
	return []byte(TraceString(e)), nil
}

// Tag 在错误链尚无本包调用栈时附加栈，不改变 Error() 文本。
// err 为 nil 或链中已有栈时返回原值；语义与不传消息的 Wrap 相同。
func Tag(err error) error {
	if err == nil {
		return nil
	}

	if HasStack(err) {
		return err
	}

	return newStackError("", err)
}

// Is 按标准库规则检查当前节点及其原因，支持自定义 Is 和 Join 分支。
func Is(err, target error) bool { return errors.Is(err, target) }

// As 按标准库规则查找首个匹配并写入 target；target 必须是标准库接受的非 nil 指针。
func As(err error, target any) bool { return errors.As(err, target) }

// AsType 按标准库的深度优先顺序提取首个 T；未匹配时返回 T 的零值和 false。
func AsType[T error](err error) (T, bool) {
	var target T
	if errors.As(err, &target) {
		return target, true
	}
	// 自定义 As 即使修改 target 后返回 false，也不能让未匹配结果携带该值。
	var zero T
	return zero, false
}

// HasMsg 按节点展示的消息精确匹配，不做子串搜索；空目标返回 false。
func HasMsg(err error, msg string) bool {
	if err == nil || msg == "" {
		return false
	}
	var stackBuf [8]error
	stack := stackBuf[:0]
	stack = append(stack, err)
	for depth := 0; len(stack) > 0 && depth < maxChainDepth; depth++ {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if current == nil {
			continue
		}
		info := inspectError(current)
		if info.hasMsg && info.msg == msg {
			return true
		}
		switch {
		case len(info.children) > 0:
			stack = pushChildren(stack, info.children)
		case info.next != nil:
			stack = append(stack, info.next)
		}
	}
	return false
}

// composeErrorMessage 用箭头连接本层与原因消息，空消息和相同消息不重复输出。
func composeErrorMessage(msg string, err error) string {
	switch {
	case msg == "":
		if err == nil {
			return ""
		}
		return err.Error()
	case err == nil:
		return msg
	}

	cause := err.Error()
	if cause == "" || cause == msg {
		return msg
	}
	return msg + errorJoinSep + cause
}

// Unwrap 只调用 Unwrap() error；nil、叶子节点和 Join 均返回 nil。
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// messageError 只保存本层消息和底层错误，调用栈仍由链中的 stackError 持有。
type messageError struct {
	msg string // 附加的错误消息
	err error  // 被包装的底层错误
}

// Error 用分隔符连接本层和底层消息，空消息或相同消息不重复输出。
func (e *messageError) Error() string {
	if e == nil {
		return ""
	}
	return composeErrorMessage(e.msg, e.err)
}

// Unwrap 返回被包装的底层错误，支持错误链展开。
func (e *messageError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Is 只匹配同一个包装对象；消息和原因相同的独立包装仍是不同错误。
func (e *messageError) Is(target error) bool {
	te, ok := target.(*messageError)
	return ok && e == te
}

// String 返回错误的文本追踪格式，等同于 TraceString。
func (e *messageError) String() string { return TraceString(e) }

// GoString 返回错误的 JSON 追踪格式，等同于 TraceJSON。
func (e *messageError) GoString() string { return TraceJSON(e) }

// Format 实现 fmt.Formatter 接口，支持格式化动词（%v/%s/%q 等）。
func (e *messageError) Format(s fmt.State, verb rune) {
	formatError(s, verb, e)
}

// MarshalJSON 实现 json.Marshaler 接口，返回 JSON 追踪格式。
func (e *messageError) MarshalJSON() ([]byte, error) {
	return []byte(TraceJSON(e)), nil
}

// MarshalText 实现 encoding.TextMarshaler 接口，返回文本追踪格式。
func (e *messageError) MarshalText() ([]byte, error) {
	return []byte(TraceString(e)), nil
}
