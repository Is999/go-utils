package errors

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

const (
	defaultStackDepth = 32
	maxStackDepth     = 64
	maxChainDepth     = 1024
)

var (
	stackDepthLimit   atomic.Int32
	traceOutputEnable atomic.Bool
)

func init() {
	stackDepthLimit.Store(defaultStackDepth)
	traceOutputEnable.Store(true)
}

// unwrapper 表示支持标准单链展开的错误。
type unwrapper interface {
	Unwrap() error
}

// multiUnwrapper 表示支持多链展开的错误，例如 errors.Join。
type multiUnwrapper interface {
	Unwrap() []error
}

// New 创建一个带链路追踪栈的错误。
func New(msg string) error {
	return newStackError(msg, nil)
}

// Errorf 创建一个带链路追踪栈的格式化错误。
func Errorf(format string, args ...any) error {
	return newStackError(fmt.Sprintf(format, args...), nil)
}

// StackDepth 返回当前每一层追踪错误捕获的最大栈帧数。
// 每一层错误在创建时各自采集调用栈；打印日志时，会沿错误链汇总这些创建点信息。
func StackDepth() int {
	if depth := int(stackDepthLimit.Load()); depth > 0 {
		return depth
	}
	return defaultStackDepth
}

// SetStackDepth 设置每一层追踪错误捕获的最大栈帧数。
// 小于 1 时按 1 处理；大于 maxStackDepth 时按 maxStackDepth 截断。
// 它只影响后续新创建的追踪错误，不影响已创建错误。
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
// 关闭后仍会输出错误链消息和错误码，但不会渲染 trace 字段，可减少日志输出成本。
func TraceEnabled() bool {
	return traceOutputEnable.Load()
}

// SetTraceEnabled 设置是否输出链路追踪栈。
// 它只影响 Trace/TraceString/TraceJSON 以及 fmt 的格式化输出，不影响错误创建、比较和展开逻辑。
func SetTraceEnabled(enabled bool) bool {
	traceOutputEnable.Store(enabled)
	return enabled
}

// Wrap 包装错误。
// 1. err 为 nil 时返回 nil。
// 2. 如果底层链路已存在追踪栈，则只附加消息，避免重复采集栈。
// 3. 否则创建新的带栈错误节点。
func Wrap(err error, msg ...string) error {
	if err == nil {
		return nil
	}

	message := ""
	if len(msg) > 0 {
		message = msg[0]
	}

	if HasStack(err) {
		if message == "" {
			return err
		}
		return &messageError{msg: message, err: err}
	}

	if message == "" {
		message = err.Error()
	}
	return newStackError(message, err)
}

// Wrapf 使用格式化消息包装错误。
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

// WithMessage 仅附加一层消息，不采集栈，适合高频返回路径。
func WithMessage(err error, msg string) error {
	if err == nil {
		return nil
	}
	return &messageError{msg: msg, err: err}
}

// WithMessagef 仅附加一层格式化消息，不采集栈。
func WithMessagef(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return &messageError{msg: fmt.Sprintf(format, args...), err: err}
}

// Join 合并多个错误，语义与标准库 errors.Join 一致。
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// WithContext 使用 context 的值包装错误，提取 ctx 中的 key-value 到错误链中。
// 提取的 key 仅限 string 类型，确保兼容 slog 属性。
// 如果 ctx 中没有可提取的值，直接返回原错误，不创建额外对象。
func WithContext(ctx context.Context, err error) error {
	if err == nil || ctx == nil {
		return err
	}
	values := extractContextValues(ctx)
	if len(values) == 0 {
		return err
	}
	return &contextError{err: err, values: values}
}

// extractContextValues 从 context 中提取所有 string 类型的 key-value 对。
func extractContextValues(ctx context.Context) []string {
	var result []string
	for {
		select {
		case <-ctx.Done():
			return result
		default:
		}
		if values := ctx.Value(contextValuesKey{}); values != nil {
			if kv, ok := values.([]string); ok {
				for i := 0; i < len(kv); i += 2 {
					result = append(result, kv[i], kv[i+1])
				}
			}
		}
		return result
	}
}

// contextValuesKey 是用于在 context 中存储错误相关 key-value 的 key 类型。
type contextValuesKey struct{}

// WithContextErr 将键值对存储到 context 中，供后续 WithContext 提取。
// 使用此函数存储的值会被 WithContext 自动提取并附加到错误链中。
// 键值必须都是字符串类型。
func WithContextErr(ctx context.Context, key, value string) context.Context {
	kv := []string{key, value}
	return context.WithValue(ctx, contextValuesKey{}, kv)
}

// WithContextErrs 批量将键值对存储到 context 中。
func WithContextErrs(ctx context.Context, kvs ...string) context.Context {
	if len(kvs)%2 != 0 {
		panic("errors.WithContextErrs: kvs must be key-value pairs")
	}
	kv := make([]string, 0, len(kvs))
	for i := 0; i < len(kvs); i += 2 {
		kv = append(kv, kvs[i], kvs[i+1])
	}
	return context.WithValue(ctx, contextValuesKey{}, kv)
}

// contextError 包装错误并携带 context 中的关键键值对。
type contextError struct {
	err    error
	values []string
}

func (e *contextError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *contextError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *contextError) Is(target error) bool {
	te, ok := target.(*contextError)
	return ok && e == te
}

func (e *contextError) String() string   { return TraceString(e) }
func (e *contextError) GoString() string { return TraceJSON(e) }

func (e *contextError) Format(s fmt.State, verb rune) {
	formatError(s, verb, e)
}

func (e *contextError) MarshalJSON() ([]byte, error) {
	return []byte(TraceJSON(e)), nil
}

func (e *contextError) MarshalText() ([]byte, error) {
	return []byte(TraceString(e)), nil
}

// Is 语义与标准库 errors.Is 一致。
func Is(err, target error) bool { return errors.Is(err, target) }

// As 语义与标准库 errors.As 一致。
func As(err error, target any) bool { return errors.As(err, target) }

// Type 从错误链中提取第一个类型为 T 的错误。
func Type[T error](err error) (T, bool) {
	var target T
	if err == nil {
		return target, false
	}
	if errors.As(err, &target) {
		return target, true
	}
	return target, false
}

// HasMsg 检查错误链中是否包含指定的消息。
// 消息比较是精确匹配，用于快速判断业务错误类型。
func HasMsg(err error, msg string) bool {
	if err == nil || msg == "" {
		return false
	}
	return err.Error() == msg
}

// Unwrap 语义与标准库 errors.Unwrap 一致。
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// messageError 只附加消息，不采集栈。
type messageError struct {
	msg string
	err error
}

func (e *messageError) Error() string {
	if e == nil {
		return ""
	}
	return e.msg
}

func (e *messageError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *messageError) Is(target error) bool {
	te, ok := target.(*messageError)
	return ok && e == te
}

func (e *messageError) String() string   { return TraceString(e) }
func (e *messageError) GoString() string { return TraceJSON(e) }

func (e *messageError) Format(s fmt.State, verb rune) {
	formatError(s, verb, e)
}

func (e *messageError) MarshalJSON() ([]byte, error) {
	return []byte(TraceJSON(e)), nil
}

func (e *messageError) MarshalText() ([]byte, error) {
	return []byte(TraceString(e)), nil
}
