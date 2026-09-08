package utils_test

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Is999/go-utils"
)

func TestContains(t *testing.T) {
	type args[T utils.Ordered] struct {
		s   T
		arr []T
	}

	type testCase[T utils.Ordered] struct {
		name string
		args args[T]
		want bool
	}
	tests := []testCase[int]{
		{name: "int-001", args: args[int]{s: -9, arr: []int{10, 5, 18, 1, 2, 6, 3, 16, 0, 4, 12, 9, 8, -9, 7}}, want: true},
		{name: "int-002", args: args[int]{s: 23, arr: []int{10, 5, 18, 1, 2, 6, 3, 16, 0, 4, 12, 9, 8, -9, 7}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Contains(tt.args.s, tt.args.arr); got != tt.want {
				t.Errorf("Has() = %v, want %v", got, tt.want)
			}
		})
	}

	tests1 := []testCase[string]{
		{name: "str-001", args: args[string]{s: "张三", arr: []string{"张三", "李四", "王五", "老六", "赵七"}}, want: true},
		{name: "str-002", args: args[string]{s: "啊九", arr: []string{"张三", "李四", "王五", "老六", "赵七"}}, want: false},
	}
	for _, tt1 := range tests1 {
		t.Run(tt1.name, func(t *testing.T) {
			if got1 := utils.Contains(tt1.args.s, tt1.args.arr); got1 != tt1.want {
				t.Errorf("Contains() = %v, want %v", got1, tt1.want)
			}
		})
	}

	tests2 := []testCase[rune]{
		{name: "rune-001", args: args[rune]{s: '三', arr: []rune{'三', '四', '五', '六', '七'}}, want: true},
		{name: "rune-002", args: args[rune]{s: '九', arr: []rune{'三', '四', '五', '六', '七'}}, want: false},
	}
	for _, tt2 := range tests2 {
		t.Run(tt2.name, func(t *testing.T) {
			if got2 := utils.Contains(tt2.args.s, tt2.args.arr); got2 != tt2.want {
				t.Errorf("Contains() = %v, want %v", got2, tt2.want)
			}
		})
	}

	tests3 := []testCase[byte]{
		{name: "byte-001", args: args[byte]{s: 'A', arr: []byte{'a', 'b', 'C', 'D', '9', '*', 'A'}}, want: true},
		{name: "byte-002", args: args[byte]{s: 'B', arr: []byte{'a', 'b', 'C', 'D', '9', '*', 'A'}}, want: false},
		{name: "byte-003", args: args[byte]{s: 'd', arr: []byte{'a', 'b', 'C', 'D', '9', '*', 'A'}}, want: false},
	}

	for _, tt3 := range tests3 {
		t.Run(tt3.name, func(t *testing.T) {
			if got3 := utils.Contains(tt3.args.s, tt3.args.arr); got3 != tt3.want {
				t.Errorf("Contains() = %v, want %v", got3, tt3.want)
			}
		})
	}
}

func TestHasCount(t *testing.T) {
	type args[T int] struct {
		s   T
		arr []T
	}
	tests := []struct {
		name      string
		args      args[int]
		wantCount int
	}{
		{name: "int-001", args: args[int]{s: 8, arr: []int{10, 5, 18, 1, 2, 6, 3, 16, 0, 4, 12, 9, 8, -9, 7}}, wantCount: 1},
		{name: "int-002", args: args[int]{s: 8, arr: []int{10, 5, 18, 1, 8, 6, 3, 16, 0, 4, 12, 9, 8, -9, 7}}, wantCount: 2},
		{name: "int-003", args: args[int]{s: 2, arr: []int{10, 5, 18, 1, 8, 6, 3, 16, 0, 4, 12, 9, 8, -9, 7}}, wantCount: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotCount := utils.HasCount(tt.args.s, tt.args.arr); gotCount != tt.wantCount {
				t.Errorf("HasCount() = %v, want %v", gotCount, tt.wantCount)
			}
		})
	}
}

// TestUnique 验证整数与字符串的去重结果及首次出现顺序。
func TestUnique(t *testing.T) {
	type args[T utils.Ordered] struct {
		a []T
	}
	type testCase[T utils.Ordered] struct {
		name string
		args args[T]
		want []T // 按首次出现顺序排列的完整结果。
	}
	tests := []testCase[int]{
		{name: "int-000", args: args[int]{a: []int{-0, -1, 1, -2, 2, 0, 1, 2, 3, 4, 5, 6, 7, 7, 8, 9}}, want: []int{0, -1, 1, -2, 2, 3, 4, 5, 6, 7, 8, 9}},
		{name: "int-001", args: args[int]{a: []int{10, 5, 18, 12, 2, 6, 5, 12, -12, 4, 12, 9, 8, -9, 7}}, want: []int{10, 5, 18, 12, 2, 6, -12, 4, 9, 8, -9, 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Unique(tt.args.a); !slices.Equal(got, tt.want) {
				t.Errorf("Unique() = %v, want %v", got, tt.want)
			}
		})
	}

	tests2 := []testCase[string]{
		{name: "string-001", args: args[string]{a: []string{"abc", "cba", "acb", "abc", "acb", "bac", "bca"}}, want: []string{"abc", "cba", "acb", "bac", "bca"}},
	}
	for _, tt2 := range tests2 {
		t.Run(tt2.name, func(t *testing.T) {
			if got2 := utils.Unique(tt2.args.a); !slices.Equal(got2, tt2.want) {
				t.Errorf("Unique() = %v, want %v", got2, tt2.want)
			}
		})
	}
}

// TestEmptySetResults 验证空集合结果仍是非 nil 切片。
func TestEmptySetResults(t *testing.T) {
	for name, got := range map[string][]int{
		"Unique":            utils.Unique[int](nil),
		"Diff":              utils.Diff[int](nil, nil),
		"Intersect":         utils.Intersect[int](nil, nil),
		"DiffExcluded":      utils.Diff([]int{1, 1}, []int{1}),
		"IntersectDisjoint": utils.Intersect([]int{1}, []int{2}),
		"IntersectEmpty":    utils.Intersect([]int{1}, nil),
	} {
		if got == nil || len(got) != 0 {
			t.Errorf("%s = %#v, want non-nil empty slice", name, got)
		}
	}
}

// TestSetResultIsolation 验证修改结果不会覆盖输入切片。
func TestSetResultIsolation(t *testing.T) {
	s1 := []int{3, 1, 3, 2}
	s2 := []int{2, 4}
	for _, tt := range []struct {
		name string
		got  []int
		want []int
	}{
		{name: "Unique", got: utils.Unique(s1), want: []int{3, 1, 2}},
		{name: "Diff", got: utils.Diff(s1, s2), want: []int{3, 1, 3}},
		{name: "DiffEmpty", got: utils.Diff(s1, nil), want: []int{3, 1, 3, 2}},
		{name: "Intersect", got: utils.Intersect(s1, s2), want: []int{2}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if !slices.Equal(tt.got, tt.want) {
				t.Fatalf("result = %v, want %v", tt.got, tt.want)
			}
			tt.got[0] = 99
			if !slices.Equal(s1, []int{3, 1, 3, 2}) || !slices.Equal(s2, []int{2, 4}) {
				t.Fatalf("result changed input: s1=%v s2=%v", s1, s2)
			}
		})
	}
}

// BenchmarkUnique 固定 200 项输入，计量包含结果切片分配的去重成本。
func BenchmarkUnique(b *testing.B) {
	var l = 200
	var s1 = make([]int64, 0, l)
	r := rand.New(rand.NewPCG(1, 2)) // 固定输入，避免每次运行的数据分布不同。
	for range l {
		s1 = append(s1, utils.Rand(int64(l), int64(l)*2, r))
	}
	for b.Loop() {
		utils.Unique(s1)
	}
}

func TestDiff(t *testing.T) {
	type args[T utils.Ordered] struct {
		s1 []T
		s2 []T
	}
	type testCase[T utils.Ordered] struct {
		name string
		args args[T]
		want []T
	}

	tests := []testCase[int]{
		{name: "int-001", args: args[int]{s1: []int{1, 2, 3, 4, 5, 9}, s2: []int{0, 1, 2, 3, 6, 7}}, want: []int{4, 5, 9}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Diff(tt.args.s1, tt.args.s2); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Diff() = %v, want %v", got, tt.want)
			}
		})
	}

	tests2 := []testCase[string]{
		{name: "string-001", args: args[string]{s1: []string{"黄色", "绿色", "红色", "粉色", "白色", "粉色"}, s2: []string{"橙色", "紫色", "黑色", "粉色", "灰色", "绿色"}}, want: []string{"黄色", "红色", "白色"}},
	}
	for _, tt2 := range tests2 {
		t.Run(tt2.name, func(t *testing.T) {
			if got2 := utils.Diff(tt2.args.s1, tt2.args.s2); !reflect.DeepEqual(got2, tt2.want) {
				t.Errorf("Diff() = %+v, want %+v", got2, tt2.want)
			}
		})
	}
}

// BenchmarkDiff 固定两组 200 项输入，包含结果切片的分配。
func BenchmarkDiff(b *testing.B) {
	var l = 200
	var s1 = make([]int64, 0, l)
	var s2 = make([]int64, 0, l)
	r := rand.New(rand.NewPCG(1, 2)) // 固定输入，避免每次运行的数据分布不同。
	for range 200 {
		s1 = append(s1, utils.Rand(int64(l), int64(l)*2, r))
		s2 = append(s2, utils.Rand(int64(l), int64(l)*2, r))
	}
	for b.Loop() {
		utils.Diff(s1, s2)
	}
}

func TestIntersect(t *testing.T) {
	type args[T utils.Ordered] struct {
		s1 []T
		s2 []T
	}
	type testCase[T utils.Ordered] struct {
		name string
		args args[T]
		want []T
	}
	tests := []testCase[int]{
		{name: "int-001", args: args[int]{s1: []int{1, 2, 3, 4, 5}, s2: []int{0, 1, 2, 3, 6, 7}}, want: []int{1, 2, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Intersect(tt.args.s1, tt.args.s2); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Intersect() = %v, want %v", got, tt.want)
			}
		})
	}

	tests2 := []testCase[string]{
		{name: "string-001", args: args[string]{s1: []string{"黄色", "绿色", "红色", "粉色", "白色", "粉色"}, s2: []string{"橙色", "紫色", "黑色", "粉色", "灰色", "绿色"}}, want: []string{"绿色", "粉色", "粉色"}},
	}
	for _, tt2 := range tests2 {
		t.Run(tt2.name, func(t *testing.T) {
			if got2 := utils.Intersect(tt2.args.s1, tt2.args.s2); !reflect.DeepEqual(got2, tt2.want) {
				t.Errorf("Intersect() = %v, want %v", got2, tt2.want)
			}
		})
	}
}

// BenchmarkIntersect 固定两组 200 项输入，包含结果切片的分配。
func BenchmarkIntersect(b *testing.B) {
	var l = 200
	var s1 = make([]int64, 0, l)
	var s2 = make([]int64, 0, l)
	r := rand.New(rand.NewPCG(1, 2)) // 固定输入，避免每次运行的数据分布不同。
	for range 200 {
		s1 = append(s1, utils.Rand(int64(l), int64(l)*2, r))
		s2 = append(s2, utils.Rand(int64(l), int64(l)*2, r))
	}
	for b.Loop() {
		utils.Intersect(s1, s2)
	}
}

func TestSumSlice(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		got := utils.SumSlice([]int{1, 2, 3, 4, 5})
		if got != 15 {
			t.Errorf("SumSlice() = %v, want 15", got)
		}
	})

	t.Run("float64", func(t *testing.T) {
		got := utils.SumSlice([]float64{1.1, 2.2, 3.3})
		if got < 6.59 || got > 6.61 {
			t.Errorf("SumSlice() = %v, want ~6.6", got)
		}
	})

	t.Run("empty", func(t *testing.T) {
		got := utils.SumSlice([]int{})
		if got != 0 {
			t.Errorf("SumSlice() = %v, want 0", got)
		}
	})

	t.Run("negative", func(t *testing.T) {
		got := utils.SumSlice([]int{-1, -2, 3})
		if got != 0 {
			t.Errorf("SumSlice() = %v, want 0", got)
		}
	})
}
