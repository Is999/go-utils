package utils

import (
	"strconv"

	"github.com/Is999/go-utils/errors"
)

// ToInt 将十进制字符串转为 int，忽略解析错误。
// 接受正负号，不去除空白；语法错误返回 0，溢出返回对应的整数边界值。
func ToInt(s string) (i int) {
	i, _ = strconv.Atoi(s)
	return
}

// ToInt64 将十进制字符串转为 int64，忽略解析错误。
// 接受正负号，不去除空白；语法错误返回 0，溢出返回对应的整数边界值。
func ToInt64(s string) (i int64) {
	i, _ = strconv.ParseInt(s, 10, 64)
	return
}

// ToFloat64 将字符串转为 float64，忽略解析错误。
// 语法错误返回 0；溢出时返回对应符号的 Inf，NaN 和 Inf 沿用 strconv.ParseFloat 的语义。
func ToFloat64(s string) (i float64) {
	i, _ = strconv.ParseFloat(s, 64)
	return
}

// BinOct 将二进制字符串转为八进制，数值范围限 int64，结果不带进制前缀。
func BinOct(str string) (string, error) {
	return convertBase(str, 2, 8)
}

// BinDec 解析带可选正负号的二进制整数；格式错误或超出 int64 时返回 0 和错误。
func BinDec(str string) (int64, error) {
	return parseBase(str, 2)
}

// BinHex 将二进制字符串转为小写十六进制，数值范围限 int64，结果不带前缀。
func BinHex(str string) (string, error) {
	return convertBase(str, 2, 16)
}

// OctBin 将八进制字符串转为二进制，数值范围限 int64，结果不带进制前缀。
func OctBin(data string) (string, error) {
	return convertBase(data, 8, 2)
}

// OctDec 解析带可选正负号的八进制整数；格式错误或超出 int64 时返回 0 和错误。
func OctDec(str string) (int64, error) {
	return parseBase(str, 8)
}

// OctHex 将八进制字符串转为小写十六进制，数值范围限 int64，结果不带前缀。
func OctHex(data string) (string, error) {
	return convertBase(data, 8, 16)
}

// DecBin 返回无进制前缀的二进制表示，负数保留负号。
func DecBin(number int64) string {
	return strconv.FormatInt(number, 2)
}

// DecOct 返回无进制前缀的八进制表示，负数保留负号。
func DecOct(number int64) string {
	return strconv.FormatInt(number, 8)
}

// DecHex 返回无进制前缀的小写十六进制表示，负数保留负号。
func DecHex(number int64) string {
	return strconv.FormatInt(number, 16)
}

// HexBin 将十六进制字符串转为二进制，数值范围限 int64，结果不带进制前缀。
func HexBin(data string) (string, error) {
	return convertBase(data, 16, 2)
}

// HexOct 将十六进制字符串转为八进制，数值范围限 int64，结果不带进制前缀。
func HexOct(str string) (string, error) {
	return convertBase(str, 16, 8)
}

// HexDec 解析带可选正负号的十六进制整数；不接受 0x 前缀，失败返回 0 和错误。
func HexDec(str string) (int64, error) {
	return parseBase(str, 16)
}

// parseBase 使用显式进制，不自动识别进制前缀，也不接受下划线；所有解析失败均返回 0 和带栈错误。
func parseBase(str string, base int) (int64, error) {
	i, err := strconv.ParseInt(str, base, 64)
	if err != nil {
		return 0, errors.Tag(err)
	}
	return i, nil
}

// convertBase 沿用 parseBase 的 int64 范围；失败返回空串，不输出部分结果。
func convertBase(str string, fromBase, toBase int) (string, error) {
	i, err := parseBase(str, fromBase)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(i, toBase), nil
}
