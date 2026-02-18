package utils_test

import (
	"testing"

	"github.com/Is999/go-utils"
)

func TestStr2Int(t *testing.T) {
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
			if gotI := utils.Str2Int(tt.args.s); gotI != tt.want {
				t.Errorf("Str2Int() = %v, want %v", gotI, tt.want)
			}
		})
	}
}

func TestStr2Int64(t *testing.T) {
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
			if gotI := utils.Str2Int64(tt.args.s); gotI != tt.want {
				t.Errorf("Str2Int64() = %v, want %v", gotI, tt.want)
			}
		})
	}
}

func TestStr2Float(t *testing.T) {
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
			if gotI := utils.Str2Float(tt.args.s); gotI != tt.want {
				t.Errorf("Str2Float() = %v, want %v", gotI, tt.want)
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
			if n, err := utils.BinDec(bin); err != nil && n != tt.args.number {
				t.Errorf("BinDec() = %v, want %v", n, tt.args.number)
			}

			// 十进制 八进制转换
			oct := utils.DecOct(tt.args.number)
			if oct == tt.want {
				t.Errorf("DecOct() = %v, want %v", oct, tt.want)
			}
			if n, err := utils.OctDec(oct); err != nil && n != tt.args.number {
				t.Errorf("OctDec() = %v, want %v", n, tt.args.number)
			}

			// 十进制 十六进制转换
			hex := utils.DecHex(tt.args.number)
			if hex == tt.want {
				t.Errorf("DecHex() = %v, want %v", hex, tt.want)
			}
			if n, err := utils.HexDec(hex); err != nil && n != tt.args.number {
				t.Errorf("HexDec() = %v, want %v", n, tt.args.number)
			}

			// 二进制 八进制转换
			if got, err := utils.BinOct(bin); err != nil {
				t.Errorf("BinOct() = %v, want %v", got, tt.want)
			} else if n, err := utils.OctBin(got); err != nil && n != bin {
				t.Errorf("OctBin() = %v, want %v", n, bin)
			}

			// 二进制 十六进制转换
			if got, err := utils.BinHex(bin); err != nil {
				t.Errorf("BinHex() = %v, want %v", got, tt.want)
			} else if n, err := utils.HexBin(got); err != nil && n != bin {
				t.Errorf("HexBin() = %v, want %v", n, bin)
			}

			// 八进制 十六进制转换
			if got, err := utils.OctHex(oct); err != nil {
				t.Errorf("OctHex() = %v, want %v", got, tt.want)
			} else if n, err := utils.HexOct(got); err != nil && n != bin {
				t.Errorf("HexOct() = %v, want %v", n, bin)
			}
		})
	}
}
