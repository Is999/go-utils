package utils

import (
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 字符集常量用于随机字符串与校验码生成。
const (
	// ALPHA 英文字母：A-Za-z
	ALPHA = `ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz`

	// DIGIT 数字：0-9
	DIGIT = `0123456789`

	// ALNUM 英文字母+数字：A-Za-z0-9
	ALNUM = ALPHA + DIGIT
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

	// map 遍历无序，先排序可保证重叠替换规则结果稳定。
	keys := make([]string, 0, length)
	for old := range oldnew {
		keys = append(keys, old)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, length*2)
	for _, old := range keys {
		pairs = append(pairs, old, oldnew[old])
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

// RandStr 随机生成字符串，使用 ALPHA 规则
//
//	n 生成字符串长度
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func RandStr(n int, r ...*rand.Rand) string {
	return RandStr3(n, ALPHA, r...)
}

// RandStr2 随机生成字符串，使用 ALNUM 规则。
// 为兼容旧行为，首字符固定从 ALPHA 中选择，避免数字开头。
//
//	n 生成字符串长度
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func RandStr2(n int, r ...*rand.Rand) string {
	if n <= 0 {
		return ""
	}
	s := make([]byte, n)
	if len(r) == 0 || r[0] == nil {
		s[0] = ALPHA[rand.IntN(len(ALPHA))]
		for i := 1; i < n; i++ {
			s[i] = ALNUM[rand.IntN(len(ALNUM))]
		}
		return string(s)
	}

	randSourceMu.Lock()
	s[0] = ALPHA[r[0].IntN(len(ALPHA))]
	for i := 1; i < n; i++ {
		s[i] = ALNUM[r[0].IntN(len(ALNUM))]
	}
	randSourceMu.Unlock()
	return string(s)
}

// RandStr3 随机生成字符串
//
//	n 生成字符串长度
//	alpha 生成随机字符串的种子
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func RandStr3(n int, alpha string, r ...*rand.Rand) string {
	if n <= 0 || len(alpha) == 0 {
		return ""
	}
	l := len(alpha)
	s := make([]byte, n)
	if len(r) == 0 || r[0] == nil {
		for i := 0; i < n; i++ {
			s[i] = alpha[rand.IntN(l)]
		}
		return string(s)
	}

	randSourceMu.Lock()
	for i := 0; i < n; i++ {
		s[i] = alpha[r[0].IntN(l)]
	}
	randSourceMu.Unlock()
	return string(s)
}

// UniqID 生成一个长度范围 16-32 位的唯一 ID 字符串（可排序字符串）。
// UniqID 只生成字符串标识，不承诺全局强唯一；强唯一场景建议使用业务唯一键或 UUID/ULID。
//
//	l 生成 UniqID 长度: 取值范围[16-32], 小于16按16位处理, 大于32按32位处理
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func UniqID(l uint8, r ...*rand.Rand) string {
	// 16-32 位
	if l > 32 {
		l = 32
	} else if l < 16 {
		l = 16
	}

	// 生成UniqId前半部分(使用时间戳生成UniqId前12位字符)
	nano := time.Now().UnixNano()
	ts := strconv.FormatInt(nano, 36) // int64时间戳转36位字符串

	// 使用 strings.Builder 减少多段拼接产生的临时对象。
	var b strings.Builder
	b.Grow(int(l))
	b.WriteString(ts)

	// 生成UniqId后半部分
	total := int(l) - len(ts) // UniqId后半部分需生成的字符长度
	// 计算生成次数(int64转换36位字符串 最大值可转换12个长度的'z', total超出12位需多次生成)
	n := total / 12

	if len(r) == 0 {
		r = append(r, RandSource)
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

// UniqId 生成一个长度范围 16-32 位的唯一 ID 字符串。
//
// Deprecated: 请使用 UniqID。
func UniqId(l uint8, r ...*rand.Rand) string {
	return UniqID(l, r...)
}

// RandSource rand
var RandSource = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))
