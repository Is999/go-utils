package errors_test

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestType(t *testing.T) {
	source := &typedError{msg: "typed"}
	err := errors.Wrapf(fmt.Errorf("standard wrap: %w", source), "custom wrap")

	got, ok := errors.Type[*typedError](err)
	if !ok {
		t.Fatal("Type() should find typed source error")
	}
	if got != source {
		t.Fatalf("Type() = %v, want %v", got, source)
	}
}

func TestTraceJSONIsValidJSON(t *testing.T) {
	err := errors.Wrap(errors.Wrap(io.EOF, "inner"), "outer")

	trace := errors.TraceJSON(err)
	if !json.Valid([]byte(trace)) {
		t.Fatalf("TraceJSON() returned invalid JSON: %s", trace)
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
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
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
		}()
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
