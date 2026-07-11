package errors_test

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Is999/go-utils/errors"
)

type typedError struct {
	msg string
}

func (e *typedError) Error() string {
	return e.msg
}

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		msg  string
	}{
		{name: "001", msg: "test error"},
		{name: "002", msg: "另一个错误"},
		{name: "003", msg: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.msg)
			if err == nil {
				t.Error("New() returned nil")
			}
			if err.Error() != tt.msg {
				t.Errorf("New() error message = %v, want %v", err.Error(), tt.msg)
			}
		})
	}
}

func TestErrorf(t *testing.T) {
	tests := []struct {
		name   string
		format string
		args   []any
		want   string
	}{
		{name: "001", format: "error: %s", args: []any{"test"}, want: "error: test"},
		{name: "002", format: "code: %d, msg: %s", args: []any{100, "error"}, want: "code: 100, msg: error"},
		{name: "003", format: "no args", args: nil, want: "no args"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.Errorf(tt.format, tt.args...)
			if err == nil {
				t.Error("Errorf() returned nil")
			}
			if err.Error() != tt.want {
				t.Errorf("Errorf() error message = %v, want %v", err.Error(), tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	wrapErr := errors.New("wrap error")

	tests := []struct {
		name    string
		err     error
		msg     []string
		wantNil bool
	}{
		{name: "001", err: nil, msg: nil, wantNil: true},
		{name: "002", err: originalErr, msg: nil, wantNil: false},
		{name: "003", err: originalErr, msg: []string{"wrapped"}, wantNil: false},
		{name: "004", err: wrapErr, msg: nil, wantNil: false},
		{name: "005", err: wrapErr, msg: []string{"double wrapped"}, wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.Wrap(tt.err, tt.msg...)
			if tt.wantNil && err != nil {
				t.Errorf("Wrap() = %v, want nil", err)
			}
			if !tt.wantNil && err == nil {
				t.Error("Wrap() returned nil, want non-nil")
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	originalErr := fmt.Errorf("original error")

	tests := []struct {
		name    string
		err     error
		format  string
		args    []any
		wantNil bool
	}{
		{name: "001", err: nil, format: "test", args: nil, wantNil: true},
		{name: "002", err: originalErr, format: "wrapped: %s", args: []any{"info"}, wantNil: false},
		{name: "003", err: originalErr, format: "code: %d", args: []any{500}, wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.Wrapf(tt.err, tt.format, tt.args...)
			if tt.wantNil && err != nil {
				t.Errorf("Wrapf() = %v, want nil", err)
			}
			if !tt.wantNil && err == nil {
				t.Error("Wrapf() returned nil, want non-nil")
			}
		})
	}
}

func TestTag(t *testing.T) {
	// 标准库错误（无追踪栈）
	stdlibErr := fmt.Errorf("stdlib error")

	// 本包错误（有追踪栈）
	trackedErr := errors.New("tracked error")

	// 第三方错误（无追踪栈）
	thirdPartyErr := &typedError{msg: "third party error"}

	tests := []struct {
		name            string
		err             error
		wantNil         bool
		shouldHaveStack bool
	}{
		{name: "001", err: nil, wantNil: true, shouldHaveStack: false},
		{name: "002", err: stdlibErr, wantNil: false, shouldHaveStack: true},
		{name: "003", err: trackedErr, wantNil: false, shouldHaveStack: true},
		{name: "004", err: thirdPartyErr, wantNil: false, shouldHaveStack: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := errors.Tag(tt.err)

			if tt.wantNil && wrapped != nil {
				t.Errorf("Tag() = %v, want nil", wrapped)
			}
			if !tt.wantNil && wrapped == nil {
				t.Error("Tag() returned nil, want non-nil")
			}

			if !tt.wantNil && wrapped != nil {
				// 验证错误消息是否正确保留
				if wrapped.Error() != tt.err.Error() {
					t.Errorf("Tag() error message = %v, want %v", wrapped.Error(), tt.err.Error())
				}

				// 验证是否包含追踪栈
				hasStack := errors.HasStack(wrapped)
				if hasStack != tt.shouldHaveStack {
					t.Errorf("Tag() hasStack = %v, want %v", hasStack, tt.shouldHaveStack)
				}

				// 验证追踪输出不为空
				trace := errors.TraceString(wrapped)
				if trace == "" {
					t.Error("Tag() should produce non-empty trace")
				}
			}
		})
	}

	// 测试重复包装已有追踪的错误不会重复创建
	t.Run("005_no_double_stack", func(t *testing.T) {
		first := errors.Tag(stdlibErr)
		second := errors.Tag(first)

		// 应该返回同一个对象，避免重复包装
		if first != second {
			t.Error("Tag() should return same object for already tracked errors")
		}
	})
}

func TestErrorMessageChainSemantics(t *testing.T) {
	baseErr := fmt.Errorf("db连接失败")

	t.Run("tag_keeps_original_message", func(t *testing.T) {
		tagged := errors.Tag(baseErr)
		if got, want := tagged.Error(), "db连接失败"; got != want {
			t.Fatalf("Tag().Error() = %q, want %q", got, want)
		}

		taggedAgain := errors.Tag(tagged)
		if got, want := taggedAgain.Error(), "db连接失败"; got != want {
			t.Fatalf("Tag(Tag(err)).Error() = %q, want %q", got, want)
		}
	})

	t.Run("wrap_builds_message_chain", func(t *testing.T) {
		wrapped := errors.Wrap(baseErr, "查询用户失败")
		if got, want := wrapped.Error(), "查询用户失败 -> db连接失败"; got != want {
			t.Fatalf("Wrap().Error() = %q, want %q", got, want)
		}
	})

	t.Run("layered_wrap_builds_full_chain", func(t *testing.T) {
		wrap1 := errors.Wrap(baseErr, "查询用户失败")
		wrap2 := errors.Wrap(wrap1, "业务处理失败")
		if got, want := wrap2.Error(), "业务处理失败 -> 查询用户失败 -> db连接失败"; got != want {
			t.Fatalf("layered Wrap().Error() = %q, want %q", got, want)
		}

		tagged := errors.Tag(wrap2)
		if got, want := tagged.Error(), "业务处理失败 -> 查询用户失败 -> db连接失败"; got != want {
			t.Fatalf("Tag(Wrap(Wrap(err))).Error() = %q, want %q", got, want)
		}

		taggedAgain := errors.Tag(tagged)
		if got, want := taggedAgain.Error(), "业务处理失败 -> 查询用户失败 -> db连接失败"; got != want {
			t.Fatalf("Tag(Tag(Wrap(Wrap(err)))).Error() = %q, want %q", got, want)
		}
	})

	t.Run("wrapf_builds_message_chain", func(t *testing.T) {
		wrapped := errors.Wrapf(baseErr, "查询用户%d失败", 123)
		if got, want := wrapped.Error(), "查询用户123失败 -> db连接失败"; got != want {
			t.Fatalf("Wrapf().Error() = %q, want %q", got, want)
		}
	})
}

func TestAs(t *testing.T) {
	err := errors.New("test error")
	wrapErr := errors.Wrap(err, "wrapped")

	// 测试 As 函数
	t.Run("001", func(t *testing.T) {
		var target interface{ Error() string }
		if !errors.As(err, &target) {
			t.Error("As() should return true for wrapError")
		}
	})

	t.Run("002", func(t *testing.T) {
		var target interface{ Error() string }
		if !errors.As(wrapErr, &target) {
			t.Error("As() should return true for wrapped wrapError")
		}
	})
}

func TestTrace(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "001", err: errors.New("test error")},
		{name: "002", err: errors.Wrap(fmt.Errorf("original"), "wrapped")},
		{name: "003", err: fmt.Errorf("standard error")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trace := errors.Trace(tt.err)
			if trace == nil {
				t.Error("Trace() returned nil")
			}
			// 验证实现了 slog.LogValuer 接口
			var _ slog.LogValuer = trace
		})
	}
}

func TestIs(t *testing.T) {
	err := fmt.Errorf("原始测试错误")
	err2 := errors.New("原始测试错误")
	type args struct {
		err    error
		target error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "001", args: args{
			err:    errors.Wrapf(io.EOF, "包装测试错误"),
			target: io.EOF,
		}, want: true},
		{name: "002", args: args{
			err:    errors.Wrapf(fmt.Errorf("原始测试错误"), "包装测试错误"),
			target: io.EOF,
		}, want: false},
		{name: "003", args: args{
			err:    errors.Wrapf(fmt.Errorf("原始测试错误"), "包装测试错误"),
			target: fmt.Errorf("原始测试错误"),
		}, want: false},
		{name: "004", args: args{
			err:    errors.Wrapf(err, "包装测试错误"),
			target: err,
		}, want: true},
		{name: "005", args: args{
			err:    errors.Wrapf(errors.New("原始测试错误"), "包装测试错误"),
			target: err,
		}, want: false},
		{name: "006", args: args{
			err:    errors.Wrapf(errors.New("原始测试错误"), "包装测试错误"),
			target: errors.New("原始测试错误"),
		}, want: false},
		{name: "007", args: args{
			err:    errors.Wrapf(err2, "包装测试错误"),
			target: err2,
		}, want: true},
		{name: "008", args: args{
			err:    err2,
			target: err2,
		}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.args.err, tt.args.target); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrappingPreservesSentinelForIs(t *testing.T) {
	thirdPartyNil := stderrors.New("redis: nil")
	tests := []struct {
		name   string
		source error
	}{
		{name: "stdlib eof", source: io.EOF},
		{name: "third party sentinel", source: thirdPartyNil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := errors.WithContextErr(context.Background(), "request_id", "r-1")
			err := errors.Wrap(tt.source, "repository read failed")
			err = errors.WithMessage(err, "service failed")
			err = errors.WithCode(err, 50001)
			err = errors.WithContext(ctx, err)
			err = errors.Wrapf(err, "handler %s", "failed")
			err = errors.Tag(err)

			if !errors.Is(err, tt.source) {
				t.Fatalf("errors.Is() should match source sentinel %v after wrapping", tt.source)
			}
			if !stderrors.Is(err, tt.source) {
				t.Fatalf("stdlib errors.Is() should match source sentinel %v after wrapping", tt.source)
			}
			if !errors.HasCode(err, 50001) {
				t.Fatal("HasCode() should keep outer business code")
			}
			if !errors.HasMsg(err, "handler failed") || !errors.HasMsg(err, "repository read failed") {
				t.Fatalf("HasMsg() should traverse wrapped messages, got %s", errors.TraceString(err))
			}
		})
	}
}

func TestWithContextKeepsValuesAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(errors.WithContextErr(context.Background(), "request_id", "r-1"))
	cancel()

	trace := errors.TraceJSON(errors.WithContext(ctx, io.EOF))
	if !strings.Contains(trace, `"request_id":"r-1"`) {
		t.Fatalf("WithContext() lost values after cancellation: %s", trace)
	}
}

func TestJoinWrappingPreservesAllSentinelsForIs(t *testing.T) {
	thirdPartyNil := stderrors.New("redis: nil")
	err := errors.WithCode(
		errors.WithMessage(
			errors.Join(
				errors.Wrap(io.EOF, "profile read failed"),
				errors.Wrap(thirdPartyNil, "cache read failed"),
			),
			"submit failed",
		),
		42201,
	)

	for _, target := range []error{io.EOF, thirdPartyNil} {
		if !errors.Is(err, target) {
			t.Fatalf("errors.Is() should match joined source %v", target)
		}
		if !stderrors.Is(err, target) {
			t.Fatalf("stdlib errors.Is() should match joined source %v", target)
		}
	}
}

func TestUnwrap(t *testing.T) {
	err := fmt.Errorf("原始测试错误")
	type args struct {
		err error
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{name: "001", args: args{err: errors.Wrapf(io.EOF, "包装测试错误")}, wantErr: io.EOF},
		{name: "002", args: args{err: errors.Wrapf(err, "包装测试错误")}, wantErr: err},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := errors.Unwrap(tt.args.err); !errors.Is(err, tt.wantErr) {
				t.Errorf("Unwrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSourceSourcesAndChain(t *testing.T) {
	left := errors.Wrap(io.EOF, "left")
	rightSource := &typedError{msg: "right"}
	right := errors.Wrap(rightSource, "right")
	joined := errors.Join(left, nil, right)
	err := errors.Wrap(joined, "top")

	if !errors.Is(err, io.EOF) {
		t.Fatal("Is() should match source in joined chain")
	}
	if got := errors.Source(err); !errors.Is(got, io.EOF) {
		t.Fatalf("Source() = %v, want %v", got, io.EOF)
	}
	if got := errors.Cause(err); !errors.Is(got, io.EOF) {
		t.Fatalf("Cause() = %v, want %v", got, io.EOF)
	}
	if got := errors.Root(err); !errors.Is(got, io.EOF) {
		t.Fatalf("Root() = %v, want %v", got, io.EOF)
	}

	sources := errors.Sources(err)
	if len(sources) != 2 {
		t.Fatalf("Sources() len = %d, want 2: %#v", len(sources), sources)
	}
	if !errors.Is(sources[0], io.EOF) {
		t.Fatalf("Sources()[0] = %v, want %v", sources[0], io.EOF)
	}
	if sources[1] != rightSource {
		t.Fatalf("Sources()[1] = %v, want %v", sources[1], rightSource)
	}

	chain := errors.Chain(err)
	if len(chain) < 6 {
		t.Fatalf("Chain() len = %d, want at least 6", len(chain))
	}
	if chain[0] != err {
		t.Fatalf("Chain()[0] = %v, want original err", chain[0])
	}
}

func TestAsType(t *testing.T) {
	source := &typedError{msg: "typed"}
	err := errors.Wrapf(fmt.Errorf("standard wrap: %w", source), "custom wrap")

	got, ok := errors.AsType[*typedError](err)
	if !ok {
		t.Fatal("AsType() should find typed source error")
	}
	if got != source {
		t.Fatalf("AsType() = %v, want %v", got, source)
	}
}

func TestAsTypeDepthFirst(t *testing.T) {
	left := &typedError{msg: "left"}
	right := &typedError{msg: "right"}
	err := errors.Join(errors.Wrap(left, "left wrap"), errors.Wrap(right, "right wrap"))

	got, ok := errors.AsType[*typedError](err)
	if !ok {
		t.Fatal("AsType() should find typed error in joined chain")
	}
	if got != left {
		t.Fatalf("AsType() = %v, want first depth-first match %v", got, left)
	}
	stdGot, stdOK := stderrors.AsType[*typedError](err)
	if stdOK != ok || stdGot != got {
		t.Fatalf("AsType() = (%v, %v), std errors.AsType() = (%v, %v)", got, ok, stdGot, stdOK)
	}
}

func TestTraceJSONIsValidJSON(t *testing.T) {
	err := errors.Wrap(errors.Wrap(io.EOF, "inner"), "outer")

	trace := errors.TraceJSON(err)
	if !json.Valid([]byte(trace)) {
		t.Fatalf("TraceJSON() returned invalid JSON: %s", trace)
	}
}

func TestTraceJSONEscapesControlAndInvalidUTF8(t *testing.T) {
	// msg 是模拟外部系统返回的异常消息，包含控制字符和非法 UTF-8，TraceJSON 必须仍保持可解析。
	msg := "outer\x00line\n" + string([]byte{0xff})
	// ctx 带入请求上下文字段，覆盖 ctx key/value 与 msg 使用同一转义入口的边界。
	ctx := errors.WithContextErr(context.Background(), "request\x00id", "value"+string([]byte{0xff}))
	err := errors.WithContext(ctx, errors.New(msg))

	trace := errors.TraceJSON(err)
	if !json.Valid([]byte(trace)) {
		t.Fatalf("TraceJSON() returned invalid JSON: %s", trace)
	}
}

func TestTraceOutputKeepsWrappedMetadataAndSource(t *testing.T) {
	old := errors.TraceEnabled()
	defer errors.SetTraceEnabled(old)
	errors.SetTraceEnabled(false)

	ctx := errors.WithContextErr(context.Background(), "request_id", "r-1")
	err := errors.WithContext(ctx, errors.WithCode(errors.WithMessage(errors.Wrap(io.EOF, "read failed"), "service failed"), 50001))

	if !errors.Is(err, io.EOF) {
		t.Fatal("Is() should match io.EOF before trace rendering")
	}

	traceText := errors.TraceString(err)
	for _, want := range []string{"code=50001", "service failed", "read failed", "EOF"} {
		if !strings.Contains(traceText, want) {
			t.Fatalf("TraceString() missing %q, got %s", want, traceText)
		}
	}

	traceJSON := errors.TraceJSON(err)
	if !json.Valid([]byte(traceJSON)) {
		t.Fatalf("TraceJSON() returned invalid JSON: %s", traceJSON)
	}
	for _, want := range []string{`"code":50001`, `"request_id":"r-1"`, `"msg":"service failed"`, `"msg":"read failed"`, `"msg":"EOF"`} {
		if !strings.Contains(traceJSON, want) {
			t.Fatalf("TraceJSON() missing %q, got %s", want, traceJSON)
		}
	}

	if !errors.Is(err, io.EOF) {
		t.Fatal("Is() should still match io.EOF after trace rendering")
	}
}

func TestWrappedErrorsFormatAndMarshal(t *testing.T) {
	ctx := errors.WithContextErr(context.Background(), "request_id", "r-1")
	wrapped := []error{
		errors.New("stack failed"),
		errors.WithMessagef(io.EOF, "message %s", "failed"),
		errors.WithContext(ctx, io.EOF),
	}

	for _, err := range wrapped {
		if err == nil {
			t.Fatal("wrapped error = nil")
		}
		if strings.TrimSpace(err.Error()) == "" {
			t.Fatalf("Error() returned empty output for %T", err)
		}
		if strings.TrimSpace(fmt.Sprintf("%v", err)) == "" {
			t.Fatalf("fmt %%v returned empty output for %T", err)
		}
		if strings.TrimSpace(fmt.Sprintf("%+v", err)) == "" {
			t.Fatalf("fmt %%+v returned empty output for %T", err)
		}
		if strings.TrimSpace(fmt.Sprintf("%#v", err)) == "" {
			t.Fatalf("fmt %%#v returned empty output for %T", err)
		}
		if stringer, ok := err.(interface{ String() string }); ok && stringer.String() == "" {
			t.Fatalf("String() returned empty output for %T", err)
		}
		if goStringer, ok := err.(interface{ GoString() string }); ok && goStringer.GoString() == "" {
			t.Fatalf("GoString() returned empty output for %T", err)
		}
		if data, marshalErr := json.Marshal(err); marshalErr != nil || !json.Valid(data) {
			t.Fatalf("json.Marshal(%T) = %q, err=%v", err, data, marshalErr)
		}
		if textMarshaler, ok := err.(interface{ MarshalText() ([]byte, error) }); ok {
			text, marshalErr := textMarshaler.MarshalText()
			if marshalErr != nil || len(text) == 0 {
				t.Fatalf("MarshalText(%T) = %q, err=%v", err, text, marshalErr)
			}
		}
	}

	if !errors.Is(wrapped[1], io.EOF) || !errors.Is(wrapped[2], io.EOF) {
		t.Fatal("wrapped formatted errors should still match io.EOF")
	}
	if !errors.HasMsg(wrapped[1], "message failed") {
		t.Fatal("WithMessagef() message should be visible in chain")
	}
}

func TestWithCode(t *testing.T) {
	err := errors.WithCode(errors.Wrap(io.EOF, "read failed"), 409)

	code, ok := errors.Code(err)
	if !ok {
		t.Fatal("Code() should find code")
	}
	if code != 409 {
		t.Fatalf("Code() = %d, want 409", code)
	}
	if !errors.Is(err, io.EOF) {
		t.Fatal("Is() should still match wrapped source")
	}

	var coder errors.Coder
	if !errors.As(err, &coder) {
		t.Fatal("As() should match Coder")
	}
	if coder.Code() != 409 {
		t.Fatalf("Coder.Code() = %d, want 409", coder.Code())
	}

	trace := errors.TraceJSON(err)
	if !strings.Contains(trace, `"code":409`) {
		t.Fatalf("TraceJSON() should contain code, got %s", trace)
	}
}

// TestHasCodeTraversesWholeChain 验证业务码查询覆盖单链和 Join 的全部节点。
func TestHasCodeTraversesWholeChain(t *testing.T) {
	nested := errors.WithCode(errors.WithCode(io.EOF, 40002), 40001)
	if !errors.HasCode(nested, 40002) {
		t.Fatal("HasCode() should match code in the inner error")
	}

	err := errors.Join(
		errors.WithCode(io.EOF, 40001),
		errors.WithCode(io.ErrUnexpectedEOF, 40002),
	)

	if !errors.HasCode(err, 40002) {
		t.Fatal("HasCode() should match code in the second joined branch")
	}
}

func TestHasMsgTraversesChain(t *testing.T) {
	err := errors.Wrap(errors.New("inner"), "outer")
	if !errors.HasMsg(err, "outer") {
		t.Fatal("HasMsg() should match outer message")
	}
	if !errors.HasMsg(err, "inner") {
		t.Fatal("HasMsg() should match inner message")
	}
	if errors.HasMsg(err, "missing") {
		t.Fatal("HasMsg() should not match missing message")
	}
}

func TestTraceRetainsOuterLightweightWrapper(t *testing.T) {
	err := errors.Wrap(errors.New("inner"), "outer")

	gotText := errors.TraceString(err)
	if !strings.Contains(gotText, "outer") || !strings.Contains(gotText, "inner") {
		t.Fatalf("TraceString() should contain outer and inner messages, got %s", gotText)
	}

	gotJSON := errors.TraceJSON(err)
	if !strings.Contains(gotJSON, `"msg":"outer"`) || !strings.Contains(gotJSON, `"msg":"inner"`) {
		t.Fatalf("TraceJSON() should contain outer and inner messages, got %s", gotJSON)
	}

	gotFmt := fmt.Sprintf("%+v", err)
	if !strings.Contains(gotFmt, "outer") || !strings.Contains(gotFmt, "inner") {
		t.Fatalf("fmt %%+v should contain outer and inner messages, got %s", gotFmt)
	}
	if !json.Valid([]byte(gotFmt)) {
		t.Fatalf("fmt %%+v should return JSON trace, got %s", gotFmt)
	}
}

func TestTraceRetainsOuterStdlibWrapper(t *testing.T) {
	err := fmt.Errorf("stdlib outer: %w", errors.New("inner"))

	gotText := errors.TraceString(err)
	if !strings.Contains(gotText, "stdlib outer") || !strings.Contains(gotText, "inner") {
		t.Fatalf("TraceString() should contain stdlib outer and inner messages, got %s", gotText)
	}

	gotJSON := errors.TraceJSON(err)
	if !json.Valid([]byte(gotJSON)) {
		t.Fatalf("TraceJSON() returned invalid JSON: %s", gotJSON)
	}
	if !strings.Contains(gotJSON, "stdlib outer") || !strings.Contains(gotJSON, `"msg":"inner"`) {
		t.Fatalf("TraceJSON() should contain stdlib outer and nested inner messages, got %s", gotJSON)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	logger.Error("stdlib failed", "trace", errors.Trace(err))
	logLine := buf.String()
	if !strings.Contains(logLine, "stdlib outer") || !strings.Contains(logLine, "inner") {
		t.Fatalf("Trace() slog output should contain stdlib outer and inner messages, got %s", logLine)
	}
}

func TestSetStackDepth(t *testing.T) {
	old := errors.StackDepth()
	defer errors.SetStackDepth(old)

	if got := errors.SetStackDepth(0); got != 1 {
		t.Fatalf("SetStackDepth(0) = %d, want 1", got)
	}
	if got := errors.StackDepth(); got != 1 {
		t.Fatalf("StackDepth() = %d, want 1", got)
	}

	err := errors.New("depth test")
	var trace struct {
		Trace []string `json:"trace"`
	}
	if decodeErr := json.Unmarshal([]byte(errors.TraceJSON(err)), &trace); decodeErr != nil {
		t.Fatalf("TraceJSON() unmarshal error = %v", decodeErr)
	}
	if len(trace.Trace) != 1 {
		t.Fatalf("trace depth = %d, want 1", len(trace.Trace))
	}
}

func TestSetTraceEnabled(t *testing.T) {
	old := errors.TraceEnabled()
	defer errors.SetTraceEnabled(old)

	errors.SetTraceEnabled(false)

	err := errors.WithCode(errors.Wrap(io.EOF, "read failed"), 40001)

	traceText := errors.TraceString(err)
	if strings.Contains(traceText, "errors/errors_test.go:") {
		t.Fatalf("TraceString() should not contain trace location when disabled, got %s", traceText)
	}
	if !strings.Contains(traceText, "code=40001") || !strings.Contains(traceText, "read failed") {
		t.Fatalf("TraceString() should keep message chain when disabled, got %s", traceText)
	}

	traceJSON := errors.TraceJSON(err)
	if strings.Contains(traceJSON, `"trace"`) {
		t.Fatalf("TraceJSON() should not contain trace field when disabled, got %s", traceJSON)
	}
	if !strings.Contains(traceJSON, `"code":40001`) || !strings.Contains(traceJSON, `"msg":"read failed"`) {
		t.Fatalf("TraceJSON() should keep code and message when disabled, got %s", traceJSON)
	}
}

func TestLayeredErrorFinalLogOutput(t *testing.T) {
	t.Helper()

	level3 := func() error {
		return errors.New("db query failed")
	}
	level2 := func() error {
		if err := level3(); err != nil {
			return errors.Wrap(err, "load user failed")
		}
		return nil
	}
	level1 := func() error {
		if err := level2(); err != nil {
			return errors.WithCode(errors.WithMessage(err, "handle request failed"), 50001)
		}
		return nil
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	err := level1()
	if err == nil {
		t.Fatal("level1() returned nil")
	}
	logger.Error("request failed", "trace", errors.Trace(err))

	logLine := strings.TrimSpace(buf.String())
	t.Log(logLine)

	if !strings.Contains(logLine, `"msg":"request failed"`) {
		t.Fatalf("log should contain top log message, got %s", logLine)
	}
	if !strings.Contains(logLine, `"code":50001`) {
		t.Fatalf("log should contain error code, got %s", logLine)
	}
	if !strings.Contains(logLine, `handle request failed`) ||
		!strings.Contains(logLine, `load user failed`) ||
		!strings.Contains(logLine, `db query failed`) {
		t.Fatalf("log should contain full business error chain, got %s", logLine)
	}
	if !strings.Contains(logLine, `errors/errors_test.go:`) {
		t.Fatalf("log should contain project-root-relative file path, got %s", logLine)
	}
	if strings.Contains(logLine, `/Users/`) {
		t.Fatalf("log should not contain absolute file path, got %s", logLine)
	}
	if strings.Contains(logLine, `testing/testing.go`) || strings.Contains(logLine, `runtime/`) {
		t.Fatalf("log should not contain extra runtime/testing chain, got %s", logLine)
	}
}

func TestRecommendedUsageWithTrace(t *testing.T) {
	t.Helper()

	// 推荐模式：
	// 1. 真正失败点使用 New/Wrap 抓取栈。
	// 2. 中间传播层使用 WithMessage/WithCode 补充业务语义，避免重复抓栈。
	// 3. 出口层统一打印 Trace(err)。
	repository := func() error {
		return errors.New("select order failed")
	}
	service := func() error {
		if err := repository(); err != nil {
			return errors.Wrap(err, "load order failed")
		}
		return nil
	}
	handler := func() error {
		if err := service(); err != nil {
			return errors.WithCode(errors.WithMessage(err, "create payment failed"), 40021)
		}
		return nil
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	err := handler()
	if err == nil {
		t.Fatal("handler() returned nil")
	}
	logger.Error("checkout failed", "trace", errors.Trace(err))

	logLine := strings.TrimSpace(buf.String())
	t.Log(logLine)

	if !strings.Contains(logLine, `"msg":"checkout failed"`) {
		t.Fatalf("log should contain top message, got %s", logLine)
	}
	if !strings.Contains(logLine, `"code":40021`) {
		t.Fatalf("log should contain business code, got %s", logLine)
	}
	if !strings.Contains(logLine, `create payment failed`) ||
		!strings.Contains(logLine, `load order failed`) ||
		!strings.Contains(logLine, `select order failed`) {
		t.Fatalf("log should contain full business chain, got %s", logLine)
	}
	if !strings.Contains(logLine, `errors/errors_test.go:`) {
		t.Fatalf("log should contain relative trace path, got %s", logLine)
	}
}

func TestRecommendedUsageWithoutTrace(t *testing.T) {
	t.Helper()

	old := errors.TraceEnabled()
	defer errors.SetTraceEnabled(old)
	errors.SetTraceEnabled(false)

	repository := func() error {
		return errors.New("query cache failed")
	}
	service := func() error {
		if err := repository(); err != nil {
			return errors.WithMessage(err, "refresh profile failed")
		}
		return nil
	}
	handler := func() error {
		if err := service(); err != nil {
			return errors.WithCode(err, 30011)
		}
		return nil
	}

	err := handler()
	if err == nil {
		t.Fatal("handler() returned nil")
	}

	traceText := errors.TraceString(err)
	t.Log(traceText)

	if !strings.Contains(traceText, "code=30011") {
		t.Fatalf("TraceString() should keep code when trace disabled, got %s", traceText)
	}
	if !strings.Contains(traceText, "refresh profile failed") || !strings.Contains(traceText, "query cache failed") {
		t.Fatalf("TraceString() should keep business chain when trace disabled, got %s", traceText)
	}
	if strings.Contains(traceText, "errors/errors_test.go:") {
		t.Fatalf("TraceString() should not contain file location when trace disabled, got %s", traceText)
	}

	traceJSON := errors.TraceJSON(err)
	if strings.Contains(traceJSON, `"trace"`) {
		t.Fatalf("TraceJSON() should not contain trace field when disabled, got %s", traceJSON)
	}
}

func TestRecommendedJoinUsage(t *testing.T) {
	t.Helper()

	old := errors.TraceEnabled()
	defer errors.SetTraceEnabled(old)
	errors.SetTraceEnabled(false)

	validateProfile := func() error {
		return errors.WithMessage(io.EOF, "profile is empty")
	}
	validateAddress := func() error {
		return errors.WithMessage(io.ErrUnexpectedEOF, "address is invalid")
	}
	service := func() error {
		errs := []error{
			validateProfile(),
			validateAddress(),
		}
		return errors.WithCode(errors.WithMessage(errors.Join(errs...), "submit user failed"), 42201)
	}

	err := service()
	if err == nil {
		t.Fatal("service() returned nil")
	}

	traceText := errors.TraceString(err)
	t.Log(traceText)

	if !strings.Contains(traceText, "code=42201") || !strings.Contains(traceText, "submit user failed") {
		t.Fatalf("TraceString() should contain top level code and message, got %s", traceText)
	}
	if !strings.Contains(traceText, "joined=") {
		t.Fatalf("TraceString() should contain joined marker, got %s", traceText)
	}
	if !strings.Contains(traceText, "profile is empty") || !strings.Contains(traceText, "address is invalid") {
		t.Fatalf("TraceString() should contain all joined error messages, got %s", traceText)
	}
	if strings.Contains(traceText, "\n") {
		t.Fatalf("TraceString() should not contain joined newline noise, got %q", traceText)
	}
	if strings.Contains(traceText, "profile is empty\naddress is invalid") {
		t.Fatalf("TraceString() should not duplicate stdlib join text, got %q", traceText)
	}

	traceJSON := errors.TraceJSON(err)
	t.Log(traceJSON)
	if !strings.Contains(traceJSON, `"errs":[`) {
		t.Fatalf("TraceJSON() should use errs for joined children, got %s", traceJSON)
	}
	if !strings.Contains(traceJSON, `"code":42201,"msg":"submit user failed","errs":[`) {
		t.Fatalf("TraceJSON() should flatten code/msg/errs into current node, got %s", traceJSON)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	logger.Error("submit failed", "trace", errors.Trace(err))
	logLine := strings.TrimSpace(buf.String())
	t.Log(logLine)
	if !strings.Contains(logLine, `"errs":[`) {
		t.Fatalf("Trace() slog output should use errs for joined children, got %s", logLine)
	}
	if !strings.Contains(logLine, `"code":42201,"msg":"submit user failed","errs":[`) {
		t.Fatalf("Trace() slog output should flatten code/msg/errs into current node, got %s", logLine)
	}
	if !strings.Contains(logLine, `profile is empty`) || !strings.Contains(logLine, `address is invalid`) {
		t.Fatalf("Trace() slog output should contain joined child messages, got %s", logLine)
	}

	sources := errors.Sources(err)
	if len(sources) != 2 {
		t.Fatalf("Sources() len = %d, want 2", len(sources))
	}
}

// TestTraceJoinChildContext 验证 slog 输出保留 Join 子错误的上下文字段。
func TestTraceJoinChildContext(t *testing.T) {
	old := errors.TraceEnabled()
	defer errors.SetTraceEnabled(old)
	errors.SetTraceEnabled(false)

	ctx := errors.WithContextErr(context.Background(), "request_id", "r-1")
	err := errors.Join(errors.WithContext(ctx, io.EOF), io.ErrUnexpectedEOF)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return attr
		},
	}))
	logger.Error("joined failed", "trace", errors.Trace(err))

	logLine := strings.TrimSpace(buf.String())
	if !strings.Contains(logLine, `"ctx":{"request_id":"r-1"}`) {
		t.Fatalf("Trace() joined child should preserve context, got %s", logLine)
	}
}

func ExampleTrace_enabled() {
	repository := func() error {
		return errors.New("load config failed")
	}
	service := func() error {
		if err := repository(); err != nil {
			return errors.Wrap(err, "bootstrap service failed")
		}
		return nil
	}
	handler := func() error {
		if err := service(); err != nil {
			return errors.WithCode(errors.WithMessage(err, "start app failed"), 50010)
		}
		return nil
	}

	err := handler()
	fmt.Println(errors.Code(err))
	fmt.Println(strings.Contains(errors.TraceJSON(err), `"trace"`))
	fmt.Println(strings.Contains(errors.TraceJSON(err), `"msg":"start app failed"`))
	// Output:
	// 50010 true
	// true
	// true
}

func ExampleTrace_disabled() {
	old := errors.TraceEnabled()
	defer errors.SetTraceEnabled(old)
	errors.SetTraceEnabled(false)

	repository := func() error {
		return errors.New("query cache failed")
	}
	service := func() error {
		if err := repository(); err != nil {
			return errors.WithMessage(err, "refresh profile failed")
		}
		return nil
	}
	handler := func() error {
		if err := service(); err != nil {
			return errors.WithCode(err, 30011)
		}
		return nil
	}

	err := handler()
	fmt.Println(errors.TraceString(err))
	// Output:
	// code=30011; cause=refresh profile failed; cause=query cache failed
}

func TestConcurrentSafe(t *testing.T) {
	source := &typedError{msg: "source"}
	err := errors.Wrap(errors.Wrap(source, "inner"), "outer")

	var wg sync.WaitGroup
	var failed atomic.Bool
	for range 32 {
		wg.Go(func() {
			for range 200 {
				_ = err.Error()
				_ = fmt.Sprintf("%s", err)
				_ = fmt.Sprintf("%+v", err)
				_ = fmt.Sprintf("%#v", err)
				_ = errors.TraceString(err)
				_ = errors.TraceJSON(err)
				if !errors.Is(err, source) {
					failed.Store(true)
				}
				if errors.Source(err) != source {
					failed.Store(true)
				}
			}
		})
	}
	wg.Wait()

	if failed.Load() {
		t.Fatal("concurrent reads returned inconsistent results")
	}
}

func TestWithContextErrsSafe(t *testing.T) {
	t.Run("odd kvs should not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("WithContextErrs() should not panic, got %v", r)
			}
		}()
		ctx := errors.WithContextErrs(testNilContext(), "request_id", "r-1", "dangling")
		if ctx == nil {
			t.Fatal("WithContextErrs() returned nil context")
		}
	})

	t.Run("strict api should return error", func(t *testing.T) {
		ctx, err := errors.WithContextErrsE(context.Background(), "request_id", "r-1", "dangling")
		if err == nil {
			t.Fatal("WithContextErrsE() expected error")
		}
		if ctx == nil {
			t.Fatal("WithContextErrsE() returned nil context")
		}
	})
}

func testNilContext() context.Context {
	return nil
}

func TestErrorFormat(t *testing.T) {
	err := errors.New("test error")
	wrapErr := errors.Wrap(fmt.Errorf("original"), "wrapped")

	// 测试 %s 格式化
	s := fmt.Sprintf("%s", err)
	if s == "" {
		t.Error("Error format with s returned empty string")
	}

	// 测试 %v 格式化
	v := fmt.Sprintf("%v", err)
	if v == "" {
		t.Error("Error format with v returned empty string")
	}

	// 测试 %+v 格式化
	pv := fmt.Sprintf("%+v", err)
	if pv == "" {
		t.Error("Error format with +v returned empty string")
	}
	if !json.Valid([]byte(pv)) {
		t.Errorf("Error format with +v should return JSON, got %s", pv)
	}

	// 测试 %#v 格式化
	hv := fmt.Sprintf("%#v", wrapErr)
	if hv == "" {
		t.Error("Error format with #v returned empty string")
	}

	// 测试 %q 格式化
	q := fmt.Sprintf("%q", err)
	if q == "" {
		t.Error("Error format with q returned empty string")
	}
}

// lazyTracePayload 表示 TraceJSON 输出中与本用例相关的最小结构。
// 只解析 trace 字段，是为了验证懒裁剪后的项目栈帧数量和路径格式，不绑定完整 JSON 协议。
type lazyTracePayload struct {
	Trace []string `json:"trace"` // Trace 是 JSON 输出中的栈帧数组，元素应为项目相对路径定位。
}

// TestLazyStackProjectFrameTrimming 验证创建错误时延迟解析栈帧后，最终链路追踪输出保持原有项目裁剪效果。
// 文本、JSON、slog 三个入口都应只暴露项目内连续调用栈，不能泄露绝对路径或 runtime/testing 框架帧。
func TestLazyStackProjectFrameTrimming(t *testing.T) {
	oldDepth := errors.StackDepth()
	defer errors.SetStackDepth(oldDepth)
	errors.SetStackDepth(8)

	err := lazyStackProjectEntry()
	if err == nil {
		t.Fatal("lazyStackProjectEntry() returned nil")
	}

	// 文本入口常用于第三方日志库，必须保持首个业务位置为项目相对路径。
	traceText := errors.TraceString(err)
	assertProjectTraceNoFramework(t, traceText)

	// JSON 入口会渲染完整 trace 数组，懒裁剪后仍应保留多层项目调用链。
	traceJSON := errors.TraceJSON(err)
	assertProjectTraceNoFramework(t, traceJSON)
	var payload lazyTracePayload
	if decodeErr := json.Unmarshal([]byte(traceJSON), &payload); decodeErr != nil {
		t.Fatalf("TraceJSON() unmarshal error = %v", decodeErr)
	}
	if len(payload.Trace) < 2 {
		t.Fatalf("TraceJSON() trace length = %d, want at least 2: %s", len(payload.Trace), traceJSON)
	}
	for _, frame := range payload.Trace {
		if !strings.Contains(frame, "errors/errors_test.go:") {
			t.Fatalf("TraceJSON() frame should use project-relative test path, got %s", frame)
		}
	}

	// slog 入口会通过 stackTrace.LogValue 延迟渲染，需复用同一套项目帧裁剪规则。
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	logger.Error("lazy stack failed", "trace", errors.Trace(err))
	assertProjectTraceNoFramework(t, buf.String())
}

// assertProjectTraceNoFramework 校验追踪输出的路径边界。
// 数据来源可以是 TraceString、TraceJSON 或 slog JSON，统一要求只保留项目相对路径并过滤测试框架帧。
func assertProjectTraceNoFramework(t *testing.T, output string) {
	t.Helper()
	if !strings.Contains(output, "errors/errors_test.go:") {
		t.Fatalf("trace should contain project-relative path, got %s", output)
	}
	if strings.Contains(output, "/Users/") {
		t.Fatalf("trace should not contain absolute path, got %s", output)
	}
	if strings.Contains(output, "testing/testing.go") || strings.Contains(output, "runtime/") {
		t.Fatalf("trace should not contain runtime/testing frames, got %s", output)
	}
}

// lazyStackProjectEntry 构造多层项目内调用栈的入口。
// 通过固定的测试调用链模拟业务 handler 到 repository 的传播路径，便于验证连续项目帧裁剪。
func lazyStackProjectEntry() error {
	return lazyStackProjectMiddle()
}

// lazyStackProjectMiddle 构造多层项目内调用栈的中间层。
// 该层没有额外包装错误，确保测试关注点停留在栈帧采集和渲染边界。
func lazyStackProjectMiddle() error {
	return lazyStackProjectLeaf()
}

// lazyStackProjectLeaf 构造多层项目内调用栈的失败点。
// 这里使用 errors.New 采集栈，验证懒解析后首帧仍定位到真实业务失败位置。
func lazyStackProjectLeaf() error {
	return errors.New("lazy stack failed")
}
