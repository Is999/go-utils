package utils_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	utils "github.com/Is999/go-utils"
)

// slogAdapter 复用标准库的日志方法，只转换 With 和 Enabled 的接口签名。
type slogAdapter struct {
	*slog.Logger // 字段绑定和级别过滤均交给原 handler，保留其并发保证。
}

// With 派生带固定字段的日志器，不修改父日志器。
func (l slogAdapter) With(args ...any) utils.Logger {
	return slogAdapter{l.Logger.With(args...)}
}

// Enabled 沿用 handler 的过滤规则；两种日志级别使用相同的数值定义。
func (l slogAdapter) Enabled(ctx context.Context, level utils.LogLevel) bool {
	return l.Logger.Enabled(ctx, slog.Level(level))
}

// ExampleLogger 展示字段继承与级别过滤，不占用只能初始化一次的全局 Configure。
func ExampleLogger() {
	var logger utils.Logger = slogAdapter{slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		// 示例去掉时间，便于核对每次运行的输出。
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return attr
		},
	}))}

	logger.Debug("hidden")
	logger.With("request_id", "req-42").Info("ready", "attempt", 1)
	fmt.Println(logger.Enabled(context.Background(), utils.LevelDebug))

	// Output:
	// level=INFO msg=ready request_id=req-42 attempt=1
	// false
}
