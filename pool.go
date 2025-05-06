package utils

import "sync"

type Pool[T any] struct {
	pool sync.Pool
	// 可选：重置函数，用于清理对象状态
	reset func(*T)
}

func NewPool[T any](newFn func() *T, resetFn func(*T)) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return newFn()
			},
		},
		reset: resetFn,
	}
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
