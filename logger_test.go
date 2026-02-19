package utils_test

import (
	"context"
	"testing"

	utils "github.com/Is999/go-utils"
)

// mockLogger 模拟第三方日志库（如 zap、logrus）的实现
type mockLogger struct {
	debugCalled   bool
	infoCalled    bool
	warnCalled    bool
	errorCalled   bool
	withCalled    bool
	enabledCalled bool
	lastMsg       string
	lastArgs      []any
	lastLevel     utils.LogLevel
	isEnabled     bool
}

func (m *mockLogger) Debug(msg string, args ...any) {
	m.debugCalled = true
	m.lastMsg = msg
	m.lastArgs = args
}

func (m *mockLogger) Info(msg string, args ...any) {
	m.infoCalled = true
	m.lastMsg = msg
	m.lastArgs = args
}

func (m *mockLogger) Warn(msg string, args ...any) {
	m.warnCalled = true
	m.lastMsg = msg
	m.lastArgs = args
}

func (m *mockLogger) Error(msg string, args ...any) {
	m.errorCalled = true
	m.lastMsg = msg
	m.lastArgs = args
}

func (m *mockLogger) With(args ...any) utils.Logger {
	m.withCalled = true
	m.lastArgs = args
	return m
}

func (m *mockLogger) Enabled(ctx context.Context, level utils.LogLevel) bool {
	m.enabledCalled = true
	m.lastLevel = level
	return m.isEnabled
}

// TestThirdPartyLoggerCompatibility 测试第三方日志库兼容性
func TestThirdPartyLoggerCompatibility(t *testing.T) {
	mock := &mockLogger{isEnabled: true}

	// 设置自定义 logger
	utils.Configure(utils.WithLogger(mock))

	logger := utils.Log()

	// 测试 Debug 方法
	logger.Debug("debug message", "key", "value")
	if !mock.debugCalled {
		t.Error("Debug() was not called")
	}
	if mock.lastMsg != "debug message" {
		t.Errorf("Debug() message = %v, want %v", mock.lastMsg, "debug message")
	}

	// 测试 Info 方法
	logger.Info("info message")
	if !mock.infoCalled {
		t.Error("Info() was not called")
	}
	if mock.lastMsg != "info message" {
		t.Errorf("Info() message = %v, want %v", mock.lastMsg, "info message")
	}

	// 测试 Warn 方法
	logger.Warn("warn message")
	if !mock.warnCalled {
		t.Error("Warn() was not called")
	}

	// 测试 Error 方法
	logger.Error("error message")
	if !mock.errorCalled {
		t.Error("Error() was not called")
	}

	// 测试 With 方法
	newLogger := logger.With("request_id", "12345")
	if !mock.withCalled {
		t.Error("With() was not called")
	}
	if newLogger == nil {
		t.Error("With() returned nil")
	}

	// 测试 Enabled 方法
	enabled := logger.Enabled(context.Background(), utils.LevelInfo)
	if !mock.enabledCalled {
		t.Error("Enabled() was not called")
	}
	if !enabled {
		t.Error("Enabled() returned false, want true")
	}
	if mock.lastLevel != utils.LevelInfo {
		t.Errorf("Enabled() level = %v, want %v", mock.lastLevel, utils.LevelInfo)
	}
}

// TestLogLevels 测试日志级别定义
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

	// 验证级别大小关系
	if utils.LevelDebug >= utils.LevelInfo {
		t.Error("LevelDebug should be less than LevelInfo")
	}
	if utils.LevelInfo >= utils.LevelWarn {
		t.Error("LevelInfo should be less than LevelWarn")
	}
	if utils.LevelWarn >= utils.LevelError {
		t.Error("LevelWarn should be less than LevelError")
	}
}

// TestDefaultSlogLogger 测试默认的 slog logger 实现
func TestDefaultSlogLogger(t *testing.T) {
	// 使用默认 logger
	logger := utils.Log()
	if logger == nil {
		t.Fatal("Log() returned nil")
	}

	// 测试各级别方法不会 panic
	logger.Debug("debug test")
	logger.Info("info test")
	logger.Warn("warn test")
	logger.Error("error test")

	// 测试 With 返回新的 Logger
	newLogger := logger.With("key", "value")
	if newLogger == nil {
		t.Error("With() returned nil")
	}

	// 测试 Enabled 方法
	ctx := context.Background()
	for _, level := range []utils.LogLevel{
		utils.LevelDebug,
		utils.LevelInfo,
		utils.LevelWarn,
		utils.LevelError,
	} {
		// 不期望 panic
		_ = logger.Enabled(ctx, level)
	}
}
