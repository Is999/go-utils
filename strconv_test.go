package utils_test

import (
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
		want string
	}{
		{name: "001", args: args{735826}, want: ""},
		{name: "001", args: args{109234}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 十进制 二进制转换
			bin := utils.DecBin(tt.args.number)
			if bin == tt.want {
				t.Errorf("DecBin() = %v, want %v", bin, tt.want)
			}
			n, err := utils.BinDec(bin)
			if err != nil {
				t.Fatalf("BinDec() error = %v", err)
			}
			if n != tt.args.number {
				t.Errorf("BinDec() = %v, want %v", n, tt.args.number)
			}

			// 十进制 八进制转换
			oct := utils.DecOct(tt.args.number)
			if oct == tt.want {
				t.Errorf("DecOct() = %v, want %v", oct, tt.want)
			}
			n, err = utils.OctDec(oct)
			if err != nil {
				t.Fatalf("OctDec() error = %v", err)
			}
			if n != tt.args.number {
				t.Errorf("OctDec() = %v, want %v", n, tt.args.number)
			}

			// 十进制 十六进制转换
			hex := utils.DecHex(tt.args.number)
			if hex == tt.want {
				t.Errorf("DecHex() = %v, want %v", hex, tt.want)
			}
			n, err = utils.HexDec(hex)
			if err != nil {
				t.Fatalf("HexDec() error = %v", err)
			}
			if n != tt.args.number {
				t.Errorf("HexDec() = %v, want %v", n, tt.args.number)
			}

			// 二进制 八进制转换
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

			// 二进制 十六进制转换
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

			// 八进制 十六进制转换
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
