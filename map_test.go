package utils_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/Is999/go-utils"
)

func TestMapKeys(t *testing.T) {
	type args[K utils.Ordered] struct {
		elements map[K]string
	}

	tests := []struct {
		name string
		args args[string]
		want []string
	}{
		{name: "001", args: args[string]{elements: map[string]string{
			"key01": "001",
			"key04": "004",
			"key02": "002",
			"key05": "005",
			"key03": "003",
		}}, want: []string{"key01", "key04", "key02", "key05", "key03"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got utils.Slice[string] = utils.MapKeys(tt.args.elements)
			sort.Sort(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual([]string(got), tt.want) {
				t.Errorf("MapKeys() = %v,  want %v", got, tt.want)
			}
		})
	}
}

func TestMapValues(t *testing.T) {
	type args[K utils.Ordered, V any] struct {
		m         map[K]V
		isReverse bool
	}
	tests := []struct {
		name string
		args args[string, string]
		want []string // 按键顺序排列的值，避免按值重算期望。
	}{
		{name: "001", args: args[string, string]{m: map[string]string{
			"key01": "001",
			"key04": "004",
			"key02": "002",
			"key05": "005",
			"key03": "003",
		}}, want: []string{"001", "002", "003", "004", "005"}},
		{name: "002", args: args[string, string]{m: map[string]string{
			"key01": "001",
			"key04": "004",
			"key02": "002",
			"key05": "005",
			"key03": "003",
		}, isReverse: true}, want: []string{"005", "004", "003", "002", "001"}},
		{name: "003", args: args[string, string]{m: map[string]string{
			"abceg": "abceg",
			"dbdrf": "dbdrf",
			"xdghf": "xdghf",
			"abreq": "abreq",
			"xbghf": "xbghf",
		}, isReverse: false}, want: []string{"abceg", "abreq", "dbdrf", "xbghf", "xdghf"}},
		{name: "004", args: args[string, string]{m: map[string]string{
			"abceg": "abceg",
			"dbdrf": "dbdrf",
			"xdghf": "xdghf",
			"abreq": "abreq",
			"xbghf": "xbghf",
		}, isReverse: true}, want: []string{"xdghf", "xbghf", "dbdrf", "abreq", "abceg"}},
		// 键和值的顺序不同，避免按值排序也能通过测试。
		{name: "values_follow_ascending_keys", args: args[string, string]{m: map[string]string{
			"a": "003", "b": "001", "c": "002",
		}}, want: []string{"003", "001", "002"}},
		{name: "values_follow_descending_keys", args: args[string, string]{m: map[string]string{
			"a": "003", "b": "001", "c": "002",
		}, isReverse: true}, want: []string{"002", "001", "003"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.MapValues(tt.args.m, tt.args.isReverse); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapDiff(t *testing.T) {
	type args[K utils.Ordered, V utils.Ordered] struct {
		m1 map[K]V
		m2 map[K]V
	}
	type testCase[K utils.Ordered, V utils.Ordered] struct {
		name string
		args args[K, V]
		want []V
	}
	tests := []testCase[string, string]{
		{name: "001", args: args[string, string]{m1: map[string]string{
			"key01": "001",
			"key02": "002",
			"key03": "003",
			"key04": "004",
			"key05": "005",
		}, m2: map[string]string{
			"key01": "001",
			"key02": "012",
			"key03": "003",
			"key04": "014",
			"key05": "005",
		}}, want: []string{"002", "004"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.MapDiff(tt.args.m1, tt.args.m2)
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapDiff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapIntersect(t *testing.T) {
	type args[K utils.Ordered, V utils.Ordered] struct {
		m1 map[K]V
		m2 map[K]V
	}
	type testCase[K utils.Ordered, V utils.Ordered] struct {
		name string
		args args[K, V]
		want []V
	}
	tests := []testCase[string, string]{
		{name: "001", args: args[string, string]{m1: map[string]string{
			"key01": "001",
			"key02": "002",
			"key03": "003",
			"key04": "004",
			"key05": "005",
		}, m2: map[string]string{
			"key01": "001",
			"key02": "012",
			"key03": "003",
			"key04": "014",
			"key05": "005",
		}}, want: []string{"001", "003", "005"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.MapIntersect(tt.args.m1, tt.args.m2)
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapIntersect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapDiffKey(t *testing.T) {
	type args[K utils.Ordered, V any] struct {
		m1 map[K]V
		m2 map[K]V
	}
	type testCase[K utils.Ordered, V any] struct {
		name string
		args args[K, V]
		want []K
	}
	tests := []testCase[string, string]{
		{name: "001", args: args[string, string]{m1: map[string]string{
			"key01": "001",
			"key02": "002",
			"key03": "003",
			"key04": "004",
			"key05": "005",
		}, m2: map[string]string{
			"key01": "001",
			"key12": "012",
			"key03": "003",
			"key14": "014",
			"key05": "005",
		}}, want: []string{"key02", "key04"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.MapDiffKey(tt.args.m1, tt.args.m2)
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapDiffKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapIntersectKey(t *testing.T) {
	type args[K utils.Ordered, V any] struct {
		m1 map[K]V
		m2 map[K]V
	}
	type testCase[K utils.Ordered, V any] struct {
		name string
		args args[K, V]
		want []K
	}
	tests := []testCase[string, string]{
		{name: "001", args: args[string, string]{m1: map[string]string{
			"key01": "001",
			"key02": "002",
			"key03": "003",
			"key04": "004",
			"key05": "005",
		}, m2: map[string]string{
			"key01": "001",
			"key12": "012",
			"key03": "003",
			"key14": "014",
			"key05": "005",
		}}, want: []string{"key01", "key03", "key05"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.MapIntersectKey(tt.args.m1, tt.args.m2)
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapIntersectKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapFilter(t *testing.T) {
	type args[K utils.Ordered, V any] struct {
		m map[K]V
		f func(key K, val V) bool
	}
	type testCase[K utils.Ordered, V any] struct {
		name string
		args args[K, V]
		want map[K]V
	}
	tests := []testCase[string, int]{
		{name: "001", args: args[string, int]{m: map[string]int{
			"key01": 100,
			"key02": 200,
			"key03": 300,
			"key04": 400,
			"key05": 500,
		}, f: func(key string, val int) bool {
			if key == "key04" || val == 200 {
				return false
			}
			return true
		}}, want: map[string]int{
			"key01": 100,
			"key03": 300,
			"key05": 500,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.MapFilter(tt.args.m, tt.args.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapRange(t *testing.T) {
	// entry 保存每次回调的实参，包括触发停止的当前项。
	type entry struct {
		// key 用于核对排序及停止位置。
		key string
		// val 用于核对键值对应关系。
		val int
	}
	type args[K utils.Ordered, V any] struct {
		m         map[K]V
		f         func(key K, val V) bool
		isReverse bool
	}
	type testCase[K utils.Ordered, V any] struct {
		name string
		args args[K, V]
		want []entry // 预期回调顺序，包含触发停止的项。
	}
	tests := []testCase[string, int]{
		{name: "升序按键停止", args: args[string, int]{m: map[string]int{
			"key01": 100,
			"key02": 200,
			"key03": 300,
			"key04": 400,
			"key05": 500,
		}, f: func(key string, val int) bool {
			return key != "key04"
		}}, want: []entry{{"key01", 100}, {"key02", 200}, {"key03", 300}, {"key04", 400}}},
		{name: "降序按值停止", args: args[string, int]{m: map[string]int{
			"key01": 100,
			"key02": 200,
			"key03": 300,
			"key04": 400,
			"key05": 500,
		}, f: func(key string, val int) bool {
			return val != 200
		}, isReverse: true}, want: []entry{{"key05", 500}, {"key04", 400}, {"key03", 300}, {"key02", 200}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []entry
			utils.MapRange(tt.args.m, func(key string, val int) bool {
				// 返回 false 的当前项也已调用回调，记录后才停止。
				got = append(got, entry{key, val})
				return tt.args.f(key, val)
			}, tt.args.isReverse)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMapRangeReadsUpdatedValue 确认排序只快照键，后续回调仍读取修改后的值。
func TestMapRangeReadsUpdatedValue(t *testing.T) {
	m := map[int]int{1: 10, 2: 20, 3: 30}
	var got []int
	utils.MapRange(m, func(key, val int) bool {
		got = append(got, val)
		if key == 1 {
			// 键集合先固定，值应在各次回调前读取。
			m[2] = 200
		}
		return true
	})
	if want := []int{10, 200, 30}; !reflect.DeepEqual(got, want) {
		t.Errorf("MapRange() values = %v, want %v", got, want)
	}
}

func TestSumMap(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2, "c": 3}
		got := utils.SumMap(m)
		if got != 6 {
			t.Errorf("SumMap() = %v, want 6", got)
		}
	})

	t.Run("float64", func(t *testing.T) {
		m := map[string]float64{"x": 1.5, "y": 2.5, "z": 3.0}
		got := utils.SumMap(m)
		if got != 7.0 {
			t.Errorf("SumMap() = %v, want 7.0", got)
		}
	})

	t.Run("empty", func(t *testing.T) {
		m := map[string]int{}
		got := utils.SumMap(m)
		if got != 0 {
			t.Errorf("SumMap() = %v, want 0", got)
		}
	})
}
