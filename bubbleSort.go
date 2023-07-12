package utils

// Bubble 冒泡排序
type Bubble[T Number] []T

// Sort 排序: 正序
func (b Bubble[T]) Sort() {
	BubbleSort(b, false)
}

// RSort 排序: 倒序
func (b Bubble[T]) RSort() {
	BubbleSort(b, true)
}

// BubbleSort 冒泡排序
//
//	arr 需要排序的slice
//	isDesc 是否倒序: true 是; false 否
func BubbleSort[T Number](arr []T, isDesc ...bool) {
	s := len(arr) - 1 // 外循环次数
	k := s            // 内循环次数

	desc := false
	if len(isDesc) > 0 {
		desc = isDesc[0]
	}

	var p int // 标记循环最后一次交换的位置

	for i := 0; i < s; i++ {
		f := 0 // 标记: 每次循环置为0, 内循环有交换置为1
		for j := 0; j < k; j++ {
			if desc && arr[j] < arr[j+1] {
				arr[j], arr[j+1] = arr[j+1], arr[j]
				f = 1 // 有交交换标记置为1
				p = j // 记录交换位置
			} else if !desc && arr[j] > arr[j+1] {
				arr[j], arr[j+1] = arr[j+1], arr[j]
				f = 1 // 有交交换标记置为1
				p = j // 记录交换位置
			}
		}

		k = p // 最后一次交互的位置赋值给内部循环次数变量k

		// 标记为0, 说明后面的元素已经有序, 否则为乱序
		if f == 0 {
			break
		}
	}
}
