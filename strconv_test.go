package utils_test

import (
	"math"
	"testing"

	"github.com/Is999/go-utils"
)

func TestToInt(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "001", args: args{"011"}, want: 11},
		{name: "002", args: args{"-10"}, want: -10},
		{name: "003", args: args{"10.00"}, want: 0},
		{name: "004", args: args{"A"}, want: 0},
		{name: "005", args: args{""}, want: 0},
		{name: "positive_overflow", args: args{"999999999999999999999999999"}, want: math.MaxInt},
		{name: "negative_overflow", args: args{"-999999999999999999999999999"}, want: math.MinInt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotI := utils.ToInt(tt.args.s); gotI != tt.want {
				t.Errorf("ToInt() = %v, want %v", gotI, tt.want)
			}
		})
	}
}

func TestToInt64(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{name: "001", args: args{"011"}, want: 11},
		{name: "002", args: args{"-10"}, want: -10},
		{name: "003", args: args{"10.00"}, want: 0},
		{name: "004", args: args{"A"}, want: 0},
		{name: "005", args: args{""}, want: 0},
		{name: "positive_overflow", args: args{"9223372036854775808"}, want: math.MaxInt64},
		{name: "negative_overflow", args: args{"-9223372036854775809"}, want: math.MinInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotI := utils.ToInt64(tt.args.s); gotI != tt.want {
				t.Errorf("ToInt64() = %v, want %v", gotI, tt.want)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{name: "001", args: args{"011"}, want: 11},
		{name: "002", args: args{"-10"}, want: -10},
		{name: "003", args: args{"10.00"}, want: 10},
		{name: "004", args: args{"A"}, want: 0},
		{name: "005", args: args{""}, want: 0},
		{name: "006", args: args{"11.345"}, want: 11.345},
		{name: "positive_overflow", args: args{"1e1000"}, want: math.Inf(1)},
		{name: "negative_overflow", args: args{"-1e1000"}, want: math.Inf(-1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotI := utils.ToFloat64(tt.args.s); gotI != tt.want {
				t.Errorf("ToFloat64() = %v, want %v", gotI, tt.want)
			}
		})
	}
}

func TestDecBin(t *testing.T) {
	type args struct {
		number int64
	}
	tests := []struct {
		name string
		args args
		bin  string // 无前缀的二进制期望。
		oct  string // 无前缀的八进制期望。
		hex  string // 小写且无前缀的十六进制期望。
	}{
		{name: "735826", args: args{735826}, bin: "10110011101001010010", oct: "2635122", hex: "b3a52"},
		{name: "109234", args: args{109234}, bin: "11010101010110010", oct: "325262", hex: "1aab2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin := utils.DecBin(tt.args.number)
			if bin != tt.bin {
				t.Errorf("DecBin() = %v, want %v", bin, tt.bin)
			}
			n, err := utils.BinDec(bin)
			if err != nil {
				t.Fatalf("BinDec() error = %v", err)
			}
			if n != tt.args.number {
				t.Errorf("BinDec() = %v, want %v", n, tt.args.number)
			}

			oct := utils.DecOct(tt.args.number)
			if oct != tt.oct {
				t.Errorf("DecOct() = %v, want %v", oct, tt.oct)
			}
			n, err = utils.OctDec(oct)
			if err != nil {
				t.Fatalf("OctDec() error = %v", err)
			}
			if n != tt.args.number {
				t.Errorf("OctDec() = %v, want %v", n, tt.args.number)
			}

			hex := utils.DecHex(tt.args.number)
			if hex != tt.hex {
				t.Errorf("DecHex() = %v, want %v", hex, tt.hex)
			}
			n, err = utils.HexDec(hex)
			if err != nil {
				t.Fatalf("HexDec() error = %v", err)
			}
			if n != tt.args.number {
				t.Errorf("HexDec() = %v, want %v", n, tt.args.number)
			}

			octFromBin, err := utils.BinOct(bin)
			if err != nil {
				t.Fatalf("BinOct() error = %v", err)
			}
			binFromOct, err := utils.OctBin(octFromBin)
			if err != nil {
				t.Fatalf("OctBin() error = %v", err)
			}
			if binFromOct != bin {
				t.Errorf("OctBin() = %v, want %v", binFromOct, bin)
			}

			hexFromBin, err := utils.BinHex(bin)
			if err != nil {
				t.Fatalf("BinHex() error = %v", err)
			}
			binFromHex, err := utils.HexBin(hexFromBin)
			if err != nil {
				t.Fatalf("HexBin() error = %v", err)
			}
			if binFromHex != bin {
				t.Errorf("HexBin() = %v, want %v", binFromHex, bin)
			}

			hexFromOct, err := utils.OctHex(oct)
			if err != nil {
				t.Fatalf("OctHex() error = %v", err)
			}
			octFromHex, err := utils.HexOct(hexFromOct)
			if err != nil {
				t.Fatalf("HexOct() error = %v", err)
			}
			if octFromHex != oct {
				t.Errorf("HexOct() = %v, want %v", octFromHex, oct)
			}
		})
	}
}
