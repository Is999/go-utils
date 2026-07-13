package utils

import (
	"math"
	"math/rand"
	"sync"
)

// randSourceMu 保护随机数源的互斥锁。
var randSourceMu sync.Mutex

// Rand 返回 min ~ max 之间的随机数，返回值可能包含 min 和 max。
func Rand(minInt, maxInt int64, r ...*rand.Rand) int64 {
	if minInt == maxInt {
		return minInt
	}
	if minInt > maxInt {
		minInt, maxInt = maxInt, minInt
	}
	span := uint64(maxInt) - uint64(minInt) + 1
	if len(r) == 0 || r[0] == nil {
		if span == 0 {
			return int64(rand.Uint64())
		}
		return int64(uint64(minInt) + randUint64N(span, rand.Uint64))
	}

	randSourceMu.Lock()
	if span == 0 {
		value := r[0].Uint64()
		randSourceMu.Unlock()
		return int64(value)
	}
	value := randUint64N(span, r[0].Uint64)
	randSourceMu.Unlock()
	return int64(uint64(minInt) + value)
}

// randUint64N 返回 [0, n) 内的随机数。
func randUint64N(n uint64, next func() uint64) uint64 {
	limit := ^uint64(0) - (^uint64(0) % n)
	for {
		v := next()
		if v < limit {
			return v % n
		}
	}
}

// Round 对 num 进行四舍五入，并保留指定小数位。
func Round(num float64, precision int) float64 {
	aux := math.Pow(10, math.Abs(float64(precision)))
	if precision >= 0 {
		return math.Round(num*aux) / aux
	}
	return math.Round(num/aux) * aux
}
