package utils

import (
	"math"
	"math/rand/v2"
	"sync"
)

// randSourceMu 串行化本包对调用方传入的随机源的访问，不保护调用方直接使用该源。
var randSourceMu sync.Mutex

// Rand 返回闭区间 [minInt, maxInt] 内的随机整数；上下界颠倒时自动交换。
// 只使用 r 第一项，省略或 nil 时用全局随机源；自定义源经包级锁串行访问。
func Rand(minInt, maxInt int64, r ...*rand.Rand) int64 {
	if minInt == maxInt {
		return minInt
	}
	if minInt > maxInt {
		minInt, maxInt = maxInt, minInt
	}
	// 整个 int64 区间的宽度为 2^64，在 uint64 中溢出为 0。
	span := uint64(maxInt) - uint64(minInt) + 1
	if len(r) == 0 || r[0] == nil {
		if span == 0 {
			return int64(rand.Uint64())
		}
		return int64(uint64(minInt) + rand.Uint64N(span))
	}

	randSourceMu.Lock()
	if span == 0 {
		value := r[0].Uint64()
		randSourceMu.Unlock()
		return int64(value)
	}
	value := r[0].Uint64N(span)
	randSourceMu.Unlock()
	return int64(uint64(minInt) + value)
}

// Round 按十进制位舍入，恰好居中时远离零；负 precision 表示舍入整数位。
// 使用 float64 运算，结果仍受浮点表示与溢出规则影响。
func Round(num float64, precision int) float64 {
	aux := math.Pow(10, math.Abs(float64(precision)))
	if precision >= 0 {
		return math.Round(num*aux) / aux
	}
	return math.Round(num/aux) * aux
}
