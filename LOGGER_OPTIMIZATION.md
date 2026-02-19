# Logger 和 Error 优化说明

## 概述 / Overview

本次优化重点解决 logger 和 error 实现代码与第三方日志库的兼容性问题，使得 utils 库能够无缝集成 zap、logrus 等流行的日志库。

This optimization focuses on solving the compatibility issues between logger/error implementation and third-party logging libraries, enabling seamless integration with popular logging libraries like zap and logrus.

## 主要变更 / Main Changes

### 1. 自定义日志级别 / Custom Log Level

引入了 `LogLevel` 类型替代 `slog.Level`，实现与第三方日志库的解耦。

Introduced `LogLevel` type to replace `slog.Level`, achieving decoupling from third-party logging libraries.

```go
type LogLevel int

const (
    LevelDebug LogLevel = -4
    LevelInfo  LogLevel = 0
    LevelWarn  LogLevel = 4
    LevelError LogLevel = 8
)
```

### 2. 更新 Logger 接口 / Updated Logger Interface

```go
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    With(args ...any) Logger
    Enabled(ctx context.Context, level LogLevel) bool  // 使用 LogLevel 而非 slog.Level
}
```

### 3. 内部适配层 / Internal Adaptation Layer

在 `slogLogger` 实现中添加了 `toSlogLevel()` 转换函数，确保内部默认实现仍然使用 `log/slog`，同时对外提供统一的接口。

Added `toSlogLevel()` conversion function in `slogLogger` implementation to ensure internal default implementation still uses `log/slog` while providing a unified interface externally.

## 如何集成第三方日志库 / How to Integrate Third-Party Logging Libraries

### 集成 Zap / Integrate with Zap

```go
import (
    "context"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    utils "github.com/Is999/go-utils"
)

type zapLoggerAdapter struct {
    logger *zap.SugaredLogger
}

func (z *zapLoggerAdapter) Debug(msg string, args ...any) {
    z.logger.Debugw(msg, args...)
}

func (z *zapLoggerAdapter) Info(msg string, args ...any) {
    z.logger.Infow(msg, args...)
}

func (z *zapLoggerAdapter) Warn(msg string, args ...any) {
    z.logger.Warnw(msg, args...)
}

func (z *zapLoggerAdapter) Error(msg string, args ...any) {
    z.logger.Errorw(msg, args...)
}

func (z *zapLoggerAdapter) With(args ...any) utils.Logger {
    return &zapLoggerAdapter{logger: z.logger.With(args...)}
}

func (z *zapLoggerAdapter) Enabled(ctx context.Context, level utils.LogLevel) bool {
    zapLevel := toZapLevel(level)
    return z.logger.Core().Enabled(zapLevel)
}

func toZapLevel(level utils.LogLevel) zapcore.Level {
    switch level {
    case utils.LevelDebug:
        return zapcore.DebugLevel
    case utils.LevelInfo:
        return zapcore.InfoLevel
    case utils.LevelWarn:
        return zapcore.WarnLevel
    case utils.LevelError:
        return zapcore.ErrorLevel
    default:
        return zapcore.InfoLevel
    }
}

// 使用 / Usage
func main() {
    zapLogger, _ := zap.NewProduction()
    adapter := &zapLoggerAdapter{logger: zapLogger.Sugar()}
    utils.Configure(utils.WithLogger(adapter))
    
    logger := utils.Log()
    logger.Info("Application started", "version", "1.0.0")
}
```

### 集成 Logrus / Integrate with Logrus

```go
import (
    "context"
    "github.com/sirupsen/logrus"
    utils "github.com/Is999/go-utils"
)

type logrusLoggerAdapter struct {
    logger *logrus.Logger
}

func (l *logrusLoggerAdapter) Debug(msg string, args ...any) {
    l.logger.WithFields(argsToFields(args)).Debug(msg)
}

func (l *logrusLoggerAdapter) Info(msg string, args ...any) {
    l.logger.WithFields(argsToFields(args)).Info(msg)
}

func (l *logrusLoggerAdapter) Warn(msg string, args ...any) {
    l.logger.WithFields(argsToFields(args)).Warn(msg)
}

func (l *logrusLoggerAdapter) Error(msg string, args ...any) {
    l.logger.WithFields(argsToFields(args)).Error(msg)
}

func (l *logrusLoggerAdapter) With(args ...any) utils.Logger {
    entry := l.logger.WithFields(argsToFields(args))
    return &logrusLoggerAdapter{logger: entry.Logger}
}

func (l *logrusLoggerAdapter) Enabled(ctx context.Context, level utils.LogLevel) bool {
    logrusLevel := toLogrusLevel(level)
    return l.logger.IsLevelEnabled(logrusLevel)
}

func argsToFields(args ...any) logrus.Fields {
    fields := make(logrus.Fields)
    for i := 0; i < len(args)-1; i += 2 {
        if key, ok := args[i].(string); ok {
            fields[key] = args[i+1]
        }
    }
    return fields
}

func toLogrusLevel(level utils.LogLevel) logrus.Level {
    switch level {
    case utils.LevelDebug:
        return logrus.DebugLevel
    case utils.LevelInfo:
        return logrus.InfoLevel
    case utils.LevelWarn:
        return logrus.WarnLevel
    case utils.LevelError:
        return logrus.ErrorLevel
    default:
        return logrus.InfoLevel
    }
}

// 使用 / Usage
func main() {
    logrusLogger := logrus.New()
    adapter := &logrusLoggerAdapter{logger: logrusLogger}
    utils.Configure(utils.WithLogger(adapter))
    
    logger := utils.Log()
    logger.Info("Application started", "version", "1.0.0")
}
```

## Error 包的优化 / Error Package Optimization

### 保持向后兼容 / Maintain Backward Compatibility

`errors` 包保留了 `slog.LogValuer` 接口的实现，以保持与现有代码的兼容性。这是一个合理的设计选择：

The `errors` package retains the implementation of `slog.LogValuer` interface to maintain compatibility with existing code. This is a reasonable design choice:

1. **错误追踪是可选的** / Error tracing is optional
   - 第三方日志库可以选择性地处理 `slog.LogValuer` 接口
   - Third-party logging libraries can optionally handle `slog.LogValuer` interface

2. **不影响核心功能** / Does not affect core functionality
   - 错误的核心功能（Error(), Wrap(), Unwrap() 等）完全独立
   - Core error functionality (Error(), Wrap(), Unwrap(), etc.) is completely independent

3. **结构化日志支持** / Structured logging support
   - 支持结构化日志输出错误追踪信息
   - Supports structured logging for error tracing information

### 使用示例 / Usage Example

```go
import (
    "github.com/Is999/go-utils/errors"
    utils "github.com/Is999/go-utils"
)

func processRequest() error {
    err := doSomething()
    if err != nil {
        return errors.Wrap(err, "failed to process request")
    }
    return nil
}

func main() {
    logger := utils.Log()
    
    if err := processRequest(); err != nil {
        // 使用 errors.Trace 获取详细的错误追踪信息
        logger.Error("Request failed", "error", err, "trace", errors.Trace(err))
    }
}
```

## 关键优势 / Key Benefits

1. **完全解耦** / Complete Decoupling
   - Logger 接口不再依赖 `log/slog` 包
   - Logger interface no longer depends on `log/slog` package

2. **易于集成** / Easy Integration
   - 只需实现 6 个方法即可集成任何日志库
   - Only need to implement 6 methods to integrate any logging library

3. **向后兼容** / Backward Compatible
   - 所有现有代码无需修改即可继续工作
   - All existing code continues to work without modifications

4. **灵活性** / Flexibility
   - 可以在运行时切换不同的日志实现
   - Can switch different logging implementations at runtime

## 测试验证 / Test Verification

运行以下命令验证所有功能：

Run the following commands to verify all functionality:

```bash
# 运行所有测试
go test ./...

# 运行日志相关测试
go test -v -run TestThirdPartyLoggerCompatibility
go test -v -run TestLogLevels
go test -v -run TestDefaultSlogLogger

# 运行示例
go test -v -run ExampleZapLoggerAdapter
go test -v -run ExampleLogrusLoggerAdapter
```

## 迁移指南 / Migration Guide

如果您之前在自定义 Logger 实现中使用了 `slog.Level`，需要进行以下调整：

If you previously used `slog.Level` in custom Logger implementations, you need to make the following adjustments:

### 修改前 / Before

```go
func (l *CustomLogger) Enabled(ctx context.Context, level slog.Level) bool {
    return true
}
```

### 修改后 / After

```go
func (l *CustomLogger) Enabled(ctx context.Context, level utils.LogLevel) bool {
    return true
}
```

## 总结 / Summary

本次优化通过引入自定义 `LogLevel` 类型和更新 `Logger` 接口，成功实现了与第三方日志库的完全兼容，同时保持了向后兼容性和良好的扩展性。

This optimization successfully achieves full compatibility with third-party logging libraries by introducing custom `LogLevel` type and updating the `Logger` interface, while maintaining backward compatibility and good extensibility.
