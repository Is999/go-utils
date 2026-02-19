package utils_test

import (
	"context"
	"fmt"

	utils "github.com/Is999/go-utils"
)

// 示例 1: 兼容 zap 日志库的适配器
// Example 1: Adapter for zap logging library

// zapLoggerAdapter 将 zap Logger 适配到 utils.Logger 接口
// zapLoggerAdapter adapts zap Logger to utils.Logger interface
//
// 使用示例 / Usage example:
// import "go.uber.org/zap"
//
// zapLogger, _ := zap.NewProduction()
// adapter := &zapLoggerAdapter{logger: zapLogger.Sugar()}
// utils.Configure(utils.WithLogger(adapter))
type zapLoggerAdapter struct {
	// logger *zap.SugaredLogger  // 取消注释以使用 / uncomment to use
	logger interface{} // 占位符，实际使用时替换为 *zap.SugaredLogger
}

func (z *zapLoggerAdapter) Debug(msg string, args ...any) {
	// z.logger.Debugw(msg, args...)
	fmt.Printf("[ZAP DEBUG] %s %v\n", msg, args)
}

func (z *zapLoggerAdapter) Info(msg string, args ...any) {
	// z.logger.Infow(msg, args...)
	fmt.Printf("[ZAP INFO] %s %v\n", msg, args)
}

func (z *zapLoggerAdapter) Warn(msg string, args ...any) {
	// z.logger.Warnw(msg, args...)
	fmt.Printf("[ZAP WARN] %s %v\n", msg, args)
}

func (z *zapLoggerAdapter) Error(msg string, args ...any) {
	// z.logger.Errorw(msg, args...)
	fmt.Printf("[ZAP ERROR] %s %v\n", msg, args)
}

func (z *zapLoggerAdapter) With(args ...any) utils.Logger {
	// 在真实实现中，应该创建一个新的 logger 实例并附加上下文
	// In real implementation, should create a new logger instance with context
	// return &zapLoggerAdapter{logger: z.logger.With(args...)}

	// 为了演示，这里返回一个新实例（实际上是同一个 logger）
	// For demonstration, return a new instance (actually the same logger)
	return &zapLoggerAdapter{logger: z.logger}
}

func (z *zapLoggerAdapter) Enabled(ctx context.Context, level utils.LogLevel) bool {
	// 可以根据 zap 的日志级别配置来判断
	// Can determine based on zap's log level configuration
	// zapLevel := zapcore.Level(level)
	// return z.logger.Core().Enabled(zapLevel)
	return true
}

// 示例 2: 兼容 logrus 日志库的适配器
// Example 2: Adapter for logrus logging library

// logrusLoggerAdapter 将 logrus Logger 适配到 utils.Logger 接口
// logrusLoggerAdapter adapts logrus Logger to utils.Logger interface
//
// 使用示例 / Usage example:
// import "github.com/sirupsen/logrus"
//
// logrusLogger := logrus.New()
// adapter := &logrusLoggerAdapter{logger: logrusLogger}
// utils.Configure(utils.WithLogger(adapter))
type logrusLoggerAdapter struct {
	// logger *logrus.Logger  // 取消注释以使用 / uncomment to use
	logger interface{} // 占位符，实际使用时替换为 *logrus.Logger
}

func (l *logrusLoggerAdapter) Debug(msg string, args ...any) {
	// l.logger.WithFields(argsToFields(args)).Debug(msg)
	fmt.Printf("[LOGRUS DEBUG] %s %v\n", msg, args)
}

func (l *logrusLoggerAdapter) Info(msg string, args ...any) {
	// l.logger.WithFields(argsToFields(args)).Info(msg)
	fmt.Printf("[LOGRUS INFO] %s %v\n", msg, args)
}

func (l *logrusLoggerAdapter) Warn(msg string, args ...any) {
	// l.logger.WithFields(argsToFields(args)).Warn(msg)
	fmt.Printf("[LOGRUS WARN] %s %v\n", msg, args)
}

func (l *logrusLoggerAdapter) Error(msg string, args ...any) {
	// l.logger.WithFields(argsToFields(args)).Error(msg)
	fmt.Printf("[LOGRUS ERROR] %s %v\n", msg, args)
}

func (l *logrusLoggerAdapter) With(args ...any) utils.Logger {
	// 在真实实现中，应该创建一个带有附加字段的新 logger
	// In real implementation, should create a new logger with additional fields
	// return &logrusLoggerAdapter{logger: l.logger.WithFields(argsToFields(args))}

	// 为了演示，这里返回一个新实例（实际上是同一个 logger）
	// For demonstration, return a new instance (actually the same logger)
	return &logrusLoggerAdapter{logger: l.logger}
}

func (l *logrusLoggerAdapter) Enabled(ctx context.Context, level utils.LogLevel) bool {
	// 可以根据 logrus 的日志级别配置来判断
	// Can determine based on logrus's log level configuration
	// logrusLevel := toLogrusLevel(level)
	// return l.logger.IsLevelEnabled(logrusLevel)
	return true
}

// argsToFields 将 key-value 参数对转换为 logrus.Fields
// argsToFields converts key-value argument pairs to logrus.Fields
// func argsToFields(args ...any) logrus.Fields {
// 	fields := make(logrus.Fields)
// 	for i := 0; i < len(args)-1; i += 2 {
// 		if key, ok := args[i].(string); ok {
// 			fields[key] = args[i+1]
// 		}
// 	}
// 	return fields
// }

// toLogrusLevel 将 utils.LogLevel 转换为 logrus.Level
// toLogrusLevel converts utils.LogLevel to logrus.Level
// func toLogrusLevel(level utils.LogLevel) logrus.Level {
// 	switch level {
// 	case utils.LevelDebug:
// 		return logrus.DebugLevel
// 	case utils.LevelInfo:
// 		return logrus.InfoLevel
// 	case utils.LevelWarn:
// 		return logrus.WarnLevel
// 	case utils.LevelError:
// 		return logrus.ErrorLevel
// 	default:
// 		return logrus.InfoLevel
// 	}
// }

// Example_zapLoggerAdapter 演示如何使用 zap 日志库适配器
// Example_zapLoggerAdapter demonstrates how to use zap logger adapter
func Example_zapLoggerAdapter() {
	// 创建适配器实例
	zapAdapter := &zapLoggerAdapter{}

	// 直接使用适配器进行日志输出（演示目的）
	// Use adapter directly for logging output (demonstration purpose)
	zapAdapter.Info("Application started", "version", "1.0.0")
	zapAdapter.Debug("Debug information", "key", "value")

	// 使用 With 添加上下文字段（注意：此演示实现不保留上下文）
	// Use With to add context fields (note: this demo implementation doesn't retain context)
	childLogger := zapAdapter.With("request_id", "12345")
	childLogger.Info("Processing request")

	// Output:
	// [ZAP INFO] Application started [version 1.0.0]
	// [ZAP DEBUG] Debug information [key value]
	// [ZAP INFO] Processing request []
}

// Example_logrusLoggerAdapter 演示如何使用 logrus 日志库适配器
// Example_logrusLoggerAdapter demonstrates how to use logrus logger adapter
func Example_logrusLoggerAdapter() {
	// 创建适配器实例
	logrusAdapter := &logrusLoggerAdapter{}

	// 直接使用适配器（因为 Configure 只能调用一次）
	// Use adapter directly (since Configure can only be called once)
	logrusAdapter.Info("Application started", "version", "2.0.0")
	logrusAdapter.Warn("Warning message")

	// Output:
	// [LOGRUS INFO] Application started [version 2.0.0]
	// [LOGRUS WARN] Warning message []
}
