package utils_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"testing"

	utils "github.com/Is999/go-utils"
)

// TestLogLevels 固定与 slog 对应的级别数值，供第三方适配器直接转换。
func TestLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		level utils.LogLevel
		value int
	}{
		{name: "LevelDebug", level: utils.LevelDebug, value: -4},
		{name: "LevelInfo", level: utils.LevelInfo, value: 0},
		{name: "LevelWarn", level: utils.LevelWarn, value: 4},
		{name: "LevelError", level: utils.LevelError, value: 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.level) != tt.value {
				t.Errorf("LogLevel %s = %v, want %v", tt.name, int(tt.level), tt.value)
			}
		})
	}
}

// TestSlogLoggerSource 通过公开 Logger 方法记录日志，源码位置应指向调用方而非适配器。
func TestSlogLoggerSource(t *testing.T) {
	for _, withFields := range []bool{false, true} {
		name := "default"
		if withFields {
			name = "with fields"
		}
		t.Run(name, func(t *testing.T) {
			previous := slog.Default() // 全局替换仅限当前用例，避免影响其他日志测试。
			t.Cleanup(func() { slog.SetDefault(previous) })
			logger := utils.Log()   // 默认实例应跟随后续的 SetDefault。
			var output bytes.Buffer // 捕获实际 JSONHandler 输出，包括调用位置和属性。
			slog.SetDefault(slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelDebug,
			})))
			if withFields {
				logger = logger.With("component", "worker")
				// 子 Logger 已绑定原 Handler，切换全局默认值不应改变它的输出目标。
				slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
			}
			for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
				output.Reset()
				var writeLog func(string, ...any) // 方法值也应保留实际调用位置。
				switch level {
				case slog.LevelDebug:
					writeLog = logger.Debug
				case slog.LevelInfo:
					writeLog = logger.Info
				case slog.LevelWarn:
					writeLog = logger.Warn
				case slog.LevelError:
					writeLog = logger.Error
				}
				pc, file, line, _ := runtime.Caller(0)
				writeLog("ready", "attempt", 2, slog.String("state", "ok"), "dangling")

				var record struct {
					Source    slog.Source `json:"source"`    // Handler 实际输出的调用位置。
					Level     string      `json:"level"`     // 沿用 slog 的级别名称。
					Message   string      `json:"msg"`       // 调用方消息。
					Component string      `json:"component"` // With 绑定的字段。
					Attempt   int         `json:"attempt"`   // 普通键值对参数。
					State     string      `json:"state"`     // slog.Attr 参数。
					BadKey    string      `json:"!BADKEY"`   // 未配对参数沿用 slog 的表示方式。
				}
				if err := json.Unmarshal(output.Bytes(), &record); err != nil {
					t.Fatal(err)
				}
				if record.Source.File != file || record.Source.Line != line+1 || record.Source.Function != runtime.FuncForPC(pc).Name() {
					t.Errorf("%s source = %+v, want %s:%d", level, record.Source, file, line+1)
				}
				if record.Level != level.String() || record.Message != "ready" || record.Attempt != 2 || record.State != "ok" || record.BadKey != "dangling" {
					t.Errorf("%s changed message or attributes: %s", level, output.Bytes())
				}
				if withFields && record.Component != "worker" {
					t.Errorf("%s lost With field: %s", level, output.Bytes())
				}
			}
		})
	}
}

// TestSlogLoggerLevelFilter 保证包装方法仍由底层 Handler 决定是否输出。
func TestSlogLoggerLevelFilter(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	var output bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelError})))
	logger := utils.Log()
	for _, level := range []utils.LogLevel{utils.LevelDebug, utils.LevelInfo, utils.LevelWarn} {
		if logger.Enabled(t.Context(), level) {
			t.Errorf("Enabled(%d) = true below handler threshold", level)
		}
	}
	if !logger.Enabled(t.Context(), utils.LevelError) {
		t.Fatal("Enabled(LevelError) = false at handler threshold")
	}
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	if output.Len() != 0 {
		t.Fatalf("filtered logs reached output: %s", output.Bytes())
	}
	logger.Error("error")
	if output.Len() == 0 {
		t.Fatal("enabled error log was filtered")
	}
}

// TestLogConcurrentAccess 覆盖共享 Logger 和 With 子实例的并发输出。
func TestLogConcurrentAccess(t *testing.T) {
	if logger := utils.Log(); logger == nil {
		t.Fatal("Log() returned nil")
	}
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	var output bytes.Buffer // 标准 Handler 负责并发写入同步，等待结束后再读取。
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))

	var wg sync.WaitGroup
	for worker := range 32 {
		wg.Go(func() {
			utils.Log().With("worker", worker).Info("ready")
		})
	}
	wg.Wait()
	if count := bytes.Count(output.Bytes(), []byte{'\n'}); count != 32 {
		t.Fatalf("concurrent logs = %d, want 32", count)
	}
}
