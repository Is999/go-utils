package utils

import (
	crand "crypto/rand"
	"io"
	"math/big"
	"math/rand/v2"
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
	// uniqIDMinBase36ByLen 缓存 base36 指定位数的最小值，数据来源是 36 进制位数边界，用于避免 UniqID 每次解析字符串。
	uniqIDMinBase36ByLen [13]int64
	// uniqIDMaxBase36ByLen 缓存 base36 指定位数的最大值，索引范围 1-12，对应 UniqID 随机段的最大块长度。
	uniqIDMaxBase36ByLen [13]int64
)

// init 初始化字符串工具的包级缓存，当前只预计算 UniqID 随机段使用的 base36 边界。
func init() {
	initUniqIDBase36Bounds()
}

// Replacer 是可复用字符串替换器。
//
// 业务意图：
//   - 当同一批替换规则被反复使用时，调用方可复用已排序和已构建的 strings.Replacer。
//   - 替换规则来源于 map，构造时会固定排序，避免 map 遍历无序导致重叠替换规则结果不稳定。
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
// 性能保护：
//   - 只在构造替换器时排序一次，避免复用场景每次 Replace 都重复排序。
//   - 对空 map 直接返回 nil，调用方可跳过底层 Replacer 分配。
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
		return substrASCII(str, start, length)
	}
	return substrRunes(str, start, length)
}

// substrRunes 按 Unicode 字符截取非 ASCII 字符串，保持历史对中文等多字节字符按 rune 计数的行为。
func substrRunes(str string, start, length int) string {
	runes := []rune(str)
	begin, end, ok := substrRange(len(runes), start, length)
	if !ok {
		return ""
	}
	return string(runes[begin:end])
}

// substrASCII 按 byte 下标截取纯 ASCII 字符串。
//
// 纯 ASCII 的 byte 数等于字符数，可直接切片避免 []rune 分配，是 Substr 的高频性能保护路径。
func substrASCII(str string, start, length int) string {
	begin, end, ok := substrRange(len(str), start, length)
	if !ok {
		return ""
	}
	return str[begin:end]
}

// substrRange 根据历史 Substr 语义计算截取范围。
//
// 边界说明：
//   - length 为 0、空字符串、start 超出末尾时返回 false。
//   - start 为负数时从尾部倒算，越过头部则降级为 0。
//   - length 为负数时表示结束字符索引，和旧实现保持 len+length+1 的闭区间语义。
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

	end := length
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

// StrRev 反转字符串
func StrRev(str string) string {
	return string(Reverse([]rune(str)))
}

// RandStr 随机生成字符串，使用 ALPHA 规则。
// 注意：该函数基于 math/rand，仅适用于测试数据、临时标识等非安全场景。
// 禁止用于 token、验证码、重置链接、签名密钥等安全敏感用途；安全场景请使用 SecureRandStr。
//
//	n 生成字符串长度
//	r 随机种子 rand.NewSource(time.Now().UnixNano()) : 批量生成时传入r参数可提升生成随机数效率
func RandStr(n int, r ...*rand.Rand) string {
	return RandStr3(n, ALPHA, r...)
}

// RandStr2 随机生成字符串，使用 ALNUM 规则。
// 为兼容旧行为，首字符固定从 ALPHA 中选择，避免数字开头。
// 注意：该函数基于 math/rand，仅适用于测试数据、临时标识等非安全场景。
// 禁止用于 token、验证码、重置链接、签名密钥等安全敏感用途；安全场景请使用 SecureRandStr2。
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

// RandStr3 随机生成字符串。
// 注意：该函数基于 math/rand，仅适用于测试数据、临时标识等非安全场景。
// 禁止用于 token、验证码、重置链接、签名密钥等安全敏感用途；安全场景请使用 SecureRandStr3。
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

// SecureRandStr 使用密码学安全随机源生成字符串，使用 ALPHA 规则。
//
// 参数说明：
//   - n：生成字符串长度
//
// 返回值：随机字符串、错误信息
func SecureRandStr(n int) (string, error) {
	return SecureRandStr3(n, ALPHA)
}

// SecureRandStr2 使用密码学安全随机源生成字符串，使用 ALNUM 规则。
// 为兼容历史命名习惯，首字符固定从 ALPHA 中选择，避免数字开头。
//
// 参数说明：
//   - n：生成字符串长度
//
// 返回值：随机字符串、错误信息
func SecureRandStr2(n int) (string, error) {
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

// SecureRandStr3 使用密码学安全随机源按自定义字符集生成字符串。
//
// 参数说明：
//   - n：生成字符串长度
//   - alpha：候选字符集
//
// 返回值：随机字符串、错误信息
func SecureRandStr3(n int, alpha string) (string, error) {
	return secureRandString(n, alpha)
}

// UniqID 生成一个长度范围 16-32 位的唯一 ID 字符串（可排序字符串）。
// UniqID 只生成字符串标识，不承诺全局强唯一；强唯一场景建议使用业务唯一键或 UUID/ULID。
// 注意：该函数基于时间戳与 math/rand，仅适用于非安全场景。
// 禁止用于 token、验证码、重置链接等安全敏感用途；安全场景请使用 SecureUniqID。
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

		// 复用预计算的 base36 边界，避免在高频 ID 生成路径里重复 strings.Repeat 和 ParseInt。
		minInt := uniqIDMinBase36ByLen[num]
		maxInt := uniqIDMaxBase36ByLen[num]

		// 随机生成 minInt - maxInt 之间的数, 并转换成36位字符串
		rv := strconv.FormatInt(Rand(minInt, maxInt, r...), 36)

		b.WriteString(rv)
	}

	return b.String()
}

// SecureUniqID 使用密码学安全随机源生成长度范围 16-32 位的字符串标识。
// 与 UniqID 不同，SecureUniqID 不依赖时间戳，不保证可排序。
//
// 参数说明：
//   - l：生成长度，取值范围 [16-32]
//
// 返回值：随机字符串、错误信息
func SecureUniqID(l uint8) (string, error) {
	if l > 32 {
		l = 32
	} else if l < 16 {
		l = 16
	}
	return SecureRandStr2(int(l))
}

// UniqId 生成一个长度范围 16-32 位的唯一 ID 字符串。
//
// Deprecated: 请使用 UniqID。
func UniqId(l uint8, r ...*rand.Rand) string {
	return UniqID(l, r...)
}

// RandSource rand
var RandSource = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))

// initUniqIDBase36Bounds 初始化 UniqID 随机段的 base36 数值边界。
//
// 边界说明：
//   - 随机段一次最多生成 12 位 base36 字符，36^12-1 仍在 int64 范围内。
//   - 索引 0 不使用，保留零值，调用方仅访问 1-12。
func initUniqIDBase36Bounds() {
	var min int64 = 1
	var max int64 = 35
	for digits := 1; digits <= 12; digits++ {
		uniqIDMinBase36ByLen[digits] = min
		uniqIDMaxBase36ByLen[digits] = max
		min *= 36
		max = max*36 + 35
	}
}

// secureRandString 使用密码学安全随机源按字符集生成字符串。
func secureRandString(n int, alpha string) (string, error) {
	if n <= 0 || len(alpha) == 0 {
		return "", nil
	}

	s := make([]byte, n)
	if err := secureRandBytes(s, alpha); err != nil {
		return "", errors.Tag(err)
	}
	return string(s), nil
}

// secureRandBytes 使用密码学安全随机源填充目标字节切片。
//
// 参数说明：
//   - dst：待填充的目标切片，调用方负责按目标长度预分配。
//   - alpha：候选字符集，按字节索引以保持历史随机字符串接口的 ASCII 字符集语义。
//
// 性能保护：
//   - 常见字符集长度不超过 256 时，批量读取随机字节并做拒绝采样，避免每个字符一次 big.Int 分配。
//   - 超过 256 的非常规字符集退回到 crand.Int，优先保证分布均匀和行为正确。
func secureRandBytes(dst []byte, alpha string) error {
	if len(dst) == 0 || len(alpha) == 0 {
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
		readSize := need + need/4 + 1
		if readSize > len(randomBuf) {
			readSize = len(randomBuf)
		}
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
