package utils

import (
	"fmt"
	"strconv"
)

// Ternary 类似于三目运算
//
//	expr bool表达式
//	trueVal expr为true时返回值
//	falseVal expr为false时返回值
func Ternary[T any](expr bool, trueVal, falseVal T) T {
	if expr {
		return trueVal
	}
	return falseVal
}

// NumberFormat 以千位分隔符方式格式化一个数字
//
//	number 要格式化的数字
//	decimals 保留几位小数
//	decPoint 小数点[.]
//	thousandsSep 千位分隔符[,]
func NumberFormat(number float64, decimals uint, decPoint, thousandsSep string) string {
	// 负数处理
	neg := false
	if number < 0 {
		number = -number
		neg = true
	}

	// 格式化并保留指定小数位
	dec := int(decimals)
	str := fmt.Sprintf("%."+strconv.Itoa(dec)+"F", number)

	// 默认分割(无千分位分割)
	if decPoint == "." && thousandsSep == "" {
		if neg {
			str = "-" + str
		}
		return str
	}

	prefix, suffix := "", ""
	if dec > 0 {
		l := len(str)
		prefix = str[:l-(dec+1)]
		suffix = str[l-dec:]
	} else {
		prefix = str
	}

	sep := []byte(thousandsSep)
	n, l1, l2 := 0, len(prefix), len(sep)

	// 千分位分割符数量
	c := (l1 - 1) / 3
	tmp := make([]byte, l2*c+l1)
	pos := len(tmp) - 1
	for i := l1 - 1; i >= 0; i, n, pos = i-1, n+1, pos-1 {
		if l2 > 0 && n > 0 && n%3 == 0 {
			for j := range sep {
				tmp[pos] = sep[l2-j-1]
				pos--
			}
		}
		tmp[pos] = prefix[i]
	}

	s := string(tmp)
	if dec > 0 {
		s += decPoint + suffix
	}

	if neg {
		s = "-" + s
	}

	return s
}

// LogArgsFormat 根据参数生成Format
func LogArgsFormat(args []any) string {
	if len(args) == 0 {
		return "\n"
	}

	b := make([]byte, 0, len(args)*3)
	for range args {
		b = append(b, "%v "...)
	}
	b[len(b)-1] = '\n' // Replace the last space with a newline.
	return string(b)
}
