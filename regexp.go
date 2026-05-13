package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// regexpQQ 校验 QQ 号格式，来源于原始 QQ 正则规则：非 0 开头且总长度为 5-12 位数字。
var regexpQQ = regexp.MustCompile(`^[1-9][0-9]{4,11}$`)

// regexpEmail 校验邮箱格式，保持历史规则：本地部分允许字母数字以及 -_. 分隔，域名后缀限制 1-2 段。
var regexpEmail = regexp.MustCompile(`^[a-z0-9A-Z]+([-_.][a-z0-9A-Z]+)*@[a-z0-9A-Z]+([-_][a-z0-9A-Z]+)*(\.[a-zA-Z]{2,4}){1,2}$`)

// regexpMobile 校验中国大陆手机号格式，来源于当前号段边界：1 开头，第二位为 3-9，总长度 11 位。
var regexpMobile = regexp.MustCompile(`^1[3-9]\d{9}$`)

// regexpPhone 校验中国大陆固定电话或服务号码格式，兼容 3/4 位区号加横线以及 5-11 位纯数字。
var regexpPhone = regexp.MustCompile(`^(\d{3}-\d{8}|\d{4}-\d{7}|\d{5,11})$`)

// regexpAlpha 校验纯英文字母，限定为 ASCII 字母以保持与历史正则一致。
var regexpAlpha = regexp.MustCompile(`^[a-zA-Z]+$`)

// regexpZh 校验纯中文汉字，使用 Unicode Han 字符类，标点和数字不允许通过。
var regexpZh = regexp.MustCompile(`^\p{Han}+$`)

// regexpMixStr 校验英文、数字和常见中英文符号，业务意图是允许展示文本但拒绝中文汉字和换行。
var regexpMixStr = regexp.MustCompile("^[A-Za-z0-9~!@#$%^&*()_+{}|:\"<>?\\-=\\[\\]\\\\;',./ \\t！￥…（）—「」：“”《》？【】、；‘’，。`]+$")

// regexpAlnum 校验纯英文字母与数字，限定 ASCII 范围以避免 Unicode 数字扩大历史行为。
var regexpAlnum = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// regexpDomain 校验不含路径参数的域名或 http(s) URL，保持历史顶级域长度 2-6 的边界。
var regexpDomain = regexp.MustCompile(`^(http(s)?://)?([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,6}(/)?$`)

// regexpTimeMonth 校验 yyyy-MM、yyyy/MM、yyyy.MM 的月份格式，后续不再做日期语义校验。
var regexpTimeMonth = regexp.MustCompile(`^[123]\d{3}[-/.](0?[1-9]|1[0-2])$`)

// regexpTimeDay 校验 yyyy-MM-dd、yyyy/MM/dd、yyyy.MM.dd 的日期外形，真实日期和分隔符一致性由代码补充校验。
var regexpTimeDay = regexp.MustCompile(`^[123]\d{3}[-/.](0?[1-9]|1[0-2])[-/.](0?[1-9]|[12][0-9]|3[01])$`)

// regexpTimestamp 校验日期时间外形，真实日期与分隔符一致性由代码补充校验。
var regexpTimestamp = regexp.MustCompile(`^[123]\d{3}[-/.](0?[1-9]|1[0-2])[-/.](0?[1-9]|[12][0-9]|3[01]) (\d|[01]\d|2[0-3])(:(\d|[0-5]\d)){2}$`)

// regexpSymbols 校验空白分隔符、符号或标点，业务上用于判断字符串是否包含特殊符号。
var regexpSymbols = regexp.MustCompile(`\p{Z}|\p{S}|\p{P}`)

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
	return "validation." + string(e.Reason)
}

// newValidationError 创建一个结构化校验错误。
// 内部统一收敛校验失败的 target、reason 和长度约束信息。
func newValidationError(reason ValidationReason, min, max uint8) error {
	return &ValidationError{
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
	return strings.TrimSpace(value) == ""
}

// QQ QQ号验证
func QQ(value string) bool {
	return regexpQQ.MatchString(value)
}

// Email 电子邮件验证
func Email(value string) bool {
	return regexpEmail.MatchString(value)
}

// Mobile 中国大陆手机号码验证
func Mobile(value string) bool {
	return regexpMobile.MatchString(value)
}

// Phone 中国大陆电话号码验证
func Phone(value string) bool {
	return regexpPhone.MatchString(value)
}

// Numeric 有符号数字验证
func Numeric(value string) bool {
	return validDecimalNumber(value, true, -1)
}

// UnNumeric 无符号数字验证
func UnNumeric(value string) bool {
	return validDecimalNumber(value, false, -1)
}

// UnInteger 无符号整数(正整数)验证
func UnInteger(value string) bool {
	return validUnsignedInteger(value, false)
}

// UnIntZero 无符号整数(正整数+0)验证
func UnIntZero(value string) bool {
	return validUnsignedInteger(value, true)
}

// Amount 金额验证
//
//	amount 金额字符串
//	decimal 保留小数位长度
//	signed 带符号的金额: 默认无符号
func Amount(amount string, decimal uint8, signed ...bool) bool {
	allowSign := len(signed) > 0 && signed[0]
	decimalLimit := 0
	if decimal > 0 {
		decimalLimit = int(decimal)
	}
	return validDecimalNumber(amount, allowSign, decimalLimit)
}

// Alpha 英文字母验证
func Alpha(value string) bool {
	return regexpAlpha.MatchString(value)
}

// Zh 中文字符验证
func Zh(value string) bool {
	return regexpZh.MatchString(value)
}

// MixStr 英文、数字、特殊字符(不包含换行符)
func MixStr(value string) bool {
	return regexpMixStr.MatchString(value)
}

// Alnum 英文字母+数字验证
func Alnum(value string) bool {
	return regexpAlnum.MatchString(value)
}

// Domain 域名(64位内正确的域名，可包含中文、字母、数字和.-)
func Domain(value string) bool {
	return regexpDomain.MatchString(value)
}

// TimeMonth 时间格式验证 yyyy-MM yyyy/MM
func TimeMonth(value string) bool {
	return regexpTimeMonth.MatchString(value)
}

// TimeDay 时间格式验证 yyyy-MM-dd
func TimeDay(value string) bool {
	// 不支持的反向引用: `^[123]\d{3}([-/.])(?:0?[1-9]|1[0-2])\1(?:0?[1-9]|[12][0-9]|3[01])$`
	matched := regexpTimeDay.MatchString(value)

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
	matched := regexpTimestamp.MatchString(value)
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
		return newValidationError(ValidationReasonLengthOutOfRange, min, max)
	}

	// 先检查连续下划线，保持历史错误优先级：即使字符串还存在其它非法字符，也先返回连续下划线原因。
	if hasConsecutiveUnderscore(value) {
		return newValidationError(ValidationReasonConsecutiveUnderscore, min, max)
	}
	if !validAccountChars(value) {
		return newValidationError(ValidationReasonInvalidFormat, min, max)
	}
	return nil
}

// PassWord 密码(字母开头，允许字母数字下划线，长度在 min - max之间)
func PassWord(value string, min, max uint8) error {
	// 验证长度
	l := len(value)
	if l < int(min) || l > int(max) {
		return newValidationError(ValidationReasonLengthOutOfRange, min, max)
	}

	if !validASCIILetterDigitUnderscore(value) {
		return newValidationError(ValidationReasonInvalidCharset, min, max)
	}
	return nil
}

// PassWord2 强密码(必须包含大小写字母和数字的组合，不能使用特殊字符，长度在min-max之间)
func PassWord2(value string, min, max uint8) error {
	// 验证长度
	l := len(value)
	if l < int(min) || l > int(max) {
		return newValidationError(ValidationReasonLengthOutOfRange, min, max)
	}

	// 一次扫描同时收集字符类别和非法字符，减少多次正则匹配产生的 CPU 与分配。
	hasLower, hasUpper, hasDigit, validCharset := passwordASCIIFlags(value, false)
	if !hasLower {
		return newValidationError(ValidationReasonMissingLowercase, min, max)
	}
	if !hasUpper {
		return newValidationError(ValidationReasonMissingUppercase, min, max)
	}
	if !hasDigit {
		return newValidationError(ValidationReasonMissingDigit, min, max)
	}
	if !validCharset {
		return newValidationError(ValidationReasonInvalidCharset, min, max)
	}
	return nil
}

// PassWord3 强密码(必须包含大小写字母和数字的组合，可以使用特殊字符，长度在min-max之间)
func PassWord3(value string, min, max uint8) error {
	// 验证长度
	l := len(value)
	if l < int(min) || l > int(max) {
		return newValidationError(ValidationReasonLengthOutOfRange, min, max)
	}

	// 特殊字符允许通过，但仍需收集大小写与数字要求；最终边界沿用历史 `.` 不匹配换行的行为。
	hasLower, hasUpper, hasDigit, noNewline := passwordASCIIFlags(value, true)
	if !hasLower {
		return newValidationError(ValidationReasonMissingLowercase, min, max)
	}
	if !hasUpper {
		return newValidationError(ValidationReasonMissingUppercase, min, max)
	}
	if !hasDigit {
		return newValidationError(ValidationReasonMissingDigit, min, max)
	}
	if !noNewline || !runeCountInRange(value, int(min), int(max)) {
		return newValidationError(ValidationReasonInvalidFormat, min, max)
	}
	return nil
}

// HasSymbols 是否包含符号
func HasSymbols(value string) bool {
	return regexpSymbols.MatchString(value)
}

// validDecimalNumber 校验十进制数字字符串，覆盖 Numeric、UnNumeric 和 Amount 的共同边界。
//
// 参数说明：
//   - value：待校验字符串，数据来源通常是用户输入或接口参数。
//   - allowSign：是否允许首位出现 + 或 -，用于区分有符号和无符号金额。
//   - maxDecimal：允许的小数位上限；-1 表示不限制小数位，0 表示不允许小数。
func validDecimalNumber(value string, allowSign bool, maxDecimal int) bool {
	if value == "" {
		return false
	}

	i := 0
	if value[0] == '+' || value[0] == '-' {
		if !allowSign || len(value) == 1 {
			return false
		}
		i = 1
	}

	// 整数部分只允许单个 0 或非 0 开头数字，避免 00、01 这类金额歧义。
	if i >= len(value) {
		return false
	}
	if value[i] == '0' {
		i++
	} else if isASCIIDigitNonZero(value[i]) {
		for i < len(value) && isASCIIDigit(value[i]) {
			i++
		}
	} else {
		return false
	}

	if i == len(value) {
		return true
	}
	if value[i] != '.' || maxDecimal == 0 {
		return false
	}
	i++
	if i == len(value) {
		return false
	}

	decimalLen := 0
	for i < len(value) {
		if !isASCIIDigit(value[i]) {
			return false
		}
		decimalLen++
		if maxDecimal > 0 && decimalLen > maxDecimal {
			return false
		}
		i++
	}
	return true
}

// validUnsignedInteger 校验无符号整数，allowZero 控制是否允许单独的 0 通过。
func validUnsignedInteger(value string, allowZero bool) bool {
	if value == "" {
		return false
	}
	if value == "0" {
		return allowZero
	}
	if !isASCIIDigitNonZero(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		if !isASCIIDigit(value[i]) {
			return false
		}
	}
	return true
}

// validAccountChars 校验账号字符边界：必须以 ASCII 字母开头，后续仅允许字母、数字和下划线。
func validAccountChars(value string) bool {
	if value == "" || !isASCIILetter(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		if !isASCIILetter(value[i]) && !isASCIIDigit(value[i]) && value[i] != '_' {
			return false
		}
	}
	return true
}

// hasConsecutiveUnderscore 判断是否存在连续下划线，供账号校验维持历史错误分类优先级。
func hasConsecutiveUnderscore(value string) bool {
	previousUnderscore := false
	for i := 0; i < len(value); i++ {
		if value[i] == '_' {
			if previousUnderscore {
				return true
			}
			previousUnderscore = true
			continue
		}
		previousUnderscore = false
	}
	return false
}

// validASCIILetterDigitUnderscore 校验密码字符集，保持历史 `\w` 的 ASCII 字母、数字、下划线语义。
func validASCIILetterDigitUnderscore(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		if !isASCIILetter(value[i]) && !isASCIIDigit(value[i]) && value[i] != '_' {
			return false
		}
	}
	return true
}

// passwordASCIIFlags 一次扫描密码字符类别，allowSpecial 表示是否允许特殊字符参与最终格式校验。
func passwordASCIIFlags(value string, allowSpecial bool) (hasLower, hasUpper, hasDigit, valid bool) {
	valid = true
	for i := 0; i < len(value); i++ {
		ch := value[i]
		switch {
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case isASCIIDigit(ch):
			hasDigit = true
		case allowSpecial:
			if ch == '\n' {
				valid = false
			}
		default:
			valid = false
		}
	}
	return hasLower, hasUpper, hasDigit, valid
}

// runeCountInRange 校验字符串的 Unicode 字符数边界，用于模拟历史正则 `.` 按字符计数的行为。
func runeCountInRange(value string, min, max int) bool {
	count := 0
	for range value {
		count++
		if count > max {
			return false
		}
	}
	return count >= min
}

// isASCIILetter 判断单字节是否为 ASCII 英文字母，避免 Unicode 扩展导致历史校验范围变化。
func isASCIILetter(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}

// isASCIIDigit 判断单字节是否为 ASCII 数字。
func isASCIIDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// isASCIIDigitNonZero 判断单字节是否为 1-9，用于拒绝带前导零的数字格式。
func isASCIIDigitNonZero(ch byte) bool {
	return ch >= '1' && ch <= '9'
}
