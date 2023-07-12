package utils

import (
	"reflect"
	"testing"
)

func TestBubbleSort(t *testing.T) {
	// golang ide 2022.3.3版本之前 嵌套式结构体使用泛型会显示语法错误，但不影响编译运行
	type args[T Number] struct {
		arr []T
	}

	type testCase[T Number] struct {
		name string
		args args[T]
		want []T
	}

	// int 类型数据测试
	tests := []testCase[int]{
		{name: "int-001", args: args[int]{arr: []int{10, 5, 18, 1, 2, 6, 3, 16, 0, 4, 12, 9, 8, 9, 7}}, want: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}},
		{name: "int-002", args: args[int]{arr: []int{18, 16, 12, 10, 9, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}}, want: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}},
		{name: "int-003", args: args[int]{arr: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}}, want: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}},
		{name: "int-004", args: args[int]{arr: []int{0, 1, 2, 3, 4, 5, 6, 7, -1, 8, 9, 10, 12}}, want: []int{-1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12}},
		{name: "int-005", args: args[int]{arr: []int{0, -1, 2, 3, 4, 5, 6, 7, 7, 8, 9, 10, 12}}, want: []int{-1, 0, 2, 3, 4, 5, 6, 7, 7, 8, 9, 10, 12}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			BubbleSort(tt.args.arr)
			if !reflect.DeepEqual(tt.args.arr, tt.want) {
				t.Errorf("bubbleTest() = %v, want %v", tt.args.arr, tt.want)
			}
		})
	}

	// Float 类型数据测试
	tests1 := []testCase[float64]{
		{name: "Float-001", args: args[float64]{arr: []float64{10, 5, 18.32, 1, 2, 6.09, 3, 16, 0, 4.12, 12, 9.0, 8, 9.01, 7}}, want: []float64{0, 1, 2, 3, 4.12, 5, 6.09, 7, 8, 9.0, 9.01, 10, 12, 16, 18.32}},
		{name: "Float-002", args: args[float64]{arr: []float64{18.32, 16, 12, 10, 9.0, 9.01, 8, 7, 6.09, 5, 4.12, 3, 2, 1, 0}}, want: []float64{0, 1, 2, 3, 4.12, 5, 6.09, 7, 8, 9.0, 9.01, 10, 12, 16, 18.32}},
		{name: "Float-003", args: args[float64]{arr: []float64{0, 1, 2, 3, 4.12, 5, 6.09, 7, 8, 9.0, 9.01, 10, 12, 16, 18.32}}, want: []float64{0, 1, 2, 3, 4.12, 5, 6.09, 7, 8, 9.0, 9.01, 10, 12, 16, 18.32}},
		{name: "Float-004", args: args[float64]{arr: []float64{0, 1, 2, 3, 4, 5, 6, 7, -1, 8, 9, 10, 12}}, want: []float64{-1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12}},
		{name: "Float-005", args: args[float64]{arr: []float64{0, -1, 2, 3, 4, 5, 6.05, 7, 7.0, 8, 9, 10, 12}}, want: []float64{-1, 0, 2, 3, 4, 5, 6.05, 7, 7, 8, 9, 10, 12}},
	}

	for _, tt1 := range tests1 {
		t.Run(tt1.name, func(t *testing.T) {
			BubbleSort(tt1.args.arr)
			if !reflect.DeepEqual(tt1.args.arr, tt1.want) {
				t.Errorf("bubbleTest() = %v, want %v", tt1.args.arr, tt1.want)
			}
		})
	}
}

func TestBubble_RSort(t *testing.T) {
	type args[T Number] struct {
		arr []T
	}

	tests := []struct {
		name string
		args args[int]
		want []int
	}{
		{name: "001", args: args[int]{arr: []int{10, 5, 18, 1, 2, 6, 3, 16, 0, 4, 12, 9, 8, 9, 7}}, want: []int{18, 16, 12, 10, 9, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}},
		{name: "002", args: args[int]{arr: []int{18, 16, 12, 10, 9, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}}, want: []int{18, 16, 12, 10, 9, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}},
		{name: "003", args: args[int]{arr: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}}, want: []int{18, 16, 12, 10, 9, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}},
		{name: "004", args: args[int]{arr: []int{0, 1, 2, 3, 4, 5, 6, 7, -1, 8, 9, 10, 12}}, want: []int{12, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0, -1}},
		{name: "005", args: args[int]{arr: []int{0, -1, 2, 3, 4, 5, 6, 7, 7, 8, 9, 10, 12}}, want: []int{12, 10, 9, 8, 7, 7, 6, 5, 4, 3, 2, 0, -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Bubble[int](tt.args.arr).RSort()
			if !reflect.DeepEqual(tt.args.arr, tt.want) {
				t.Errorf("RSort() = %v, want %v", tt.args.arr, tt.want)
			}
		})
	}
}

func TestBubble_Sort(t *testing.T) {
	type args[T Number] struct {
		arr []T
	}

	tests := []struct {
		name string
		args args[int]
		want []int
	}{
		{name: "001", args: args[int]{arr: []int{10, 5, 18, 1, 2, 6, 3, 16, 0, 4, 12, 9, 8, 9, 7}}, want: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}},
		{name: "002", args: args[int]{arr: []int{18, 16, 12, 10, 9, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}}, want: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}},
		{name: "003", args: args[int]{arr: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}}, want: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}},
		{name: "004", args: args[int]{arr: []int{0, 1, 2, 3, 4, 5, 6, 7, -1, 8, 9, 10, 12}}, want: []int{-1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12}},
		{name: "005", args: args[int]{arr: []int{0, -1, 2, 3, 4, 5, 6, 7, 7, 8, 9, 10, 12}}, want: []int{-1, 0, 2, 3, 4, 5, 6, 7, 7, 8, 9, 10, 12}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Bubble[int](tt.args.arr)
			b.Sort()
			if !reflect.DeepEqual(tt.args.arr, tt.want) {
				t.Errorf("Sort() = %v, want %v", tt.args.arr, tt.want)
			}
		})
	}
}
