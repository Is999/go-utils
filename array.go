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

// ArrUnique 去除s中重复的值
func ArrUnique[T Ordered](s []T) []T {
	m := make(map[T]struct{}, len(s))
	for i := 0; i < len(s); i++ {
		if _, ok := m[s[i]]; !ok {
			m[s[i]] = struct{}{}
		}
	}

	a := make([]T, 0, len(m))
	for k := range m {
		a = append(a, k)
	}

	return a
}

// ArrDiff 计算s1与s2的差集
func ArrDiff[T Ordered](s1, s2 []T) []T {
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

// ArrIntersect 计算s1与s2的交集
func ArrIntersect[T Ordered](s1, s2 []T) []T {
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
