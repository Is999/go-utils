package utils

import "sync"

// Pool 是带可选重置回调的泛型对象池。
//
// 适用场景：
//   - 频繁复用临时对象，降低 GC 压力。
//   - 需要在对象归还前清理内部状态。
type Pool[T any] struct {
	pool sync.Pool
	// 可选：重置函数，用于清理对象状态
	reset func(*T)
}

// PoolOption 对象池配置项。
type PoolOption[T any] func(*Pool[T])

// WithPoolReset 设置对象重置函数。
//
// 参数说明：
//   - resetFn：对象归还前的状态清理函数
func WithPoolReset[T any](resetFn func(*T)) PoolOption[T] {
	return func(p *Pool[T]) {
		p.reset = resetFn
	}
}

// NewPool 创建泛型对象池。
//
// 参数说明：
//   - newFn：对象工厂函数；为空时回退为 `new(T)`
//   - opts：对象池配置项
//
// 返回值：泛型对象池指针
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

// Get 获取对象。
//
// 返回值：池中对象指针
func (p *Pool[T]) Get() *T {
	return p.pool.Get().(*T)
}

// Put 放回对象。
// 若配置了重置函数，会先执行清理逻辑再归还对象。
//
// 参数说明：
//   - x：待归还对象
func (p *Pool[T]) Put(x *T) {
	if x == nil {
		return
	}
	if p.reset != nil {
		p.reset(x)
	}
	p.pool.Put(x)
}
