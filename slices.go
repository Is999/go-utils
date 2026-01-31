package utils

// IsHas 检查s中是否存在v。 1.21版本以上推荐使用标准库 slices.Contains(s,v)
func IsHas[T Ordered](v T, s []T) bool {
	for i := 0; i < len(s); i++ {
		if v == s[i] {
			return true
		}
	}
	return false
}

// HasCount 统计v在s中出现次数
func HasCount[T Ordered](v T, s []T) (count int) {
	for i := 0; i < len(s); i++ {
		if v == s[i] {
			count++
		}
	}
	return
}

// Reverse 反转s 1.21版本以上推荐使用标准库 slices.Reverse(s)
func Reverse[T Ordered](s []T) []T {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

// Unique 去除s中重复的值
func Unique[T Ordered](s []T) []T {
	m := make(map[T]struct{}, len(s))
	a := make([]T, 0, len(s))
	for _, v := range s {
		if _, exists := m[v]; !exists {
			m[v] = struct{}{}
			a = append(a, v)
		}
	}

	return a[:]
}

// Diff 计算s1与s2的差集即s1中有而s2中没有的元素
func Diff[T Ordered](s1, s2 []T) []T {
	m := make(map[T]struct{}, len(s2))
	for i := 0; i < len(s2); i++ {
		m[s2[i]] = struct{}{}
	}
	var s = make([]T, 0, len(s1))
	for i := 0; i < len(s1); i++ {
		if _, ok := m[s1[i]]; !ok {
			s = append(s, s1[i])
		}
	}
	return s
}

// Intersect 计算s1与s2的交集即s1中有而s2中也有的元素
func Intersect[T Ordered](s1, s2 []T) []T {
	m := make(map[T]struct{}, len(s2))
	for i := 0; i < len(s2); i++ {
		m[s2[i]] = struct{}{}
	}
	var s = make([]T, 0, len(s1))
	for i := 0; i < len(s1); i++ {
		if _, ok := m[s1[i]]; ok {
			s = append(s, s1[i])
		}
	}
	return s
}

// SumSlice 求和
func SumSlice[T Number](nums []T) T {
	var sum T
	for _, v := range nums {
		sum += v
	}
	return sum
}
