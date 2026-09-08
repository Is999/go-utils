package utils

import "slices"

// Contains 按 == 比较元素；nil 或空切片返回 false。
func Contains[T comparable](v T, s []T) bool {
	return slices.Contains(s, v)
}

// HasCount 按 == 统计 v 的出现次数；nil 或空切片返回 0。
func HasCount[T comparable](v T, s []T) (count int) {
	for i := range s {
		if v == s[i] {
			count++
		}
	}
	return
}

// Reverse 原地反转 s 并返回同一切片；共享底层数组的切片也会看到变化。
func Reverse[T any](s []T) []T {
	slices.Reverse(s)
	return s
}

// Unique 按 == 去重并保留首次出现顺序，元素本身不做深拷贝。
// 返回独立结果切片，空输入返回非 nil 切片。
func Unique[T comparable](s []T) []T {
	out := make([]T, 0, len(s))
	seen := make(map[T]struct{}, len(s))
	for i := range s {
		if _, exists := seen[s[i]]; exists {
			continue
		}
		seen[s[i]] = struct{}{}
		out = append(out, s[i])
	}
	return out
}

// Diff 返回 s1 中不在 s2 中的元素，保留 s1 的顺序和重复值。
// 返回独立结果切片，空结果为非 nil 切片。
func Diff[T comparable](s1, s2 []T) []T {
	out := make([]T, 0, len(s1))
	// 没有待筛选元素时，无需扫描 s2。
	if len(s1) == 0 {
		return out
	}
	// 排除集合为空时只需复制，不对元素做哈希比较。
	if len(s2) == 0 {
		return append(out, s1...)
	}
	// 预先索引排除集合，避免两个切片逐项交叉查找。
	excluded := make(map[T]struct{}, len(s2))
	for i := range s2 {
		excluded[s2[i]] = struct{}{}
	}
	for i := range s1 {
		if _, ok := excluded[s1[i]]; !ok {
			out = append(out, s1[i])
		}
	}
	return out
}

// Intersect 返回 s1 中也在 s2 中的元素，保留 s1 的顺序和重复值。
// 返回独立结果切片，空结果为非 nil 切片。
func Intersect[T comparable](s1, s2 []T) []T {
	out := make([]T, 0, len(s1))
	// 任一集合为空时交集已确定，无需扫描另一侧。
	if len(s1) == 0 || len(s2) == 0 {
		return out
	}
	// 预先索引命中集合，避免两个切片逐项交叉查找。
	included := make(map[T]struct{}, len(s2))
	for i := range s2 {
		included[s2[i]] = struct{}{}
	}
	for i := range s1 {
		if _, ok := included[s1[i]]; ok {
			out = append(out, s1[i])
		}
	}
	return out
}

// SumSlice 按切片顺序求和，空切片返回 0；整数溢出沿用元素类型的运算规则。
func SumSlice[T Number](nums []T) T {
	var sum T
	for _, v := range nums {
		sum += v
	}
	return sum
}
