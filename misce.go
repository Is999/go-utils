package utils

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/Is999/go-utils/errors"
)

// 重试退避常量用于限制失败后的等待时间。
const (
	// retryBaseDelay 定义第一次失败后的基础退避时间。
	retryBaseDelay = 200 * time.Millisecond
	// retryMaxDelay 定义退避时间上限，避免长时间阻塞调用方。
	retryMaxDelay = 3 * time.Second
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

// Retry 尝试执行 fn，如果 fn 返回错误则按指数退避策略重试。
// maxRetries 表示最大尝试次数，包含首次执行。
// 重试间隔从 100ms~200ms 区间起步，按 2 倍递增，并在每次退避上附加随机抖动，最大不超过 3s。
func Retry(maxRetries uint8, fn func(tries int) error) error {
	if fn == nil {
		return errors.New("无效的执行方法")
	}
	var (
		err   error
		tries = 0
	)
	if maxRetries == 0 {
		maxRetries = 1
	}
	for {
		tries++
		if err = fn(tries); err == nil {
			break
		}

		// 判断退出条件
		if tries >= int(maxRetries) {
			break
		}

		// 延迟重试
		time.Sleep(retryDelay(tries))
	}

	if err != nil {
		// 重试失败，返回错误信息
		return errors.Wrapf(err, "%s 尝试 %d 次后依然失败", GetFunctionName(fn), maxRetries)
	}
	return nil
}

// retryDelay 计算第 attempt 次失败后的重试等待时间。
// 为降低并发失败时的重试雪崩风险，会在指数退避的基础上附加小幅随机抖动。
func retryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return retryBaseDelay / 2
	}

	// 指数退避从 retryBaseDelay 开始，attempt=1 表示第一次失败后的等待。
	delay := retryBaseDelay
	for i := 1; i < attempt && delay < retryMaxDelay; i++ {
		delay *= 2
	}
	if delay > retryMaxDelay {
		delay = retryMaxDelay
	}

	// 使用 equal jitter：保留一半确定性延迟，另一半随机化。
	half := delay / 2
	if half <= 0 {
		return delay
	}
	return half + time.Duration(rand.Int64N(int64(half)))
}
