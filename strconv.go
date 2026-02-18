package utils

import (
	"strconv"

	"github.com/Is999/go-utils/errors"
)

// Str2Int string 转 int，失败返回零值。
func Str2Int(s string) (i int) {
	i, _ = strconv.Atoi(s)
	return
}

// Str2Int64 string 转 int64，失败返回零值。
func Str2Int64(s string) (i int64) {
	i, _ = strconv.ParseInt(s, 10, 64)
	return
}

// Str2Float string 转 float64，失败返回零值。
func Str2Float(s string) (i float64) {
	i, _ = strconv.ParseFloat(s, 64)
	return
}

// BinOct 二进制转换为八进制
func BinOct(str string) (string, error) {
	i, err := strconv.ParseInt(str, 2, 0)
	if err != nil {
		return "", errors.Wrap(err)
	}
	return strconv.FormatInt(i, 8), nil
}

// BinDec 二进制转换为十进制
func BinDec(str string) (int64, error) {
	return strconv.ParseInt(str, 2, 0)
}

// BinHex 二进制转换为十六进制
func BinHex(str string) (string, error) {
	i, err := strconv.ParseInt(str, 2, 0)
	if err != nil {
		return "", errors.Wrap(err)
	}
	return strconv.FormatInt(i, 16), nil
}

// OctBin 八进制转换为二进制
func OctBin(data string) (string, error) {
	i, err := strconv.ParseInt(data, 8, 0)
	if err != nil {
		return "", errors.Wrap(err)
	}
	return strconv.FormatInt(i, 2), nil
}

// OctDec 八进制转换为十进制
func OctDec(str string) (int64, error) {
	return strconv.ParseInt(str, 8, 0)
}

// OctHex 八进制转换为十六进制
func OctHex(data string) (string, error) {
	i, err := strconv.ParseInt(data, 8, 0)
	if err != nil {
		return "", errors.Wrap(err)
	}
	return strconv.FormatInt(i, 16), nil
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
	i, err := strconv.ParseInt(data, 16, 0)
	if err != nil {
		return "", errors.Wrap(err)
	}
	return strconv.FormatInt(i, 2), nil
}

// HexOct 十六进制转换为八进制
func HexOct(str string) (string, error) {
	i, err := strconv.ParseInt(str, 16, 0)
	if err != nil {
		return "", errors.Wrap(err)
	}
	return strconv.FormatInt(i, 8), nil
}

// HexDec 十六进制转换为十进制
func HexDec(str string) (int64, error) {
	return strconv.ParseInt(str, 16, 0)
}
