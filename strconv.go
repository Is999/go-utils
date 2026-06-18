package utils

import (
	"strconv"

	"github.com/Is999/go-utils/errors"
)

// ToInt string 转 int，失败返回零值。
func ToInt(s string) (i int) {
	i, _ = strconv.Atoi(s)
	return
}

// ToInt64 string 转 int64，失败返回零值。
func ToInt64(s string) (i int64) {
	i, _ = strconv.ParseInt(s, 10, 64)
	return
}

// ToFloat64 string 转 float64，失败返回零值。
func ToFloat64(s string) (i float64) {
	i, _ = strconv.ParseFloat(s, 64)
	return
}

// BinOct 二进制转换为八进制
func BinOct(str string) (string, error) {
	return convertBase(str, 2, 8)
}

// BinDec 二进制转换为十进制
func BinDec(str string) (int64, error) {
	return parseBase(str, 2)
}

// BinHex 二进制转换为十六进制
func BinHex(str string) (string, error) {
	return convertBase(str, 2, 16)
}

// OctBin 八进制转换为二进制
func OctBin(data string) (string, error) {
	return convertBase(data, 8, 2)
}

// OctDec 八进制转换为十进制
func OctDec(str string) (int64, error) {
	return parseBase(str, 8)
}

// OctHex 八进制转换为十六进制
func OctHex(data string) (string, error) {
	return convertBase(data, 8, 16)
}

// DecBin 十进制转换为二进制
func DecBin(number int64) string {
	return strconv.FormatInt(number, 2)
}

// DecOct 十进制转换为八进制
func DecOct(number int64) string {
	return strconv.FormatInt(number, 8)
}

// DecHex 十进制转换为十六进制
func DecHex(number int64) string {
	return strconv.FormatInt(number, 16)
}

// HexBin 十六进制转换为二进制
func HexBin(data string) (string, error) {
	return convertBase(data, 16, 2)
}

// HexOct 十六进制转换为八进制
func HexOct(str string) (string, error) {
	return convertBase(str, 16, 8)
}

// HexDec 十六进制转换为十进制
func HexDec(str string) (int64, error) {
	return parseBase(str, 16)
}

// parseBase 按指定进制解析 int64，并统一包装解析错误。
func parseBase(str string, base int) (int64, error) {
	i, err := strconv.ParseInt(str, base, 64)
	if err != nil {
		return 0, errors.Tag(err)
	}
	return i, nil
}

// convertBase 将字符串从源进制转换为目标进制。
func convertBase(str string, fromBase, toBase int) (string, error) {
	i, err := parseBase(str, fromBase)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(i, toBase), nil
}
