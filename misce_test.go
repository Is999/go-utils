package utils_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Is999/go-utils"
)

func TestTernary(t *testing.T) {
	type args[T utils.Ordered] struct {
		condition bool
		trueVal   T
		falseVal  T
	}

	type testCase[T utils.Ordered] struct {
		name string
		args args[T]
		want T
	}
	// int 类型
	tests := []testCase[int]{
		{name: "int-001", args: args[int]{condition: false, trueVal: 1, falseVal: 3}, want: 3},
		{name: "int-002", args: args[int]{condition: true, trueVal: 1, falseVal: 3}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Ternary(tt.args.condition, tt.args.trueVal, tt.args.falseVal); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Ternary() = %v, want %v", got, tt.want)
			}
		})
	}

	// string 类型
	var tests1 = []testCase[string]{
		{name: "str-001", args: args[string]{condition: false, trueVal: "正确", falseVal: "错误"}, want: "错误"},
		{name: "str-002", args: args[string]{condition: true, trueVal: "正确", falseVal: "错误"}, want: "正确"},
	}
	for _, tt1 := range tests1 {
		t.Run(tt1.name, func(t *testing.T) {
			if got1 := utils.Ternary(tt1.args.condition, tt1.args.trueVal, tt1.args.falseVal); !reflect.DeepEqual(got1, tt1.want) {
				t.Errorf("Ternary() = %v, want %v", got1, tt1.want)
			}
		})
	}
}

func TestNumberFormat(t *testing.T) {
	type args struct {
		number       float64
		decimals     uint
		decPoint     string
		thousandsSep string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "001", args: args{number: 4247312.5423197, decimals: 3}, want: "4247312542"},
		{name: "002", args: args{number: 4247312.5423197, decimals: 3, decPoint: ".", thousandsSep: ""}, want: "4247312.542"},
		{name: "003", args: args{number: 4247312.5423197, decimals: 3, decPoint: ".", thousandsSep: ","}, want: "4,247,312.542"},
		{name: "004", args: args{number: 4247312.5423197, decimals: 3, decPoint: " ", thousandsSep: "-"}, want: "4-247-312 542"},
		{name: "005", args: args{number: 4247312.5423197, decimals: 0, decPoint: ".", thousandsSep: ","}, want: "4,247,313"},
		{name: "006", args: args{number: 4247312.5423197, decimals: 5, decPoint: ".", thousandsSep: ","}, want: "4,247,312.54232"},
		{name: "007", args: args{number: 4247312.5423197, decimals: 9, decPoint: ".", thousandsSep: ","}, want: "4,247,312.542319700"},
		{name: "009", args: args{number: -4247312.5423197, decimals: 3}, want: "-4247312542"},
		{name: "010", args: args{number: -4247312.5423197, decimals: 3, decPoint: ".", thousandsSep: ""}, want: "-4247312.542"},
		{name: "011", args: args{number: -4247312.5423197, decimals: 3, decPoint: ".", thousandsSep: ","}, want: "-4,247,312.542"},
		{name: "012", args: args{number: -4247312.5423197, decimals: 3, decPoint: " ", thousandsSep: "-"}, want: "-4-247-312 542"},
		{name: "013", args: args{number: -4247312.5423197, decimals: 0, decPoint: ".", thousandsSep: ","}, want: "-4,247,313"},
		{name: "014", args: args{number: -4247312.5423197, decimals: 5, decPoint: ".", thousandsSep: ","}, want: "-4,247,312.54232"},
		{name: "015", args: args{number: -4247312.5423197, decimals: 9, decPoint: ".", thousandsSep: ","}, want: "-4,247,312.542319700"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.NumberFormat(tt.args.number, tt.args.decimals, tt.args.decPoint, tt.args.thousandsSep); got != tt.want {
				t.Errorf("NumberFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetry_Success(t *testing.T) {
	callCount := 0
	err := utils.Retry(3, func(tries int) error {
		callCount++
		return nil
	})
	if err != nil {
		t.Errorf("Retry() error = %v, want nil", err)
	}
	if callCount != 1 {
		t.Errorf("Retry() callCount = %d, want 1", callCount)
	}
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	callCount := 0
	err := utils.Retry(5, func(tries int) error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	})
	if err != nil {
		t.Errorf("Retry() error = %v, want nil", err)
	}
	if callCount != 3 {
		t.Errorf("Retry() callCount = %d, want 3", callCount)
	}
}

func TestRetry_MaxRetriesExhausted(t *testing.T) {
	callCount := 0
	err := utils.Retry(3, func(tries int) error {
		callCount++
		return errors.New("persistent error")
	})
	if err == nil {
		t.Error("Retry() error = nil, want error")
	}
	if callCount != 3 {
		t.Errorf("Retry() callCount = %d, want 3", callCount)
	}
}
