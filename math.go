package utils

import (
	"math"
	"math/rand/v2"
	"sync"
)

// randSourceMu 保护随机数源的互斥锁。
var randSourceMu sync.Mutex

// Rand 返回 min ~ max 之间的随机数，返回值可能包含 min 和 max。
//
// 参数说明：
//   - minInt：最小值
//   - maxInt：最大值
//   - r：可选的随机数生成器，批量生成时传入 r 参数可提升生成随机数效率
//
// 返回值：min ~ max 之间的随机整数
func Rand(minInt, maxInt int64, r ...*rand.Rand) int64 {
	if minInt == maxInt {
		return minInt
	}
	if minInt > maxInt {
		minInt, maxInt = maxInt, minInt
	}
	span := uint64(maxInt) - uint64(minInt) + 1
	if span == 0 {
		// 当调用方传入完整 int64 空间时，区间大小为 2^64，无法作为 Uint64N 参数，直接使用全量 uint64 映射。
		return int64(randUint64(r...))
	}
	return int64(uint64(minInt) + randUint64N(span, r...))
}

// randUint64 生成全范围 uint64 随机值，用于覆盖完整 int64 区间这类无法计算上界的边界场景。
func randUint64(r ...*rand.Rand) uint64 {
	if len(r) == 0 || r[0] == nil {
		return rand.Uint64()
	}
	randSourceMu.Lock()
	value := r[0].Uint64()
	randSourceMu.Unlock()
	return value
}

// randUint64N 生成 [0,n) 范围内的随机 uint64。
//
// 该函数使用 rand/v2 的 Uint64N 避免取模偏差；当调用方传入共享随机源时继续使用全局锁，
// 业务意图是兼容既有 RandSource 的并发调用方式。
func randUint64N(n uint64, r ...*rand.Rand) uint64 {
	if len(r) == 0 || r[0] == nil {
		return rand.Uint64N(n)
	}
	randSourceMu.Lock()
	value := r[0].Uint64N(n)
	randSourceMu.Unlock()
	return value
}

// Round 对 num 进行四舍五入，并保留指定小数位。
//
// 参数说明：
//   - num：待处理的浮点数
//   - precision：保留的小数位数（可以为负数）
//
// 返回值：四舍五入后的浮点数
func Round(num float64, precision int) float64 {
	aux := math.Pow(10, math.Abs(float64(precision)))
	if precision >= 0 {
		return math.Round(num*aux) / aux
	}
	return math.Round(num/aux) * aux
}
