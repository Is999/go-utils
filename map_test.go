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
		want utils.Slice[string]
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
		}, isReverse: true}, want: []string{"001", "002", "003", "004", "005"}},
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
		}, isReverse: true}, want: []string{"abceg", "dbdrf", "xdghf", "abreq", "xbghf"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.isReverse {
				sort.Sort(sort.Reverse(tt.want))
				//t.Logf("Reverse want = %v \n", tt.want)
			} else {
				sort.Sort(tt.want)
			}
			if got := utils.MapValues(tt.args.m, tt.args.isReverse); !reflect.DeepEqual(got, []string(tt.want)) {
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
	type args[K utils.Ordered, V any] struct {
		m         map[K]V
		f         func(key K, val V) bool
		isReverse bool
	}
	type testCase[K utils.Ordered, V any] struct {
		name string
		args args[K, V]
	}
	tests := []testCase[string, int]{
		{name: "001", args: args[string, int]{m: map[string]int{
			"key01": 100,
			"key02": 200,
			"key03": 300,
			"key04": 400,
			"key05": 500,
		}, f: func(key string, val int) bool {
			if key == "key04" {
				return false
			}
			//t.Logf("key %v, val %v", key, val)
			return true
		}}},
		{name: "002", args: args[string, int]{m: map[string]int{
			"key01": 100,
			"key02": 200,
			"key03": 300,
			"key04": 400,
			"key05": 500,
		}, f: func(key string, val int) bool {
			if val == 200 {
				return false
			}
			//t.Logf("key %v, val %v", key, val)
			return true
		}, isReverse: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utils.MapRange(tt.args.m, tt.args.f, tt.args.isReverse)
		})
	}
}

func TestSumMap(t *testing.T) {
	// int 类型
	t.Run("int", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2, "c": 3}
		got := utils.SumMap(m)
		if got != 6 {
			t.Errorf("SumMap() = %v, want 6", got)
		}
	})

	// float64 类型
	t.Run("float64", func(t *testing.T) {
		m := map[string]float64{"x": 1.5, "y": 2.5, "z": 3.0}
		got := utils.SumMap(m)
		if got != 7.0 {
			t.Errorf("SumMap() = %v, want 7.0", got)
		}
	})

	// 空 map
	t.Run("empty", func(t *testing.T) {
		m := map[string]int{}
		got := utils.SumMap(m)
		if got != 0 {
			t.Errorf("SumMap() = %v, want 0", got)
		}
	})
}
