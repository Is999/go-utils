package utils

import (
	"context"
	"log/slog"
)

// slogLogger 基于 log/slog 的默认 Logger 实现。
// 当 l 为 nil 时，委托给 slog.Default()，从而跟随 slog.SetDefault() 的变更。
type slogLogger struct {
	l *slog.Logger
}

func newSlogLogger() *slogLogger {
	return &slogLogger{}
}

func (s *slogLogger) logger() *slog.Logger {
	if s.l != nil {
		return s.l
	}
	return slog.Default()
}

func (s *slogLogger) Debug(msg string, args ...any) { s.logger().Debug(msg, args...) }
func (s *slogLogger) Info(msg string, args ...any)  { s.logger().Info(msg, args...) }
func (s *slogLogger) Warn(msg string, args ...any)  { s.logger().Warn(msg, args...) }
func (s *slogLogger) Error(msg string, args ...any) { s.logger().Error(msg, args...) }

func (s *slogLogger) With(args ...any) Logger {
	return &slogLogger{l: s.logger().With(args...)}
}

// Enabled 将自定义 LogLevel 转换为 slog.Level 进行判断
func (s *slogLogger) Enabled(ctx context.Context, level LogLevel) bool {
	return s.logger().Enabled(ctx, toSlogLevel(level))
}

// toSlogLevel 将自定义 LogLevel 转换为 slog.Level
func toSlogLevel(level LogLevel) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		// 对于自定义级别，直接转换（假设用户知道 slog 的级别语义）
		// For custom levels, directly convert (assumes user understands slog level semantics)
		return slog.Level(level)
	}
}

// Log 获取全局 Logger 实例。
// 若未通过 Configure 设置自定义 Logger，则返回基于 log/slog 标准库的默认实现。
//
// 兼容第三方日志库说明:
// 本库 Logger 接口设计兼容 log/slog。若需使用 zap、logrus 等第三方库，
// 需编写适配器实现 Logger 接口，并通过 Configure(WithLogger(adapter)) 注入。
// Example (Zap):
//
//	type ZapAdapter struct { l *zap.Logger }
//	func (z *ZapAdapter) Debug(msg string, args ...any) { ... }
//	func (z *ZapAdapter) Enabled(ctx context.Context, level LogLevel) bool { ... }
func Log() Logger {
	return config.logger
}
