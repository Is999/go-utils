package utils_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

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

func TestFormatNumber(t *testing.T) {
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
		{name: "NaN", args: args{number: math.NaN(), decimals: 2, decPoint: ".", thousandsSep: ","}, want: "NaN"},
		{name: "positive infinity", args: args{number: math.Inf(1), decimals: 2, decPoint: ".", thousandsSep: ","}, want: "+Inf"},
		{name: "negative infinity", args: args{number: math.Inf(-1), decimals: 2, decPoint: ".", thousandsSep: ","}, want: "-Inf"},
		{name: "negative zero", args: args{number: math.Copysign(0, -1), decimals: 2, decPoint: ".", thousandsSep: ","}, want: "-0.00"},
		{name: "rounded negative zero", args: args{number: -0.001, decimals: 2, decPoint: "."}, want: "-0.00"},
		{name: "rounding adds group", args: args{number: -999.999, decimals: 2, decPoint: ".", thousandsSep: ","}, want: "-1,000.00"},
		{name: "multibyte separators", args: args{number: -1234567.5, decimals: 2, decPoint: "点", thousandsSep: "万"}, want: "-1万234万567点50"},
		{name: "zero decimals", args: args{number: -12.5, decimals: 0, decPoint: "点", thousandsSep: ","}, want: "-12"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.FormatNumber(tt.args.number, tt.args.decimals, tt.args.decPoint, tt.args.thousandsSep); got != tt.want {
				t.Errorf("FormatNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

// BenchmarkFormatNumber 对比直接输出与千分位拼接的分配，正负数使用相同精度。
func BenchmarkFormatNumber(b *testing.B) {
	for _, tt := range []struct {
		name   string
		number float64
		sep    string
	}{
		{"positive/plain", 1234567.89, ""},
		{"negative/plain", -1234567.89, ""},
		{"positive/grouped", 1234567.89, ","},
		{"negative/grouped", -1234567.89, ","},
	} {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = utils.FormatNumber(tt.number, 2, ".", tt.sep)
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

func TestRetry_ZeroMaxRetriesRunsOnce(t *testing.T) {
	callCount := 0
	err := utils.Retry(0, func(tries int) error {
		callCount++
		return errors.New("persistent error")
	})
	if err == nil {
		t.Fatal("Retry() error = nil, want error")
	}
	if callCount != 1 {
		t.Fatalf("Retry() callCount = %d, want 1", callCount)
	}
}

func TestRetryContext_CancelStopsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	err := utils.RetryContext(ctx, 5, func(ctx context.Context, tries int) error {
		callCount++
		cancel()
		return errors.New("temporary error")
	})
	if err == nil {
		t.Fatal("RetryContext() error = nil, want error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RetryContext() error = %v, want context.Canceled", err)
	}
	if callCount != 1 {
		t.Fatalf("RetryContext() callCount = %d, want 1", callCount)
	}
}

func TestRetryContext_TimeoutStopsEarly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	callCount := 0
	err := utils.RetryContext(ctx, 5, func(ctx context.Context, tries int) error {
		callCount++
		return errors.New("temporary error")
	})
	if err == nil {
		t.Fatal("RetryContext() error = nil, want error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RetryContext() error = %v, want context.DeadlineExceeded", err)
	}
	if callCount != 1 {
		t.Fatalf("RetryContext() callCount = %d, want 1", callCount)
	}
	if elapsed := time.Since(start); elapsed >= 200*time.Millisecond {
		t.Fatalf("RetryContext() elapsed = %v, want < 200ms", elapsed)
	}
}

func TestRetryFailureNamesOriginalFunction(t *testing.T) {
	// Retry 的签名适配不得改变错误中的调用方函数名或底层错误链。
	wantErr := errors.New("operation failed")
	fn := func(int) error { return wantErr }
	err := utils.Retry(1, fn)
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), utils.GetFunctionName(fn)+" 尝试 1 次后依然失败") {
		t.Fatalf("Retry() error = %v, want original function and cause", err)
	}
	contextFn := func(context.Context, int) error { return wantErr }
	err = utils.RetryContext(t.Context(), 1, contextFn)
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), utils.GetFunctionName(contextFn)+" 尝试 1 次后依然失败") {
		t.Fatalf("RetryContext() error = %v, want original function and cause", err)
	}
}

func BenchmarkRetrySuccess(b *testing.B) {
	// 首次执行成功是无退避路径，基准只包含公开入口的调度成本。
	fn := func(int) error { return nil }
	b.ReportAllocs()
	for b.Loop() {
		if err := utils.Retry(3, fn); err != nil {
			b.Fatal(err)
		}
	}
}
