package utils

import "slices"

// MapKeys 返回顺序未定的键切片；nil 或空 map 返回非 nil 空切片。
func MapKeys[K Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// MapValues 返回值切片；不传 isReverse 时顺序未定，false 按键升序，true 按键降序。
// 只使用 isReverse 第一项；排序时浮点键不能含 NaN，空 map 返回非 nil 空切片。
// 仅复制值本身，不复制值引用的底层数据。
func MapValues[K Ordered, V any](m map[K]V, isReverse ...bool) []V {
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

// MapRange 按键升序调用 f；isReverse 第一项为 true 时降序，f 返回 false 时停止。
// 开始前固定键集合，每次回调前读取当前值；回调增删元素不会更新键列表。
// 浮点键不能含 NaN；空 map 不调用 f。
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

// MapFilter 原地删除 f 返回 false 的元素，并返回同一个 map。
// 回调顺序未定；nil map 原样返回且不调用 f。
func MapFilter[K Ordered, V any](m map[K]V, f func(key K, value V) bool) map[K]V {
	for k, v := range m {
		if !f(k, v) {
			delete(m, k)
		}
	}
	return m
}

// MapDiff 计算 m1 与 m2 的值差集，即 m1 中有但 m2 中没有的值。
// 结果顺序未定，保留来自不同键的重复值；空结果为非 nil 切片。
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
// 结果顺序未定，保留来自不同键的重复值；空结果为非 nil 切片。
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

// MapDiffKey 返回 m1 中有而 m2 中没有的键；顺序未定，空结果为非 nil 切片。
func MapDiffKey[K Ordered, V any](m1, m2 map[K]V) []K {
	s := make([]K, 0, len(m1))
	for k := range m1 {
		if _, ok := m2[k]; !ok {
			s = append(s, k)
		}
	}
	return s
}

// MapIntersectKey 返回两者都有的键；顺序未定，空结果为非 nil 切片。
func MapIntersectKey[K Ordered, V any](m1, m2 map[K]V) []K {
	s := make([]K, 0, len(m1))
	for k := range m1 {
		if _, ok := m2[k]; ok {
			s = append(s, k)
		}
	}
	return s
}

// SumMap 计算值的和，空 map 返回 0；浮点累加顺序随 map 遍历顺序而定。
func SumMap[K comparable, V Number](m map[K]V) V {
	var sum V
	for _, v := range m {
		sum += v
	}
	return sum
}
