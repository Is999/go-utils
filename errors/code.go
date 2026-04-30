package errors

import (
	"errors"
	"fmt"
)

// Coder 表示携带业务错误码的 error。
type Coder interface {
	Code() int
}

// WithCode 为错误附加业务错误码，不采集栈。
func WithCode(err error, code int) error {
	if err == nil {
		return nil
	}
	return &codeError{code: code, err: err}
}

// Code 从错误链中提取第一个业务错误码。
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

// codeError 只附加业务错误码，不采集栈。
type codeError struct {
	code int
	err  error
}

func (e *codeError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *codeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *codeError) Code() int {
	if e == nil {
		return 0
	}
	return e.code
}

func (e *codeError) Is(target error) bool {
	te, ok := target.(*codeError)
	return ok && e == te
}

func (e *codeError) String() string   { return TraceString(e) }
func (e *codeError) GoString() string { return TraceJSON(e) }

func (e *codeError) Format(s fmt.State, verb rune) {
	formatError(s, verb, e)
}

func (e *codeError) MarshalJSON() ([]byte, error) {
	return []byte(TraceJSON(e)), nil
}

func (e *codeError) MarshalText() ([]byte, error) {
	return []byte(TraceString(e)), nil
}
