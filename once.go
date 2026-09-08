package utils

import (
	"context"
	"sync"

	"github.com/Is999/go-utils/errors"
)

// Once 每轮只由一个调用方执行并重试，成功或最终失败都会缓存到 Reset。
// 零值可用，使用后不可复制；并发调用共享结果，只有执行方传入的函数和重试次数生效。
type Once struct {
	mu     sync.Mutex    // 保护以下轮次状态，执行函数和等待时不持锁。
	waitCh chan struct{} // 非 nil 表示本轮执行中，发布结果后关闭并清空。
	done   bool          // 是否已有可复用结果，最终失败也会置为 true。
	err    error         // done 为 true 时的结果，nil 表示成功。
}

// Do 执行或等待本轮结果；maxRetries 包含首次执行，小于等于 0 时执行一次。
// f 不可为 nil，也不能在执行期间调用同一个 Once 的 Do、DoContext 或 Reset。
func (r *Once) Do(f func() error, maxRetries int) error {
	if f == nil {
		return errors.New("无效的执行方法")
	}
	// 保留原函数，避免失败信息显示下面的适配闭包名。
	return r.doContext(context.Background(), f, func(_ context.Context) error {
		return f()
	}, maxRetries)
}

// DoContext 与 Do 共享结果；nil ctx 按 Background 处理，执行中的 f 需自行响应取消。
// 等待方取消只结束自身等待；执行方在退避期间取消产生的错误会作为本轮结果缓存。
func (r *Once) DoContext(ctx context.Context, f func(ctx context.Context) error, maxRetries int) error {
	return r.doContext(ctx, f, f, maxRetries)
}

// doContext 返回本轮结果，只有取得执行权的调用才运行 f。
func (r *Once) doContext(ctx context.Context, originalFn any, f func(ctx context.Context) error, maxRetries int) error {
	if f == nil {
		return errors.New("无效的执行方法")
	}
	if maxRetries <= 0 {
		maxRetries = 1
	}
	ctx = ensureContext(ctx)

	for {
		r.mu.Lock()
		// 缓存结果优先于当前调用的取消状态，保持同一轮结果一致。
		if r.done {
			err := r.err
			r.mu.Unlock()
			return err
		}

		// 先发布等待通道，再解锁执行，保证本轮只有一个执行方。
		if r.waitCh == nil {
			r.waitCh = make(chan struct{})
			r.mu.Unlock()
			break
		}

		// 等待期间释放状态锁，执行方才能发布结果并关闭通道。
		waitCh := r.waitCh
		r.mu.Unlock()

		select {
		case <-waitCh:
		case <-ctx.Done():
			return errors.Tag(ctx.Err())
		}
	}

	err := r.doWithRetryContext(ctx, originalFn, f, maxRetries)

	// 结果写入与通知在同一临界区完成，等待方醒来后重新检查状态。
	r.mu.Lock()
	r.err = err
	r.done = true
	close(r.waitCh)
	r.waitCh = nil
	r.mu.Unlock()
	return err
}

// Reset 等待正在执行的轮次结束，再清空缓存结果；不会中断目标函数。
func (r *Once) Reset() {
	for {
		r.mu.Lock()
		if r.waitCh == nil {
			r.done = false
			r.err = nil
			r.mu.Unlock()
			return
		}
		waitCh := r.waitCh
		r.mu.Unlock()
		<-waitCh
	}
}

// doWithRetryContext 尝试执行 f，次数耗尽时保留最后一次失败原因。
func (r *Once) doWithRetryContext(ctx context.Context, originalFn any, f func(ctx context.Context) error, maxRetries int) (finalErr error) {
	// panic 也转为错误，让外层能发布结果并唤醒等待方。
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			finalErr = errors.Errorf("%s panic: %v", GetFunctionName(originalFn), recoverErr)
		}
	}()

	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = f(ctx)
		if err == nil {
			return nil
		}
		if attempt < maxRetries {
			// 使用有上限的指数退避，避免失败风暴下重试间隔失控。
			if err = waitRetry(ctx, attempt); err != nil {
				return errors.Tag(err)
			}
		}
	}
	return errors.Wrapf(err, "%s 尝试 %d 次后依然失败", GetFunctionName(originalFn), maxRetries)
}
