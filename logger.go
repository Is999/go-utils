package utils

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

// slogLogger 适配 slog，并发输出由底层 Handler 负责。
type slogLogger struct {
	l *slog.Logger // 底层 slog Logger，nil 时使用 slog.Default()。
}

// logger 未绑定实例时动态读取 slog.Default，以跟随默认日志器变更。
func (s *slogLogger) logger() *slog.Logger {
	if s.l != nil {
		return s.l
	}
	return slog.Default()
}

// Debug 输出调试级别日志。
func (s *slogLogger) Debug(msg string, args ...any) { s.log(slog.LevelDebug, msg, args...) }

// Info 输出信息级别日志。
func (s *slogLogger) Info(msg string, args ...any) { s.log(slog.LevelInfo, msg, args...) }

// Warn 输出警告级别日志。
func (s *slogLogger) Warn(msg string, args ...any) { s.log(slog.LevelWarn, msg, args...) }

// Error 输出错误级别日志。
func (s *slogLogger) Error(msg string, args ...any) { s.log(slog.LevelError, msg, args...) }

// log 为四个级别统一采集调用位置，避免 source 指向本包的包装方法。
func (s *slogLogger) log(level slog.Level, msg string, args ...any) {
	logger := s.logger() // 同一记录的过滤与输出使用同一个底层实例。
	ctx := context.Background()
	// 被过滤的级别不采集调用栈，也不构造日志记录。
	if !logger.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr         // 日志记录只需要调用方的一帧。
	runtime.Callers(3, pcs[:]) // 跳过 runtime.Callers、log 和公开的级别方法。
	record := slog.NewRecord(time.Now(), level, msg, pcs[0])
	record.Add(args...) // 属性解析交给 slog，保留 Attr 和未配对参数的原有行为。
	// 沿用 slog 的调用约定，Handler 错误不向业务调用方传播。
	_ = logger.Handler().Handle(ctx, record)
}

// With 创建带固定字段的子 Logger，并固定当前底层实例，不再跟随后续 slog.SetDefault。
func (s *slogLogger) With(args ...any) Logger {
	return &slogLogger{l: s.logger().With(args...)}
}

// Enabled 将自定义 LogLevel 转换为 slog.Level 进行判断
func (s *slogLogger) Enabled(ctx context.Context, level LogLevel) bool {
	return s.logger().Enabled(ctx, slog.Level(level))
}

// Log 返回全局共享 Logger，默认跟随 slog.Default；第三方实现通过 Configure(WithLogger(...)) 注入。
func Log() Logger {
	return configValue.Load().logger
}
