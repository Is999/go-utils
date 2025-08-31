package utils

import (
	"sort"
)

// MapKeys 获取map的所有key
func MapKeys[K Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// MapValues 有序获取map的所有value，对map的key排序并按排序后的key返回其value
//
//	isReverse 是否降序排列：true 降序，false 升序
func MapValues[K Ordered, V any](m map[K]V, isReverse ...bool) []V {
	var keys Slice[K] = MapKeys(m)

	// 排序
	if len(isReverse) > 0 && isReverse[0] {
		sort.Sort(sort.Reverse(keys))
	} else {
		sort.Sort(keys)
	}

	vals := make([]V, keys.Len())
	for i, key := range keys {
		vals[i] = m[key]
	}
	return vals
}

// MapRange 有序遍历map元素，对map的key排序并按排序后的key遍历m
//
//	f 函数接收key与value，返回一个bool值，如果f函数返回false则终止遍历
//	isReverse 是否降序排列：true 降序，false 升序
func MapRange[K Ordered, V any](m map[K]V, f func(key K, value V) bool, isReverse ...bool) {
	var keys Slice[K] = MapKeys(m)

	// 排序
	if len(isReverse) > 0 && isReverse[0] {
		sort.Sort(sort.Reverse(keys))
	} else {
		sort.Sort(keys)
	}

	for _, key := range keys {
		if !f(key, m[key]) {
			break
		}
	}
}

// MapFilter 使用回调函数过滤map的元素
//
//	f 函数接收key与value，返回一个bool值，如果f函数返回false则过滤掉该元素（删除该元素）
func MapFilter[K Ordered, V any](m map[K]V, f func(key K, value V) bool) map[K]V {
	for k, v := range m {
		if !f(k, v) {
			delete(m, k)
		}
	}
	return m
}

// MapDiff 计算m1与m2的值差集
func MapDiff[K, V Ordered](m1, m2 map[K]V) []V {
	var s1 = make([]V, 0, len(m1))
	for _, v := range m1 {
		s1 = append(s1, v)
	}
	var s2 = make([]V, 0, len(m2))
	for _, v2 := range m2 {
		s2 = append(s2, v2)
	}
	return Diff(s1, s2)
}

// MapIntersect 计算m1与m2的值交集
func MapIntersect[K, V Ordered](m1, m2 map[K]V) []V {
	var s1 = make([]V, 0, len(m1))
	for _, v := range m1 {
		s1 = append(s1, v)
	}
	var s2 = make([]V, 0, len(m2))
	for _, v2 := range m2 {
		s2 = append(s2, v2)
	}
	return Intersect(s1, s2)
}

// MapDiffKey 计算m1与m2的键差集
func MapDiffKey[K Ordered, V any](m1, m2 map[K]V) []K {
	var s = make([]K, 0, len(m1))
	for k := range m1 {
		if _, ok := m2[k]; !ok {
			s = append(s, k)
		}
	}
	return s
}

// MapIntersectKey 计算m1与m2的键交集
func MapIntersectKey[K Ordered, V any](m1, m2 map[K]V) []K {
	var s = make([]K, 0, len(m1))
	for k := range m1 {
		if _, ok := m2[k]; ok {
			s = append(s, k)
		}
	}
	return s
}

// SumMap 计算map的值和
func SumMap[K Ordered, V Number](m map[K]V) V {
	var sum V
	for _, v := range m {
		sum += v
	}
	return sum
}
