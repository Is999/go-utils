package errors_test

import (
	"io"
	"strconv"
	"testing"

	"github.com/Is999/go-utils/errors"
)

// exposedCode 只实现 Coder，验证第三方 As 不需要返回一个 error 对象。
type exposedCode int

// Code 返回通过自定义 As 暴露的业务码。
func (c exposedCode) Code() int { return int(c) }

// codeAliasError 将业务码保存在错误对象之外，模拟第三方错误的 As 扩展。
type codeAliasError struct {
	error // 底层错误继续参与标准展开，允许查询另一层业务码。
}

// As 只接受 *Coder；换成另一接口类型会破坏这一扩展契约。
func (e codeAliasError) As(target any) bool {
	if coder, ok := target.(*errors.Coder); ok {
		*coder = exposedCode(41)
		return true
	}
	return false
}

// Unwrap 保留底层业务码，供 HasCode 查询自定义 As 之外的节点。
func (e codeAliasError) Unwrap() error { return e.error }

// TestCodeCustomAs 覆盖不实现 error 的 Coder、外层优先级和多分支查找。
func TestCodeCustomAs(t *testing.T) {
	inner := errors.WithCode(io.EOF, 42)
	err := errors.Wrap(codeAliasError{inner}, "read")
	if got, ok := errors.Code(err); !ok || got != 41 {
		t.Fatalf("Code() = (%d, %v), want (41, true)", got, ok)
	}
	err = errors.Join(err, errors.WithCode(io.ErrClosedPipe, 43))
	for _, code := range []int{41, 42, 43} {
		if !errors.HasCode(err, code) {
			t.Errorf("HasCode(%d) = false", code)
		}
	}
	if errors.HasCode(err, 44) {
		t.Error("HasCode matched a missing code")
	}
}

// BenchmarkHasCodeChain 覆盖业务码在多层消息包装内的命中与未命中查询。
func BenchmarkHasCodeChain(b *testing.B) {
	for _, depth := range []int{1, 8, 64} {
		err := errors.WithCode(io.EOF, 42)
		for range depth {
			err = errors.WithMessage(err, "read")
		}
		for _, code := range []int{42, 43} {
			b.Run(strconv.Itoa(depth)+"/"+strconv.Itoa(code), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					benchBool = errors.HasCode(err, code)
				}
			})
		}
	}
}
