package utils_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Is999/go-utils"
)

func TestWithJSON(t *testing.T) {
	// 测试 WithJSON 选项
	opt := utils.WithJSON(json.Marshal, json.Unmarshal)
	if opt == nil {
		t.Error("WithJSON() returned nil")
	}
}

func TestWithJSONNil(t *testing.T) {
	// 测试传入nil参数
	opt := utils.WithJSON(nil, nil)
	if opt == nil {
		t.Error("WithJSON() with nil params returned nil")
	}
}

func TestMarshal(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name    string
		data    any
		wantErr bool
	}{
		{name: "001", data: testStruct{Name: "test", Age: 18}, wantErr: false},
		{name: "002", data: map[string]any{"key": "value"}, wantErr: false},
		{name: "003", data: []int{1, 2, 3}, wantErr: false},
		{name: "004", data: "string", wantErr: false},
		{name: "005", data: 123, wantErr: false},
		{name: "006", data: nil, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.Marshal(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) == 0 {
				t.Errorf("Marshal() got empty result")
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name    string
		data    string
		target  any
		wantErr bool
	}{
		{name: "001", data: `{"name":"test","age":18}`, target: &testStruct{}, wantErr: false},
		{name: "002", data: `{"key":"value"}`, target: &map[string]any{}, wantErr: false},
		{name: "003", data: `[1,2,3]`, target: &[]int{}, wantErr: false},
		{name: "004", data: `invalid json`, target: &testStruct{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.Unmarshal([]byte(tt.data), tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// testLogger 用于测试的自定义 Logger 实现
type testLogger struct {
	logs []string
}

func (l *testLogger) Debug(msg string, args ...any)                    { l.logs = append(l.logs, "DEBUG: "+msg) }
func (l *testLogger) Info(msg string, args ...any)                     { l.logs = append(l.logs, "INFO: "+msg) }
func (l *testLogger) Warn(msg string, args ...any)                     { l.logs = append(l.logs, "WARN: "+msg) }
func (l *testLogger) Error(msg string, args ...any)                    { l.logs = append(l.logs, "ERROR: "+msg) }
func (l *testLogger) With(args ...any) utils.Logger                    { return l }
func (l *testLogger) Enabled(_ context.Context, _ utils.LogLevel) bool { return true }

func TestWithLogger(t *testing.T) {
	// 测试 WithLogger 选项
	opt := utils.WithLogger(&testLogger{})
	if opt == nil {
		t.Error("WithLogger() returned nil")
	}
}

func TestWithLoggerNil(t *testing.T) {
	// 测试传入nil参数
	opt := utils.WithLogger(nil)
	if opt == nil {
		t.Error("WithLogger() with nil param returned nil")
	}
}

func TestLog(t *testing.T) {
	// 默认 Logger 不为 nil
	logger := utils.Log()
	if logger == nil {
		t.Error("Log() returned nil")
	}

	// 默认 Logger 应能正常调用各级别方法（不 panic）
	logger.Debug("debug test")
	logger.Info("info test")
	logger.Warn("warn test")
	logger.Error("error test")

	// With 返回新的 Logger
	child := logger.With("key", "value")
	if child == nil {
		t.Error("Logger.With() returned nil")
	}

	// Enabled 应返回 bool
	_ = logger.Enabled(context.Background(), utils.LevelInfo)
}
