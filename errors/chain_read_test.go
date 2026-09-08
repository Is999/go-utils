package errors_test

import (
	"io"
	"slices"
	"sync"
	"testing"

	"github.com/Is999/go-utils/errors"
)

// sharedErrors 直接返回自有数组，用于检查读取错误链时是否改写外部存储。
// nil 子项超出标准 Unwrap 约定，但过滤这些子项也不应改变调用方的数据。
type sharedErrors []error

// Error 在子项全为空时提供叶子消息，覆盖渲染回退路径。
func (e sharedErrors) Error() string { return "several failures" }

// Unwrap 始终暴露同一数组，使读取方的误写能被断言和 race 检测发现。
func (e sharedErrors) Unwrap() []error { return e }

// TestErrorReadersPreserveChildren 固定过滤 nil 后的输出和子错误顺序，并检查原始数组。
func TestErrorReadersPreserveChildren(t *testing.T) {
	for _, tc := range []struct {
		name     string
		children sharedErrors
		text     string
		json     string
		hasEOF   bool
	}{
		{"without_nil", sharedErrors{io.EOF, io.ErrClosedPipe}, "joined=[EOF | io: read/write on closed pipe]", `{"errs":[{"msg":"EOF"},{"msg":"io: read/write on closed pipe"}]}`, true},
		{"with_nil", sharedErrors{nil, io.EOF, nil, io.ErrClosedPipe, nil}, "joined=[EOF | io: read/write on closed pipe]", `{"errs":[{"msg":"EOF"},{"msg":"io: read/write on closed pipe"}]}`, true},
		{"all_nil", sharedErrors{nil, nil}, "several failures", `{"msg":"several failures"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, reader := range []string{"TraceString", "TraceJSON", "HasMsg"} {
				t.Run(reader, func(t *testing.T) {
					// 每个入口使用独立数组，确保一次读取造成的写入不会被下一次读取掩盖。
					children := slices.Clone(tc.children)
					switch reader {
					case "TraceString":
						if got := errors.TraceString(children); got != tc.text {
							t.Errorf("TraceString() = %q, want %q", got, tc.text)
						}
					case "TraceJSON":
						if got := errors.TraceJSON(children); got != tc.json {
							t.Errorf("TraceJSON() = %q, want %q", got, tc.json)
						}
					case "HasMsg":
						if got := errors.HasMsg(children, "EOF"); got != tc.hasEOF {
							t.Errorf("HasMsg() = %v, want %v", got, tc.hasEOF)
						}
					}
					if !slices.Equal(children, tc.children) {
						t.Errorf("%s changed Unwrap children: got %v, want %v", reader, []error(children), []error(tc.children))
					}
				})
			}
		})
	}
}

// TestErrorReadersConcurrent 多个日志和消息查询共享同一错误对象，读取方不能争写其子错误数组。
func TestErrorReadersConcurrent(t *testing.T) {
	children := sharedErrors{nil, io.EOF, nil, io.ErrClosedPipe}
	want := slices.Clone(children)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 20 {
				_ = errors.TraceString(children)
				_ = errors.TraceJSON(children)
				if !errors.HasMsg(children, "EOF") {
					t.Error("HasMsg() lost EOF")
				}
			}
		})
	}
	wg.Wait()
	if !slices.Equal(children, want) {
		t.Errorf("concurrent readers changed Unwrap children: got %v, want %v", []error(children), []error(want))
	}
}
