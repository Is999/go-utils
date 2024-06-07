package utils_test

import (
	"github.com/Is999/go-utils"
	"sync"
	"testing"
)

func TestRand(t *testing.T) {
	type args struct {
		min int64
		max int64
	}
	type testCase struct {
		name string
		args args
	}
	tests := []testCase{
		{name: "000", args: args{min: 0, max: 0}},
		{name: "001", args: args{min: 2, max: 2}},
		{name: "002", args: args{min: 0, max: 1}},
		{name: "003", args: args{min: 0, max: 2}},
		{name: "004", args: args{min: -1, max: 2}},
		{name: "005", args: args{min: -2, max: 2}},
		{name: "006", args: args{min: -5, max: 0}},
		{name: "007", args: args{min: 5, max: 10}},
		{name: "008", args: args{min: 10, max: 5}},
		{name: "009", args: args{min: 10000, max: 100000}},
	}
	r := utils.GetRandPool()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var minInt, maxInt = tt.args.min, tt.args.max
			if minInt > maxInt {
				minInt, maxInt = maxInt, minInt
			}
			var wg = &sync.WaitGroup{}
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(min, max int64, i int, tt testCase) {
					defer wg.Done()
					for j := 0; j < 10; j++ {
						if got := utils.Rand(tt.args.min, tt.args.max, r); !(got >= min && got <= max) {
							t.Errorf("%v-%v%v Rand() = %v, want %v-%v", tt.name, i, j, got, tt.args.min, tt.args.max)
							break
						} else {
							//t.Logf("%v-%v%v Rand() = %v, want %v-%v", tt.name, i, j, got, tt.args.min, tt.args.max)
						}
					}
				}(minInt, maxInt, i, tt)
			}
			wg.Wait()
		})
	}
}

// go test -bench=Rand$ -run ^$  -count 5 -benchmem
func BenchmarkRand(t *testing.B) {
	type args struct {
		min int64
		max int64
	}
	type testCase struct {
		name string
		args args
	}
	tests := []testCase{
		//{name: "001", args: args{min: -1, max: 2}},
		{name: "002", args: args{min: 10000, max: 100000}},
	}
	r := utils.GetRandPool()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.B) {
			for n := 0; n < t.N; n++ {
				if got := utils.Rand(tt.args.min, tt.args.max, r); !(got >= tt.args.min && got <= tt.args.max) {
					t.Errorf("Rand() = %v, want %v-%v", got, tt.args.min, tt.args.max)
					break
				}
			}
		})
	}
}

func TestRound(t *testing.T) {
	type args struct {
		value     float64
		precision int
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{name: "000", args: args{value: 0, precision: 0}, want: 0},
		{name: "001", args: args{value: 0.5, precision: 0}, want: 1},
		{name: "002", args: args{value: 1.4, precision: 0}, want: 1},
		{name: "003", args: args{value: 1, precision: 1}, want: 1},
		{name: "004", args: args{value: 23.256, precision: 1}, want: 23.3},
		{name: "005", args: args{value: 23.256, precision: 2}, want: 23.26},
		{name: "006", args: args{value: 23.256, precision: 4}, want: 23.256},
		{name: "007", args: args{value: 23.244, precision: 1}, want: 23.2},
		{name: "008", args: args{value: 23.244, precision: 2}, want: 23.24},
		{name: "009", args: args{value: 23.244, precision: 4}, want: 23.244},
		{name: "010", args: args{value: 45323.244, precision: -4}, want: 50000},
		{name: "011", args: args{value: 45323.244, precision: -3}, want: 45000},
		{name: "012", args: args{value: 45323.244, precision: -2}, want: 45300},
		{name: "013", args: args{value: 45323.244, precision: -1}, want: 45320},
		{name: "014", args: args{value: 45323.244, precision: 0}, want: 45323},
		{name: "015", args: args{value: -45323.244, precision: -4}, want: -50000},
		{name: "016", args: args{value: -45323.244, precision: -3}, want: -45000},
		{name: "017", args: args{value: -45323.244, precision: -2}, want: -45300},
		{name: "018", args: args{value: -45323.244, precision: -1}, want: -45320},
		{name: "019", args: args{value: -45323.244, precision: 0}, want: -45323},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Round(tt.args.value, tt.args.precision); got != tt.want {
				t.Errorf("Round() = %v, want %v", got, tt.want)
			}
		})
	}
}
