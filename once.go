package utils

import (
	"fmt"
	"sync"
	"time"
)

type RetryOnce struct {
	once sync.Once
	mu   sync.Mutex
	done bool
	err  error
}

// Do 执行带有重试机制的函数调用
// 参数:
//
//	f: 要执行的目标函数，无参数且返回error类型
//	maxRetries: 最大重试次数
//
// 返回值:
//
//	error: 执行成功时返回nil，失败时返回包含重试次数的错误信息
//
// 特性:
//   - 线程安全，使用互斥锁保证并发安全
//   - 使用指数退避策略
//   - 通过sync.Once保证每次重试只执行一次目标函数
func (r *RetryOnce) Do(f func() error, maxRetries int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查是否已完成执行（包括成功或最终失败）
	if r.done {
		return r.err
	}

	attempt := 0
	for {
		// 使用sync.Once保证目标函数在单次循环中只执行一次
		r.once.Do(func() {
			r.err = f()
		})

		// 成功执行后标记完成状态
		if r.err == nil {
			r.done = true
			return r.err
		}

		// 重试次数达到上限时终止循环
		attempt++
		if attempt >= maxRetries {
			break
		}

		// 重置sync.Once准备下次重试，并执行指数退避等待
		r.once = sync.Once{}
		// 指数退避, 最大延迟1秒
		maxDelay := 100 << (attempt - 1)
		if maxDelay > 1000 {
			maxDelay = 1000
		}
		// 延迟重试
		time.Sleep(time.Millisecond * time.Duration(maxDelay))
	}

	// 标记最终失败状态并构造错误信息
	r.done = true
	r.err = fmt.Errorf("failed after %d retries: %v", maxRetries, r.err)
	return r.err
}

// Reset 重置 RetryOnce 实例状态，使其可以再次执行重试操作
func (r *RetryOnce) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.once = sync.Once{}
	r.done = false
	r.err = nil
}
