package utils

import "sync"

// Pool 复用临时对象，缓存可随 GC 丢弃，不保证归还后还能取到同一对象。
// 通过 NewPool 创建，可并发取还；对象由取出方独占，Pool 使用后不可复制。
type Pool[T any] struct {
	pool  sync.Pool // 对象生命周期和缓存回收沿用标准库规则。
	reset func(*T)  // 可选回调，在对象放回池前执行；不同 Put 可并发调用。
}

// PoolOption 对象池配置项。
type PoolOption[T any] func(*Pool[T])

// WithPoolReset 设置归还前的重置函数；回调共享的外部状态由调用方同步。
func WithPoolReset[T any](resetFn func(*T)) PoolOption[T] {
	return func(p *Pool[T]) {
		p.reset = resetFn
	}
}

// NewPool 创建对象池；newFn 为 nil 时使用 new(T)，nil 选项会被忽略。
func NewPool[T any](newFn func() *T, opts ...PoolOption[T]) *Pool[T] {
	if newFn == nil {
		newFn = func() *T {
			return new(T)
		}
	}

	p := &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return newFn()
			},
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}
	return p
}

// Get 获取可独占使用的对象；池中无对象时调用创建函数。
func (p *Pool[T]) Get() *T {
	return p.pool.Get().(*T)
}

// Put 重置并归还对象，忽略 nil；归还后调用方不得继续使用或重复归还该对象。
func (p *Pool[T]) Put(x *T) {
	if x == nil {
		return
	}
	if p.reset != nil {
		p.reset(x)
	}
	p.pool.Put(x)
}
