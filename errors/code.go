package errors

import (
	"errors"
	"fmt"
)

// Coder 提供调用方定义的业务错误码，也可通过错误的 As 方法对外暴露。
type Coder interface {
	// Code 返回业务码，0 也可以是有效值。
	Code() int
}

// WithCode 为错误附加业务错误码，不采集调用栈；err 为 nil 时返回 nil。
// 多次包装时各层保留自己的码，Code 返回从外向内找到的第一个码。
func WithCode(err error, code int) error {
	if err == nil {
		return nil
	}
	return &codeError{code: code, err: err}
}

// Code 从外向内提取第一个 Coder，Join 分支按从左到右的深度优先顺序查找。
// 同时支持第三方错误通过 As 暴露的 Coder；未找到时返回 (0, false)。
func Code(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	// Coder 不要求实现 error，且第三方 As 接收 *Coder，因此保留 errors.As 的目标类型。
	var coder Coder
	if errors.As(err, &coder) {
		return coder.Code(), true
	}
	return 0, false
}

// HasCode 检查各层及 Join 分支中的业务码，外层码不同也继续向内查找。
func HasCode(err error, code int) bool {
	if err == nil {
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

		switch e := current.(type) {
		case Coder:
			if e.Code() == code {
				return true
			}
		case interface{ As(any) bool }:
			// 仅自定义 As 需要指针目标，避免普通节点的堆分配。
			var coder Coder
			// 标准库 As 继续查找原因，保留外层遍历预算之外的匹配结果。
			if errors.As(current, &coder) && coder.Code() == code {
				return true
			}
		}

		children, next := unwrapNode(current)
		switch {
		case len(children) > 0:
			stack = pushChildren(stack, children)
		case next != nil:
			stack = append(stack, next)
		}
	}
	return false
}

// codeError 保留本层业务码，不改变底层错误的消息和展开行为。
type codeError struct {
	code int   // 业务错误码
	err  error // 被包装的底层错误
}

// Error 返回被包装错误的错误消息。
func (e *codeError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

// Unwrap 返回被包装的底层错误，支持错误链展开。
func (e *codeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Code 返回错误码。
func (e *codeError) Code() int {
	if e == nil {
		return 0
	}
	return e.code
}

// Is 只匹配同一个包装对象；按业务码比较应使用 HasCode。
func (e *codeError) Is(target error) bool {
	te, ok := target.(*codeError)
	return ok && e == te
}

// String 返回错误的文本追踪格式，等同于 TraceString。
func (e *codeError) String() string { return TraceString(e) }

// GoString 返回错误的 JSON 追踪格式，等同于 TraceJSON。
func (e *codeError) GoString() string { return TraceJSON(e) }

// Format 实现 fmt.Formatter 接口，支持格式化动词（%v/%s/%q 等）。
func (e *codeError) Format(s fmt.State, verb rune) {
	formatError(s, verb, e)
}

// MarshalJSON 实现 json.Marshaler 接口，返回 JSON 追踪格式。
func (e *codeError) MarshalJSON() ([]byte, error) {
	return []byte(TraceJSON(e)), nil
}

// MarshalText 实现 encoding.TextMarshaler 接口，返回文本追踪格式。
func (e *codeError) MarshalText() ([]byte, error) {
	return []byte(TraceString(e)), nil
}
