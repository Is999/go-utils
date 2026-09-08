package utils

import (
	"context"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/Is999/go-utils/errors"
)

const (
	// retryBaseDelay 定义第一次失败后的基础退避时间。
	retryBaseDelay = 200 * time.Millisecond
	// retryMaxDelay 定义退避时间上限，避免长时间阻塞调用方。
	retryMaxDelay = 3 * time.Second
)

// Ternary 按 expr 选择返回值；两个候选参数在调用前都会求值，不提供短路执行。
func Ternary[T any](expr bool, trueVal, falseVal T) T {
	if expr {
		return trueVal
	}
	return falseVal
}

// FormatNumber 保留 decimals 位小数并插入指定分隔符，舍入沿用 strconv.FormatFloat。
// thousandsSep 为空时不加千分位符；NaN/Inf 原样输出，不应用精度或分隔符。
func FormatNumber(number float64, decimals uint, decPoint, thousandsSep string) string {
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return strconv.FormatFloat(number, 'f', -1, 64)
	}

	dec := int(decimals)
	// 舍入和负零符号沿用 strconv 的结果。
	str := strconv.FormatFloat(number, 'f', dec, 64)
	if decPoint == "." && thousandsSep == "" {
		return str
	}

	// 符号不参与千分位计数。
	prefix, neg := strings.CutPrefix(str, "-")
	suffix := ""
	if dec > 0 {
		// 'f' 格式固定保留 dec 位小数，可直接按尾长切分。
		suffix = prefix[len(prefix)-dec:]
		prefix = prefix[:len(prefix)-dec-1]
	}

	groups := (len(prefix) - 1) / 3
	var b strings.Builder
	b.Grow(len(prefix) + groups*len(thousandsSep) + len(decPoint) + len(suffix) + 1)
	if neg {
		b.WriteByte('-')
	}

	first := len(prefix) % 3
	if first == 0 {
		first = 3
	}
	b.WriteString(prefix[:first])
	for i := first; i < len(prefix); i += 3 {
		b.WriteString(thousandsSep)
		b.WriteString(prefix[i : i+3])
	}
	if dec > 0 {
		b.WriteString(decPoint)
		b.WriteString(suffix)
	}
	return b.String()
}

// Retry 尝试执行 fn，如果 fn 返回错误则按指数退避策略重试。
// maxRetries 包含首次执行，0 按 1 次处理；tries 从 1 开始传给 fn。
// 重试间隔从 100ms~200ms 区间起步，按 2 倍递增，并在每次退避上附加随机抖动，最大不超过 3s。
func Retry(maxRetries uint8, fn func(tries int) error) error {
	if fn == nil {
		return errors.New("无效的执行方法")
	}
	return retryContext(context.Background(), maxRetries, fn, func(_ context.Context, tries int) error {
		return fn(tries)
	})
}

// RetryContext 尝试执行 fn，如果 fn 返回错误则按指数退避策略重试。
// ctx 为 nil 时使用 Background；maxRetries 和 tries 的含义同 Retry。
// 取消会中断重试间隔，正在执行的 fn 需自行响应 ctx；首次调用不预检查取消状态。
func RetryContext(ctx context.Context, maxRetries uint8, fn func(ctx context.Context, tries int) error) error {
	if fn == nil {
		return errors.New("无效的执行方法")
	}
	return retryContext(ctx, maxRetries, fn, fn)
}

// retryContext 共用 Retry 的次数和退避约定，首次调用不预检查取消状态。
func retryContext(ctx context.Context, maxRetries uint8, originalFn any, fn func(ctx context.Context, tries int) error) error {
	ctx = ensureContext(ctx)
	if maxRetries == 0 {
		maxRetries = 1
	}

	var err error
	for tries := 1; tries <= int(maxRetries); tries++ {
		if err = fn(ctx, tries); err == nil {
			return nil
		}
		if tries < int(maxRetries) {
			if err = waitRetry(ctx, tries); err != nil {
				return err
			}
		}
	}
	// 只在最终失败时查询原回调名，避免诊断指向 Retry 的适配闭包。
	return errors.Wrapf(err, "%s 尝试 %d 次后依然失败", GetFunctionName(originalFn), maxRetries)
}

// ensureContext 归一化 context，避免调用方传入 nil 导致 panic。
func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// waitRetry 等待下一次重试窗口，context 取消会中断等待。
func waitRetry(ctx context.Context, attempt int) error {
	delay := retryDelay(attempt)
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		// 在取消出口附栈，上层直接传回同一错误。
		return errors.Tag(ctx.Err())
	}
}

// retryDelay 计算失败后的等待时间，在指数上限的一半至上限之间随机取值。
func retryDelay(attempt int) time.Duration {
	// 指数退避从 retryBaseDelay 开始，attempt=1 表示第一次失败后的等待。
	delay := retryBaseDelay
	for i := 1; i < attempt && delay < retryMaxDelay; i++ {
		delay *= 2
	}
	if delay > retryMaxDelay {
		delay = retryMaxDelay
	}

	// 使用 equal jitter：保留一半确定性延迟，另一半随机化。
	half := delay / 2
	return half + time.Duration(rand.Int64N(int64(half)))
}
