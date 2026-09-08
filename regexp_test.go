package utils_test

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/Is999/go-utils"
)

// BenchmarkAccount 对比短账号、最大长度账号与末尾连续下划线的扫描成本。
func BenchmarkAccount(b *testing.B) {
	for _, value := range []string{"Account8", strings.Repeat("a", 255), strings.Repeat("a", 253) + "__"} {
		name := "short"
		if len(value) == 255 {
			name = "long"
			if strings.HasSuffix(value, "__") {
				name = "underscores"
			}
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				utils.Account(value, 1, 255)
			}
		})
	}
}

// BenchmarkNumericValidation 固定有效值与超精度值，区分整段扫描和长度边界的成本。
func BenchmarkNumericValidation(b *testing.B) {
	for _, tc := range []struct {
		name, value string
		validate    func(string) bool
	}{
		{"qq_short", "12345", utils.QQ},
		{"qq_long", "123456789012", utils.QQ},
		{"qq_invalid", "12345678901x", utils.QQ},
		{"amount", "12345678.90", func(s string) bool { return utils.Amount(s, 2) }},
		{"amount_precision", "12345678." + strings.Repeat("1", 64), func(s string) bool { return utils.Amount(s, 2) }},
		{"numeric", "12345678.90123456", utils.Numeric},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				tc.validate(tc.value)
			}
		})
	}
}

// BenchmarkDateValidation 区分有效日期、分隔符不一致和真实日期不存在的检查成本。
func BenchmarkDateValidation(b *testing.B) {
	for _, tc := range []struct {
		name, value string
		validate    func(string) bool
	}{
		{"date", "2024-02-29", utils.TimeDay},
		{"date_mixed_separator", "2024-02/29", utils.TimeDay},
		{"date_non_leap", "2023-02-29", utils.TimeDay},
		{"timestamp", "2024-02-29 23:59:59", utils.Timestamp},
		{"timestamp_short", "2024/2/29 3:4:5", utils.Timestamp},
		{"timestamp_mixed_separator", "2024-02/29 23:59:59", utils.Timestamp},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				tc.validate(tc.value)
			}
		})
	}
}

// TestDateValidationBoundaries 固定日期范围、闰年以及日期和时间组合时允许的位数与空白。
func TestDateValidationBoundaries(t *testing.T) {
	for _, date := range []struct {
		value string
		valid bool
	}{
		{"1000-1-1", true}, {"3999.12.31", true}, {"2024/2/9", true}, {"2024/02/09", true},
		{"2000-2-29", true}, {"2024-2-29", true}, {"1900-2-29", false}, {"2023-2-29", false},
		{"2024-4-31", false}, {"2024-0-1", false}, {"2024-1-0", false}, {"4000-1-1", false},
		{"999-1-1", false}, {"", false}, {"2024-2-29 ", false}, {" 2024-2-29", false},
		{"2024-2-29\n", false}, {"2024-2\xff29", false},
	} {
		if got := utils.TimeDay(date.value); got != date.valid {
			t.Errorf("TimeDay(%q) = %v, want %v", date.value, got, date.valid)
		}
		for _, clock := range []struct {
			value string
			valid bool
		}{
			{"0:0:0", true}, {"03:04:05", true}, {"23:59:59", true},
			{"24:0:0", false}, {"1:60:0", false}, {"1:1:60", false}, {"", false},
			{" 1:1:1", false}, {"1:1:1 ", false}, {"1:1:1\n", false},
		} {
			value := date.value + " " + clock.value
			if got, want := utils.Timestamp(value), date.valid && clock.valid; got != want {
				t.Errorf("Timestamp(%q) = %v, want %v", value, got, want)
			}
		}
	}
	// 两处分隔符必须相同，单独允许的分隔符不能交叉使用。
	for _, first := range []string{"-", "/", "."} {
		for _, second := range []string{"-", "/", "."} {
			date := "2024" + first + "2" + second + "29"
			if got := utils.TimeDay(date); got != (first == second) {
				t.Errorf("TimeDay(%q) = %v", date, got)
			}
			if got := utils.Timestamp(date + " 3:4:5"); got != (first == second) {
				t.Errorf("Timestamp(%q) = %v", date, got)
			}
		}
	}
}

// TestNumericValidationMatchesPatterns 用独立正则规则核对数字解析的符号、前导零和精度边界。
func TestNumericValidationMatchesPatterns(t *testing.T) {
	for _, tc := range []struct {
		name, pattern string
		validate      func(string) bool
	}{
		{"qq", `^[1-9][0-9]{4,11}$`, utils.QQ},
		{"numeric", `^[+-]?(0|[1-9][0-9]*)(\.[0-9]+)?$`, utils.Numeric},
		{"unsigned", `^(0|[1-9][0-9]*)(\.[0-9]+)?$`, utils.UnNumeric},
		{"amount_integer", `^(0|[1-9][0-9]*)$`, func(s string) bool { return utils.Amount(s, 0) }},
		{"amount_decimal", `^[+-]?(0|[1-9][0-9]*)(\.[0-9]{1,2})?$`, func(s string) bool { return utils.Amount(s, 2, true) }},
		{"amount_max_precision", `^(0|[1-9][0-9]*)(\.[0-9]{1,255})?$`, func(s string) bool { return utils.Amount(s, 255) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reference := regexp.MustCompile(tc.pattern)
			for _, value := range []string{
				"", "+", "-", "0", "00", "01", "+0", "-0", "+01", "1.", ".1", "0.00", "1.001",
				"1234", "12345", "123456789012", "1234567890123", "012345", "12345\n", "12\xff34", "１２３４５",
				"0." + strings.Repeat("1", 255), "0." + strings.Repeat("1", 256),
			} {
				if got, want := tc.validate(value), reference.MatchString(value); got != want {
					t.Errorf("validate(%q) = %v, want %v", value, got, want)
				}
			}
		})
	}
}

func TestAmount(t *testing.T) {
	type args struct {
		value   string
		decimal uint8
		signed  bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "100", decimal: 0}, want: true},
		{name: "s-002", args: args{value: "5.23", decimal: 2}, want: true},
		{name: "s-003", args: args{value: "-120", decimal: 2, signed: true}, want: true},
		{name: "s-004", args: args{value: "-100.45", decimal: 2, signed: true}, want: true},
		{name: "s-005", args: args{value: "0", decimal: 2}, want: true},
		{name: "s-006", args: args{value: "0.00", decimal: 2}, want: true},
		{name: "s-007", args: args{value: "+1", decimal: 2, signed: true}, want: true},
		{name: "s-008", args: args{value: "5.3", decimal: 1}, want: true},
		{name: "s-009", args: args{value: "-15.4324", decimal: 4, signed: true}, want: true},
		{name: "e-001", args: args{value: "5.432", decimal: 2}, want: false}, // 保留小数位长度错误
		{name: "e-002", args: args{value: "321,875.34", decimal: 2}, want: false},
		{name: "e-003", args: args{value: "00", decimal: 2}, want: false},
		{name: "e-004", args: args{value: "00.0", decimal: 2}, want: false},
		{name: "e-005", args: args{value: "01", decimal: 2, signed: true}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Amount(tt.args.value, tt.args.decimal, tt.args.signed); got != tt.want {
				t.Errorf("Amount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "abc@qq.com"}, want: true},
		{name: "s-002", args: args{value: "abc0.12-12@qq.com.cn"}, want: true},
		{name: "s-003", args: args{value: "abc_012-12@qq.com.cn"}, want: true},
		{name: "s-004", args: args{value: "abc_012-12@qq.com.cn"}, want: true},
		{name: "s-005", args: args{value: "abc_1@qq.com"}, want: true},
		{name: "s-006", args: args{value: "abc01212.w@gamil.com.cn"}, want: true},
		{name: "e-001", args: args{value: "_abc01212@qq.com"}, want: false},         // 错误: 不能_开头
		{name: "e-002", args: args{value: "abc01@212@qq.com"}, want: false},         // 错误: 不能出现多个 @
		{name: "e-003", args: args{value: "abc01212@gamil.com.cn.tt"}, want: false}, // 错误: 超过2次.xx
		{name: "e-004", args: args{value: "abc_@qq.com"}, want: false},              // 错误: 末尾不能出现 -_.
		{name: "e-005", args: args{value: "abc01212.w@gamil"}, want: false},         // 错误: 至少出现一次 .xx
		{name: "e-006", args: args{value: "abc_-q@qq.com"}, want: false},            // 错误: 不能连续出现-_.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Email(tt.args.value); got != tt.want {
				t.Errorf("Email() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlnum(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "0123456789"}, want: true},
		{name: "s-002", args: args{value: "hello"}, want: true},
		{name: "s-003", args: args{value: "hello0123456789"}, want: true},
		{name: "e-001", args: args{value: "中文"}, want: false},
		{name: "e-002", args: args{value: "hello01234 56789"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Alnum(tt.args.value); got != tt.want {
				t.Errorf("Alnum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlpha(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "helloWorld"}, want: true},
		{name: "e-001", args: args{value: "0123456789"}, want: false},       // 数字
		{name: "e-002", args: args{value: "中文"}, want: false},               // 中文
		{name: "e-003", args: args{value: "hello0123456789"}, want: false},  // 包含数字
		{name: "e-004", args: args{value: "hello01234 56789"}, want: false}, // 包含数字和特殊字符
		{name: "e-005", args: args{value: "hello world"}, want: false},      //  包含特殊字符
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Alpha(tt.args.value); got != tt.want {
				t.Errorf("Alpha() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnInteger(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "100"}, want: true},
		{name: "e-001", args: args{value: "+123"}, want: false},    // 错误: 包含符号
		{name: "e-002", args: args{value: "5.23"}, want: false},    // 错误: 小数
		{name: "e-003", args: args{value: "5.432"}, want: false},   // 不接受小数。
		{name: "e-004", args: args{value: "-120"}, want: false},    // 错误: 负数
		{name: "e-005", args: args{value: "-100.45"}, want: false}, // 错误: 负数
		{name: "e-006", args: args{value: "0"}, want: false},       // 不接受 0
		{name: "e-007", args: args{value: "321,875"}, want: false}, // 错误: 千分位格式
		{name: "e-008", args: args{value: "0.00"}, want: false},    // 错误: 非整数
		{name: "e-009", args: args{value: "ab"}, want: false},      // 错误: 非数字
		{name: "e-010", args: args{value: "00.0"}, want: false},    // 错误: 非整数
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.UnInteger(tt.args.value); got != tt.want {
				t.Errorf("UnInteger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMixStr(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "001", args: args{value: "中文"}, want: false},
		{name: "002", args: args{value: "`hello`"}, want: true},
		{name: "003", args: args{value: "_"}, want: true},
		{name: "004", args: args{value: "_中文0123456789hello"}, want: false},
		{name: "005", args: args{value: "hello world"}, want: true},
		{name: "006", args: args{value: "012 `HelleWorld!`	~!@#$%^&*()_+{}|:\",./"}, want: true},
		{name: "007", args: args{value: "‘012’ “HelleWorld!”	~!@#$%^&*()_+{}|:\"<?-=>[]\\;',./ ~！……&*（@#￥%)——+「|：“《>？-=【、；‘，。、’】》”」）"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.MixStr(tt.args.value); got != tt.want {
				t.Errorf("MixStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMobile(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "13148888999"}, want: true},
		{name: "e-001", args: args{value: "123456789"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Mobile(tt.args.value); got != tt.want {
				t.Errorf("Mobile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmpty(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "   "}, want: true},
		{name: "s-002", args: args{value: ""}, want: true},
		{name: "e-001", args: args{value: "100.23"}, want: false}, // 非空
		{name: "e-002", args: args{value: "abc"}, want: false},    // 非空
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Empty(tt.args.value); got != tt.want {
				t.Errorf("Empty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNumeric(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "-100.23"}, want: true},
		{name: "s-002", args: args{value: "100.23"}, want: true},
		{name: "s-003", args: args{value: `+100.23`}, want: true},
		{name: "s-004", args: args{value: "100.030"}, want: true},
		{name: "s-005", args: args{value: "0.00"}, want: true},
		{name: "e-001", args: args{value: "0100.03"}, want: false},
		{name: "e-002", args: args{value: "+0100.23"}, want: false},
		{name: "e-003", args: args{value: "00.23"}, want: false},
		{name: "e-004", args: args{value: "00"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Numeric(tt.args.value); got != tt.want {
				t.Errorf("Numeric() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPhone(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "123456789"}, want: true},
		{name: "s-002", args: args{value: "4001-1046618"}, want: true},
		{name: "s-003", args: args{value: "533-74177002"}, want: true},
		{name: "s-004", args: args{value: "51983045392"}, want: true},
		{name: "s-005", args: args{value: "2010014"}, want: true},
		{name: "s-006", args: args{value: "13148888999"}, want: true},
		{name: "s-007", args: args{value: "95599"}, want: true},
		{name: "e-001", args: args{value: "53374-177002"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Phone(tt.args.value); got != tt.want {
				t.Errorf("Phone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQQ(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "20205"}, want: true},
		{name: "s-002", args: args{value: "125339782132"}, want: true},
		{name: "e-001", args: args{value: "1999"}, want: false},          // 长度
		{name: "e-002", args: args{value: "2020-01-01"}, want: false},    // 符号
		{name: "e-003", args: args{value: "1253397821256"}, want: false}, // 长度
		{name: "e-004", args: args{value: "abc12322"}, want: false},      // 英文
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.QQ(tt.args.value); got != tt.want {
				t.Errorf("QQ() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeDay(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "2020-01-01"}, want: true},
		{name: "s-002", args: args{value: "2020-02-29"}, want: true},
		{name: "s-003", args: args{value: "2020-01-31"}, want: true},
		{name: "s-004", args: args{value: "2020.01.20"}, want: true},
		{name: "s-005", args: args{value: "2020/1/31"}, want: true},
		{name: "s-006", args: args{value: "2020/6/9"}, want: true},
		{name: "s-007", args: args{value: "2020-9-08"}, want: true},
		{name: "s-008", args: args{value: "2020/01/20"}, want: true},
		{name: "s-009", args: args{value: "2020.5.4"}, want: true},
		{name: "e-001", args: args{value: "1999-02-31"}, want: false},          // 格式正确, 时间错误
		{name: "e-002", args: args{value: "1999-02-29"}, want: false},          // 格式正确, 时间错误
		{name: "e-003", args: args{value: "2020-01-32"}, want: false},          // 时间错误
		{name: "e-004", args: args{value: "2020-01/20"}, want: false},          // 分割符错误
		{name: "e-005", args: args{value: "2020/01-20"}, want: false},          // 分割符错误
		{name: "e-006", args: args{value: "2020.01/20"}, want: false},          // 分割符错误
		{name: "e-007", args: args{value: "2020/04/31"}, want: false},          // 时间错误
		{name: "e-008", args: args{value: "2020/06/31"}, want: false},          // 时间错误
		{name: "e-009", args: args{value: "2020/09/31"}, want: false},          // 时间错误
		{name: "e-010", args: args{value: "2020/11/31"}, want: false},          // 时间错误
		{name: "e-011", args: args{value: "2020.5.0"}, want: false},            // 时间错误
		{name: "e-012", args: args{value: "2020.05.00"}, want: false},          // 时间错误
		{name: "e-013", args: args{value: "2020.0.11"}, want: false},           // 时间错误
		{name: "e-014", args: args{value: "2020.00.11"}, want: false},          // 时间错误
		{name: "e-015", args: args{value: "2020-11-12 23:23:23"}, want: false}, // 格式错误
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.TimeDay(tt.args.value); got != tt.want {
				t.Errorf("TimeDay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeMonth(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "2020-01"}, want: true},
		{name: "s-002", args: args{value: "2020-02"}, want: true},
		{name: "s-003", args: args{value: "1999-02"}, want: true},
		{name: "s-004", args: args{value: "1999-2"}, want: true},
		{name: "s-005", args: args{value: "1999-10"}, want: true},
		{name: "e-001", args: args{value: "1999-02-27"}, want: false},
		{name: "e-002", args: args{value: "1999-13"}, want: false},
		{name: "e-003", args: args{value: "1999-0"}, want: false},
		{name: "e-004", args: args{value: "1999-00"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.TimeMonth(tt.args.value); got != tt.want {
				t.Errorf("TimeMonth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimestamp(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "2020-01-01 23:59:59"}, want: true},
		{name: "s-002", args: args{value: "2020-02-29 00:00:00"}, want: true},
		{name: "s-003", args: args{value: "2020-01-31 01:01:59"}, want: true},
		{name: "s-004", args: args{value: "2020.01.20 00:59:59"}, want: true},
		{name: "s-005", args: args{value: "2020/01/20 23:23:23"}, want: true},
		{name: "s-006", args: args{value: "2020/1/31 23:23:23"}, want: true},
		{name: "s-007", args: args{value: "2020/6/9 23:23:23"}, want: true},
		{name: "s-008", args: args{value: "2020-9-08 23:23:23"}, want: true},
		{name: "s-009", args: args{value: "2020.5.4 23:23:23"}, want: true},
		{name: "s-010", args: args{value: "2020.10.11 22:00:00"}, want: true},
		{name: "e-001", args: args{value: "1999-02-31 23:23:23"}, want: false}, // 格式正确, 时间错误
		{name: "e-002", args: args{value: "1999-02-29 23:23:23"}, want: false}, // 格式正确, 时间错误
		{name: "e-003", args: args{value: "2020-01-32 23:23:23"}, want: false}, // 时间错误
		{name: "e-004", args: args{value: "2020-01/20 23:23:23"}, want: false}, // 分割符错误
		{name: "e-005", args: args{value: "2020/01-20 23:23:23"}, want: false}, // 分割符错误
		{name: "e-006", args: args{value: "2020.01/20 23:23:23"}, want: false}, // 分割符错误
		{name: "e-007", args: args{value: "2020/04/31 23:23:23"}, want: false}, // 时间错误
		{name: "e-008", args: args{value: "2020/06/31 23:23:23"}, want: false}, // 时间错误
		{name: "e-009", args: args{value: "2020/09/31 23:23:23"}, want: false}, // 时间错误
		{name: "e-010", args: args{value: "2020/11/31 23:23:23"}, want: false}, // 时间错误
		{name: "e-011", args: args{value: "2020.5.0 23:23:23"}, want: false},   // 时间错误
		{name: "e-012", args: args{value: "2020.05.00 23:23:23"}, want: false}, // 时间错误
		{name: "e-013", args: args{value: "2020.0.11 23:23:23"}, want: false},  // 时间错误
		{name: "e-014", args: args{value: "2020.00.11 23:23:23"}, want: false}, // 时间错误
		{name: "e-015", args: args{value: "2020.10.11 24:00:00"}, want: false}, // 小时上限为 23。
		{name: "e-016", args: args{value: "2020.10.11 22:60:00"}, want: false}, // 时间错误
		{name: "e-017", args: args{value: "2020.10.11 22:00:60"}, want: false}, // 时间错误
		{name: "e-018", args: args{value: "2020.10.11-22:00:00"}, want: false}, // 格式错误
		{name: "e-019", args: args{value: "2020.10.11 22.00.00"}, want: false}, // 格式错误
		{name: "e-020", args: args{value: "2020-10-11 22.00.00"}, want: false}, // 格式错误
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Timestamp(tt.args.value); got != tt.want {
				t.Errorf("Timestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnNumeric(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "100"}, want: true},
		{name: "s-002", args: args{value: "5.23"}, want: true},
		{name: "s-003", args: args{value: "5.432"}, want: true},
		{name: "s-004", args: args{value: "0"}, want: true},
		{name: "s-005", args: args{value: "0.00"}, want: true},
		{name: "e-001", args: args{value: "-120"}, want: false},    // 负数
		{name: "e-002", args: args{value: "-100.45"}, want: false}, // 负数
		{name: "e-003", args: args{value: "321,875"}, want: false}, // 千分位格式
		{name: "e-004", args: args{value: "ab"}, want: false},      // 非数字
		{name: "e-005", args: args{value: "00.0"}, want: false},    // 重复 0
		{name: "e-006", args: args{value: "+123"}, want: false},    // 出现符号
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.UnNumeric(tt.args.value); got != tt.want {
				t.Errorf("UnNumeric() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnIntZero(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "100"}, want: true},
		{name: "s-002", args: args{value: "0"}, want: true},
		{name: "e-001", args: args{value: "5.23"}, want: false},
		{name: "e-002", args: args{value: "5.432"}, want: false},
		{name: "e-003", args: args{value: "-120"}, want: false},
		{name: "e-004", args: args{value: "-100.45"}, want: false},
		{name: "e-005", args: args{value: "321,875"}, want: false},
		{name: "e-006", args: args{value: "0.00"}, want: false},
		{name: "e-007", args: args{value: "ab"}, want: false},
		{name: "e-008", args: args{value: "00.0"}, want: false},
		{name: "e-009", args: args{value: "+123"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.UnIntZero(tt.args.value); got != tt.want {
				t.Errorf("UnIntZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZh(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "中文汉字"}, want: true},
		{name: "e-001", args: args{value: "中文汉字523"}, want: false},    // 数字
		{name: "e-002", args: args{value: "中文汉字,博大精深."}, want: false}, //英文符号
		{name: "e-003", args: args{value: "中文汉字，博大精深。"}, want: false}, //中文符号
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Zh(tt.args.value); got != tt.want {
				t.Errorf("Zh() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDomain(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "s-001", args: args{value: "https://wanwang.aliyun.com"}, want: true},
		{name: "s-002", args: args{value: "https://wanwang.aliyun.com/"}, want: true},
		{name: "s-003", args: args{value: "https://wanwang.aliyun.com.cn"}, want: true},
		{name: "s-004", args: args{value: "https://wanwang.aliyun.com.cn/"}, want: true},
		{name: "s-005", args: args{value: "https://wan-wang.aliyun.com.cn/"}, want: true},
		{name: "e-001", args: args{value: "https://wan_wang.aliyun.com.cn/"}, want: false},     // 下划线不属于允许的域名字符
		{name: "e-002", args: args{value: "https://wanwang.aliyun.com.cn//"}, want: false},     // 末尾至多一个斜线。
		{name: "e-003", args: args{value: "https://wanwang.aliyun.com.cn/incex"}, want: false}, // 不能带路径或参数
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Domain(tt.args.value); got != tt.want {
				t.Errorf("Domain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPassword(t *testing.T) {
	type args struct {
		value string
		min   uint8
		max   uint8
	}
	tests := []struct {
		name      string
		args      args
		wantValid bool // true 表示应通过校验。
	}{
		{name: "s-001", args: args{value: "ABC12321cb", min: 8, max: 12}, wantValid: true},
		{name: "s-002", args: args{value: "ABC123_1cb", min: 8, max: 12}, wantValid: true},
		{name: "s-003", args: args{value: "ABC123__1cb", min: 8, max: 12}, wantValid: true},
		{name: "s-004", args: args{value: "ABC123___1cb", min: 8, max: 12}, wantValid: true},
		{name: "s-005", args: args{value: "ABCEFGHJKL", min: 8, max: 12}, wantValid: true},
		{name: "s-006", args: args{value: "abcefghjkl", min: 8, max: 12}, wantValid: true},
		{name: "s-007", args: args{value: "123456789", min: 8, max: 12}, wantValid: true},
		{name: "e-001", args: args{value: "ABC123#1cb", min: 8, max: 12}, wantValid: false}, // 不能使用特殊字符
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.Password(tt.args.value, tt.args.min, tt.args.max); (err == nil) != tt.wantValid {
				t.Errorf("Password() error = %v, wantValid %v, input = %q", err, tt.wantValid, tt.args.value)
			}
		})
	}
}

func TestStrongPassword(t *testing.T) {
	type args struct {
		value string
		min   uint8
		max   uint8
	}
	tests := []struct {
		name      string
		args      args
		wantValid bool // true 表示应通过校验。
	}{
		{name: "s-001", args: args{value: "ABC12321cb", min: 8, max: 12}, wantValid: true},
		{name: "e-001", args: args{value: "ABC123#1cb", min: 8, max: 12}, wantValid: false},   // 不能使用特殊字符
		{name: "e-002", args: args{value: "ABC123_1cb", min: 8, max: 12}, wantValid: false},   // 不能使用特殊字符
		{name: "e-003", args: args{value: "ABC123__1cb", min: 8, max: 12}, wantValid: false},  // 不能使用特殊字符
		{name: "e-004", args: args{value: "ABC123___1cb", min: 8, max: 12}, wantValid: false}, // 不能使用特殊字符
		{name: "e-005", args: args{value: "ABCEFGHJKL", min: 8, max: 12}, wantValid: false},   // 必须包含小写字母
		{name: "e-006", args: args{value: "abcefghjkl", min: 8, max: 12}, wantValid: false},   // 必须包含大写字母
		{name: "e-007", args: args{value: "123456789", min: 8, max: 12}, wantValid: false},    // 必须包含小写字母
		{name: "e-008", args: args{value: "ABCEFghjkl", min: 8, max: 12}, wantValid: false},   // 必须包含数字
		{name: "e-009", args: args{value: "ABCE56789", min: 8, max: 12}, wantValid: false},    // 必须包含小写字母
		{name: "e-010", args: args{value: "abce56789", min: 8, max: 12}, wantValid: false},    // 必须包含大写字母
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.StrongPassword(tt.args.value, tt.args.min, tt.args.max); (err == nil) != tt.wantValid {
				t.Errorf("StrongPassword() error = %v, wantValid %v, input = %q", err, tt.wantValid, tt.args.value)
			}
		})
	}
}

func TestStrongPasswordWithSymbols(t *testing.T) {
	type args struct {
		value string
		min   uint8
		max   uint8
	}
	tests := []struct {
		name      string
		args      args
		wantValid bool // true 表示应通过校验。
	}{
		{name: "s-001", args: args{value: "ABC*-f&#21c", min: 8, max: 12}, wantValid: true},
		{name: "s-002", args: args{value: "*-Af&#2c", min: 8, max: 12}, wantValid: true},
		{name: "s-003", args: args{value: "ABc你123*", min: 8, max: 8}, wantValid: true},
		{name: "e-001", args: args{value: "Fg2B*-AAf&#256cb", min: 8, max: 12}, wantValid: false}, // 超过最大字符数
		{name: "e-002", args: args{value: "ABCEFGHJKL", min: 8, max: 12}, wantValid: false},       // 必须包含小写字母
		{name: "e-003", args: args{value: "abcefghjkl", min: 8, max: 12}, wantValid: false},       // 必须包含大写字母
		{name: "e-004", args: args{value: "123456789", min: 8, max: 12}, wantValid: false},        // 必须包含小写字母
		{name: "e-005", args: args{value: "ABCEFghjkl", min: 8, max: 12}, wantValid: false},       // 必须包含数字
		{name: "e-006", args: args{value: "ABCE56789", min: 8, max: 12}, wantValid: false},        // 必须包含小写字母
		{name: "e-007", args: args{value: "abce56789", min: 8, max: 12}, wantValid: false},        // 必须包含大写字母
		{name: "e-008", args: args{value: "ABc你123*4", min: 8, max: 8}, wantValid: false},         // 超过表内最大字符数
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.StrongPasswordWithSymbols(tt.args.value, tt.args.min, tt.args.max); (err == nil) != tt.wantValid {
				t.Errorf("StrongPasswordWithSymbols() error = %v, wantValid %v, input = %q", err, tt.wantValid, tt.args.value)
			}
		})
	}
}

func TestAccount(t *testing.T) {
	type args struct {
		value string
		min   uint8
		max   uint8
	}
	tests := []struct {
		name      string
		args      args
		wantValid bool // true 表示应通过校验。
	}{
		{name: "s-001", args: args{value: "ABC12321cb", min: 8, max: 12}, wantValid: true},
		{name: "s-002", args: args{value: "ABC123_1cb", min: 8, max: 12}, wantValid: true},
		{name: "s-003", args: args{value: "ABCEFGHJKL", min: 8, max: 12}, wantValid: true},
		{name: "s-004", args: args{value: "abcefghjkl", min: 8, max: 12}, wantValid: true},
		{name: "e-001", args: args{value: "ABC123#1cb", min: 8, max: 12}, wantValid: false},   // 不能使用特殊字符
		{name: "e-002", args: args{value: "ABC123__1cb", min: 8, max: 12}, wantValid: false},  // 不接受连续下划线。
		{name: "e-003", args: args{value: "ABC123___1cb", min: 8, max: 12}, wantValid: false}, // 不接受连续下划线。
		{name: "e-004", args: args{value: "123456789", min: 8, max: 12}, wantValid: false},    // 非字母开头
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.Account(tt.args.value, tt.args.min, tt.args.max); (err == nil) != tt.wantValid {
				t.Errorf("Account() error = %v, wantValid %v", err, tt.wantValid)
			}
		})
	}
}

func TestValidationErrorClassification(t *testing.T) {
	tests := []struct {
		name       string
		run        func() error
		wantReason utils.ValidationReason
		wantMin    uint8
		wantMax    uint8
	}{
		{
			name:       "account_length",
			run:        func() error { return utils.Account("abc", 8, 12) },
			wantReason: utils.ValidationReasonLengthOutOfRange,
			wantMin:    8,
			wantMax:    12,
		},
		{
			name:       "account_consecutive_underscore",
			run:        func() error { return utils.Account("abc__123", 8, 12) },
			wantReason: utils.ValidationReasonConsecutiveUnderscore,
			wantMin:    8,
			wantMax:    12,
		},
		{
			name:       "account_length_before_underscore",
			run:        func() error { return utils.Account("__", 8, 12) },
			wantReason: utils.ValidationReasonLengthOutOfRange,
			wantMin:    8,
			wantMax:    12,
		},
		{
			name:       "account_underscore_before_charset",
			run:        func() error { return utils.Account("!abc__12", 8, 12) },
			wantReason: utils.ValidationReasonConsecutiveUnderscore,
			wantMin:    8,
			wantMax:    12,
		},
		{
			name:       "password_charset",
			run:        func() error { return utils.Password("ABC123#1cb", 8, 12) },
			wantReason: utils.ValidationReasonInvalidCharset,
			wantMin:    8,
			wantMax:    12,
		},
		{
			name:       "password2_missing_lowercase",
			run:        func() error { return utils.StrongPassword("ABCE56789", 8, 12) },
			wantReason: utils.ValidationReasonMissingLowercase,
			wantMin:    8,
			wantMax:    12,
		},
		{
			name:       "password3_missing_digit",
			run:        func() error { return utils.StrongPasswordWithSymbols("ABC*-f&#xy", 8, 12) },
			wantReason: utils.ValidationReasonMissingDigit,
			wantMin:    8,
			wantMax:    12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}

			var validationErr *utils.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ValidationError, got %T", err)
			}
			if validationErr.Reason != tt.wantReason {
				t.Fatalf("ValidationError.Reason = %q, want %q", validationErr.Reason, tt.wantReason)
			}
			if validationErr.Min != tt.wantMin || validationErr.Max != tt.wantMax {
				t.Fatalf("ValidationError range = [%d,%d], want [%d,%d]", validationErr.Min, validationErr.Max, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestValidationErrorDefaultMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil_receiver",
			err:  (*utils.ValidationError)(nil),
			want: "",
		},
		{
			name: "account_length",
			err:  utils.Account("abc", 8, 12),
			want: "长度在8-12之间",
		},
		{
			name: "account_consecutive_underscore",
			err:  utils.Account("abc__123", 8, 12),
			want: "不能连续出现下划线",
		},
		{
			name: "password_charset",
			err:  utils.Password("ABC123#1cb", 8, 12),
			want: "必须包含大小写字母和数字的组合，不能使用特殊字符，长度在8-12之间",
		},
		{
			name: "password2_missing_uppercase",
			err:  utils.StrongPassword("abce56789", 8, 12),
			want: "必须包含至少一个大写字母",
		},
		{
			name: "password3_missing_digit",
			err:  utils.StrongPasswordWithSymbols("ABC*-f&#xy", 8, 12),
			want: "必须包含至少一个数字",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("ValidationError.Error() = %q, want %q", got, tt.want)
			}

			var validationErr *utils.ValidationError
			if !errors.As(tt.err, &validationErr) {
				t.Fatalf("expected ValidationError, got %T", tt.err)
			}
			if got := validationErr.DefaultMessage(); got != tt.want {
				t.Fatalf("ValidationError.DefaultMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidationErrorMessageKey(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "account_length",
			err:  utils.Account("abc", 8, 12),
			want: "validation.length_out_of_range",
		},
		{
			name: "password_charset",
			err:  utils.Password("ABC123#1cb", 8, 12),
			want: "validation.invalid_charset",
		},
		{
			name: "password2_missing_lowercase",
			err:  utils.StrongPassword("ABCE56789", 8, 12),
			want: "validation.missing_lowercase",
		},
		{
			name: "password3_missing_digit",
			err:  utils.StrongPasswordWithSymbols("ABC*-f&#xy", 8, 12),
			want: "validation.missing_digit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var validationErr *utils.ValidationError
			if !errors.As(tt.err, &validationErr) {
				t.Fatalf("expected ValidationError, got %T", tt.err)
			}
			if got := validationErr.MessageKey(); got != tt.want {
				t.Fatalf("ValidationError.MessageKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_hasSymbols(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "001", args: args{value: "A"}, want: false},
		{name: "002", args: args{value: "a"}, want: false},
		{name: "003", args: args{value: "1"}, want: false},
		{name: "004", args: args{value: "aB"}, want: false},
		{name: "005", args: args{value: "A1"}, want: false},
		{name: "006", args: args{value: "中文"}, want: false},
		{name: "007", args: args{value: "&"}, want: true},
		{name: "008", args: args{value: "$"}, want: true},
		{name: "009", args: args{value: "$."}, want: true},
		{name: "010", args: args{value: "A$"}, want: true},
		{name: "011", args: args{value: "A$*12#"}, want: true},
		{name: "012", args: args{value: "$*12#"}, want: true},
		{name: "013", args: args{value: "中文*"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.HasSymbols(tt.args.value); got != tt.want {
				t.Errorf("HasSymbols() = %v, want %v", got, tt.want)
			}
		})
	}
}
