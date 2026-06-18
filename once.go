package utils

import (
	"context"
	"sync"

	"github.com/Is999/go-utils/errors"
)

// Once 提供带重试能力的一次性执行控制器。
// 同一轮生命周期内只会有一个 goroutine 真正执行目标函数，其余调用方等待最终结果。
type Once struct {
	mu      sync.Mutex    // 状态锁
	waitCh  chan struct{} // 当前执行轮次的等待通道
	running bool          // 是否已有执行中的任务
	done    bool          // 是否已有最终结果
	err     error         // 缓存的最终错误
}

// Do 执行带重试能力的一次性调用。
//
//   - f：待执行函数，无参数并返回 error
//
//   - maxRetries：最大尝试次数，包含首次执行；小于等于 0 时按 1 次处理
//
//   - error：执行成功返回 nil，最终失败返回带重试次数的错误
//
// 特性：
//   - 线程安全：同一时刻仅一个 goroutine 执行目标函数，其余 goroutine 等待结果
//   - 使用有上限的指数退避策略，避免重试间隔无限增大
//   - 成功或最终失败后都会缓存结果，后续调用直接复用；需重新执行时调用 Reset
func (r *Once) Do(f func() error, maxRetries int) error {
	if f == nil {
		return errors.New("无效的执行方法")
	}
	return r.doContext(context.Background(), GetFunctionName(f), func(_ context.Context) error {
		return f()
	}, maxRetries)
}

// DoContext 执行带重试能力的一次性调用。
// 当 ctx 被取消时，等待中的调用方会立即返回；真正执行中的 goroutine 也会在重试等待阶段响应取消。
func (r *Once) DoContext(ctx context.Context, f func(ctx context.Context) error, maxRetries int) error {
	return r.doContext(ctx, GetFunctionName(f), f, maxRetries)
}

// doContext 是 Do/DoContext 的统一实现。
func (r *Once) doContext(ctx context.Context, fnName string, f func(ctx context.Context) error, maxRetries int) error {
	if f == nil {
		return errors.New("无效的执行方法")
	}
	if maxRetries <= 0 {
		maxRetries = 1
	}
	ctx = ensureContext(ctx)

	for {
		r.mu.Lock()
		// 已有最终结果时直接复用，避免重复执行。
		if r.done {
			err := r.err
			r.mu.Unlock()
			return err
		}

		// 当前无执行中的任务时，由当前 goroutine 负责执行。
		if !r.running {
			r.running = true
			r.waitCh = make(chan struct{})
			r.mu.Unlock()
			break
		}

		// 其余 goroutine 等待执行结果落定。
		waitCh := r.waitCh
		r.mu.Unlock()

		select {
		case <-waitCh:
		case <-ctx.Done():
			return errors.Tag(ctx.Err())
		}
	}

	err := r.doWithRetryContext(ctx, fnName, f, maxRetries)

	r.mu.Lock()
	r.err = err
	r.done = true
	r.running = false
	if r.waitCh != nil {
		close(r.waitCh)
		r.waitCh = nil
	}
	r.mu.Unlock()
	return err
}

// Reset 重置 Once 状态，使控制器进入下一轮可执行状态。
func (r *Once) Reset() {
	for {
		r.mu.Lock()
		if !r.running {
			r.done = false
			r.err = nil
			r.waitCh = nil
			r.mu.Unlock()
			return
		}
		waitCh := r.waitCh
		r.mu.Unlock()
		if waitCh != nil {
			<-waitCh
		}
	}
}

// doWithRetryContext 执行带重试的目标函数。
// 若目标函数发生 panic，会被转换为 error 返回，避免等待方永久阻塞。
func (r *Once) doWithRetryContext(ctx context.Context, fnName string, f func(ctx context.Context) error, maxRetries int) (finalErr error) {
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			finalErr = errors.Errorf("%s panic: %v", fnName, recoverErr)
		}
	}()

	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = f(ctx)
		if err == nil {
			return nil
		}
		if attempt >= maxRetries {
			break
		}

		// 使用有上限的指数退避，避免失败风暴下重试间隔失控。
		if err = waitRetry(ctx, attempt); err != nil {
			return errors.Tag(err)
		}
	}
	return errors.Wrapf(err, "%s 尝试 %d 次后依然失败", fnName, maxRetries)
}
