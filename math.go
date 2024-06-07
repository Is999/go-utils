package utils

import (
	"math"
	"math/rand/v2"
)

// Rand 返回min~max之间的随机数，值可能包含min和max
//
//	minInt 最小值
//	maxInt 最大值
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func Rand(minInt, maxInt int64, r ...*rand.Rand) int64 {
	if minInt == maxInt {
		return minInt
	}
	if minInt > maxInt {
		minInt, maxInt = maxInt, minInt
	}
	if len(r) == 0 {
		source := GetRandPool()
		defer RandPool.Put(source)
		r = append(r, source)
	}
	return r[0].Int64()%(maxInt-minInt+1) + minInt
}

// Round 对num进行四舍五入，并保留指定小数位
//
//	precision 保留小数位
func Round(num float64, precision int) float64 {
	aux := math.Pow(10, math.Abs(float64(precision)))
	if precision >= 0 {
		return math.Round(num*aux) / aux
	} else {
		return math.Round(num/aux) * aux
	}
}
