package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// regexpEmail 校验邮箱格式，保持历史规则：本地部分允许字母数字以及 -_. 分隔，域名后缀限制 1-2 段。
var regexpEmail = regexp.MustCompile(`^[a-z0-9A-Z]+([-_.][a-z0-9A-Z]+)*@[a-z0-9A-Z]+([-_][a-z0-9A-Z]+)*(\.[a-zA-Z]{2,4}){1,2}$`)

// regexpMobile 只检查 1[3-9] 开头的 11 位外形，不查询号段分配或号码状态。
var regexpMobile = regexp.MustCompile(`^1[3-9]\d{9}$`)

// regexpPhone 校验中国大陆固定电话或服务号码格式，兼容 3/4 位区号加横线以及 5-11 位纯数字。
var regexpPhone = regexp.MustCompile(`^(\d{3}-\d{8}|\d{4}-\d{7}|\d{5,11})$`)

// regexpAlpha 校验纯英文字母，限定为 ASCII 字母以保持与历史正则一致。
var regexpAlpha = regexp.MustCompile(`^[a-zA-Z]+$`)

// regexpZh 校验纯中文汉字，使用 Unicode Han 字符类，标点和数字不允许通过。
var regexpZh = regexp.MustCompile(`^\p{Han}+$`)

// regexpMixStr 只接受列出的字母、数字和中英文符号，允许空格与制表符，不含汉字或换行。
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

// regexpSymbols 匹配 Unicode 分隔符、符号或标点；不包含制表符、换行等控制字符。
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
// 业务侧应通过 errors.As 提取后，根据 Reason 自行决定提示文案。
type ValidationError struct {
	Reason ValidationReason // 校验失败原因
	Min    uint8            // 最小长度，单位由具体校验函数决定。
	Max    uint8            // 最大长度，与 Min 使用同一单位。
}

// Error 返回默认中文文案；需区分失败原因时读取 Reason，nil 接收者返回空串。
func (e *ValidationError) Error() string {
	return e.DefaultMessage()
}

// DefaultMessage 返回默认中文提示文案。
func (e *ValidationError) DefaultMessage() string {
	if e == nil {
		return ""
	}
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

// MessageKey 返回稳定的文案键。
// 业务侧可基于该键做国际化映射、错误码映射或统一前端提示管理。
func (e *ValidationError) MessageKey() string {
	if e == nil {
		return ""
	}
	return "validation." + string(e.Reason)
}

// Empty 判断字符串是否为空或全部由 Unicode 空白字符组成。
func Empty(value string) bool {
	return strings.TrimSpace(value) == ""
}

// QQ 校验 5-12 位、非零开头的 ASCII 数字。
func QQ(value string) bool {
	return len(value) >= 5 && len(value) <= 12 && validUnsignedInteger(value, false)
}

// Email 按固定格式校验邮箱；不检查可投递性，域名后缀限定为 1-2 段、每段 2-4 个字母。
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

// Numeric 校验可带正负号的 ASCII 十进制整数或小数，不接受多位前导零、空白和指数写法。
func Numeric(value string) bool {
	return validDecimalNumber(value, true, -1)
}

// UnNumeric 使用 Numeric 的数字规则，但不接受正负号。
func UnNumeric(value string) bool {
	return validDecimalNumber(value, false, -1)
}

// UnInteger 校验不带符号和前导零的 ASCII 正整数，不接受 0。
func UnInteger(value string) bool {
	return validUnsignedInteger(value, false)
}

// UnIntZero 使用 UnInteger 的规则，并允许单独的 0。
func UnIntZero(value string) bool {
	return validUnsignedInteger(value, true)
}

// Amount 校验 ASCII 十进制格式，不检查数值大小；整数部分不允许多位前导零。
// decimal 是最大小数位数，0 禁止小数点；signed 第一项为 true 时允许正负号。
func Amount(amount string, decimal uint8, signed ...bool) bool {
	allowSign := len(signed) > 0 && signed[0]
	return validDecimalNumber(amount, allowSign, int(decimal))
}

// Alpha 校验非空 ASCII 英文字母串。
func Alpha(value string) bool {
	return regexpAlpha.MatchString(value)
}

// Zh 校验非空 Unicode Han 字符串，不接受标点。
func Zh(value string) bool {
	return regexpZh.MatchString(value)
}

// MixStr 校验字母、数字及固定符号集合；允许空格、制表符，不接受汉字和换行。
func MixStr(value string) bool {
	return regexpMixStr.MatchString(value)
}

// Alnum 校验非空 ASCII 字母数字串。
func Alnum(value string) bool {
	return regexpAlnum.MatchString(value)
}

// Domain 校验 ASCII 域名，可带 http(s):// 和末尾斜线；不接受中文、端口或路径。
// 普通标签最长 63 字节，顶级域限 2-6 个字母，不检查域名是否存在。
func Domain(value string) bool {
	return regexpDomain.MatchString(value)
}

// TimeMonth 校验年和月，年份限 1000-3999，月份可为一位或两位，分隔符为 -、/ 或 .。
func TimeMonth(value string) bool {
	return regexpTimeMonth.MatchString(value)
}

// TimeDay 校验真实日期，年和月规则同 TimeMonth，日可为一位或两位，两个分隔符须相同。
func TimeDay(value string) bool {
	if !regexpTimeDay.MatchString(value) {
		return false
	}
	// 正则已固定年和月的位置；再次找到相同分隔符才允许校验真实日期。
	month, day, matched := strings.Cut(value[5:], value[4:5])
	return matched && CheckDate(ToInt(value[:4]), ToInt(month), ToInt(day))
}

// Timestamp 校验 TimeDay 日期加一个空格及 24 小时时分秒，时分秒可为一位或两位。
func Timestamp(value string) bool {
	if !regexpTimestamp.MatchString(value) {
		return false
	}
	// 时间范围已由正则确认，日期部分还需检查分隔符一致性和月末、闰年边界。
	date, _, _ := strings.Cut(value, " ")
	month, day, matched := strings.Cut(date[5:], date[4:5])
	return matched && CheckDate(ToInt(date[:4]), ToInt(month), ToInt(day))
}

// Account 校验 ASCII 字母开头、后续为字母数字下划线的账号，长度按字节计。
// 长度超限、连续下划线、格式错误按此顺序报告；min/max 不自动交换。
func Account(value string, min, max uint8) error {
	l := len(value)
	if l < int(min) || l > int(max) {
		return &ValidationError{Reason: ValidationReasonLengthOutOfRange, Min: min, Max: max}
	}

	// 连续下划线优先于其他非法字符报告。
	if strings.Contains(value, "__") {
		return &ValidationError{Reason: ValidationReasonConsecutiveUnderscore, Min: min, Max: max}
	}
	if !validAccountChars(value) {
		return &ValidationError{Reason: ValidationReasonInvalidFormat, Min: min, Max: max}
	}
	return nil
}

// Password 校验非空 ASCII 字母数字下划线串，不限制首字符，长度按字节计。
func Password(value string, min, max uint8) error {
	l := len(value)
	if l < int(min) || l > int(max) {
		return &ValidationError{Reason: ValidationReasonLengthOutOfRange, Min: min, Max: max}
	}

	if !validASCIILetterDigitUnderscore(value) {
		return &ValidationError{Reason: ValidationReasonInvalidCharset, Min: min, Max: max}
	}
	return nil
}

// StrongPassword 要求同时包含 ASCII 大小写字母和数字，且不得有其他字符，长度按字节计。
func StrongPassword(value string, min, max uint8) error {
	l := len(value)
	if l < int(min) || l > int(max) {
		return &ValidationError{Reason: ValidationReasonLengthOutOfRange, Min: min, Max: max}
	}

	// 失败顺序沿用原规则：小写、大写、数字、字符集。
	hasLower, hasUpper, hasDigit, validCharset := passwordASCIIFlags(value, false)
	if !hasLower {
		return &ValidationError{Reason: ValidationReasonMissingLowercase, Min: min, Max: max}
	}
	if !hasUpper {
		return &ValidationError{Reason: ValidationReasonMissingUppercase, Min: min, Max: max}
	}
	if !hasDigit {
		return &ValidationError{Reason: ValidationReasonMissingDigit, Min: min, Max: max}
	}
	if !validCharset {
		return &ValidationError{Reason: ValidationReasonInvalidCharset, Min: min, Max: max}
	}
	return nil
}

// StrongPasswordWithSymbols 要求同时包含 ASCII 大小写字母和数字，允许其他字符但不接受换行。
// 长度按 rune 数计，非法 UTF-8 字节各按一个替换字符计数。
func StrongPasswordWithSymbols(value string, min, max uint8) error {
	if !runeCountInRange(value, int(min), int(max)) {
		return &ValidationError{Reason: ValidationReasonLengthOutOfRange, Min: min, Max: max}
	}

	// 与 StrongPassword 保持相同的字符类别检查顺序，最后检查换行。
	hasLower, hasUpper, hasDigit, noNewline := passwordASCIIFlags(value, true)
	if !hasLower {
		return &ValidationError{Reason: ValidationReasonMissingLowercase, Min: min, Max: max}
	}
	if !hasUpper {
		return &ValidationError{Reason: ValidationReasonMissingUppercase, Min: min, Max: max}
	}
	if !hasDigit {
		return &ValidationError{Reason: ValidationReasonMissingDigit, Min: min, Max: max}
	}
	if !noNewline {
		return &ValidationError{Reason: ValidationReasonInvalidFormat, Min: min, Max: max}
	}
	return nil
}

// HasSymbols 检查 Unicode 分隔符、符号或标点；空格算符号，制表符和换行不算。
func HasSymbols(value string) bool {
	return regexpSymbols.MatchString(value)
}

// validDecimalNumber 校验十进制数字字符串，覆盖 Numeric、UnNumeric 和 Amount 的共同边界。
// maxDecimal 为 0 时只接受整数，负数表示不限制小数位；符号由 allowSign 控制。
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

	// 保留原数字格式边界：整数部分只能是单个 0 或非零开头。
	switch {
	case value[i] == '0':
		i++
	case isASCIIDigitNonZero(value[i]):
		for i < len(value) && isASCIIDigit(value[i]) {
			i++
		}
	default:
		return false
	}

	if i == len(value) {
		return true
	}
	if value[i] != '.' || maxDecimal == 0 {
		return false
	}
	i++
	// 小数位数由剩余字节数确定，先检查精度，再逐字节验证 ASCII 数字。
	if i == len(value) || maxDecimal > 0 && len(value)-i > maxDecimal {
		return false
	}

	for ; i < len(value); i++ {
		if !isASCIIDigit(value[i]) {
			return false
		}
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
