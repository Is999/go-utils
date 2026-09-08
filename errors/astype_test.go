package errors_test

import (
	"io"
	"testing"

	"github.com/Is999/go-utils/errors"
)

// asMissError 故意修改目标却拒绝匹配，验证返回值不会泄漏失败分支的数据。
type asMissError struct{}

func (asMissError) Error() string { return "no match" }

func (asMissError) As(target any) bool {
	if p, ok := target.(**typedError); ok {
		*p = &typedError{msg: "discarded"}
	}
	return false
}

func TestAsTypeMiss(t *testing.T) {
	for _, err := range []error{nil, io.EOF, asMissError{}, errors.Join(asMissError{}, io.EOF)} {
		if got, ok := errors.AsType[*typedError](err); got != nil || ok {
			t.Fatalf("AsType(%v) = (%v, %v), want (nil, false)", err, got, ok)
		}
	}
}
