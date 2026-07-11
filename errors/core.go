package errors

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

// 全局常量定义

const (
	defaultStackDepth = 32     // 默认每层错误捕获的栈帧深度
	maxStackDepth     = 64     // 每层错误捕获栈帧深度的上限
	maxChainDepth     = 1024   // 错误链遍历的最大深度，防止异常链过深导致的性能问题
	errorJoinSep      = " -> " // 错误链输出分隔符
)

// 全局原子变量，用于并发安全的配置控制

var (
	stackDepthLimit   atomic.Int32 // 每一层追踪错误捕获的最大栈帧数（原子操作）
	traceOutputEnable atomic.Bool  // 是否输出链路追踪栈的全局开关（原子操作）
)

// init 初始化函数，设置全局变量的默认值
func init() {
	stackDepthLimit.Store(defaultStackDepth) // 初始栈深度为 32
	traceOutputEnable.Store(true)            // 默认开启链路追踪输出
}

// ============================ 接口定义 ============================

// unwrapper 表示支持标准单链展开的错误接口。
// 标准库 errors 包中的错误通过 Unwrap() 方法形成单向链表。
type unwrapper interface {
	Unwrap() error
}

// multiUnwrapper 表示支持多链展开的错误接口，用于 errors.Join 等场景。
// 返回的切片中可能包含多个错误节点，形成树状结构。
type multiUnwrapper interface {
	Unwrap() []error
}

// ============================ 核心错误创建 ============================

// New 创建一个带链路追踪栈的新错误。
// 内部会调用 runtime.Callers 捕获当前调用栈信息。
func New(msg string) error {
	return newStackError(msg, nil)
}

// Errorf 创建一个带链路追踪栈的格式化错误。
// 使用 fmt.Sprintf 格式化消息，内部同样会捕获调用栈。
func Errorf(format string, args ...any) error {
	return newStackError(fmt.Sprintf(format, args...), nil)
}

// ============================ 全局配置管理 ============================

// StackDepth 返回当前每一层追踪错误捕获的最大栈帧数。
// 该值控制 runtime.Callers 捕获栈帧的数量，影响栈追踪的详细程度和性能开销。
func StackDepth() int {
	if depth := int(stackDepthLimit.Load()); depth > 0 {
		return depth
	}
	return defaultStackDepth
}

// SetStackDepth 设置每一层追踪错误捕获的最大栈帧数。
// 该设置为全局配置，只影响后续新创建的追踪错误，不影响已创建的错误对象。
//
// 边界处理规则：
//   - depth < 1 时，按 1 处理（最小栈深度）
//   - depth > maxStackDepth 时，按 maxStackDepth 处理（最大栈深度）
func SetStackDepth(depth int) int {
	switch {
	case depth < 1:
		depth = 1
	case depth > maxStackDepth:
		depth = maxStackDepth
	}
	stackDepthLimit.Store(int32(depth))
	return depth
}

// TraceEnabled 返回当前是否输出链路追踪栈。
// 关闭追踪后，TraceString/TraceJSON 和 fmt 格式化输出将不包含 trace 字段，
// 但仍会输出错误链消息和错误码，可有效减少日志输出量和性能开销。
func TraceEnabled() bool {
	return traceOutputEnable.Load()
}

// SetTraceEnabled 设置是否输出链路追踪栈。
// 该设置为全局配置，只影响 TraceString/TraceJSON/fmt 格式化输出，
// 不影响错误的创建、比较和展开逻辑。
func SetTraceEnabled(enabled bool) bool {
	traceOutputEnable.Store(enabled)
	return enabled
}

// ============================ 错误包装函数 ============================

// Wrap 包装错误，附加调用信息。
// 根据底层错误是否已包含追踪栈，自动选择最优包装策略，避免重复采集栈。
//
// err 为 nil 时直接返回 nil；错误链路已有追踪栈时只创建轻量 messageError；
// 底层错误无追踪栈时创建完整 stackError 并采集调用栈。
func Wrap(err error, msg ...string) error {
	if err == nil {
		return nil
	}

	// 如果没有附加消息，则仅附加一层调用信息
	if len(msg) == 0 {
		return Tag(err)
	}

	message := msg[0]
	if HasStack(err) {
		return &messageError{msg: message, err: err}
	}

	return newStackError(message, err)
}

// Wrapf 使用格式化消息包装错误。
// 内部逻辑同 Wrap，但消息通过 fmt.Sprintf 动态生成。
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

// WithMessage 仅附加一层消息，不采集调用栈。
// 设计用于高频返回路径，避免每次 Wrap 都采集栈带来的性能开销。
// 错误链中会保留原始错误的栈信息。
func WithMessage(err error, msg string) error {
	if err == nil {
		return nil
	}
	return &messageError{msg: msg, err: err}
}

// WithMessagef 仅附加一层格式化消息，不采集调用栈。
// 同 WithMessage，但消息通过 fmt.Sprintf 动态生成。
func WithMessagef(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return &messageError{msg: fmt.Sprintf(format, args...), err: err}
}

// ============================ 多错误合并 ============================

// Join 合并多个错误，语义与标准库 errors.Join 完全一致。
// 合并后的错误支持 multiUnwrapper 接口，可通过 Unwrap() 获取所有子错误。
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// ============================ 上下文感知错误 ============================

// WithContext 使用 context 中的值包装错误。
// 自动提取 ctx 中通过 WithContextErr/WithContextErrs 存储的键值对，
// 将其附加到错误链中，便于日志追踪和调试。
//
// 提取的 key-value 必须是字符串类型，确保兼容 slog 属性。
// ctx 为 nil 或没有可提取值时直接返回原错误，不创建额外对象。
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

// contextValuesKey 是用于在 context 中存储错误相关键值对的 key 类型。
// 使用空结构体作为 key，避免与其他 context key 冲突。
type contextValuesKey struct{}

// WithContextErr 将一对键值存储到 context 中。
// 存储的值后续会被 WithContext 自动提取并附加到错误链。
// 适用于在业务处理链中传递请求级标识（如 user_id、request_id）。
func WithContextErr(ctx context.Context, key, value string) context.Context {
	return context.WithValue(normalizeContext(ctx), contextValuesKey{}, []string{key, value})
}

// WithContextErrs 批量将多对键值存储到 context 中。
// 出于稳定性考虑，奇数个参数时不会 panic，而是忽略最后一个不完整项。
//
// 注意：kvs 长度为奇数时，会忽略最后一个不完整项
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

// WithContextErrsE 批量将多对键值存储到 context 中，并返回显式错误。
// 与 WithContextErrs 不同，当前函数在输入参数不完整时会返回错误，便于调用方感知配置问题。
func WithContextErrsE(ctx context.Context, kvs ...string) (context.Context, error) {
	parent := normalizeContext(ctx)
	if len(kvs)%2 != 0 {
		return parent, Errorf("WithContextErrsE 参数必须成对出现，当前参数个数=%d", len(kvs))
	}
	kv := append([]string(nil), kvs...)
	return context.WithValue(parent, contextValuesKey{}, kv), nil
}

// normalizeContext 规范化 context 父对象。
// 当传入 nil context 时，自动回退到 context.Background()，避免库函数内部 panic。
func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// contextError 包装错误并携带 context 中的关键键值对。
// 实现 unwrapper 接口，支持错误链遍历。
//
// values 保存扁平化键值对，格式为 [key1, value1, key2, value2, ...]。
type contextError struct {
	err    error    // 被包装的原始错误
	values []string // context 中提取的键值对
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

// Is 实现 errors.Is 接口，支持错误比较。
// 当目标错误也是 contextError 时，比较其 values 是否完全一致。
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

// Tag 统一包装任意类型的错误，智能判断是否需要添加追踪链路。
// 这是 Wrap 的增强版本，专门用于处理来自标准库或第三方包的错误。
//
// err 为 nil 时直接返回 nil；错误已有追踪栈时直接返回原错误；
// 错误没有追踪栈时创建带追踪栈的新错误。
//
// 使用场景：
//   - 统一处理来自不同来源的错误（标准库、第三方包、本包错误）
//   - 确保所有错误都有完整的追踪链路，便于调试
//   - 避免对已有追踪的错误重复包装，提高性能
//
// 例如：
//
//	// 包装标准库错误
//	err := os.Open("file.txt")  // 标准库错误，无追踪
//	wrappedErr := errors.Tag(err)  // 现在有追踪链路了
//
//	// 包装已有追踪的错误
//	trackedErr := errors.New("already tracked")  // 已有追踪
//	result := errors.Tag(trackedErr)     // 直接返回原错误
func Tag(err error) error {
	if err == nil {
		return nil
	}

	// 如果错误已经有追踪栈，直接返回，避免重复包装
	if HasStack(err) {
		return err
	}

	// 创建带追踪栈的新错误
	return newStackError("", err)
}

// ============================ 标准库兼容函数 ============================

// Is 语义与标准库 errors.Is 完全一致。
// 沿错误链向上遍历，比较是否存在与目标错误相等的错误节点。
func Is(err, target error) bool { return errors.Is(err, target) }

// As 语义与标准库 errors.As 完全一致。
// 沿错误链向上遍历，查找是否存在类型匹配的错误。
func As(err error, target any) bool { return errors.As(err, target) }

// AsType 从错误链中提取第一个类型为 T 的错误。
// 语义对齐 Go 1.26 errors.AsType：按深度优先顺序遍历单链和多链错误。
//
// 类型参数说明：
//   - T：目标错误类型，必须实现 error 接口
func AsType[T error](err error) (T, bool) {
	return errors.AsType[T](err)
}

// HasMsg 检查错误链中是否包含指定的消息内容。
// 使用精确匹配（==），用于快速判断特定业务错误类型。
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

// composeErrorMessage 组合错误消息。
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

// Unwrap 语义与标准库 errors.Unwrap 完全一致。
// 返回错误链中的下一个错误。
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// ============================ messageError 轻量级错误包装 ============================

// messageError 仅附加消息，不采集调用栈的轻量级错误包装类型。
// 设计用于高频路径，避免每次包装都采集栈。
// 内部会复用底层错误的栈信息，不重复采集。
//
// 它保存附加消息和被包装的底层错误。
type messageError struct {
	msg string // 附加的错误消息
	err error  // 被包装的底层错误
}

// Error 返回附加的错误消息。
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

// Is 实现 errors.Is 接口，支持错误比较。
// 当目标错误也是 messageError 时，比较其 msg 和 err 是否完全一致。
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
