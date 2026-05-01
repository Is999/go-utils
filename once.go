package utils

import (
	"fmt"
	"sync"
	"time"
)

// Once 提供带重试能力的一次性执行控制器。
// 同一轮生命周期内只会有一个 goroutine 真正执行目标函数，其余调用方等待最终结果。
type Once struct {
	mu      sync.Mutex
	cond    *sync.Cond
	running bool
	done    bool
	err     error
}

// Do 执行带重试能力的一次性调用。
//
// 参数说明：
//
//   - f：待执行函数，无参数并返回 error
//   - maxRetries：最大尝试次数，包含首次执行；小于等于 0 时按 1 次处理
//
// 返回值：
//
//   - error：执行成功返回 nil，最终失败返回带重试次数的错误
//
// 特性：
//   - 线程安全：同一时刻仅一个 goroutine 执行目标函数，其余 goroutine 等待结果
//   - 使用有上限的指数退避策略，避免重试间隔无限增大
//   - 成功或最终失败后都会缓存结果，后续调用直接复用；需重新执行时调用 Reset
func (r *Once) Do(f func() error, maxRetries int) error {
	if f == nil {
		return fmt.Errorf("Once.Do() f 不能为空")
	}
	if maxRetries <= 0 {
		maxRetries = 1
	}

	r.mu.Lock()
	r.initCondLocked()

	for {
		// 已有最终结果时直接复用，避免重复执行。
		if r.done {
			err := r.err
			r.mu.Unlock()
			return err
		}

		// 当前无执行中的任务时，由当前 goroutine 负责执行。
		if !r.running {
			r.running = true
			break
		}

		// 其余 goroutine 等待执行结果落定。
		r.cond.Wait()
	}
	r.mu.Unlock()

	err := r.doWithRetry(f, maxRetries)

	r.mu.Lock()
	r.err = err
	r.done = true
	r.running = false
	r.cond.Broadcast()
	r.mu.Unlock()
	return err
}

// Reset 重置 Once 状态，使控制器进入下一轮可执行状态。
func (r *Once) Reset() {
	r.mu.Lock()
	r.initCondLocked()
	for r.running {
		r.cond.Wait()
	}
	r.done = false
	r.err = nil
	r.mu.Unlock()
}

// initCondLocked 初始化条件变量。
// 调用方必须先持有互斥锁。
func (r *Once) initCondLocked() {
	if r.cond == nil {
		r.cond = sync.NewCond(&r.mu)
	}
}

// doWithRetry 执行带重试的目标函数。
// 若目标函数发生 panic，会被转换为 error 返回，避免等待方永久阻塞。
func (r *Once) doWithRetry(f func() error, maxRetries int) (finalErr error) {
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			finalErr = fmt.Errorf("Once.Do() panic: %v", recoverErr)
		}
	}()

	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = f()
		if err == nil {
			return nil
		}
		if attempt >= maxRetries {
			break
		}

		// 使用有上限的指数退避，避免失败风暴下重试间隔失控。
		time.Sleep(retryDelay(attempt))
	}
	return fmt.Errorf("failed after %d attempts: %v", maxRetries, err)
}
