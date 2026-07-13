package utils

import (
	crand "crypto/rand"
	"io"
	"math/big"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Is999/go-utils/errors"
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

var (
	// uniqueIDMinBase36ByLen 缓存 base36 指定位数的最小值，数据来源是 36 进制位数边界，用于避免 UniqueID 每次解析字符串。
	uniqueIDMinBase36ByLen [13]int64
	// uniqueIDMaxBase36ByLen 缓存 base36 指定位数的最大值，索引范围 1-12，对应 UniqueID 随机段的最大块长度。
	uniqueIDMaxBase36ByLen [13]int64
)

// init 初始化字符串工具的包级缓存，当前只预计算 UniqueID 随机段使用的 base36 边界。
func init() {
	min, max := int64(1), int64(35)
	for digits := 1; digits <= 12; digits++ {
		uniqueIDMinBase36ByLen[digits] = min
		uniqueIDMaxBase36ByLen[digits] = max
		min *= 36
		max = max*36 + 35
	}
}

// Replacer 是可复用字符串替换器。
//
// 同一批替换规则被反复使用时，调用方可复用已排序和已构建的 strings.Replacer。
// 替换规则来源于 map，构造时会固定排序，避免 map 遍历无序导致重叠替换规则结果不稳定。
type Replacer struct {
	replacer *strings.Replacer // 底层标准库替换器；nil 表示无替换规则，Replace 会原样返回输入。
}

// NewReplacer 根据 map 规则创建可复用字符串替换器。
//
// oldnew 字段含义：
//   - key：需要被替换的原字符串。
//   - value：替换后的目标字符串。
func NewReplacer(oldnew map[string]string) *Replacer {
	pairs := replacePairs(oldnew)
	if len(pairs) == 0 {
		return &Replacer{}
	}
	return &Replacer{replacer: strings.NewReplacer(pairs...)}
}

// Replace 使用已构建规则替换字符串。
//
// 空规则或 nil 接收者会原样返回输入，便于调用方在条件化构造替换器时不额外判空。
func (r *Replacer) Replace(s string) string {
	if r == nil || r.replacer == nil {
		return s
	}
	return r.replacer.Replace(s)
}

// replacePairs 将 map 替换规则转换为 strings.NewReplacer 需要的有序 pairs。
//
// 该函数只在构造替换器时排序一次，避免复用场景每次 Replace 都重复排序。
// 对空 map 直接返回 nil，调用方可跳过底层 Replacer 分配。
func replacePairs(oldnew map[string]string) []string {
	length := len(oldnew)
	if length == 0 {
		return nil
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
	return pairs
}

// Replace 字符串替换
//
//	s 源字符串
//	oldnew 替换规则，map类型，map 的键为要替换的字符串，map 的值为替换后的字符串。
func Replace(s string, oldnew map[string]string) string {
	pairs := replacePairs(oldnew)
	if len(pairs) == 0 {
		return s
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
	if isASCIIString(str) {
		begin, end, ok := substrRange(len(str), start, length)
		if !ok {
			return ""
		}
		return str[begin:end]
	}
	runes := []rune(str)
	begin, end, ok := substrRange(len(runes), start, length)
	if !ok {
		return ""
	}
	return string(runes[begin:end])
}

// substrRange 根据历史 Substr 语义计算截取范围。
//
// length 为 0、空字符串、start 超出末尾时返回 false。
// start 为负数时从尾部倒算，越过头部则降级为 0。
// length 为负数时表示结束字符索引，和旧实现保持 len+length+1 的闭区间语义。
func substrRange(size, start, length int) (int, int, bool) {
	if length == 0 || size == 0 || start > size {
		return 0, 0, false
	}

	// 负数 start 来自历史接口约定，用于从字符串尾部倒数起点。
	if start < 0 {
		start += size
		if start < 0 {
			start = 0
		}
	}

	var end int
	if length < 0 {
		// 结束位置越过字符串头部时直接返回空，避免 size+length+1 在极端负数下溢出。
		if length < -size {
			return 0, 0, false
		}
		end = size + length + 1
	} else {
		// 正数 length 表示截取长度，先比较剩余空间再相加，避免 start+length 在极端值下溢出。
		if length > size-start {
			end = size
		} else {
			end = start + length
		}
	}

	if start >= end {
		return 0, 0, false
	}
	return start, end, true
}

// isASCIIString 判断字符串是否全为 ASCII 字符。
//
// 数据来源为原始字符串字节；只要存在最高位为 1 的字节，就说明需要走 rune 路径以保持 Unicode 语义。
func isASCIIString(str string) bool {
	for i := 0; i < len(str); i++ {
		if str[i]&0x80 != 0 {
			return false
		}
	}
	return true
}

// ReverseString 反转字符串
func ReverseString(str string) string {
	return string(Reverse([]rune(str)))
}

// RandomLetters 随机生成字符串，使用 ALPHA 规则。
// 注意：该函数基于 math/rand，仅适用于测试数据、临时标识等非安全场景。
// 禁止用于 token、验证码、重置链接、签名密钥等安全敏感用途；安全场景请使用 SecureRandomLetters。
//
//	n 生成字符串长度
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func RandomLetters(n int, r ...*rand.Rand) string {
	return RandomString(n, ALPHA, r...)
}

// RandomID 随机生成字符串，使用 ALNUM 规则。
// 为兼容旧行为，首字符固定从 ALPHA 中选择，避免数字开头。
// 注意：该函数基于 math/rand，仅适用于测试数据、临时标识等非安全场景。
// 禁止用于 token、验证码、重置链接、签名密钥等安全敏感用途；安全场景请使用 SecureRandomID。
//
//	n 生成字符串长度
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func RandomID(n int, r ...*rand.Rand) string {
	if n <= 0 {
		return ""
	}
	s := make([]byte, n)
	if len(r) == 0 || r[0] == nil {
		s[0] = ALPHA[rand.Intn(len(ALPHA))]
		for i := 1; i < n; i++ {
			s[i] = ALNUM[rand.Intn(len(ALNUM))]
		}
		return string(s)
	}

	randSourceMu.Lock()
	s[0] = ALPHA[r[0].Intn(len(ALPHA))]
	for i := 1; i < n; i++ {
		s[i] = ALNUM[r[0].Intn(len(ALNUM))]
	}
	randSourceMu.Unlock()
	return string(s)
}

// RandomString 随机生成字符串。
// 注意：该函数基于 math/rand，仅适用于测试数据、临时标识等非安全场景。
// 禁止用于 token、验证码、重置链接、签名密钥等安全敏感用途；安全场景请使用 SecureRandomString。
//
//	n 生成字符串长度
//	alpha 生成随机字符串的种子
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func RandomString(n int, alpha string, r ...*rand.Rand) string {
	if n <= 0 || alpha == "" {
		return ""
	}
	l := len(alpha)
	s := make([]byte, n)
	if len(r) == 0 || r[0] == nil {
		for i := 0; i < n; i++ {
			s[i] = alpha[rand.Intn(l)]
		}
		return string(s)
	}

	randSourceMu.Lock()
	for i := 0; i < n; i++ {
		s[i] = alpha[r[0].Intn(l)]
	}
	randSourceMu.Unlock()
	return string(s)
}

// SecureRandomLetters 使用密码学安全随机源生成字符串，使用 ALPHA 规则。
func SecureRandomLetters(n int) (string, error) {
	return SecureRandomString(n, ALPHA)
}

// SecureRandomID 使用密码学安全随机源生成字符串，使用 ALNUM 规则。
// 为兼容历史命名习惯，首字符固定从 ALPHA 中选择，避免数字开头。
func SecureRandomID(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}
	s := make([]byte, n)
	if err := secureRandBytes(s[:1], ALPHA); err != nil {
		return "", errors.Tag(err)
	}
	if n > 1 {
		if err := secureRandBytes(s[1:], ALNUM); err != nil {
			return "", errors.Tag(err)
		}
	}
	return string(s), nil
}

// SecureRandomString 使用密码学安全随机源按自定义字符集生成字符串。
func SecureRandomString(n int, alpha string) (string, error) {
	if n <= 0 || alpha == "" {
		return "", nil
	}

	s := make([]byte, n)
	if err := secureRandBytes(s, alpha); err != nil {
		return "", errors.Tag(err)
	}
	return string(s), nil
}

// UniqueID 生成一个长度范围 16-32 位的唯一 ID 字符串（可排序字符串）。
// UniqueID 只生成字符串标识，不承诺全局强唯一；强唯一场景建议使用业务唯一键或 UUID/ULID。
// 注意：该函数基于时间戳与 math/rand，仅适用于非安全场景。
// 禁止用于 token、验证码、重置链接等安全敏感用途；安全场景请使用 SecureUniqueID。
//
//	l 生成 UniqueID 长度: 取值范围[16-32], 小于16按16位处理, 大于32按32位处理
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func UniqueID(l uint8, r ...*rand.Rand) string {
	// 16-32 位
	if l > 32 {
		l = 32
	} else if l < 16 {
		l = 16
	}

	// 生成UniqueID前半部分(使用时间戳生成UniqueID前12位字符)
	nano := time.Now().UnixNano()
	ts := strconv.FormatInt(nano, 36) // int64时间戳转36位字符串

	// 使用 strings.Builder 减少多段拼接产生的临时对象。
	var b strings.Builder
	b.Grow(int(l))
	b.WriteString(ts)

	// 生成UniqueID后半部分
	total := int(l) - len(ts) // UniqueID后半部分需生成的字符长度
	for total > 0 {
		num := min(total, 12)
		total -= num

		minInt := uniqueIDMinBase36ByLen[num]
		maxInt := uniqueIDMaxBase36ByLen[num]
		b.WriteString(strconv.FormatInt(Rand(minInt, maxInt, r...), 36))
	}

	return b.String()
}

// SecureUniqueID 使用密码学安全随机源生成长度范围 16-32 位的字符串标识。
// 与 UniqueID 不同，SecureUniqueID 不依赖时间戳，不保证可排序。
func SecureUniqueID(l uint8) (string, error) {
	if l > 32 {
		l = 32
	} else if l < 16 {
		l = 16
	}
	return SecureRandomID(int(l))
}

// RandSource 是兼容旧调用的可复用随机源；并发场景请通过 Rand/Random* 函数使用。
var RandSource = rand.New(rand.NewSource(time.Now().UnixNano()))

// secureRandBytes 使用密码学安全随机源填充目标字节切片。
//
// 常见字符集长度不超过 256 时，批量读取随机字节并做拒绝采样，避免每个字符一次 big.Int 分配。
// 超过 256 的非常规字符集退回到 crand.Int，优先保证分布均匀和行为正确。
func secureRandBytes(dst []byte, alpha string) error {
	if len(dst) == 0 || alpha == "" {
		return nil
	}
	if len(alpha) == 1 {
		for i := range dst {
			dst[i] = alpha[0]
		}
		return nil
	}
	if len(alpha) > 256 {
		return secureRandBytesBigAlpha(dst, alpha)
	}

	alphaLen := len(alpha)
	acceptLimit := 256 - 256%alphaLen
	written := 0
	var randomBuf [256]byte
	for written < len(dst) {
		need := len(dst) - written
		readSize := min(need+need/4+1, len(randomBuf))
		if _, err := io.ReadFull(crand.Reader, randomBuf[:readSize]); err != nil {
			return errors.Tag(err)
		}

		// 拒绝采样丢弃落在不完整区间里的随机字节，避免 byte%len(alpha) 产生取模偏差。
		for i := 0; i < readSize && written < len(dst); i++ {
			randomByte := int(randomBuf[i])
			if randomByte >= acceptLimit {
				continue
			}
			dst[written] = alpha[randomByte%alphaLen]
			written++
		}
	}
	return nil
}

// secureRandBytesBigAlpha 处理超大字符集的降级路径。
//
// 该路径保留每字符一次 crand.Int 的实现，业务意图是支持历史上可能传入的任意长度 alpha，
// 边界条件是 alpha 长度超过单字节拒绝采样能表达的范围。
func secureRandBytesBigAlpha(dst []byte, alpha string) error {
	max := big.NewInt(int64(len(alpha)))
	for i := range dst {
		index, err := crand.Int(crand.Reader, max)
		if err != nil {
			return errors.Tag(err)
		}
		dst[i] = alpha[index.Int64()]
	}
	return nil
}
