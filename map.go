package utils

import "slices"

// MapKeys 获取map的所有key
func MapKeys[K Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// MapValues 获取 map 的所有 value。
//
//	isReverse 是否降序排列：true 降序，false 升序，未指定则不排序直接返回所有 value
func MapValues[K Ordered, V any](m map[K]V, isReverse ...bool) []V {
	// 未指定排序则直接返回所有 value
	if len(isReverse) == 0 {
		vals := make([]V, 0, len(m))
		for _, v := range m {
			vals = append(vals, v)
		}
		return vals
	}

	keys := MapKeys(m)
	slices.Sort(keys)
	if isReverse[0] {
		slices.Reverse(keys)
	}

	vals := make([]V, len(keys))
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
	keys := MapKeys(m)
	slices.Sort(keys)
	if len(isReverse) > 0 && isReverse[0] {
		slices.Reverse(keys)
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

// MapDiff 计算 m1 与 m2 的值差集，即 m1 中有但 m2 中没有的值。
// 返回结果保留 m1 原有遍历结果中的重复值。
func MapDiff[K comparable, V comparable](m1, m2 map[K]V) []V {
	set := make(map[V]struct{}, len(m2))
	for _, v := range m2 {
		set[v] = struct{}{}
	}

	result := make([]V, 0, len(m1))
	for _, v := range m1 {
		if _, ok := set[v]; !ok {
			result = append(result, v)
		}
	}
	return result
}

// MapIntersect 计算 m1 与 m2 的值交集，即 m1 与 m2 都有的值。
// 返回结果保留 m1 原有遍历结果中的重复值。
func MapIntersect[K comparable, V comparable](m1, m2 map[K]V) []V {
	set := make(map[V]struct{}, len(m2))
	for _, v := range m2 {
		set[v] = struct{}{}
	}

	result := make([]V, 0, len(m1))
	for _, v := range m1 {
		if _, ok := set[v]; ok {
			result = append(result, v)
		}
	}
	return result
}

// MapDiffKey 计算m1与m2的键差集即m1中有但m2中没有的键
func MapDiffKey[K Ordered, V any](m1, m2 map[K]V) []K {
	s := make([]K, 0, len(m1))
	for k := range m1 {
		if _, ok := m2[k]; !ok {
			s = append(s, k)
		}
	}
	return s
}

// MapIntersectKey 计算m1与m2的键交集即m1与m2都有的键
func MapIntersectKey[K Ordered, V any](m1, m2 map[K]V) []K {
	s := make([]K, 0, len(m1))
	for k := range m1 {
		if _, ok := m2[k]; ok {
			s = append(s, k)
		}
	}
	return s
}

// SumMap 计算 map 的值和。
func SumMap[K comparable, V Number](m map[K]V) V {
	var sum V
	for _, v := range m {
		sum += v
	}
	return sum
}
