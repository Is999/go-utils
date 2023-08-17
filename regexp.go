package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Empty 空字符串验证
func Empty(value string) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	return false
}

// QQ QQ号验证
func QQ(value string) bool {
	matched, _ := regexp.MatchString(`^[1-9][0-9]{4,11}$`, value)
	return matched
}

// Email 电子邮件验证
func Email(value string) bool {
	matched, _ := regexp.MatchString(`^[a-z0-9A-Z]+([-_.][a-z0-9A-Z]+)*@[a-z0-9A-Z]+([-_][a-z0-9A-Z]+)*(\.[a-zA-Z]{2,4}){1,2}$`, value)
	return matched
}

// Mobile 中国大陆手机号码验证
func Mobile(value string) bool {
	matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, value)
	return matched
}

// Phone 中国大陆电话号码验证
func Phone(value string) bool {
	matched, _ := regexp.MatchString(`^(\d{3}-\d{8}|\d{4}-\d{7}|\d{5,11})$`, value)
	return matched
}

// Numeric 有符号数字验证
func Numeric(value string) bool {
	// /\pN/u
	matched, _ := regexp.MatchString(`^([+-])?(0|[1-9]\d*)(\.\d+)?$`, value)
	return matched
}

// UnNumeric 无符号数字验证
func UnNumeric(value string) bool {
	matched, _ := regexp.MatchString(`^(0|[1-9]\d*)(\.\d+)?$`, value)
	return matched
}

// UnInteger 无符号整数(正整数)验证
func UnInteger(value string) bool {
	matched, _ := regexp.MatchString(`^([1-9]\d*)$`, value)
	return matched
}

// ZeroUint 无符号整数(正整数+0)验证
func ZeroUint(value string) bool {
	matched, _ := regexp.MatchString(`^(0|[1-9]\d*)$`, value)
	return matched
}

// Amount 金额验证
//
//	amount 金额字符串
//	decimal 保留小数位长度
//	signed 带符号的金额: 默认无符号
func Amount(amount string, decimal uint8, signed ...bool) bool {
	s := strings.Builder{}
	s.Grow(31) // 预分配内存
	s.WriteString(`^`)
	if len(signed) > 0 && signed[0] {
		s.WriteString(`[+-]?`)
	}
	s.WriteString(`(0|[1-9]\d*)`)
	if decimal > 0 {
		s.WriteString(`(\.\d{1,` + strconv.Itoa(int(decimal)) + `})?`)
	}
	s.WriteString(`$`)

	// 无小数位: `^(0|[1-9]\d*)$`
	// 无符号: `^(0|[1-9]\d*)(?:\.\d{1,2})?$`
	// 有符号: `^[+-]?(0|[1-9]\d*)(\.\d{1,2})?$`
	matched, _ := regexp.MatchString(s.String(), amount)
	return matched
}

// Alpha 英文字母验证
func Alpha(value string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z]+$`, value)
	return matched
}

// Zh 中文字符验证
func Zh(value string) bool {
	matched, _ := regexp.MatchString(`^\p{Han}+$`, value)
	return matched
}

// MixStr 英文、数字、特殊字符(不包含换行符)
func MixStr(value string) bool {
	matched, _ := regexp.MatchString("^[A-Za-z0-9~!@#$%^&*()_+{}|:\"<>?\\-=\\[\\]\\\\;',./ \\t！￥…（）—「」：“”《》？【】、；‘’，。`]+$", value)
	return matched
}

// AlphaNum 英文字母+数字验证
func AlphaNum(value string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, value)
	return matched
}

// Domain 域名(64位内正确的域名，可包含中文、字母、数字和.-)
func Domain(value string) bool {
	matched, _ := regexp.MatchString(`^(http(s)?://)?([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,6}(/)?$`, value)
	return matched
}

// TimeMonth 时间格式验证 yyyy-MM yyyy/MM
func TimeMonth(value string) bool {
	matched, _ := regexp.MatchString(`^[123]\d{3}[-/.](0?[1-9]|1[0-2])$`, value)
	return matched
}

// TimeDay 时间格式验证 yyyy-MM-dd
func TimeDay(value string) bool {
	// 不支持的反向引用: `^[123]\d{3}([-/.])(?:0?[1-9]|1[0-2])\1(?:0?[1-9]|[12][0-9]|3[01])$`
	matched, _ := regexp.MatchString(`^[123]\d{3}[-/.](0?[1-9]|1[0-2])[-/.](0?[1-9]|[12][0-9]|3[01])$`, value)

	if matched {
		// 验证分割符是否一致
		if strings.Count(value, string(value[4])) != 2 {
			return false
		}

		// 验证时间是否正确
		i := strings.LastIndex(value, string(value[4])) // 最后一个分割符下标
		year := value[0:4]
		month := value[5:i]
		day := value[i+1:]
		// 验证日期是否正确
		matched = CheckDate(Str2Int(year), Str2Int(month), Str2Int(day))
	}
	return matched
}

// Timestamp 时间格式验证 yyyy-MM-dd hh:mm:ss
func Timestamp(value string) bool {
	matched, _ := regexp.MatchString(`^[123]\d{3}[-/.](0?[1-9]|1[0-2])[-/.](0?[1-9]|[12][0-9]|3[01]) (\d|[01]\d|2[0-3])(:(\d|[0-5]\d)){2}$`, value)
	if matched {
		// 验证分割符是否一致
		if strings.Count(value, string(value[4])) != 2 {
			return false
		}

		// 验证时间是否正确
		i := strings.LastIndex(value, string(value[4])) // 最后一个分割符下标
		space := strings.Index(value, " ")              // 空字符串下标
		year := value[0:4]
		month := value[5:i]
		day := value[i+1 : space]
		// 验证日期是否正确
		matched = CheckDate(Str2Int(year), Str2Int(month), Str2Int(day))
	}
	return matched
}

// Account 帐号验证(字母开头，允许字母数字下划线，长度在min-max之间)
func Account(value string, min, max uint8) error {
	// 验证长度
	l := len(value)
	if l < int(min) || l > int(max) {
		return Error("长度在%d-%d之间", min, max)
	}

	// 不能连续出现下滑线'_'两次或两次以上"
	reg := regexp.MustCompile(`(_{2,})`)
	s := reg.FindString(value)
	if s != "" {
		return Error("不能连续出现下滑线'_'两次或两次以上")
	}
	matched, err := regexp.MatchString(fmt.Sprintf(`^[a-zA-Z][a-zA-Z0-9_]{%d,%d}$`, min, max), value)
	if err != nil {
		return Wrap(err)
	}
	if !matched {
		return Error("字母开头，允许字母数字下划线，长度在%d-%d之间", min, max)
	}
	return nil
}

// PassWord 密码(字母开头，允许字母数字下划线，长度在 min - max之间)
func PassWord(value string, min, max uint8) error {
	// 验证长度
	l := len(value)
	if l < int(min) || l > int(max) {
		return Error("长度在%d-%d之间", min, max)
	}

	matched, err := regexp.MatchString(fmt.Sprintf(`^\w{%d,%d}$`, min, max), value)
	if err != nil {
		return Wrap(err)
	}
	if !matched {
		return Error("允许字母数字下划线，长度在%d-%d之间", min, max)
	}
	return nil
}

// PassWord2 强密码(必须包含大小写字母和数字的组合，不能使用特殊字符，长度在min-max之间)
func PassWord2(value string, min, max uint8) error {
	// 验证长度
	l := len(value)
	if l < int(min) || l > int(max) {
		return Error("长度在%d-%d之间", min, max)
	}

	// 是否包含小写字母
	reg := regexp.MustCompile(`([a-z])`)
	s := reg.FindString(value)
	if s == "" {
		return Error("必须包含小写字母")
	}

	// 是否包含大写字母
	reg = regexp.MustCompile(`([A-Z])`)
	s = reg.FindString(value)
	if s == "" {
		return Error("必须包含大写字母")
	}

	// 是否包含数字
	reg = regexp.MustCompile(`([0-9])`)
	s = reg.FindString(value)
	if s == "" {
		return Error("必须包含数字")
	}

	// 匹配表达式
	matched, err := regexp.MatchString(fmt.Sprintf(`^[a-zA-Z0-9]{%d,%d}$`, min, max), value)
	if err != nil {
		return Wrap(err)
	}
	if !matched {
		return Error("必须包含大小写字母和数字的组合，不能使用特殊字符，长度在%d-%d之间", min, max)
	}
	return nil
}

// PassWord3 强密码(必须包含大小写字母和数字的组合，可以使用特殊字符，长度在min-max之间)
func PassWord3(value string, min, max uint8) error {
	// 验证长度
	l := len(value)
	if l < int(min) || l > int(max) {
		return Error("长度在%d-%d之间", min, max)
	}

	// 是否包含小写字母
	reg := regexp.MustCompile(`([a-z])`)
	s := reg.FindString(value)
	if s == "" {
		return Error("必须包含小写字母")
	}

	// 是否包含大写字母
	reg = regexp.MustCompile(`([A-Z])`)
	s = reg.FindString(value)
	if s == "" {
		return Error("必须包含大写字母")
	}

	// 是否包含数字
	reg = regexp.MustCompile(`([0-9])`)
	s = reg.FindString(value)
	if s == "" {
		return Error("必须包含数字")
	}

	// 匹配表达式
	matched, err := regexp.MatchString(fmt.Sprintf(`^.{%d,%d}$`, min, max), value)
	if err != nil {
		return Wrap(err)
	}
	if !matched {
		return Error("必须包含大小写字母和数字的组合，可以使用特殊字符，长度在%d-%d之间", min, max)
	}
	return nil
}

// HasSymbols 是否包含符号
func HasSymbols(value string) bool {
	matched, _ := regexp.MatchString(`\p{Z}|\p{S}|\p{P}`, value)
	return matched
}
