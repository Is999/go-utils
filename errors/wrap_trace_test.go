package errors_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/Is999/go-utils/errors"
)

// TestWrapCaller 验证只保留一帧时，所有创建入口仍指向调用方，而不是库内转发函数。
func TestWrapCaller(t *testing.T) {
	// 单帧能暴露被默认深度掩盖的 skip 偏移。
	depth := errors.StackDepth()
	t.Cleanup(func() { errors.SetStackDepth(depth) })
	errors.SetStackDepth(1)
	for _, name := range []string{"New", "Errorf", "Tag", "Wrap", "WrapMessage", "Wrapf"} {
		t.Run(name, func(t *testing.T) {
			var err error // 每个分支都在当前测试函数中触发采集。
			switch name {
			case "New":
				err = errors.New("read failed")
			case "Errorf":
				err = errors.Errorf("read %s failed", "file")
			case "Tag":
				err = errors.Tag(io.EOF)
			case "Wrap":
				err = errors.Wrap(io.EOF)
			case "WrapMessage":
				err = errors.Wrap(io.EOF, "read failed")
			case "Wrapf":
				err = errors.Wrapf(io.EOF, "read %s failed", "file")
			}
			for _, output := range []string{errors.TraceString(err), errors.TraceJSON(err)} {
				if !strings.Contains(output, "errors/wrap_trace_test.go:") {
					t.Errorf("trace lost caller: %s", output)
				}
			}
		})
	}
}

// TestWrapMessageContract 固定可变参数、原错误身份和标准库展开行为，避免修栈位置时改变包装语义。
func TestWrapMessageContract(t *testing.T) {
	// 无消息的重复包装必须保持对象身份；显式空消息仍创建节点。
	tracked := errors.Wrap(io.EOF, "read")
	if got := errors.Wrap(tracked); got != tracked {
		t.Fatal("Wrap without message replaced a tracked error")
	}
	for _, msgs := range [][]string{{""}, {"outer"}, {"outer", "ignored"}} {
		wrapped := errors.Wrap(tracked, msgs...)
		if errors.Unwrap(wrapped) != tracked || !errors.Is(wrapped, io.EOF) {
			t.Fatal("Wrap changed the cause chain")
		}
		want := "read -> EOF" // 空消息不得添加分隔符。
		if msgs[0] != "" {
			want = "outer -> " + want
		}
		if got := wrapped.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	}
}

// TestTraceJoinCarrierContext 覆盖错误码包住上下文再包住 Join 的实际日志路径。
func TestTraceJoinCarrierContext(t *testing.T) {
	ctx := errors.WithContextErr(t.Context(), "request_id", "req-42")
	err := errors.WithCode(errors.WithContext(ctx, errors.Join(io.EOF, io.ErrClosedPipe)), 500)
	// 两种结构化输出都应在扁平化后保留上下文。
	var buf bytes.Buffer
	slog.New(slog.NewJSONHandler(&buf, nil)).Error("failed", "error", errors.Trace(err))
	for _, output := range []string{errors.TraceJSON(err), buf.String()} {
		if !json.Valid([]byte(output)) || !strings.Contains(output, `"ctx":{"request_id":"req-42"}`) {
			t.Errorf("trace lost context: %s", output)
		}
	}
}

// TestTraceJoinChildStack 验证 Join 子链在两种 slog handler 中保留可读位置和追踪开关语义。
func TestTraceJoinChildStack(t *testing.T) {
	depth, enabled := errors.StackDepth(), errors.TraceEnabled()
	t.Cleanup(func() {
		errors.SetStackDepth(depth)
		errors.SetTraceEnabled(enabled)
	})
	errors.SetStackDepth(1)
	ctx := errors.WithContextErr(t.Context(), "request_id", "req-42")
	source := errors.New("read failed")
	wrapped := errors.WithCode(errors.WithContext(ctx, errors.Wrap(source, "repository failed")), 500)
	for _, state := range []struct {
		name    string
		enabled bool
	}{{"enabled", true}, {"disabled", false}} {
		errors.SetTraceEnabled(state.enabled)
		for _, tc := range []struct {
			name string
			err  error
		}{
			{"plain", errors.Join(source)},
			{"wrapped", errors.Join(wrapped, io.EOF)},
			{"nested", errors.Join(errors.WithMessage(errors.Join(wrapped, io.EOF), "batch failed"), io.ErrClosedPipe)},
		} {
			t.Run(state.name+"/"+tc.name, func(t *testing.T) {
				wantJSON := errors.TraceJSON(tc.err)
				var want any
				if err := json.Unmarshal([]byte(wantJSON), &want); err != nil {
					t.Fatal(err)
				}
				// 对照同一错误的 TraceJSON，保留子链结构，并确保 trace 元素是字符串而不是 PC 数字。
				if strings.Contains(wantJSON, `"trace":["`) != state.enabled {
					t.Fatalf("unexpected reference trace: %s", wantJSON)
				}
				var buf bytes.Buffer
				slog.New(slog.NewJSONHandler(&buf, nil)).Error("failed", "error", errors.Trace(tc.err))
				var got struct {
					Error any `json:"error"`
				}
				if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got.Error, want) {
					t.Errorf("slog child differs from TraceJSON:\n got %s\nwant %s", buf.String(), wantJSON)
				}

				buf.Reset()
				slog.New(slog.NewTextHandler(&buf, nil)).Error("failed", "error", errors.Trace(tc.err))
				if strings.Contains(buf.String(), "errors/wrap_trace_test.go:") != state.enabled {
					t.Errorf("text trace does not match enabled=%v: %s", state.enabled, buf.String())
				}
				if !state.enabled && strings.Contains(buf.String(), "trace:") {
					t.Errorf("disabled text trace retained frames: %s", buf.String())
				}
			})
		}
	}
}

// TestTraceJoinDepthLimit 保留既有截断形状：JSON 使用消息字符串，slog 数组使用带消息的对象。
func TestTraceJoinDepthLimit(t *testing.T) {
	const message = "read failed at source"
	err := errors.New(message)
	for range 1024 { // 达到现有追踪深度上限，剩余错误只能通过 Error() 展示。
		err = errors.WithCode(err, 42)
	}
	joined := errors.Join(err)

	var reference struct {
		Errors []any `json:"errs"` // Join 子链使用对象数组，截断处由 TraceJSON 写成字符串。
	}
	if err := json.Unmarshal([]byte(errors.TraceJSON(joined)), &reference); err != nil {
		t.Fatal(err)
	}
	terminal := reference.Errors[0]
	for {
		node, ok := terminal.(map[string]any)
		if !ok {
			break
		}
		terminal = node["err"]
	}
	if terminal != message {
		t.Fatalf("TraceJSON truncated message = %v, want %q", terminal, message)
	}

	// Join 数组已由 LogValue 展开，直接检查对象即可验证 handler 实际消费的内容。
	children := errors.Trace(joined).LogValue().Group()[0].Value.Any().([]any)
	node := children[0].(map[string]any)
	for node["err"] != nil {
		node = node["err"].(map[string]any)
	}
	if got := node["msg"]; got != message {
		t.Fatalf("slog truncated message = %v, want %q", got, message)
	}
}

// traceProbe 记录 Error 调用次数，区分日志级别过滤与实际错误渲染。
type traceProbe struct {
	calls int // 仅由当前测试中的同步 handler 读写。
}

// Error 只在日志消费消息时计数，构造追踪值时不应调用。
func (e *traceProbe) Error() string {
	e.calls++
	return "read failed"
}

// TestTraceFilteredJoin 验证被日志级别过滤的 Join 不触发延迟渲染。
func TestTraceFilteredJoin(t *testing.T) {
	probe := &traceProbe{}
	err := errors.Join(probe, errors.New("write failed"))
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))
	logger.Debug("filtered", "error", errors.Trace(err))
	if probe.calls != 0 || buf.Len() != 0 {
		t.Fatalf("filtered log rendered error: calls=%d, output=%s", probe.calls, buf.String())
	}
	logger.Error("visible", "error", errors.Trace(err))
	if probe.calls == 0 || buf.Len() == 0 {
		t.Fatal("visible log did not render error")
	}
}

// BenchmarkTraceContext 衡量 slog 消费上下文字段时的分配，不计 handler 和磁盘输出。
func BenchmarkTraceContext(b *testing.B) {
	ctx := errors.WithContextErrs(b.Context(), "request_id", "req-42", "operation", "read")
	trace := errors.Trace(errors.WithContext(ctx, io.EOF))
	b.ReportAllocs()
	for b.Loop() {
		_ = trace.LogValue()
	}
}
