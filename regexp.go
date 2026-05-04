package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Is999/go-utils/errors"
)

// ValidationTarget 表示校验目标类型。
// 用于区分当前错误来自账号、普通密码或强密码等不同场景。
type ValidationTarget string

const (
	// ValidationTargetAccount 表示账号校验。
	ValidationTargetAccount ValidationTarget = "account"
	// ValidationTargetPassword 表示普通密码校验。
	ValidationTargetPassword ValidationTarget = "password"
	// ValidationTargetStrongPassword 表示仅允许字母和数字的强密码校验。
	ValidationTargetStrongPassword ValidationTarget = "strong_password"
	// ValidationTargetStrongPasswordWithChars 表示允许特殊字符的强密码校验。
	ValidationTargetStrongPasswordWithChars ValidationTarget = "strong_password_with_symbols"
)

// ValidationReason 表示校验失败原因。
// 业务侧可根据该字段决定最终提示文案、错误码或国际化文案键。
type ValidationReason string

const (
	// ValidationReasonLengthOutOfRange 表示长度不在允许范围内。
	ValidationReasonLengthOutOfRange ValidationReason = "length_out_of_range"
	// ValidationReasonInvalidFormat 表示整体格式不符合要求。
	ValidationReasonInvalidFormat ValidationReason = "invalid_format"
	// ValidationReasonInvalidCharset 表示字符集不符合要求。
	ValidationReasonInvalidCharset ValidationReason = "invalid_charset"
	// ValidationReasonConsecutiveUnderscore 表示存在连续下划线。
	ValidationReasonConsecutiveUnderscore ValidationReason = "consecutive_underscore"
	// ValidationReasonMissingLowercase 表示缺少小写字母。
	ValidationReasonMissingLowercase ValidationReason = "missing_lowercase"
	// ValidationReasonMissingUppercase 表示缺少大写字母。
	ValidationReasonMissingUppercase ValidationReason = "missing_uppercase"
	// ValidationReasonMissingDigit 表示缺少数字。
	ValidationReasonMissingDigit ValidationReason = "missing_digit"
)

// ValidationError 表示结构化校验失败结果。
// 业务侧应通过 errors.As 提取后，根据 Target/Reason 自行决定提示文案。
type ValidationError struct {
	Target ValidationTarget // 校验目标类型
	Reason ValidationReason // 校验失败原因
	Min    uint8            // 最小长度约束
	Max    uint8            // 最大长度约束
}

// Error 返回默认中文提示文案。
// 当业务侧未自定义提示时，该文案可以直接返回给终端用户；
// 如业务侧已有国际化或错误码体系，仍建议优先使用 Target/Reason 自行映射。
func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.DefaultMessage()
}

// DefaultMessage 返回默认中文提示文案。
// 该方法适合作为兜底展示文案，也便于业务侧显式区分 Error() 与默认文案语义。
func (e *ValidationError) DefaultMessage() string {
	if e == nil {
		return ""
	}
	return defaultValidationMessage(e)
}

// MessageKey 返回稳定的文案键。
// 业务侧可基于该键做国际化映射、错误码映射或统一前端提示管理。
func (e *ValidationError) MessageKey() string {
	if e == nil {
		return ""
	}
	return "validation." + string(e.Target) + "." + string(e.Reason)
}

// newValidationError 创建一个结构化校验错误。
// 内部统一收敛校验失败的 target、reason 和长度约束信息。
func newValidationError(target ValidationTarget, reason ValidationReason, min, max uint8) error {
	return &ValidationError{
		Target: target,
		Reason: reason,
		Min:    min,
		Max:    max,
	}
}

// defaultValidationMessage 根据目标类型和失败原因生成默认中文提示。
// 该提示面向终端用户，要求简洁、自然且可直接展示。
func defaultValidationMessage(e *ValidationError) string {
	switch e.Reason {
	case ValidationReasonLengthOutOfRange:
		return fmt.Sprintf("长度在%d-%d之间", e.Min, e.Max)
	case ValidationReasonConsecutiveUnderscore:
		return "不能连续出现下划线"
	case ValidationReasonMissingLowercase:
		return "必须包含至少一个小写字母"
	case ValidationReasonMissingUppercase:
		return "必须包含至少一个大写字母"
	case ValidationReasonMissingDigit:
		return "必须包含至少一个数字"
	case ValidationReasonInvalidCharset:
		return fmt.Sprintf("必须包含大小写字母和数字的组合，不能使用特殊字符，长度在%d-%d之间", e.Min, e.Max)
	case ValidationReasonInvalidFormat:
		return fmt.Sprintf("字母开头，允许字母数字下划线，长度在%d-%d之间", e.Min, e.Max)
	}

	if e.Min == 0 && e.Max == 0 {
		return "输入不合法"
	}
	return fmt.Sprintf("输入不合法，长度必须在 %d 到 %d 个字符之间", e.Min, e.Max)
}

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

// UnIntZero 无符号整数(正整数+0)验证
func UnIntZero(value string) bool {
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

// Alnum 英文字母+数字验证
func Alnum(value string) bool {
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
		return newValidationError(ValidationTargetAccount, ValidationReasonLengthOutOfRange, min, max)
	}

	// 不能连续出现下滑线'_'两次或两次以上"
	reg := regexp.MustCompile(`(_{2,})`)
	s := reg.FindString(value)
	if s != "" {
		return newValidationError(ValidationTargetAccount, ValidationReasonConsecutiveUnderscore, min, max)
	}
	matched, err := regexp.MatchString(fmt.Sprintf(`^[a-zA-Z][a-zA-Z0-9_]{%d,%d}$`, min, max), value)
	if err != nil {
		return errors.Tag(err)
	}
	if !matched {
		return newValidationError(ValidationTargetAccount, ValidationReasonInvalidFormat, min, max)
	}
	return nil
}

// PassWord 密码(字母开头，允许字母数字下划线，长度在 min - max之间)
func PassWord(value string, min, max uint8) error {
	// 验证长度
	l := len(value)
	if l < int(min) || l > int(max) {
		return newValidationError(ValidationTargetPassword, ValidationReasonLengthOutOfRange, min, max)
	}

	matched, err := regexp.MatchString(fmt.Sprintf(`^\w{%d,%d}$`, min, max), value)
	if err != nil {
		return errors.Tag(err)
	}
	if !matched {
		return newValidationError(ValidationTargetPassword, ValidationReasonInvalidCharset, min, max)
	}
	return nil
}

// PassWord2 强密码(必须包含大小写字母和数字的组合，不能使用特殊字符，长度在min-max之间)
func PassWord2(value string, min, max uint8) error {
	// 验证长度
	l := len(value)
	if l < int(min) || l > int(max) {
		return newValidationError(ValidationTargetStrongPassword, ValidationReasonLengthOutOfRange, min, max)
	}

	// 是否包含小写字母
	reg := regexp.MustCompile(`([a-z])`)
	s := reg.FindString(value)
	if s == "" {
		return newValidationError(ValidationTargetStrongPassword, ValidationReasonMissingLowercase, min, max)
	}

	// 是否包含大写字母
	reg = regexp.MustCompile(`([A-Z])`)
	s = reg.FindString(value)
	if s == "" {
		return newValidationError(ValidationTargetStrongPassword, ValidationReasonMissingUppercase, min, max)
	}

	// 是否包含数字
	reg = regexp.MustCompile(`([0-9])`)
	s = reg.FindString(value)
	if s == "" {
		return newValidationError(ValidationTargetStrongPassword, ValidationReasonMissingDigit, min, max)
	}

	// 匹配表达式
	matched, err := regexp.MatchString(fmt.Sprintf(`^[a-zA-Z0-9]{%d,%d}$`, min, max), value)
	if err != nil {
		return errors.Tag(err)
	}
	if !matched {
		return newValidationError(ValidationTargetStrongPassword, ValidationReasonInvalidCharset, min, max)
	}
	return nil
}

// PassWord3 强密码(必须包含大小写字母和数字的组合，可以使用特殊字符，长度在min-max之间)
func PassWord3(value string, min, max uint8) error {
	// 验证长度
	l := len(value)
	if l < int(min) || l > int(max) {
		return newValidationError(ValidationTargetStrongPasswordWithChars, ValidationReasonLengthOutOfRange, min, max)
	}

	// 是否包含小写字母
	reg := regexp.MustCompile(`([a-z])`)
	s := reg.FindString(value)
	if s == "" {
		return newValidationError(ValidationTargetStrongPasswordWithChars, ValidationReasonMissingLowercase, min, max)
	}

	// 是否包含大写字母
	reg = regexp.MustCompile(`([A-Z])`)
	s = reg.FindString(value)
	if s == "" {
		return newValidationError(ValidationTargetStrongPasswordWithChars, ValidationReasonMissingUppercase, min, max)
	}

	// 是否包含数字
	reg = regexp.MustCompile(`([0-9])`)
	s = reg.FindString(value)
	if s == "" {
		return newValidationError(ValidationTargetStrongPasswordWithChars, ValidationReasonMissingDigit, min, max)
	}

	// 匹配表达式
	matched, err := regexp.MatchString(fmt.Sprintf(`^.{%d,%d}$`, min, max), value)
	if err != nil {
		return errors.Tag(err)
	}
	if !matched {
		return newValidationError(ValidationTargetStrongPasswordWithChars, ValidationReasonInvalidFormat, min, max)
	}
	return nil
}

// HasSymbols 是否包含符号
func HasSymbols(value string) bool {
	matched, _ := regexp.MatchString(`\p{Z}|\p{S}|\p{P}`, value)
	return matched
}
