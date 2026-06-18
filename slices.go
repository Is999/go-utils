package utils

const (
	// smallSliceLinearThreshold 是集合运算切换到线性扫描的规模阈值。
	// 业务意图：短切片通常来自少量标签、状态码、枚举值等场景，直接双层扫描比创建 map 更省分配。
	smallSliceLinearThreshold = 8
)

// IsHas 检查 s 中是否存在 v。1.21 版本以上推荐使用标准库 slices.Contains(s, v)。
func IsHas[T comparable](v T, s []T) bool {
	for i := range s {
		if v == s[i] {
			return true
		}
	}
	return false
}

// HasCount 统计v在s中出现次数
func HasCount[T comparable](v T, s []T) (count int) {
	for i := range s {
		if v == s[i] {
			count++
		}
	}
	return
}

// Reverse 反转 s。1.21 版本以上推荐使用标准库 slices.Reverse(s)。
func Reverse[T any](s []T) []T {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

// Unique 去除 s 中重复的值。
// 返回结果保持首个元素出现顺序；返回切片使用新的底层数组，避免调用方误改原始数据。
func Unique[T comparable](s []T) []T {
	if len(s) == 0 {
		return make([]T, 0)
	}
	return UniqueInto(make([]T, 0, len(s)), s)
}

// UniqueInto 将 s 去重后追加到 dst，并返回复用后的结果切片。
// dst 的历史内容会被清空但容量会被复用；适合循环内批量处理时降低临时切片分配。
func UniqueInto[T comparable](dst, s []T) []T {
	// out 是本次返回结果，数据来源为调用方传入的 s，边界为保留每个值第一次出现的位置。
	out := dst[:0]
	if len(s) == 0 {
		return out
	}
	if len(s) <= smallSliceLinearThreshold {
		for i := range s {
			if !containsComparable(out, s[i]) {
				out = append(out, s[i])
			}
		}
		return out
	}

	// seen 记录已经输出过的元素；只在中大切片启用，避免短切片为了 map 付出固定分配成本。
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

// UniqueInPlace 在 s 的原始底层数组上完成去重并返回结果切片。
// 调用方明确不再需要原始顺序完整数据时，可用该入口省掉结果切片分配；返回后 s 尾部旧数据不可再视为有效结果。
func UniqueInPlace[T comparable](s []T) []T {
	return UniqueInto(s[:0], s)
}

// Diff 计算 s1 与 s2 的差集，即 s1 中有而 s2 中没有的元素。
// 返回结果保持 s1 原有顺序，并保留 s1 中原本存在的重复值。
func Diff[T comparable](s1, s2 []T) []T {
	if len(s1) == 0 {
		return make([]T, 0)
	}
	return DiffInto(make([]T, 0, len(s1)), s1, s2)
}

// DiffInto 将 s1 与 s2 的差集追加到 dst，并返回复用后的结果切片。
// s2 被视为排除集合；返回结果保持 s1 顺序与重复值，适合权限、标签、状态列表等保序过滤场景。
func DiffInto[T comparable](dst, s1, s2 []T) []T {
	// out 复用调用方容量承载结果；边界为 s1 为空时直接返回空结果，不创建排除集合。
	out := dst[:0]
	if len(s1) == 0 {
		return out
	}
	if len(s2) == 0 {
		return append(out, s1...)
	}
	if len(s2) <= smallSliceLinearThreshold {
		for i := range s1 {
			if !containsComparable(s2, s1[i]) {
				out = append(out, s1[i])
			}
		}
		return out
	}

	// excluded 记录 s2 中需要剔除的值；只在排除集合足够大时构建，避免短集合固定分配拖慢热路径。
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

// Intersect 计算 s1 与 s2 的交集，即 s1 中有而 s2 中也有的元素。
// 返回结果保持 s1 原有顺序，并保留 s1 中原本存在的重复值。
func Intersect[T comparable](s1, s2 []T) []T {
	if len(s1) == 0 {
		return make([]T, 0)
	}
	return IntersectInto(make([]T, 0, len(s1)), s1, s2)
}

// IntersectInto 将 s1 与 s2 的交集追加到 dst，并返回复用后的结果切片。
// s2 被视为命中集合；返回结果保持 s1 顺序与重复值，适合按白名单过滤但不打乱业务排序的场景。
func IntersectInto[T comparable](dst, s1, s2 []T) []T {
	// out 复用调用方容量承载结果；任一输入为空时交集为空，可直接降级返回。
	out := dst[:0]
	if len(s1) == 0 || len(s2) == 0 {
		return out
	}
	if len(s2) <= smallSliceLinearThreshold {
		for i := range s1 {
			if containsComparable(s2, s1[i]) {
				out = append(out, s1[i])
			}
		}
		return out
	}

	// included 记录 s2 中允许保留的值；只在命中集合较大时构建，降低小集合过滤时的分配压力。
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

// containsComparable 在线性集合中查找 target。
// 该方法仅服务短切片降级路径，业务边界是 smallSliceLinearThreshold 内避免创建 map。
func containsComparable[T comparable](s []T, target T) bool {
	for i := range s {
		if s[i] == target {
			return true
		}
	}
	return false
}

// SumSlice 求和
func SumSlice[T Number](nums []T) T {
	var sum T
	for _, v := range nums {
		sum += v
	}
	return sum
}
