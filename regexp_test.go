package utils_test

import (
	"testing"

	"github.com/Is999/go-utils"
)

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
		// 正确的格式
		{name: "s-001", args: args{value: "100", decimal: 0}, want: true},
		{name: "s-002", args: args{value: "5.23", decimal: 2}, want: true},
		{name: "s-003", args: args{value: "-120", decimal: 2, signed: true}, want: true},
		{name: "s-004", args: args{value: "-100.45", decimal: 2, signed: true}, want: true},
		{name: "s-005", args: args{value: "0", decimal: 2}, want: true},
		{name: "s-006", args: args{value: "0.00", decimal: 2}, want: true},
		{name: "s-007", args: args{value: "+1", decimal: 2, signed: true}, want: true},
		{name: "s-008", args: args{value: "5.3", decimal: 1}, want: true},
		{name: "s-009", args: args{value: "-15.4324", decimal: 4, signed: true}, want: true},
		// 错误的格式
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
		// 正确的格式
		{name: "s-001", args: args{value: "abc@qq.com"}, want: true},
		{name: "s-002", args: args{value: "abc0.12-12@qq.com.cn"}, want: true},
		{name: "s-003", args: args{value: "abc_012-12@qq.com.cn"}, want: true},
		{name: "s-004", args: args{value: "abc_012-12@qq.com.cn"}, want: true},
		{name: "s-005", args: args{value: "abc_1@qq.com"}, want: true},
		{name: "s-006", args: args{value: "abc01212.w@gamil.com.cn"}, want: true},
		// 错误的格式
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
		// 正确的格式
		{name: "s-001", args: args{value: "0123456789"}, want: true},
		{name: "s-002", args: args{value: "hello"}, want: true},
		{name: "s-003", args: args{value: "hello0123456789"}, want: true},
		// 错误的格式
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
		// 正确的格式
		{name: "s-001", args: args{value: "helloWorld"}, want: true},
		// 错误的格式 非存英文字符串
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
		// 正确的格式
		{name: "s-001", args: args{value: "100"}, want: true},
		// 错误的格式
		{name: "e-001", args: args{value: "+123"}, want: false},    // 错误: 包含符号
		{name: "e-002", args: args{value: "5.23"}, want: false},    // 错误: 小数
		{name: "e-003", args: args{value: "5.432"}, want: false},   // 错误: 多出小数位
		{name: "e-004", args: args{value: "-120"}, want: false},    // 错误: 负数
		{name: "e-005", args: args{value: "-100.45"}, want: false}, // 错误: 负数
		{name: "e-006", args: args{value: "0"}, want: false},       // 错误: 0非整数
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
		// 正确的格式
		{name: "s-001", args: args{value: "13148888999"}, want: true},
		// 错误的格式
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
		// 空字符串
		{name: "s-001", args: args{value: "   "}, want: true},
		{name: "s-002", args: args{value: ""}, want: true},
		// 非空字符串
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
		// 正确的格式
		{name: "s-001", args: args{value: "-100.23"}, want: true},
		{name: "s-002", args: args{value: "100.23"}, want: true},
		{name: "s-003", args: args{value: `+100.23`}, want: true},
		{name: "s-004", args: args{value: "100.030"}, want: true},
		{name: "s-005", args: args{value: "0.00"}, want: true},
		// 错误的格式
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
		// 正确的格式
		{name: "s-001", args: args{value: "123456789"}, want: true},
		{name: "s-002", args: args{value: "4001-1046618"}, want: true},
		{name: "s-003", args: args{value: "533-74177002"}, want: true},
		{name: "s-004", args: args{value: "51983045392"}, want: true},
		{name: "s-005", args: args{value: "2010014"}, want: true},
		{name: "s-006", args: args{value: "13148888999"}, want: true},
		{name: "s-007", args: args{value: "95599"}, want: true},
		// 错误的格式
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
		// 正确的格式
		{name: "s-001", args: args{value: "20205"}, want: true},
		{name: "s-002", args: args{value: "125339782132"}, want: true},
		// 错误的格式
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
		// 正确的格式
		{name: "s-001", args: args{value: "2020-01-01"}, want: true},
		{name: "s-002", args: args{value: "2020-02-29"}, want: true},
		{name: "s-003", args: args{value: "2020-01-31"}, want: true},
		{name: "s-004", args: args{value: "2020.01.20"}, want: true},
		{name: "s-005", args: args{value: "2020/1/31"}, want: true},
		{name: "s-006", args: args{value: "2020/6/9"}, want: true},
		{name: "s-007", args: args{value: "2020-9-08"}, want: true},
		{name: "s-008", args: args{value: "2020/01/20"}, want: true},
		{name: "s-009", args: args{value: "2020.5.4"}, want: true},
		// 错误的格式
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
		// 正确格式
		{name: "s-001", args: args{value: "2020-01"}, want: true},
		{name: "s-002", args: args{value: "2020-02"}, want: true},
		{name: "s-003", args: args{value: "1999-02"}, want: true},
		{name: "s-004", args: args{value: "1999-2"}, want: true},
		{name: "s-005", args: args{value: "1999-10"}, want: true},
		// 错误格式
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
		// 正确的格式
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
		// 错误的格式
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
		{name: "e-015", args: args{value: "2020.10.11 24:00:00"}, want: false}, // 时间错误 24点 即 00:00:00
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
		// 正确的格式
		{name: "s-001", args: args{value: "100"}, want: true},
		{name: "s-002", args: args{value: "5.23"}, want: true},
		{name: "s-003", args: args{value: "5.432"}, want: true},
		{name: "s-004", args: args{value: "0"}, want: true},
		{name: "s-005", args: args{value: "0.00"}, want: true},
		// 错误的格式
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
		// 正确的格式
		{name: "s-001", args: args{value: "100"}, want: true},
		{name: "s-002", args: args{value: "0"}, want: true},
		// 错误的格式
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
		// 正确的格式
		{name: "s-001", args: args{value: "中文汉字"}, want: true},
		// 错误的格式
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
		// 正确的格式
		{name: "s-001", args: args{value: "https://wanwang.aliyun.com"}, want: true},
		{name: "s-002", args: args{value: "https://wanwang.aliyun.com/"}, want: true},
		{name: "s-003", args: args{value: "https://wanwang.aliyun.com.cn"}, want: true},
		{name: "s-004", args: args{value: "https://wanwang.aliyun.com.cn/"}, want: true},
		{name: "s-005", args: args{value: "https://wan-wang.aliyun.com.cn/"}, want: true},
		// 错误的格式
		{name: "e-001", args: args{value: "https://wan_wang.aliyun.com.cn/"}, want: false},     // 64位内正确的域名，可包含中文、字母、数字和.-
		{name: "e-002", args: args{value: "https://wanwang.aliyun.com.cn//"}, want: false},     // / 后不能有参数
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

func TestPassWord(t *testing.T) {
	type args struct {
		value string
		min   uint8
		max   uint8
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// 正确的格式
		{name: "s-001", args: args{value: "ABC12321cb", min: 8, max: 12}, wantErr: true},
		{name: "s-002", args: args{value: "ABC123_1cb", min: 8, max: 12}, wantErr: true},
		{name: "s-003", args: args{value: "ABC123__1cb", min: 8, max: 12}, wantErr: true},
		{name: "s-004", args: args{value: "ABC123___1cb", min: 8, max: 12}, wantErr: true},
		{name: "s-005", args: args{value: "ABCEFGHJKL", min: 8, max: 12}, wantErr: true},
		{name: "s-006", args: args{value: "abcefghjkl", min: 8, max: 12}, wantErr: true},
		{name: "s-007", args: args{value: "123456789", min: 8, max: 12}, wantErr: true},
		// 错误的格式
		{name: "e-001", args: args{value: "ABC123#1cb", min: 8, max: 12}, wantErr: false}, // 不能使用特殊字符
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.PassWord(tt.args.value, 8, 12); (err == nil) != tt.wantErr {
				t.Errorf("PassWord() WrapError = %v, wantErr %v, length = %v", err, tt.wantErr, len(tt.args.value))
			} else {
				if err != nil {
					//t.Logf("PassWord() WrapError = %v, length = %v", WrapError, len(tt.args.value))
				}
			}
		})
	}
}

func TestPassWord2(t *testing.T) {
	type args struct {
		value string
		min   uint8
		max   uint8
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// 正确的格式
		{name: "s-001", args: args{value: "ABC12321cb", min: 8, max: 12}, wantErr: true},
		// 错误的格式
		{name: "e-001", args: args{value: "ABC123#1cb", min: 8, max: 12}, wantErr: false},   // 不能使用特殊字符
		{name: "e-002", args: args{value: "ABC123_1cb", min: 8, max: 12}, wantErr: false},   // 不能使用特殊字符
		{name: "e-003", args: args{value: "ABC123__1cb", min: 8, max: 12}, wantErr: false},  // 不能使用特殊字符
		{name: "e-004", args: args{value: "ABC123___1cb", min: 8, max: 12}, wantErr: false}, // 不能使用特殊字符
		{name: "e-005", args: args{value: "ABCEFGHJKL", min: 8, max: 12}, wantErr: false},   // 必须包含小写字母
		{name: "e-006", args: args{value: "abcefghjkl", min: 8, max: 12}, wantErr: false},   // 必须包含大写字母
		{name: "e-007", args: args{value: "123456789", min: 8, max: 12}, wantErr: false},    // 必须包含小写字母
		{name: "e-008", args: args{value: "ABCEFghjkl", min: 8, max: 12}, wantErr: false},   // 必须包含数字
		{name: "e-009", args: args{value: "ABCE56789", min: 8, max: 12}, wantErr: false},    // 必须包含小写字母
		{name: "e-010", args: args{value: "abce56789", min: 8, max: 12}, wantErr: false},    // 必须包含大写字母
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.PassWord2(tt.args.value, 8, 12); (err == nil) != tt.wantErr {
				t.Errorf("PassWord2() WrapError = %v, wantErr %v, length = %v", err, tt.wantErr, len(tt.args.value))
			} else {
				if err != nil {
					//t.Logf("PassWord2() WrapError = %v, length = %v", WrapError, len(tt.args.value))
				}
			}
		})
	}
}

func TestPassWord3(t *testing.T) {
	type args struct {
		value string
		min   uint8
		max   uint8
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// 正确的格式
		{name: "s-001", args: args{value: "ABC*-f&#21c", min: 8, max: 12}, want: true},
		{name: "s-002", args: args{value: "*-Af&#2c", min: 8, max: 12}, want: true},
		// 错误的格式
		{name: "e-001", args: args{value: "Fg2B*-AAf&#256cb", min: 8, max: 12}, want: false}, // 长度16 不在 8 - 12之间
		{name: "e-002", args: args{value: "ABCEFGHJKL", min: 8, max: 12}, want: false},       // 必须包含小写字母
		{name: "e-003", args: args{value: "abcefghjkl", min: 8, max: 12}, want: false},       // 必须包含大写字母
		{name: "e-004", args: args{value: "123456789", min: 8, max: 12}, want: false},        // 必须包含小写字母
		{name: "e-005", args: args{value: "ABCEFghjkl", min: 8, max: 12}, want: false},       // 必须包含数字
		{name: "e-006", args: args{value: "ABCE56789", min: 8, max: 12}, want: false},        // 必须包含小写字母
		{name: "e-007", args: args{value: "abce56789", min: 8, max: 12}, want: false},        // 必须包含大写字母
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.PassWord3(tt.args.value, 8, 12); (err == nil) != tt.want {
				t.Errorf("PassWord3() WrapError = %v, want %v, length = %v", err, tt.want, len(tt.args.value))
			} else {
				if err != nil {
					//t.Logf("PassWord3() WrapError = %v, length = %v", WrapError, len(tt.args.value))
				}
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
		name    string
		args    args
		wantErr bool
	}{
		// 正确的格式
		{name: "s-001", args: args{value: "ABC12321cb", min: 8, max: 12}, wantErr: true},
		{name: "s-002", args: args{value: "ABC123_1cb", min: 8, max: 12}, wantErr: true},
		{name: "s-003", args: args{value: "ABCEFGHJKL", min: 8, max: 12}, wantErr: true},
		{name: "s-004", args: args{value: "abcefghjkl", min: 8, max: 12}, wantErr: true},
		// 错误的格式
		{name: "e-001", args: args{value: "ABC123#1cb", min: 8, max: 12}, wantErr: false},   // 不能使用特殊字符
		{name: "e-002", args: args{value: "ABC123__1cb", min: 8, max: 12}, wantErr: false},  // 不能连续出现下滑线'_'两次或两次以上
		{name: "e-003", args: args{value: "ABC123___1cb", min: 8, max: 12}, wantErr: false}, // 不能连续出现下滑线'_'两次或两次以上
		{name: "e-004", args: args{value: "123456789", min: 8, max: 12}, wantErr: false},    // 非字母开头
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.Account(tt.args.value, tt.args.min, tt.args.max); (err == nil) != tt.wantErr {
				t.Errorf("Account() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				if err != nil {
					//t.Logf("Account() WrapError = %v, length = %v", WrapError, len(tt.args.value))
				}
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
		// 错误
		{name: "001", args: args{value: "A"}, want: false},
		{name: "002", args: args{value: "a"}, want: false},
		{name: "003", args: args{value: "1"}, want: false},
		{name: "004", args: args{value: "aB"}, want: false},
		{name: "005", args: args{value: "A1"}, want: false},
		{name: "006", args: args{value: "中文"}, want: false},
		// 正确的
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
