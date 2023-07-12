package utils

import (
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const (
	// LETTERS 值为：A-Za-z
	LETTERS = `ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz`

	// ALPHANUM 值为：A-Za-z0-9
	ALPHANUM = LETTERS + `0123456789`
)

// Replace 字符串替换
//
//	s 源字符串
//	oldnew 替换规则，map类型， map的键为要替换的字符串，map的值为替换成什么字符串。
func Replace(s string, oldnew map[string]string) string {
	length := len(oldnew)
	if length == 0 {
		return s
	}
	pairs := make([]string, 0, length*2)
	for old, news := range oldnew {
		pairs = append(pairs, old, news)
	}
	return strings.NewReplacer(pairs...).Replace(s)
}

// Substr 字符串截取
//
//	str 被截取的字符串
//	start  截取的起始位置，即截取的第一个字符所在的索引：
//		- start小于0时，start = len(str) + start
//	length  截取的截止位置，即截取的最后一个字符所在的索引：
//		- length大于0时，length表示为截取子字符串的长度，截取的最后一个字符所在的索引值为：start + length
//		- length小于0时，length表示为截取的最后一个字符所在的索引，值为：len(str) + length + 1
//		- 例如：等于-1时，表示截取到最后一个字符；等于-2时，表示截取到倒数第二个字符
func Substr(str string, start, length int) string {
	end := length
	if end == 0 {
		return ""
	}

	runes := []rune(str)
	l := len(runes)
	if l == 0 || l < start {
		return ""
	}

	// 计算start值
	if start < 0 {
		start += l
		if start < 0 {
			start = 0
		}
	}

	// 计算end值
	if end < 0 {
		end += l + 1
		if end <= 0 {
			return ""
		}
	} else {
		end += start
		if l < end {
			end = l
		}
	}

	if start >= end {
		return ""
	}
	return string(runes[start:end])
}

// StrRev 反转字符串
func StrRev(str string) string {
	return string(Reverse([]rune(str)))
}

// RandStr 随机生成字符串，使用LETTERS规则
//
//	n 生成字符串长度
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func RandStr(n int, r ...rand.Source) string {
	if n <= 0 {
		return ""
	}

	if len(r) == 0 {
		r = append(r, GetSource())
		defer sourcePool.Put(r[0])
	}

	s := make([]byte, n)
	for i := 0; i < n; i++ {
		s[i] = LETTERS[int(r[0].Int63())%len(LETTERS)]
	}
	return *(*string)(unsafe.Pointer(&s))
}

// RandStr2 随机生成字符串，使用ALPHANUM规则
//
//	n 生成字符串长度
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func RandStr2(n int, r ...rand.Source) string {
	if n <= 0 {
		return ""
	}
	if len(r) == 0 {
		r = append(r, GetSource())
		defer sourcePool.Put(r[0])
	}
	s := make([]byte, n)
	s[0] = LETTERS[int(r[0].Int63())%len(LETTERS)]
	for i := 1; i < n; i++ {
		s[i] = ALPHANUM[int(r[0].Int63())%len(ALPHANUM)]
	}
	return *(*string)(unsafe.Pointer(&s))
}

// RandStr3 随机生成字符串
//
//	n 生成字符串长度
//	alpha 生成随机字符串的种子
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func RandStr3(n int, alpha string, r ...rand.Source) string {
	if n <= 0 || len(alpha) == 0 {
		return ""
	}
	if len(r) == 0 {
		r = append(r, GetSource())
		defer sourcePool.Put(r[0])
	}
	l := len(alpha)
	s := make([]byte, n)
	for i := 0; i < n; i++ {
		s[i] = alpha[int(r[0].Int63())%l]
	}
	return *(*string)(unsafe.Pointer(&s))
}

// UniqId 生成一个长度范围16-32位的唯一ID字符串(可排序的字符串)，UniqId只生成字符串并不保证唯一性。
// UniqId将int64时间戳转换成36位字符串（长度12位）剩余长度使用rand随机生成int64数字并转换成36位字符串。
//
//	l 生成UniqId长度: 取值范围[16-32], 小于16按16位处理, 大于32按32位处理
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func UniqId(l uint8, r ...rand.Source) string {
	// 16-32 位
	if l > 32 {
		l = 32
	} else if l < 16 {
		l = 16
	}

	// 生成UniqId前半部分(使用时间戳生成UniqId前12位字符)
	nano := time.Now().UnixNano()
	ts := strconv.FormatInt(nano, 36) // int64时间戳转36位字符串

	// UniqId拼接
	var b strings.Builder
	b.Grow(int(l))
	b.WriteString(ts)

	// 生成UniqId后半部分
	total := int(l) - len(ts) // UniqId后半部分需生成的字符长度
	// 计算生成次数(int64转换36位字符串 最大值可转换12个长度的'z', total超出12位需多次生成)
	n := total / 12

	//r := rand.New(rand.NewSource(nano))
	if len(r) == 0 {
		r = append(r, GetSource())
		defer sourcePool.Put(r[0])
	}

	for i := 0; i <= n && total > 0; i++ {
		// 计算随机生成最小值(min)和最大值(max), 并重新计算total值
		num := 12 // 最大随机值长度(int64转换36位字符串 最大值可转换12个长度的'z')
		if total < num {
			num = total
		}
		total -= num

		// 计算最小值, 最大值
		minInt, _ := strconv.ParseInt("1"+strings.Repeat("0", num-1), 36, 64)
		maxInt, _ := strconv.ParseInt(strings.Repeat("z", num), 36, 64)

		// 随机生成 minInt - maxInt 之间的数, 并转换成36位字符串
		rv := strconv.FormatInt(Rand(minInt, maxInt, r...), 36)

		b.WriteString(rv)
	}

	return b.String()
}

// sourcePool 随机种子
var sourcePool = &sync.Pool{New: func() interface{} {
	return rand.NewSource(time.Now().UnixNano())
}}

// GetSource 获取随机种子
func GetSource() rand.Source {
	return sourcePool.Get().(rand.Source)
}
