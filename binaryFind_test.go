package utils

import (
	"testing"
)

func TestBinary(t *testing.T) {
	type args[T Number] struct {
		s   T
		arr []T
	}
	tests := []struct {
		name string
		args args[int]
		want int
	}{
		// 正序
		{name: "001", args: args[int]{arr: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}, s: 16}, want: 13},
		{name: "002", args: args[int]{arr: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}, s: 1}, want: 1},
		{name: "003", args: args[int]{arr: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}, s: 20}, want: -1},
		{name: "004", args: args[int]{arr: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}, s: 7}, want: 7},
		{name: "005", args: args[int]{arr: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 12, 16, 18}, s: 9}, want: 9},
		// 倒序
		{name: "006", args: args[int]{arr: []int{18, 16, 12, 11, 10, 9, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}, s: 9}, want: 5},
		{name: "007", args: args[int]{arr: []int{18, 16, 12, 11, 10, 9, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}, s: 7}, want: 8},
		{name: "008", args: args[int]{arr: []int{18, 16, 12, 11, 10, 9, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}, s: 20}, want: -1},
		{name: "009", args: args[int]{arr: []int{18, 16, 12, 11, 10, 9, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}, s: 1}, want: 14},
		{name: "010", args: args[int]{arr: []int{18, 16, 12, 11, 10, 9, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}, s: 18}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Binary(tt.args.s, tt.args.arr); got != tt.want {
				t.Errorf("Binary() = %v, want %v", got, tt.want)
			}
		})
	}
}
