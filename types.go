package utils

import (
	"context"
	"crypto/cipher"
)

// Signed 接受有符号整数及以这些类型为底层类型的自定义类型。
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Unsigned 接受无符号整数、uintptr 及其自定义类型。
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Integer 合并有符号与无符号整数，不包含浮点数。
type Integer interface {
	Signed | Unsigned
}

// Float 接受 float32、float64 及其自定义类型。
type Float interface {
	~float32 | ~float64
}

// Number 接受整数与浮点数，不包含复数。
type Number interface {
	Integer | Float
}

// Ordered 接受可用 < 比较的数字与字符串；浮点 NaN 不满足全序。
type Ordered interface {
	Number | ~string
}

// Slice 供 sort.Sort 使用，排序会修改原切片；浮点元素不应包含 NaN。
type Slice[T Ordered] []T

// Len 满足 sort.Interface，nil 切片长度为 0。
func (s Slice[T]) Len() int { return len(s) }

// Less 使用类型原生的 < 比较，不单独处理 NaN。
func (s Slice[T]) Less(i, j int) bool { return s[i] < s[j] }

// Swap 原地交换元素，共享底层数组的切片可见该变化。
func (s Slice[T]) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// 密码分组与编码回调。
type (
	// CipherMode 选择分组工作模式，取值见 ECB、CBC、CTR、CFB、OFB。
	CipherMode int8

	// EncodeToString 将字节编码为文本，如 hex.EncodeToString；该步骤本身不执行加密。
	EncodeToString func([]byte) string

	// DecodeString 将文本解码为字节，如 hex.DecodeString；输入格式错误时返回 error。
	DecodeString func(string) ([]byte, error)

	// Pad 接收明文和分组字节数，返回填充后的数据；内置填充函数返回独立副本。
	// 自定义实现是否复用输入，由调用方约定。
	Pad func([]byte, int) []byte

	// Unpad 接收解密结果并去填充，填充不符合所选规则时返回 error。
	// PKCS7Unpad/ZeroUnpad 返回输入的子切片，NoUnpad 返回独立副本。
	Unpad func([]byte) ([]byte, error)

	// CipherBlock 按密钥创建 cipher.Block，如 aes.NewCipher；密钥长度由具体算法校验。
	CipherBlock func([]byte) (cipher.Block, error)
)

// 读取回调使用读取器的临时缓冲；需在返回后保留数据时，由调用方复制。
// 回调返回可被 errors.Is 匹配为 DONE 的错误时正常停止，其他错误向外返回。
type (
	// ReadBlock 接收本次读取的字节数和数据；size 等于 len(block)，数据块边界不代表行边界。
	ReadBlock func(size int, block []byte) error

	// ReadScan 接收从 1 开始的行号和去掉换行符的数据。
	// Scan 调用时 err 固定为 nil，实际读取错误由 Scan 返回；参数保留供现有调用方兼容。
	ReadScan func(num int, line []byte, err error) error

	// ReadLine 接收从 1 开始的物理行号；长行的多次回调复用同一行号，lineDone 标记本行最后一块。
	// 行尾的 LF 或 CRLF 不包含在数据中；末块可以为空，仅用于通知该行结束。
	ReadLine func(num int, line []byte, lineDone bool) error
)

// JSON 编解码回调由 WithJSON 安装，安装后可由多个 goroutine 共享调用。
type (
	// Encode 将值编码为 JSON 字节；默认实现为 json.Marshal。
	Encode func(any) ([]byte, error)

	// Decode 将 JSON 写入调用方传入的目标；默认实现为 json.Unmarshal。
	Decode func([]byte, any) error
)

// LogLevel 使用 slog 的级别数值，自定义日志适配器负责转换为其自身级别。
type LogLevel int

// 日志级别数值越大，严重程度越高。
const (
	// LevelDebug 调试级别 (-4 对应 slog.LevelDebug)
	LevelDebug LogLevel = -4
	// LevelInfo 信息级别 (0 对应 slog.LevelInfo)
	LevelInfo LogLevel = 0
	// LevelWarn 警告级别 (4 对应 slog.LevelWarn)
	LevelWarn LogLevel = 4
	// LevelError 错误级别 (8 对应 slog.LevelError)
	LevelError LogLevel = 8
)

// Logger 由 Configure 安装后共享使用；自定义实现负责内部状态的并发访问。
// 默认实现为 slog，args 按键值对解释；第三方日志库需提供该接口的适配器。
type Logger interface {
	// Debug 调试级别日志
	Debug(msg string, args ...any)
	// Info 信息级别日志
	Info(msg string, args ...any)
	// Warn 警告级别日志
	Warn(msg string, args ...any)
	// Error 错误级别日志
	Error(msg string, args ...any)
	// With 创建携带附加字段的子 Logger，不修改当前 Logger。
	With(args ...any) Logger
	// Enabled 在构造日志内容前判断级别是否启用，ctx 可携带调用方的过滤信息。
	Enabled(ctx context.Context, level LogLevel) bool
}
