package utils

import (
	"fmt"
	"strconv"
	"time"
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
	str := fmt.Sprintf("%."+strconv.Itoa(dec)+"f", number)

	// 默认分割(无千分位分割)
	if decPoint == "." && thousandsSep == "" {
		if neg {
			str = "-" + str
		}
		return str
	}

	// 分割整数和小数部分
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

// Retry 尝试执行fn, 如果fn返回错误则进行重试
// 最大重试次数为maxRetries
// 每次重试休眠100毫秒的指数倍，最大休眠1秒
func Retry(maxRetries uint8, fn func(tries int) error) error {
	var (
		err   error
		tries = 0
	)
	for {
		tries++
		if err = fn(tries); err == nil {
			break
		}

		// 判断退出条件
		if tries >= int(maxRetries) {
			break
		}

		// 延迟时间： 204ms 409ms 614ms 819ms 1024ms 1228ms ...
		maxDelay := tries << 11 / 10

		// 延迟重试
		time.Sleep(time.Millisecond * time.Duration(maxDelay))
	}

	if err != nil {
		// 重试失败，返回错误信息
		return fmt.Errorf("method %s failed after %d retries: %w", GetFunctionName(fn), tries, err)
	}
	return nil
}
