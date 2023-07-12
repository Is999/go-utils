package utils

import (
	"math"
	"math/rand"
)

// Rand 返回min~max之间的随机数，值可能包含min和max
//
//	min 最小值
//	max 最大值
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func Rand(min, max int64, r ...rand.Source) int64 {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}
	if len(r) == 0 {
		source := sourcePool.Get().(rand.Source)
		defer sourcePool.Put(source)
		r = append(r, source)
	}
	return r[0].Int63()%(max-min+1) + min
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

// Max 返回nums中最大值
func Max[T Number](nums ...T) T {
	length := len(nums)
	var max T
	if length > 0 {
		max = nums[0]
		for i := 1; i < length; i++ {
			// max = math.Max(max, nums[i])
			max = Ternary(max >= nums[i], max, nums[i])
		}
	}
	return max
}

// Min 返回nums中最小值
func Min[T Number](nums ...T) T {
	length := len(nums)
	var min T
	if length > 0 {
		min = nums[0]
		for i := 1; i < length; i++ {
			// min = math.Min(min, nums[i])
			min = Ternary(min <= nums[i], min, nums[i])
		}
	}
	return min
}
