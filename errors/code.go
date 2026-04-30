package errors

import (
	"errors"
	"fmt"
)

// ============================ 错误码接口定义 ============================

// Coder 表示携带业务错误码的错误接口。
// 实现此接口的类型可以通过 Code() 方法提供业务错误码。
//
// 使用场景：
//   - 用于业务层面的错误分类，如 10001=用户未找到、20002=余额不足
//   - 便于日志检索和监控告警配置
type Coder interface {
	// Code 返回业务错误码。
	// 错误码应该全局唯一，建议使用 5 位数字格式（xxxxx）。
	//
	// 返回值：业务错误码
	Code() int
}

// ============================ 错误码操作函数 ============================

// WithCode 为错误附加业务错误码，不采集调用栈。
// 设计的轻量级包装器，仅在错误上附加 code 信息，适合在业务入口处统一打码。
// 注意：即使多次 Wrap，错误码也不会覆盖，总是保留错误链中第一个 WithCode 设置的码。
//
// 参数说明：
//   - err：原始错误对象，nil 时返回 nil
//   - code：业务错误码，如 10001、20002
//
// 返回值：携带错误码的错误对象
func WithCode(err error, code int) error {
	if err == nil {
		return nil
	}
	return &codeError{code: code, err: err}
}

// Code 从错误链中提取第一个业务错误码。
// 沿错误链向上遍历，返回遇到的第一个 WithCode 设置的错误码。
//
// 参数说明：
//   - err：错误对象
//
// 返回值：
//   - 第一个错误码值
//   - bool 表示是否找到错误码
func Code(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	var coder Coder
	if errors.As(err, &coder) {
		return coder.Code(), true
	}
	return 0, false
}

// HasCode 检查错误链中是否包含指定的业务错误码。
// 用于快速判断错误类型，如判断是否为"余额不足"错误。
//
// 参数说明：
//   - err：错误对象
//   - code：要查找的错误码
//
// 返回值：true 表示错误链中存在指定错误码
func HasCode(err error, code int) bool {
	if err == nil {
		return false
	}
	var coder Coder
	if errors.As(err, &coder) {
		return coder.Code() == code
	}
	return false
}

// ============================ codeError 错误码包装类型 ============================

// codeError 仅附加业务错误码的轻量级错误包装类型。
// 不采集调用栈，仅携带 code 信息，适合业务入口统一打码场景。
//
// 字段说明：
//   - code：业务错误码
//   - err：被包装的底层错误
type codeError struct {
	code int   // 业务错误码
	err  error // 被包装的底层错误
}

// Error 返回被包装错误的错误消息。
//
// 返回值：底层错误的消息字符串，空错误返回空字符串
func (e *codeError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

// Unwrap 返回被包装的底层错误，支持错误链展开。
//
// 返回值：底层错误对象
func (e *codeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Code 返回错误码。
//
// 返回值：业务错误码数值
func (e *codeError) Code() int {
	if e == nil {
		return 0
	}
	return e.code
}

// Is 实现 errors.Is 接口，支持错误比较。
// 当目标错误也是 codeError 时，比较其 code 和 err 是否完全一致。
//
// 参数说明：
//   - target：目标错误对象
//
// 返回值：true 表示两个错误相等
func (e *codeError) Is(target error) bool {
	te, ok := target.(*codeError)
	return ok && e == te
}

// String 返回错误的文本追踪格式，等同于 TraceString。
//
// 返回值：错误的多行文本表示
func (e *codeError) String() string { return TraceString(e) }

// GoString 返回错误的 JSON 追踪格式，等同于 TraceJSON。
//
// 返回值：错误的 JSON 字符串表示
func (e *codeError) GoString() string { return TraceJSON(e) }

// Format 实现 fmt.Formatter 接口，支持格式化动词（%v/%s/%q 等）。
//
// 参数说明：
//   - s：格式化状态
//   - verb：格式化动词
func (e *codeError) Format(s fmt.State, verb rune) {
	formatError(s, verb, e)
}

// MarshalJSON 实现 json.Marshaler 接口，返回 JSON 追踪格式。
//
// 返回值：JSON 字节数组
func (e *codeError) MarshalJSON() ([]byte, error) {
	return []byte(TraceJSON(e)), nil
}

// MarshalText 实现 encoding.TextMarshaler 接口，返回文本追踪格式。
//
// 返回值：错误文本表示的字节数组
func (e *codeError) MarshalText() ([]byte, error) {
	return []byte(TraceString(e)), nil
}
