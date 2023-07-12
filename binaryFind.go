package utils

// Binary 二分查找
//
//	s 检索值
//	arr 检索的 slice，必须是有序的
func Binary[T Number](s T, arr []T) int {
	m := 0            // 中间下标
	l := 0            // 左移动下标
	r := len(arr) - 1 // 右移动下标

	asc := true // 标记是正序还是倒序
	if arr[l] > arr[r] {
		asc = false
	}

	// 左移下标大于右移下标结束查找
	for l <= r {
		// 计算中间下标
		m = (l + r) / 2

		// 找到返回下标
		if arr[m] == s {
			return m
		}

		// 中间值小于要查找的值, 左移下标. 否则右移下标
		if arr[m] < s {
			if asc {
				l = m + 1
			} else {
				r = m - 1
			}
		} else {
			if asc {
				r = m - 1
			} else {
				l = m + 1
			}
		}
	}

	return -1
}
