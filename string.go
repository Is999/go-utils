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
	// uniqueIDMinBase36ByLen 保存随机段的 base36 下界，索引 1-12 对应段长。
	uniqueIDMinBase36ByLen [13]int64
	// uniqueIDMaxBase36ByLen 保存随机段的 base36 上界；两张表在初始化后只读。
	uniqueIDMaxBase36ByLen [13]int64
)

// init 预计算每段随机数的范围，避免生成 ID 时重复换算。
func init() {
	min, max := int64(1), int64(35)
	for digits := 1; digits <= 12; digits++ {
		uniqueIDMinBase36ByLen[digits] = min
		uniqueIDMaxBase36ByLen[digits] = max
		min *= 36
		max = max*36 + 35
	}
}

// Replacer 保存构造时的替换规则，可并发调用 Replace；后续修改原 map 不影响它。
type Replacer struct {
	replacer *strings.Replacer // 底层标准库替换器；nil 表示无替换规则，Replace 会原样返回输入。
}

// NewReplacer 将 oldnew 的键替换为对应值；重叠规则按键的字典序确定优先级。
// 空键和非递归替换沿用 strings.NewReplacer 的规则。
func NewReplacer(oldnew map[string]string) *Replacer {
	pairs := replacePairs(oldnew)
	if len(pairs) == 0 {
		return &Replacer{}
	}
	return &Replacer{replacer: strings.NewReplacer(pairs...)}
}

// Replace 使用构造时的规则替换；零值或 nil 接收者原样返回输入。
func (r *Replacer) Replace(s string) string {
	if r == nil || r.replacer == nil {
		return s
	}
	return r.replacer.Replace(s)
}

// replacePairs 按键的字典序固定规则优先级；空 map 返回 nil。
func replacePairs(oldnew map[string]string) []string {
	length := len(oldnew)
	if length == 0 {
		return nil
	}

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

// Replace 按 NewReplacer 的规则做一次替换，空规则原样返回。
// 同一规则反复使用时可保存 NewReplacer 的结果，省去重复排序和构造。
func Replace(s string, oldnew map[string]string) string {
	pairs := replacePairs(oldnew)
	if len(pairs) == 0 {
		return s
	}
	return strings.NewReplacer(pairs...).Replace(s)
}

// Substr 按 rune 索引截取，start 从 0 开始，负值从末尾倒数并在越界时截到开头。
// length 为正时表示数量，为负时指定倒数的结束字符（包含该字符，-1 表示末尾）。
// length 为 0 或范围为空时返回空串；非法 UTF-8 按 rune 转换为替换字符。
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

// substrRange 把 Substr 的闭区间结束约定转换成切片范围，空范围返回 false。
func substrRange(size, start, length int) (int, int, bool) {
	if length == 0 || size == 0 || start > size {
		return 0, 0, false
	}

	if start < 0 {
		start += size
		if start < 0 {
			start = 0
		}
	}

	var end int
	if length < 0 {
		// 负数 length 指定倒数位置，越过字符串开头时返回空。
		if length < -size {
			return 0, 0, false
		}
		end = size + length + 1
	} else {
		// 正数 length 表示截取长度，先比较剩余空间再相加，避免 start+length 溢出。
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

// isASCIIString 用于确认字节索引与 rune 索引一致，包含高位字节时交给 rune 路径。
func isASCIIString(str string) bool {
	for i := 0; i < len(str); i++ {
		if str[i]&0x80 != 0 {
			return false
		}
	}
	return true
}

// ReverseString 按 rune 反转；组合字符可能被拆开，非法 UTF-8 转为替换字符。
func ReverseString(str string) string {
	return string(Reverse([]rune(str)))
}

// RandomLetters 生成 n 个 ASCII 字母；长度和随机源规则同 RandomString。
func RandomLetters(n int, r ...*rand.Rand) string {
	return RandomString(n, ALPHA, r...)
}

// RandomID 生成 n 个 ASCII 字母数字，首位固定为字母。
// n <= 0 返回空串；非密码学随机源和共享规则同 RandomString。
func RandomID(n int, r ...*rand.Rand) string {
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

// RandomString 从 alpha 的字节中采样，生成 n 字节；n <= 0 或 alpha 为空时返回空串。
// 只使用 r 第一项，省略或 nil 时用全局源；本包对自定义源的访问串行化，外部直接访问需自行同步。
// 使用 math/rand，需密码学随机性时使用 SecureRandomString。
func RandomString(n int, alpha string, r ...*rand.Rand) string {
	if n <= 0 || alpha == "" {
		return ""
	}
	l := len(alpha)
	s := make([]byte, n)
	if len(r) == 0 || r[0] == nil {
		for i := range n {
			s[i] = alpha[rand.IntN(l)]
		}
		return string(s)
	}

	randSourceMu.Lock()
	for i := range n {
		s[i] = alpha[r[0].IntN(l)]
	}
	randSourceMu.Unlock()
	return string(s)
}

// SecureRandomLetters 使用 crypto/rand 生成 n 个 ASCII 字母，n <= 0 返回空串。
func SecureRandomLetters(n int) (string, error) {
	return SecureRandomString(n, ALPHA)
}

// SecureRandomID 使用 crypto/rand 生成 n 个 ASCII 字母数字，首位固定为字母。
// n <= 0 返回空串和 nil 错误。
func SecureRandomID(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}
	s := make([]byte, n)
	if err := secureRandBytes(s[:1], ALPHA); err != nil {
		return "", err
	}
	if err := secureRandBytes(s[1:], ALNUM); err != nil {
		return "", err
	}
	return string(s), nil
}

// SecureRandomString 使用 crypto/rand 从 alpha 的字节中等概率采样，生成 n 字节。
// n <= 0 或 alpha 为空时返回空串和 nil 错误；重复字节会增加对应字符的采样权重。
func SecureRandomString(n int, alpha string) (string, error) {
	if n <= 0 || alpha == "" {
		return "", nil
	}

	s := make([]byte, n)
	if err := secureRandBytes(s, alpha); err != nil {
		return "", err
	}
	return string(s), nil
}

// UniqueID 以 base36 纳秒时间戳和 math/rand 随机段生成标识，长度限制在 16-32 字节。
// 不保证全局唯一或时钟回拨时的排序；随机源参数规则同 Rand，密码学随机标识使用 SecureUniqueID。
func UniqueID(l uint8, r ...*rand.Rand) string {
	if l > 32 {
		l = 32
	} else if l < 16 {
		l = 16
	}

	// 时间戳与随机段共用最大长度缓冲，避免中间字符串。
	var buf [32]byte
	id := strconv.AppendInt(buf[:0], time.Now().UnixNano(), 36)

	total := int(l) - len(id)
	for total > 0 {
		// 每段最多 12 位，避免 base36 上界超出 int64。
		num := min(total, 12)
		total -= num

		// 下界首位非零，防止格式化后缩短目标长度。
		minInt := uniqueIDMinBase36ByLen[num]
		maxInt := uniqueIDMaxBase36ByLen[num]
		id = strconv.AppendInt(id, Rand(minInt, maxInt, r...), 36)
	}

	return string(id)
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

// RandSource 是可复用随机源；经 Rand/Random* 传入时由本包加锁，直接使用时需自行同步。
var RandSource = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))

// secureRandBytes 按 alpha 的字节位置均匀采样。
func secureRandBytes(dst []byte, alpha string) error {
	// 空目标不消耗随机源，供单字符 ID 的空尾部复用。
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
			// 在随机源失败处附栈，上层直接传回同一错误。
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

// secureRandBytesBigAlpha 用于长度超过 256 的 alpha，单个随机字节无法覆盖其全部索引。
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
