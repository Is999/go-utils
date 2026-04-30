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
	return randInt64(r...)%(maxInt-minInt+1) + minInt
}

// randInt64 生成随机 int64 值。
//
// 参数说明：
//   - r：可选的随机数生成器
//
// 返回值：随机 int64 值
func randInt64(r ...*rand.Rand) int64 {
	if len(r) == 0 || r[0] == nil {
		return rand.Int64()
	}
	randSourceMu.Lock()
	n := r[0].Int64()
	randSourceMu.Unlock()
	return n
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
