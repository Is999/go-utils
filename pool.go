package utils

import "sync"

type Pool[T any] struct {
	pool sync.Pool
	// 可选：重置函数，用于清理对象状态
	reset func(*T)
}

// PoolOption 对象池配置项
type PoolOption[T any] func(*Pool[T])

// WithPoolReset 设置对象重置函数
func WithPoolReset[T any](resetFn func(*T)) PoolOption[T] {
	return func(p *Pool[T]) {
		p.reset = resetFn
	}
}

func NewPool[T any](newFn func() *T, opts ...PoolOption[T]) *Pool[T] {
	p := &Pool[T]{
		pool: sync.Pool{
			New: func() interface{} {
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

// Get 获取对象（自动类型转换）
func (p *Pool[T]) Get() *T {
	return p.pool.Get().(*T)
}

// Put 放回对象（可选执行重置逻辑）
func (p *Pool[T]) Put(x *T) {
	if p.reset != nil {
		p.reset(x)
	}
	p.pool.Put(x)
}
