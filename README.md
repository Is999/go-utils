# go-utils

#### 介绍

golang 帮助函数

#### 安装教程

```sh
go get github.com/Is999/go-utils
```

模块路径以 `go.mod` 中的 `github.com/Is999/go-utils` 为准。

### 使用说明

1. 版本要求 Go 1.26。

------

### 模块概览

本库提供基础工具方法。调用方负责业务输入校验、资源授权和可变数据的并发访问；各方法的现有行为与限制见下表及对应注释。

| 模块                    | 行为与使用边界                                                                                                                    |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------|
| AES / DES             | AES 推荐 `GCM`；`CBC` 必须显式 `utils.WithIV(...)` 或 `utils.WithRandIV(true)`；`CTR/CFB/OFB`、`ECB`、默认 key 派生 IV 默认禁用。DES/3DES 仅保留历史兼容，不建议新系统使用。 |
| RSA                   | 生成和导入密钥均要求至少 2048 位；推荐 `EncryptOAEP/DecryptOAEP` 与 `SignPSS/VerifyPSS`；`PKCS#1 v1.5` 仅用于旧协议兼容；签名拒绝 MD5/SHA1。                |
| PKCS7 / Zero          | `PKCS7Unpad` 严格校验填充字节；`ZeroPad` 仅适合能接受尾部 0 歧义的旧协议。                                                                  |
| Pool / Once           | `Pool[T]` 基于 `sync.Pool`，支持归还前 reset；`Once` 并发安全，失败按有限指数退避重试，结果会缓存，需重新执行时调用 `Reset`。                                        |
| Retry                 | `Retry` 使用 capped exponential backoff + jitter：第一次失败后约 100ms~200ms，最多 3s；`maxRetries=0` 时只执行 1 次。                           |
| IP / Logger           | `ClientIP` 仅在远端为可信代理时读取转发头；生产建议使用 `ClientIPWithTrustedProxies`。默认 logger 基于 `log/slog` 并发安全，自定义 logger 也应保证并发安全。            |
| slices / string / map | `Unique/Diff/Intersect` 返回独立结果切片；传入的 slice/map 若被外部并发修改，调用方需自行加锁。重复字符串替换可复用 `NewReplacer`。              |
| tar / zip             | 解压防 Zip Slip/Tar Slip，拒绝绝对路径、目录穿越、空字符、符号链接和特殊文件；限制条目数、单文件大小和总展开大小。                                                          |
| errors                | 支持栈追踪、错误码、上下文键值、`errors.Join`、文本/JSON/slog 输出；全局配置使用原子变量并发安全。                                                               |
| curl                  | 继承标准库默认 Transport 连接池/TLS 行为；Header 按请求克隆；可回放 body 才会重试；默认日志 body 为预览上限并读取后恢复；支持 `Clone/NewRequest` 隔离请求配置，共享已初始化的 Transport。         |
| response              | 最终状态码只提交一次；支持 1xx 临时响应；Content-Type 自动规范化；下载文件名清洗 CR/LF 和路径；文件输出拒绝目录。                                                                      |

### 全局配置 Configure

`Configure` 是全局配置入口，只需在程序入口处（如 `main` 函数）调用一次。支持自定义 JSON 编解码器和自定义 Logger。

```go
import (
	"encoding/json"

	utils "github.com/Is999/go-utils"
)

func main() {
	utils.Configure(
		utils.WithJSON(json.Marshal, json.Unmarshal), // 可选：自定义 JSON 编解码器
	)
}
```

#### JSON 编解码配置

通过 `WithJSON` 设置自定义的 JSON
编解码方法（如 [sonic](https://github.com/bytedance/sonic)、[go-json](https://github.com/goccy/go-json)
等高性能库），若未设置则默认使用标准库 `encoding/json`。

```go
import (
	utils "github.com/Is999/go-utils"
	"github.com/bytedance/sonic"
)

func main() {
	utils.Configure(utils.WithJSON(sonic.Marshal, sonic.Unmarshal))
}
```

设置后，使用 `utils.Marshal()` 和 `utils.Unmarshal()` 即会调用自定义的编解码器。

#### Logger 配置

通过 `WithLogger` 设置自定义 Logger，若未设置则默认使用标准库 `log/slog`。

默认适配器跟随 `slog.Default()`，启用 `HandlerOptions.AddSource` 时记录业务调用位置；`With` 创建的子 Logger 保留创建时的底层实例和附加字段。

使用标准库 handler 的完整可运行示例见 [ExampleLogger](example_logger_adapter_test.go)。下面保留 Zap 和 Logrus 的适配写法，使用这些示例时由应用引入相应依赖。

Logger 接口定义如下，实现 6 个方法即可集成第三方日志库（如 zap、logrus 等）：

```go
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
	Enabled(ctx context.Context, level LogLevel) bool
}
```

日志级别定义（与 `slog.Level` 值一致）：

```go
const (
	LevelDebug LogLevel = -4
	LevelInfo  LogLevel = 0
	LevelWarn  LogLevel = 4
	LevelError LogLevel = 8
)
```

##### 集成 Zap 示例

```go
import (
	"context"
	"fmt"
	"os"

	utils "github.com/Is999/go-utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLoggerAdapter struct {
	logger *zap.SugaredLogger
}

func (z *zapLoggerAdapter) Debug(msg string, args ...any) { z.logger.Debugw(msg, args...) }
func (z *zapLoggerAdapter) Info(msg string, args ...any)  { z.logger.Infow(msg, args...) }
func (z *zapLoggerAdapter) Warn(msg string, args ...any)  { z.logger.Warnw(msg, args...) }
func (z *zapLoggerAdapter) Error(msg string, args ...any) { z.logger.Errorw(msg, args...) }

func (z *zapLoggerAdapter) With(args ...any) utils.Logger {
	return &zapLoggerAdapter{logger: z.logger.With(args...)}
}

func (z *zapLoggerAdapter) Enabled(ctx context.Context, level utils.LogLevel) bool {
	var zapLevel zapcore.Level
	switch level {
	case utils.LevelDebug:
		zapLevel = zapcore.DebugLevel
	case utils.LevelInfo:
		zapLevel = zapcore.InfoLevel
	case utils.LevelWarn:
		zapLevel = zapcore.WarnLevel
	case utils.LevelError:
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	return z.logger.Desugar().Core().Enabled(zapLevel)
}

func main() {
	zapLogger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer func() {
		if err := zapLogger.Sync(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	utils.Configure(utils.WithLogger(&zapLoggerAdapter{logger: zapLogger.Sugar()}))
}
```

##### 集成 Logrus 示例

```go
import (
	"context"

	utils "github.com/Is999/go-utils"
	"github.com/sirupsen/logrus"
)

type logrusLoggerAdapter struct {
	logger *logrus.Logger
	fields logrus.Fields
}

func (l *logrusLoggerAdapter) entry() *logrus.Entry {
	if len(l.fields) > 0 {
		return l.logger.WithFields(l.fields)
	}
	return logrus.NewEntry(l.logger)
}

func (l *logrusLoggerAdapter) Debug(msg string, args ...any) {
	l.entry().WithFields(argsToFields(args)).Debug(msg)
}
func (l *logrusLoggerAdapter) Info(msg string, args ...any) {
	l.entry().WithFields(argsToFields(args)).Info(msg)
}
func (l *logrusLoggerAdapter) Warn(msg string, args ...any) {
	l.entry().WithFields(argsToFields(args)).Warn(msg)
}
func (l *logrusLoggerAdapter) Error(msg string, args ...any) {
	l.entry().WithFields(argsToFields(args)).Error(msg)
}

func (l *logrusLoggerAdapter) With(args ...any) utils.Logger {
	newFields := argsToFields(args)
	merged := make(logrus.Fields, len(l.fields)+len(newFields))
	for k, v := range l.fields {
		merged[k] = v
	}
	for k, v := range newFields {
		merged[k] = v
	}
	return &logrusLoggerAdapter{logger: l.logger, fields: merged}
}

func (l *logrusLoggerAdapter) Enabled(_ context.Context, level utils.LogLevel) bool {
	var logrusLevel logrus.Level
	switch level {
	case utils.LevelDebug:
		logrusLevel = logrus.DebugLevel
	case utils.LevelInfo:
		logrusLevel = logrus.InfoLevel
	case utils.LevelWarn:
		logrusLevel = logrus.WarnLevel
	case utils.LevelError:
		logrusLevel = logrus.ErrorLevel
	default:
		logrusLevel = logrus.InfoLevel
	}
	return l.logger.IsLevelEnabled(logrusLevel)
}

func argsToFields(args []any) logrus.Fields {
	fields := make(logrus.Fields)
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			fields[key] = args[i+1]
		}
	}
	return fields
}

func main() {
	logrusLogger := logrus.New()
	utils.Configure(utils.WithLogger(&logrusLoggerAdapter{logger: logrusLogger}))
}
```

##### Error 追踪与第三方日志库配合

`errors` 包提供了多种方式获取错误追踪信息，兼容不同日志库：

- 首次处理外部错误时使用 `Wrap(err, "操作失败")` 采集调用栈；已有本包栈时只追加消息。
- `Wrap(err)` 不追加消息；已有栈时返回原对象。`Wrap(err, "")` 会创建包装节点，两者的 `Error()` 文本相同。
- 已确定底层采集过栈的高频传播路径，可用 `WithMessage` 追加消息，省去错误链检查。
- `SetTraceEnabled(false)` 只关闭栈的输出，不关闭采集。`Is`、`As` 和 `Unwrap` 保留原始错误链。
- JSON 与 slog 追踪在 1024 层处停止展开，并保留剩余错误的消息；这不限制原始错误链。

以下四种追踪输出按需选择一种；实际代码只保留所选写法，避免同一错误被重复记录。

```go
import (
	"fmt"
	"log/slog"

	utils "github.com/Is999/go-utils"
	"github.com/Is999/go-utils/errors"
)

func reportError(originalErr error, logger utils.Logger) {
	if originalErr == nil {
		return
	}
	err := errors.Wrap(originalErr, "操作失败")

	// 方式1: 使用 slog（默认日志库），Trace 返回 slog.LogValuer 接口
	slog.Error(err.Error(), "trace", errors.Trace(err))

	// 方式2: 使用第三方日志库（zap/logrus 等），TraceString 返回简洁字符串
	logger.Error("操作失败", "error", err.Error(), "trace", errors.TraceString(err))

	// 方式3: 获取完整 JSON 格式追踪（含嵌套 wrap 信息）
	logger.Error("操作失败", "error", err.Error(), "trace", errors.TraceJSON(err))

	// 方式4: 使用 fmt 格式化
	fmt.Printf("%+v\n", err) // 等同于 TraceJSON
	fmt.Printf("%#v\n", err) // 等同于 TraceJSON
}
```

------

# Go常用标准库方法及utils包帮助函数
开发中使用频率较高的Go标准库中的方法及utils包中的帮助方法。utils包中的方法都可以在单元测试中找到使用方法示例。版本要求 >=
1.26版本。

> 使用前请核对上方模块概览和各方法注释中的边界；许可与责任条款见 [LICENSE](LICENSE)。

下文代码块包含 API 签名和用法片段，省略重复的 import；`utils` 指向本模块。示例中的数据、密钥、路径和业务对象由调用方提供。

## 1. 字符串

**strings** 面向字符串，**bytes** 面向字节切片；许多函数名称和语义相近，具体签名以各包 API 为准。

------

### 1.1 截取字符串

字符串切片下标按字节计算；需要按 Unicode 码点截取时使用 `[]rune` 或 `utils.Substr`。组合字符可能包含多个码点。

------

#### type	string

```go
str[start:end]
```

| 参数       | 描述                                                   |
|----------|------------------------------------------------------|
| *str*    | 原字符串。                                                |
| *start*  | 起始字节索引，包含此位置；省略时为 0。 |
| *end*    | 结束字节索引，不包含此位置；省略时为 len(str)。        |

------

#### func [utils.Substr](string.go)

```go
func Substr(str string, start, length int) string
```

| 参数 | 描述 |
| --- | --- |
| *str* | 原字符串，位置和数量均按 rune（Unicode 码点）计算。 |
| *start* | 起始位置；负数从末尾倒数，-1 表示最后一个 rune。 |
| *length* | 正数为截取数量；负数指定包含在结果中的倒数结束字符，-1 截到末尾、-2 截到倒数第二个；0 返回空字符串。 |

备注：负数 start 从末尾换算后仍越过开头时裁剪到开头，正数 length 超过剩余长度时截到末尾；起始位置到达末尾或截取范围为空时返回空字符串。

------

### 1.2 拼接字符串

> **推荐**：strings.Builder+预设大小的方式拼接字符串。


------

#### struct	strings.Builder

```go
var b strings.Builder
b.Grow(length)    // 预设大小
b.WriteString(s1) // 写入字符串s1
b.WriteString(s2) // 写入字符串s2
s3 := b.String()  // 拼接后的字符串
```

------

#### func	fmt.Sprintf

```go
var str = fmt.Sprintf("%s%d%s", s1, i, s2)
```

------

#### func	strings.Join

```go
var str []string = []string{s1, s2}
s := strings.Join(str, "")
```

------

#### struct	bytes.Buffer

```go
var bt bytes.Buffer
bt.WriteString(s1)
bt.WriteString(s2)
s3 := bt.String()
```

------

### 1.3 获取字符串长度

`len(str)` 返回字节数；`utf8.RuneCount` 和 `utf8.RuneCountInString` 返回 Unicode 码点数，组合字符不一定只占一个码点。

------

#### func	len

```go
len(str)
```

------

#### func	utf8.RuneCount

```go
func RuneCount(p []byte) int
```

------

#### func	utf8.RuneCountInString

```go
func RuneCountInString(s string) (n int)
```

------

#### func	bytes.Count

```go
func Count(s, sep []byte) int
```

| 参数    | 描述           |
|-------|--------------|
| *str* | 要获取长度的字符串。   |
| *sep* | 分隔符，一般传 nil。 |

备注：利用 sep 长度等于0的逻辑获取字符串长度；sep长度等于0内部调用 utf8.RuneCount(s) + 1，所以返回的**长度需要减 1**。

```go
length := bytes.Count([]byte(str), nil) - 1
```

------

#### func	strings.Count

```go
func Count(s, substr string) int
```

| 参数       | 描述         |
|----------|------------|
| *str*    | 要获取长度的字符串。 |
| *substr* | 子串，传入空即可。  |

备注：利用 substr 长度等于0的逻辑获取字符串长度；substr长度等于0内部调用 utf8.RuneCountInString(s) + 1，所以返回的**长度需要减
1**。

```go
length := strings.Count(str, "") - 1
```

------

### 1.4 分割字符串

分割字符串我们可以分为几种情况，分别为：按空格分割、按子字符串分割和按字符分割。

------

#### func	strings.Fields

```go
func Fields(s string) []string
```

备注：按一个或多个连续 Unicode 空白字符分割，并忽略首尾空白。

------

#### func	strings.FieldsFunc

```go
func FieldsFunc(s string, f func(rune) bool) []string
```

备注：将 `f(r) == true` 的连续 Unicode 字符视为分隔符，忽略空段。

------

#### func	strings.Split

```go
func Split(s, sep string) []string
```

| 参数    | 描述       |
|-------|----------|
| *s*   | 要分割的字符串。 |
| *sep* | 字符串的分割符。 |

备注：按**子字符串分割**字符串

```go
// 按单个空格分割；连续空格会产生空元素
strings.Split(s, " ")
// 按xxx字符串分割
strings.Split(s, "xxx")
```

------

### 1.5 统计子串在字符串中出现的次数

------

#### func	strings.Count

```go
func Count(s, substr string) int
```

| 参数       | 描述       |
|----------|----------|
| *s*      | 原字符串。    |
| *substr* | 要检索的字符串。 |

------

### 1.6 查找子串在字符串中出现位置

以下 Index 系列函数返回字节索引；未找到时返回 -1。IndexAny/LastIndexAny 在给定字符集合中匹配任一 rune。

------

#### func	strings.Index

```go
func Index(s, substr string) int
```

| 参数       | 描述       |
|----------|----------|
| *s*      | 原字符串。    |
| *substr* | 要检索的字符串。 |

备注：返回的是**字符串第一次出现的位置**。查找不存在的字符串返回 **-1**。

------

#### func	strings.LastIndex

```go
func LastIndex(s, substr string) int
```

| 参数       | 描述       |
|----------|----------|
| *s*      | 原字符串。    |
| *substr* | 要检索的字符串。 |

备注：返回的是**字符串最后一次出现的位置**。查找不存在的字符串返回 **-1**。

------

#### func	strings.IndexAny

```go
func IndexAny(s, chars string) int
```

| 参数      | 描述        |
|---------|-----------|
| *s*     | 原字符串。     |
| *chars* | 要检索的字符序列。 |

备注：返回 chars 中任一字符首次出现的字节索引；未找到返回 -1。

------

#### func	strings.LastIndexAny

```go
func LastIndexAny(s, chars string) int
```

| 参数      | 描述        |
|---------|-----------|
| *s*     | 原字符串。     |
| *chars* | 要检索的字符序列。 |

备注：返回 chars 中任一字符最后出现的字节索引；未找到返回 -1。

------

#### func	strings.IndexByte

```go
func IndexByte(s string, c byte) int
```

| 参数  | 描述        |
|-----|-----------|
| *s* | 原字符串。     |
| *c* | 表示要检索的字符。 |

备注：返回**第一次出现字符的索引**；反之，则返回 **-1**。

------

#### func	strings.LastIndexByte

```go
func LastIndexByte(s string, c byte) int
```

| 参数  | 描述        |
|-----|-----------|
| *s* | 原字符串。     |
| *c* | 表示要检索的字符。 |

备注：返回**最后一次出现字符的索引**；反之，则返回 **-1**。

------

#### func	strings.IndexRune

```go
func IndexRune(s string, r rune) int
```

| 参数  | 描述        |
|-----|-----------|
| *s* | 原字符串。     |
| *r* | 表示要检索的字符。 |

备注：返回**第一次出现字符的索引**；反之，则返回 **-1**。

------

#### 查找 rune 最后出现的位置

```go
index := strings.LastIndexFunc(s, func(r rune) bool { return r == '中' })
```

备注：使用 LastIndexFunc 匹配指定 rune，返回字节索引；未找到返回 -1。

------

#### func	strings.IndexFunc

```go
func IndexFunc(s string, f func(rune) bool) int
```

| 参数  | 描述               |
|-----|------------------|
| *s* | 原字符串。            |
| *f* | 表示要检索的字符的条件判断函数。 |

备注：返回**第一次出现字符的索引**；反之，则返回 **-1**。

------

#### func	strings.LastIndexFunc

```go
func LastIndexFunc(s string, f func(rune) bool) int
```

| 参数  | 描述               |
|-----|------------------|
| *s* | 原字符串。            |
| *f* | 表示要检索的字符的条件判断函数。 |

备注：返回**最后一次出现字符的索引**；反之，则返回 **-1**。

------

### 1.7 判断字符串是否包含子串

------

#### func	strings.HasPrefix

```go
func HasPrefix(s, prefix string) bool
```

| 参数       | 描述      |
|----------|---------|
| *s*      | 原字符串。   |
| *prefix* | 要检索的子串。 |

备注：检索**字符串s**是否以指定**字符串prefix开头**，如果是返回 **True**；反之返回 **False**。

------

#### func	strings.HasSuffix

```go
func HasSuffix(s, suffix string) bool
```

| 参数       | 描述      |
|----------|---------|
| *s*      | 原字符串。   |
| *suffix* | 要检索的子串。 |

备注：检索**字符串s**是否以指定**字符串suffix结尾**，如果是返回 **True**；反之返回 **False**。

------

#### func	strings.Contains

```go
func Contains(s, substr string) bool
```

| 参数       | 描述      |
|----------|---------|
| *s*      | 原字符串。   |
| *substr* | 要检索的子串。 |

备注：检索**字符串s**是否包含**字符串substr**。等价于 **strings.Index(s, substr) >= 0** 。

------

#### func	strings.ContainsRune

```go
func ContainsRune(s string, r rune) bool
```

| 参数  | 描述        |
|-----|-----------|
| *s* | 原字符串。     |
| *r* | 表示要检索的字符。 |

备注：检索**字符串s**是否包含**字符r**。等价于 **strings.IndexRune(s, r) >= 0** 。

------

#### func	strings.ContainsAny

```go
func ContainsAny(s, chars string) bool
```

| 参数      | 描述         |
|---------|------------|
| *s*     | 原字符串。      |
| *chars* | 表示要检索的字符串。 |

备注：判断 s 是否包含 chars 中任一 Unicode 字符，不要求包含完整的 chars 字符串。

------

#### func	strings.Cut

```go
func Cut(s, sep string) (before, after string, found bool)
```

| 参数    | 描述      |
|-------|---------|
| *s*   | 原字符串。   |
| *sep* | 查找的字符串。 |

备注：在**字符串s**中找到**字符串sep**, 并返回**字符串sep前后部分**以及**是否找到字符串sep**。

------

### 1.8 字符串大小写转换

------

#### func	strings.ToTitle

```go
func ToTitle(s string) string
```

备注：将所有 Unicode 字母映射为 title case；不是只转换字符串或单词的首字母。

------

#### func	strings.ToLower

```go
func ToLower(s string) string
```

备注：将**字符串转成小写**。

------

#### func	strings.ToUpper

```go
func ToUpper(s string) string
```

备注：将**字符串转成大写**。

------

### 1.9 去除字符串指定字符

------

#### func	strings.TrimSpace

```go
func TrimSpace(s string) string
```

备注：去除字符串两端的 Unicode 空白字符。

------

#### func	strings.Trim

```go
func Trim(s string, cutset string) string
```

| 参数       | 描述        |
|----------|-----------|
| *s*      | 原字符串。     |
| *cutset* | 需要去除的 Unicode 字符集合。 |

备注：从字符串左右两边连续移除属于 cutset 字符集合的字符，不把 cutset 作为完整子串匹配。

------

#### func	strings.TrimLeft

```go
func TrimLeft(s, cutset string) string
```

| 参数       | 描述        |
|----------|-----------|
| *s*      | 原字符串。     |
| *cutset* | 需要去除的 Unicode 字符集合。 |

备注：从字符串左边连续移除属于 cutset 字符集合的字符，不把 cutset 作为完整子串匹配。

------

#### func	strings.TrimRight

```go
func TrimRight(s, cutset string) string
```

| 参数       | 描述        |
|----------|-----------|
| *s*      | 原字符串。     |
| *cutset* | 需要去除的 Unicode 字符集合。 |

备注：从字符串右边连续移除属于 cutset 字符集合的字符，不把 cutset 作为完整子串匹配。

------

#### func	strings.TrimPrefix

```go
func TrimPrefix(s, prefix string) string
```

| 参数       | 描述          |
|----------|-------------|
| *s*      | 原字符串。       |
| *prefix* | 需要去除的前缀字符串。 |

备注：**去除字符串的前缀prefix**。

------

#### func	strings.TrimSuffix

```go
func TrimSuffix(s, suffix string) string
```

| 参数       | 描述          |
|----------|-------------|
| *s*      | 原字符串。       |
| *suffix* | 需要去除的后缀字符串。 |

备注：**去除字符串的后缀suffix**。

------

#### func	strings.TrimFunc

```go
func TrimFunc(s string, f func(rune) bool) string
```

| 参数  | 描述             |
|-----|----------------|
| *s* | 原字符串。          |
| *f* | 需要去除的字符串的规则函数。 |

备注：将**字符串左右两边符合函数 f 规则字符串去除**。函数 f，接受一个 **rune** 类型的参数，返回一个 **bool** 类型的变量，如果函数
f 返回 **true**，那说明符合规则，**字符将被移除**。

------

#### func	strings.TrimLeftFunc

```go
func TrimLeftFunc(s string, f func(rune) bool) string
```

| 参数  | 描述             |
|-----|----------------|
| *s* | 原字符串。          |
| *f* | 需要去除的字符串的规则函数。 |

备注：将**字符串左边符合函数 f 规则字符串去除**。函数 f，接受一个 **rune** 类型的参数，返回一个 **bool** 类型的变量，如果函数
f 返回 **true**，那说明符合规则，**字符将被移除**。

------

#### func	strings.TrimRightFunc

```go
func TrimRightFunc(s string, f func(rune) bool) string
```

| 参数  | 描述             |
|-----|----------------|
| *s* | 原字符串。          |
| *f* | 需要去除的字符串的规则函数。 |

备注：将**字符串右边符合函数 f 规则字符串去除**。函数 f，接受一个 **rune** 类型的参数，返回一个 **bool** 类型的变量，如果函数
f 返回 **true**，那说明符合规则，**字符将被移除**。

------

### 1.10 字符串遍历处理

------

#### func	strings.Map

```go
func Map(mapping func(rune) rune, s string) string
```

| 参数        | 描述               |
|-----------|------------------|
| *mapping* | 对字符串中每一个字符的处理函数。 |
| *s*       | 原字符串。            |

备注：对字符串 s 中的每一个字符都做 mapping 处理。

------

### 1.11 字符串比较

#### func	strings.Compare

```go
func Compare(a, b string) int
```

备注：比较字符串 a 和字符串 b 是否相等，如果 a > b，返回一个大于 0 的数，如果 a == b，返回 0，否则，返回负数。

------

#### func	strings.EqualFold

```go
func EqualFold(s, t string) bool
```

备注：比较字符串 s 和字符串 t 是否相等，如果相等，返回 true，否则，返回 false。该函数比较字符串大小是忽略大小写的。

------

### 1.12 字符串重复指定次数

------

#### func	strings.Repeat

```go
func Repeat(s string, count int) string
```

| 参数      | 描述      |
|---------|---------|
| *s*     | 原字符串。   |
| *count* | 要重复的次数。 |

备注：将字符串s重复count次。

------

### 1.13 字符串替换

------

#### func	strings.Replace

```go
func Replace(s, old, new string, n int) string
```

| 参数    | 描述                                      |
|-------|-----------------------------------------|
| *s*   | 原字符串。                                   |
| *old* | 要替换的字符串。                                |
| *new* | 替换成什么字符串。                               |
| *n*   | 要替换的次数，-1，那么就会将字符串 s 中的所有的 old 替换成 new。 |

备注：将字符串 s 中的 old 字符串替换成 new 字符串，替换 n 次，返回替换后的字符串。如果 n 是 -1，那么就会将字符串 s 中的所有的
old 替换成 new。

------

#### func	strings.ReplaceAll

```go
func ReplaceAll(s, old, new string) string
```

| 参数    | 描述        |
|-------|-----------|
| *s*   | 原字符串。     |
| *old* | 要替换的字符串。  |
| *new* | 替换成什么字符串。 |

备注：将字符串 s 中的 old 字符串全部替换成 new 字符串，返回替换后的字符串。

------

#### struct	strings.Replacer

```go
strings.NewReplacer(oldnew...).Replace(s)
```

备注：字符串替换

------

#### func [utils.Replace](string.go)

```go
func Replace(s string, oldnew map[string]string) string
```

| 参数       | 描述                                        |
|----------|-------------------------------------------|
| *s*      | 原字符串。                                     |
| *oldnew* | 替换规则，map类型， map的键为要替换的字符串，map的值为替换成什么字符串。 |

备注：内部会先按 key 排序再调用 strings.NewReplacer，保证 map 遍历无序时替换规则仍稳定；少量一次性替换可直接使用。循环或高频替换应优先使用 `NewReplacer` 复用已排序规则和标准库 replacer。

------

### 1.14 字符串克隆

------

#### func	strings.Clone

```go
func Clone(s string) string
```

备注：返回 s 的新副本。

------

### 1.15 字符串反转

------

#### func [utils.ReverseString](string.go)

```go
func ReverseString(str string) string
```

备注：将字符串 str 反转。

------

### 1.16 生成随机字符串

------

#### func [utils.UniqueID](string.go)

```go
func UniqueID(l uint8, r ...*rand.Rand) string
```

| 参数  | 描述                                          |
|-----|---------------------------------------------|
| *l* | 生成字符串的长度。                                   |
| *r* | 可选随机源，仅用第一项；省略或 nil 时用全局源，自定义源经包级锁串行访问。 |

备注：长度限制在 16–32 字节，由 base36 纳秒时间戳和随机段拼接；不保证全局唯一或时钟回拨时的排序。该函数基于 `math/rand/v2`，仅适用于非安全场景；token、验证码、重置链接等安全用途请使用 `SecureUniqueID`。

------

#### func [utils.RandomLetters](string.go)

```go
func RandomLetters(n int, r ...*rand.Rand) string
```

| 参数  | 描述                                          |
|-----|---------------------------------------------|
| *n* | 生成字符串的长度。                                   |
| *r* | 可选随机源，仅用第一项；省略或 nil 时用全局源，自定义源经包级锁串行访问。 |

备注：随机生成字符串 ALPHA。ALPHA 值为：A-Za-z。该函数基于 `math/rand/v2`，仅适用于测试数据、临时标识等非安全场景；安全用途请使用
`SecureRandomLetters`。

------

#### func [utils.RandomID](string.go)

```go
func RandomID(n int, r ...*rand.Rand) string
```

| 参数  | 描述                                          |
|-----|---------------------------------------------|
| *n* | 生成字符串的长度。                                   |
| *r* | 可选随机源，仅用第一项；省略或 nil 时用全局源，自定义源经包级锁串行访问。 |

备注：随机生成字符串 ALNUM。ALNUM 值为：A-Za-z0-9；为兼容旧行为，首字符固定为字母，不会以数字开头。该函数基于 `math/rand/v2`，
仅适用于非安全场景；安全用途请使用 `SecureRandomID`。

------

#### func [utils.RandomString](string.go)

```go
func RandomString(n int, alpha string, r ...*rand.Rand) string
```

| 参数      | 描述                                          |
|---------|---------------------------------------------|
| *n*     | 生成字符串的长度。                                   |
| *alpha* | 候选字节集合，重复字节会增加其被选中的概率。                                 |
| *r*     | 可选随机源，仅用第一项；省略或 nil 时用全局源，自定义源经包级锁串行访问。 |

备注：从 alpha 按字节抽样，输出 n 字节；n <= 0 或 alpha 为空时返回空串，多字节字符可能被拆开。该函数基于 `math/rand/v2`，仅适用于非安全场景；安全用途请使用
`SecureRandomString`。

------

### 1.17 字符串 Read

------

#### struct	Reader

```go
reader := strings.NewReader(s)
n, err := reader.Read(buf)
if err != nil && err != io.EOF {
	fmt.Println(err)
	return
}
fmt.Printf("read %d bytes: %q\n", n, buf[:n])
```

备注：strings.Reader 实现 io.Reader；即使同时返回 EOF，也应先使用前 n 个有效字节。

------

### 1.18 字符转义

------

#### func	html.EscapeString

```go
func EscapeString(s string) string
```

备注：将html文本中的字符转换为实体字符。

------

#### func	html.UnescapeString

```go
func UnescapeString(s string) string
```

备注：将 HTML 实体还原为对应字符，不解析 HTML 标签。

------

#### func	url.QueryEscape

```go
func QueryEscape(s string) string
```

备注：转义单个查询参数值，空格编码为 `+`；完整查询串可用 `url.Values.Encode`。

------

#### func	url.QueryUnescape

```go
func QueryUnescape(s string) (string, error)
```

备注：解码查询参数值，将 `+` 还原为空格；无效的百分号转义返回错误。

------

#### func [utils.URLPath](url.go)

```go
func URLPath(urlPath string, params url.Values) (string, error)
```

| 参数      | 描述         |
|---------|------------|
| urlPath | 基础 URL 路径。 |
| params  | 查询参数键值对。   |

备注：将 params 合并到查询字符串；同名键替换原有值，其他键保留，不修改传入的 params。

------

## 2. 编码与解码

------

### 2.1 JSON编码解码

------

#### func	json.Marshal

```go
func Marshal(v any) ([]byte, error)
```

备注：对数据进行 JSON 编码

------

#### func	json.Unmarshal

```go
func Unmarshal(data []byte, v any) error
```

备注：对数据进行 JSON 解码

------

### 2.2 BASE64编码解码

------

#### 文本数据进行 base64 编码

```go
base64.StdEncoding.EncodeToString(src)
```

#### 文本数据进行 base64 解码

```go
base64.StdEncoding.DecodeString(s)
```

#### URL或文件名进行 base64 编码

```go
base64.URLEncoding.EncodeToString(src)
```

#### URL或文件名进行 base64 解码

```go
base64.URLEncoding.DecodeString(s)
```

------

## 3. MATH 函数

------

### 3.1 math函数

------

#### func	math.Pow

```go
func Pow(x, y float64) float64
```

备注：返回x的y次方。

------

#### func	math.Abs

```go
func Abs(x float64) float64
```

备注：返回x的绝对值。

------

#### func	math.Round

```go
func Round(x float64) float64
```

备注：返回最接近的整数，从零四舍五入。

------

#### func	math.Ceil

```go
func Ceil(x float64) float64
```

备注：返回x的上舍入值。

------

#### func	math.Floor

```go
func Floor(x float64) float64
```

备注：返回x的下舍入值。

------

#### func	math.Mod

```go
func Mod(x, y float64) float64
```

备注：返回x/y的余数。

------

#### func	math.Max

```go
func Max(x, y float64) float64
```

备注：返回x和y中最大值。

------

#### func	math.Min

```go
func Min(x, y float64) float64
```

备注：返回x和y中最小值。

------

### 3.2 utils函数

------

#### func [utils.Rand](math.go)

```go
func Rand(minInt, maxInt int64, r ...*rand.Rand) int64
```

| 参数       | 描述                                          |
|----------|---------------------------------------------|
| *minInt* | 最小值。                                        |
| *maxInt* | 最大值。                                        |
| *r*      | 可选随机源，仅用第一项；省略或 nil 时用全局源，自定义源经包级锁串行访问。 |

备注：返回 minInt~maxInt 之间的随机数，值可能包含 minInt 和 maxInt；当 minInt 大于 maxInt 时会自动交换。

------

#### func [utils.Round](math.go)

```go
func Round(num float64, precision int) float64
```

| 参数          | 描述     |
|-------------|--------|
| *num*       | 原数值。   |
| *precision* | 保留小数位。 |

备注：对num进行四舍五入，并保留指定小数位。

------

## 4. 文件

------

### 4.1 获取文件信息

------

#### func	os.Stat

```go
func Stat(name string) (FileInfo, error)
```

备注：获取名为name的文件或目录信息。

------

### 4.2 创建与移除

------

#### func	os.Mkdir

```go
func Mkdir(name string, perm FileMode) error
```

备注：创建目录。

------

#### func	os.Remove

```go
func Remove(name string) error
```

备注：删除名为name的文件。

------

### 4.3 打开与创建

------

#### func	os.OpenFile

```go
func OpenFile(name string, flag int, perm FileMode) (*File, error)
```

备注：按 flag 打开或创建文件；perm 用于新建文件，受 umask 影响。成功返回的句柄由调用方 Close。

------

#### func	os.Create

```go
func Create(name string) (*File, error)
```

备注：创建文件，已存在时截断；成功返回的句柄由调用方 Close。

------

### 4.4 读取与写入

------

#### func	os.ReadFile

```go
func ReadFile(name string) ([]byte, error)
```

备注：读取名为name的文件内容。

------

#### func	os.WriteFile

```go
func WriteFile(name string, data []byte, perm FileMode) error
```

备注：将data数据写入name文件。

------

### 4.5 重命名与移动

#### func	os.Rename

```go
func Rename(oldpath, newpath string) error
```

备注：重命名文件(夹)oldpath为newpath，或移动文件。

------

### 4.6 获取目录路径

------

#### func	os.Getwd

```go
func Getwd() (dir string, err error)
```

备注：返回当前工作目录的绝对路径。

------

#### func	filepath.Abs

```go
func Abs(path string) (string, error)
```

备注：获取path的绝对路径。

------

#### func	filepath.IsAbs

```go
func IsAbs(path string) bool
```

备注：判断path路径是否是绝对路径。

------

#### func	filepath.Rel

```go
func Rel(basepath, targpath string) (string, error)
```

备注：返回一个相对路径。

------

#### func	filepath.Clean

```go
func Clean(path string) string
```

备注：按词法清理重复分隔符、`.` 和 `..`，不访问文件系统。

------

### 4.7 权限

------

#### func	os.Chmod

```go
	func Chmod(name string, mode FileMode) error
```

备注：改变文件(夹)name的权限。

------

#### func	os.Chown

```go
	func Chown(name string, uid, gid int) error
```

备注： 改变文件的所有者。

------

### 4.8 解析路径名

------

#### func	filepath.Ext

```go
	func Ext(path string) string
```

备注：返回path文件扩展名。

------

#### func	filepath.Base

```go
	func Base(path string) string
```

备注： (path为一个文件路径)可获取path中的文件名。

------

#### func	filepath.Dir

```go
	func Dir(path string) string
```

备注：获取path路径的目录。

------

### 4.9 路径的切分和拼接

------

#### func	filepath.Split

```go
	func Split(path string) (dir, file string)
```

备注：将path路径分成dir目录和file文件。

------

#### func	filepath.Join

```go
	func Join(elem ...string) string
```

备注：Join函数可以将任意数量的路径元素放入一个单一路径里。

------

### 4.10  路径判断

------

#### func [utils.IsDir](file.go)

```go
func IsDir(path string) bool
```

备注：判断给定路径是否是一个目录。

------

#### func [utils.IsFile](file.go)

```go
func IsFile(filepath string) bool
```

备注：路径可访问且不是目录时返回 true，会跟随符号链接；不等同于仅判断普通文件。

------

#### func [utils.IsExist](file.go)

```go
func IsExist(path string) bool
```

备注：仅当 Stat 确认路径不存在时返回 false；权限等其他访问错误仍返回 true。

------

### 4.11 获取文件大小

------

#### func [utils.Size](file.go)

```go
func Size(filepath string) (int64, error)
```

备注：取得文件大小。

------

#### func [utils.FormatFileSize](file.go)

```go
func FormatFileSize(size int64, decimals uint) string
```

| 参数         | 描述            |
|------------|---------------|
| *size*     | 文件实际大小(Byte)。 |
| *decimals* | 保留几位小数。       |

备注：文件大小格式化已可读式显示文件大小。

------

### 4.12 复制文件

------

#### func [utils.Copy](file.go)

```go
func Copy(src, dst string) error
```

| 参数    | 描述      |
|-------|---------|
| *src* | 拷贝的原文件。 |
| *dst* | 拷贝后的文件。 |

备注：拷贝文件。

------

### 4.13 获取目录文件

------

#### func [utils.FindFiles](file.go)

```go
func FindFiles(path string, depth bool, match ...string) (files []FileInfo, err error)
```

| 参数      | 描述                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
|---------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| *path*  | 目录路径。                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| *depth* | 深度查找: true 采用filepath.WalkDir遍历; false 只在当前目录查找。                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| *match* | 匹配规则:<br />   - `无参` : 匹配所有文件名 FindFiles(path, depth)<br />   - `*`   : 匹配所有文件名 FindFiles(path, depth, `*`) <br />  - `文件完整名`      : 精准匹配文件名 FindFiles(path, depth, fullFileName) <br />  - `e`, `文件完整名` : 精准匹配文件名 FindFiles(path, depth, `e`, fullFileName) <br />  - `p`, `文件前缀名` : 匹配前缀文件名 FindFiles(path, depth, `p`, fileNamePrefix)<br />  - `s`, `文件后缀名` : 匹配后缀文件名 FindFiles(path, depth, `s`, fileNameSuffix) <br />  - `r`, `正则表达式` : 正则匹配文件名 FindFiles(path, depth, `r`, fileNameReg) |

备注：获取目录下所有匹配文件；匹配规则会在遍历前编译成 matcher，正则只编译一次，适合目录文件较多的场景。

------

### 4.14 内容读取

------

#### func [utils.Scan](file.go)

```go
func Scan(r io.Reader, handle ReadScan, size ...int) error
```

| 参数       | 描述                                                                                                                                                                    |
|----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| *r*      | 实现io.Reader接口。                                                                                                                                                        |
| *handle* | func(num int, line []byte, err error) error；num 从 1 开始，line 不含行尾换行符，err 固定为 nil；返回可被 errors.Is 匹配为 DONE 的错误时正常停止。 |
| *size*   | 只取第一项；大于默认 64 KiB 时生效，最多 4 GiB，限制需包含行尾分隔符                                                                                                               |

备注：按行扫描，读取错误由 Scan 返回；line 仅在本次回调内有效，保留内容时需复制。Scan、Line 和 Read 均不关闭传入的 Reader，由调用方管理其生命周期。

------

#### func [utils.Line](file.go)

```go
func Line(r io.Reader, handle ReadLine) error
```

| 参数       | 描述                                                                                                                                                                                                                                 |
|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| *r*      | 实现io.Reader接口。                                                                                                                                                                                                                     |
| *handle* | func(num int, line []byte, lineDone bool) error 函数。<br /> num 行号: 当前扫描到第几行<br /> line 行数据: 当前扫描的行数据<br /> lineDone 当前行(num)数据是否读取完毕: true 当前行(num)数据读取完毕; false 当前行(num)数据未读完<br /> error 处理错误信息: 返回可被 errors.Is 匹配为 DONE 的错误时正常终止读取 |

备注：按物理行读取，长行通过多次回调交付，行号在整行结束后递增。末行恰好占满缓冲区时，可能通过空末块通知 `lineDone=true`。回调切片仅在本次调用内有效，保留内容时需自行复制。读取错误在有效尾块或结束通知交给回调后返回；回调返回 `DONE` 时正常结束，其他回调错误优先返回。

------

#### func [utils.Read](file.go)

```go
func Read(r io.Reader, handle ReadBlock) error
```

| 参数       | 描述                                                                                                                                 |
|----------|------------------------------------------------------------------------------------------------------------------------------------|
| *r*      | 实现io.Reader接口。                                                                                                                     |
| *handle* | func(size int, block []byte) error 函数。<br /> size 读取的数据块大小<br /> block 读取的数据块<br /> error 处理错误信息: 返回可被 errors.Is 匹配为 DONE 的错误时正常终止读取 |

备注：分块读取，适用于大文件或无换行数据。Reader 返回 `(0, nil)` 时继续读取，直到返回错误或回调返回 `DONE`；回调保留数据时需自行复制，因为下一次读取会复用缓冲区。

------

### 4.15 写入内容到文件

------

#### func [utils.NewWrite](file.go)

```go
func NewWrite(fileName string, opts ...WriteOption) (*WriteFile, error)
```

| 参数         | 描述                                                            |
|------------|---------------------------------------------------------------|
| *fileName* | 文件路径名。                                                        |
| *opts*     | 写入配置项: WithWriteAppend(true) 追加写入; WithWritePerm(0644) 设置文件权限 |

备注：返回一个 WriteFile 实例；调用方在写入结束后负责 Close。

补充说明：

- `NewWrite` 适合日志、顺序输出等持续写入场景。
- 若是配置文件、密钥文件、状态文件等“整文件覆盖”场景，生产环境更建议使用 `WriteFileAtomic` / `WriteStringAtomic`，避免直接
  `O_TRUNC` 导致半写文件。
- `NewWrite` 会拒绝目标文件为符号链接，降低通过通用写入口写穿到其它路径的风险。

```go
// 实例化一个 WriteFile（追加写入）
w, err := utils.NewWrite(fileName, utils.WithWriteAppend(true))
if err != nil {
	fmt.Printf("NewWrite() error = %v\n", err)
	return
}

// 写入结束后关闭句柄，关闭错误也需处理。
defer func() {
	if err := w.Close(); err != nil {
		fmt.Printf("Close() err %v\n", err)
	}
}()
```

------

#### func [utils.WriteFileAtomic](file.go) / [utils.WriteStringAtomic](file.go)

```go
func WriteFileAtomic(fileName string, data []byte, perm os.FileMode) error
func WriteStringAtomic(fileName, data string, perm os.FileMode) error
```

备注：使用同目录临时文件 + `Sync` + `Close` + `Rename` 原子覆盖目标文件，适用于配置、密钥、状态等要求“覆盖即完整替换”的场景。

```go
err := utils.WriteFileAtomic(fileName, []byte("hello world"), 0644)
if err != nil {
	fmt.Printf("WriteFileAtomic() err %v\n", err)
	return
}

err = utils.WriteStringAtomic(fileName, "hello world", 0644)
if err != nil {
	fmt.Printf("WriteStringAtomic() err %v\n", err)
	return
}
```

备注：适合一次性覆盖完整文件内容；若需要持续写入、追加写入或 `bufio.Writer` 批量写入，继续使用 `NewWrite`。

------

### 4.16 获取文件类型

------

#### func [utils.FileType](file.go)

```go
func FileType(f *os.File) (string, error)
```

备注：优先按文件扩展名匹配 MIME 类型；没有匹配时，从文件头读取最多 512 字节检测，且不改变文件偏移。短文件和空文件照常检测，其他读取错误会返回给调用方。

------

## 5. 加密、解密与摘要

------

### 5.1 MD5 摘要

------

#### func [utils.MD5](md5.go)

```go
func MD5(str string) string
```

备注：返回 MD5 十六进制摘要。MD5 已不适合密码存储、签名、完整性安全校验等安全场景，仅用于历史协议兼容或非安全哈希标识。

------

### 5.2 SHA 摘要

------

#### func [utils.SHA1](sha.go)

```go
func SHA1(str string) string
```

备注：返回 SHA-1 十六进制摘要。SHA-1 已不适合签名、证书、密码存储等安全场景，仅用于历史协议兼容或非安全哈希标识。

------

#### func [utils.SHA256](sha.go)

```go
func SHA256(str string) string
```

备注：返回 SHA-256 十六进制摘要。普通摘要场景可使用；密码存储仍应使用 bcrypt/argon2/scrypt 等专用算法。

------

#### func [utils.SHA512](sha.go)

```go
func SHA512(str string) string
```

备注：返回 SHA-512 十六进制摘要。普通摘要场景可使用；密码存储仍应使用 bcrypt/argon2/scrypt 等专用算法。

------

### 5.3 RSA 非对称加密与解密

------

#### func [utils.GenerateKeyRSA](rsa.go)

```go
func GenerateKeyRSA(path string, bits int, pkcs ...bool) ([]string, error)
```

| 参数     | 描述                                                                                                                                      |
|--------|-----------------------------------------------------------------------------------------------------------------------------------------|
| *path* | 文件名路径。                                                                                                                                  |
| *bits* | 生成秘钥位大小，生产环境要求至少 2048。                                                                                                                  |
| *pkcs* | pkcs[0] 默认 true，输出 PKIX 公钥，false 输出 PKCS1 公钥；pkcs[1] 默认 true，输出 PKCS1 私钥，false 输出 PKCS8 私钥。 |

备注：返回公钥、私钥两个文件名，默认分别为 PKIX 和 PKCS1 格式；密钥位数低于 2048 时返回错误。PKIX 公钥沿用 `public_pkcs8_` 文件名前缀；PKCS8 私钥沿用 `RSA PRIVATE KEY` 标记，导入时按 DER 内容识别格式。

------

#### RSA 加密与解密：[utils.NewRSA](rsa.go) / [(*RSA).Encrypt](rsa.go) / [(*RSA).Decrypt](rsa.go) / [(*RSA).EncryptOAEP](rsa.go) / [(*RSA).DecryptOAEP](rsa.go)

密钥支持 PEM 文本、base64 DER 文本和原始 DER 数据，也可通过 `utils.WithRSAFilePath(true)` 从文件读取。文本允许首尾空白；原始 DER 必须保留完整二进制字节，不应在外层添加空白。

```go
// 两个参数按密钥文件路径读取。
r, err := utils.NewRSA(publicKey, privateKey, utils.WithRSAFilePath(true))
if err != nil {
	fmt.Printf("utils.NewRSA() err = %v\n", err)
	return
}

marshal, err := json.Marshal(map[string]any{
	"Title":   "示例",
	"Content": "待加密内容",
})
if err != nil {
	fmt.Printf("Marshal() error = %v\n", err)
	return
}

// PKCS#1 v1.5 保留用于旧协议。
encodeString, err := r.Encrypt(string(marshal), base64.StdEncoding.EncodeToString)
if err != nil {
	fmt.Printf("Encrypt() err = %v\n", err)
	return
}

decryptString, err := r.Decrypt(encodeString, base64.StdEncoding.DecodeString)
if err != nil {
	fmt.Printf("Decrypt() err = %v\n", err)
	return
}

fmt.Println(decryptString == string(marshal))

// OAEP 摘要实例由当前调用独占。
encodeString, err = r.EncryptOAEP(string(marshal), base64.StdEncoding.EncodeToString, sha256.New())
if err != nil {
	fmt.Printf("Encrypt() err = %v\n", err)
	return
}

decryptString, err = r.DecryptOAEP(encodeString, base64.StdEncoding.DecodeString, sha256.New())
if err != nil {
	fmt.Printf("Decrypt() err = %v\n", err)
	return
}
fmt.Println(decryptString == string(marshal))
```

备注：先实例化RSA 设置公钥私钥，使用公钥加密数据， 私钥解密数据。

------

#### RSA 签名与验签：[(*RSA).Sign](rsa.go) / [(*RSA).Verify](rsa.go) / [(*RSA).SignPSS](rsa.go) / [(*RSA).VerifyPSS](rsa.go)

```go
// 两个参数按密钥文件路径读取。
r, err := utils.NewRSA(publicKey, privateKey, utils.WithRSAFilePath(true))
if err != nil {
	fmt.Printf("utils.NewRSA() err = %v\n", err)
	return
}

marshal, err := json.Marshal(map[string]any{
	"Title":   "示例",
	"Content": "待加密内容",
})
if err != nil {
	fmt.Printf("Marshal() error = %v\n", err)
	return
}

// 传入完整正文，摘要计算由签名入口完成。
sign, err := r.Sign(string(marshal), crypto.SHA256, base64.StdEncoding.EncodeToString)
if err != nil {
	fmt.Printf("Sign() err = %v\n", err)
	return
}

// 验签须使用签名时的正文和摘要算法。
if err := r.Verify(string(marshal), sign, crypto.SHA256, base64.StdEncoding.DecodeString); err != nil {
	fmt.Printf("Verify() err = %v\n", err)
	return
}
fmt.Println("Verify() = 验证成功")

sign, err = r.SignPSS(string(marshal), crypto.SHA256, base64.StdEncoding.EncodeToString, nil)
if err != nil {
	fmt.Printf("Sign() err = %v\n", err)
	return
}

if err := r.VerifyPSS(string(marshal), sign, crypto.SHA256, base64.StdEncoding.DecodeString, nil); err != nil {
	fmt.Printf("Verify() err = %v\n", err)
	return
}
fmt.Println("Verify() = 验证成功")
```

备注：先实例化RSA 设置公钥私钥，使用私钥签名，公钥验签。

------

#### 加密模式选型建议

| 场景         | 推荐方案                                            | 说明                                     |
|------------|-------------------------------------------------|----------------------------------------|
| 新系统对称加密    | `AES + GCM`                                     | 默认优先方案，带机密性和完整性校验，支持 `additionalData`。 |
| 兼容分组协议     | `AES/DES + CBC + WithIV/WithRandIV`             | 需要补位，适合和旧系统的固定协议对接。                    |
| 旧系统流模式兼容   | `CTR/CFB/OFB + utils.WithAllowUnsafeStreamMode(true)` | 默认禁用；无需块补位，但不提供完整性校验，仅用于兼容历史协议。        |
| 旧系统 ECB 兼容 | `ECB + WithAllowUnsafeECB(true)`                | 默认禁用，不建议新系统使用。                         |
| RSA 加密     | `EncryptOAEP/DecryptOAEP`                       | 新协议优先，适合加密小数据或对称密钥。                    |
| RSA 签名     | `SignPSS/VerifyPSS`                             | 新协议优先，推荐配合 `SHA256` 或更强摘要。             |

备注：

- RSA 适合加密小块数据或密钥封装，不建议直接承载大体积正文。
- `PKCS#1 v1.5` 加解密与签名接口主要保留给旧系统兼容；新系统优先使用 OAEP/PSS。
- 生产环境推荐 `2048` 位及以上 RSA 密钥，摘要算法推荐 `SHA256` / `SHA384` / `SHA512`。

------

#### RSA 密钥格式转换：[utils.RemovePEMHeaders](rsa.go) / [utils.AddPEMHeaders](rsa.go)

```go
// pubFile/priFile 为默认 GenerateKeyRSA 生成的 PKIX 公钥与 PKCS#1 私钥路径。
for keyType, fileName := range map[string]string{"public": pubFile, "private": priFile} {
	data, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	body := utils.RemovePEMHeaders(string(data))
	restored, err := utils.AddPEMHeaders(body, keyType)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(keyType, restored == strings.TrimSpace(string(data)))
}
```

------

### 5.4 AES 加密与解密

------

#### AES [加密与解密](aes.go)

```go
// 固定 IV 来自双方协议，AES 要求 16 个原始字节。
a, err := utils.AES(key, utils.WithIV(iv))
if err != nil {
	fmt.Printf("utils.AES() error = %v\n", err)
	return
}

encryptStr, err := a.Encrypt(data, utils.CBC, base64.StdEncoding.EncodeToString, utils.PKCS7Pad)
if err != nil {
	fmt.Printf("Encrypt() mode = %v error = %v\n", utils.CBC, err)
	return
}

got, err := a.Decrypt(encryptStr, utils.CBC, base64.StdEncoding.DecodeString, utils.PKCS7Unpad)
if err != nil {
	fmt.Printf("Decrypt() mode = %v error = %v\n", utils.CBC, err)
	return
}
fmt.Println(got == data)
```

备注：先实例化 AES 并设置 key；CBC/CTR/CFB/OFB 等需要显式通过 `WithIV` 设置固定 IV，或通过 `utils.WithRandIV(true)` 使用随机 IV。
`CTR/CFB/OFB` 属于非认证流模式，默认禁用，仅在兼容旧系统时通过 `utils.WithAllowUnsafeStreamMode(true)` 显式开启。`ECB`
默认禁用，仅在兼容旧系统时通过 `WithAllowUnsafeECB(true)` 显式开启；历史上“未设置 IV 时使用 key 派生
IV”的兼容行为也默认禁用，如需兼容旧密文可显式开启 `WithAllowUnsafeKeyIV(true)`。

补充：`CTR/CFB/OFB` 若用于旧协议兼容且无需补位，建议配合 `NoPad` / `NoUnpad` 使用，避免多余填充和块长度约束。

------

### 5.5 DES 加密与解密

------

#### DES 加密与解密：[utils.DES](des.go) / [utils.TripleDES](des.go)

```go
// 固定 IV 来自双方协议，DES/3DES 要求 8 个原始字节。
a, err := utils.DES(key, utils.WithIV(iv))
if err != nil {
	fmt.Printf("utils.DES() error = %v\n", err)
	return
}

encryptStr, err := a.Encrypt(data, utils.CBC, base64.StdEncoding.EncodeToString, utils.PKCS7Pad)
if err != nil {
	fmt.Printf("Encrypt() mode = %v error = %v\n", utils.CBC, err)
	return
}

got, err := a.Decrypt(encryptStr, utils.CBC, base64.StdEncoding.DecodeString, utils.PKCS7Unpad)
if err != nil {
	fmt.Printf("Decrypt() mode = %v error = %v\n", utils.CBC, err)
	return
}
fmt.Println(got == data)
```

备注：DES/3DES 仅建议用于旧系统兼容，新系统应优先使用 AES-GCM。先实例化 DES 并设置 key；CBC/CTR/CFB/OFB 等需要显式通过
`WithIV` 设置固定 IV，或通过 `utils.WithRandIV(true)` 使用随机 IV。`CTR/CFB/OFB` 属于非认证流模式，默认禁用，仅在兼容旧系统时通过
`utils.WithAllowUnsafeStreamMode(true)` 显式开启。
`ECB` 默认禁用，仅在兼容旧系统时通过 `WithAllowUnsafeECB(true)` 显式开启；历史上“未设置 IV 时使用 key 派生
IV”的兼容行为也默认禁用，如需兼容旧密文可显式开启 `WithAllowUnsafeKeyIV(true)`。

补充：`CTR/CFB/OFB` 若用于旧协议兼容且无需补位，建议配合 `NoPad` / `NoUnpad` 使用，避免多余填充和块长度约束。

------

### 5.6 pkcs7 填充与反填充

------

#### AES GCM（推荐）

```go
a, err := utils.AES(key)
if err != nil {
	fmt.Printf("utils.AES() error = %v\n", err)
	return
}

// GCM 加密，additionalData 可为空
encryptStr, err := a.EncryptGCMString(data, base64.StdEncoding.EncodeToString, []byte("request-id=r-1"))
if err != nil {
	fmt.Printf("EncryptGCMString() error = %v\n", err)
	return
}

// GCM 解密，additionalData 必须与加密时一致
got, err := a.DecryptGCMString(encryptStr, base64.StdEncoding.DecodeString, []byte("request-id=r-1"))
if err != nil {
	fmt.Printf("DecryptGCMString() error = %v\n", err)
	return
}
fmt.Println(got == data)
```

备注：GCM 属于 AEAD 认证加密模式，优先级高于 CBC/CTR/CFB/OFB；每次加密都会自动生成随机 nonce 并写入密文头部。

------

#### NoPad / NoUnpad

```go
// CTR 不提供完整性校验，仅在兼容旧协议时显式开启。
a, err := utils.AES(key, utils.WithRandIV(true), utils.WithAllowUnsafeStreamMode(true))
if err != nil {
	fmt.Println(err)
	return
}

encryptStr, err := a.Encrypt(data, utils.CTR, base64.StdEncoding.EncodeToString, utils.NoPad)
if err != nil {
	fmt.Println(err)
	return
}

got, err := a.Decrypt(encryptStr, utils.CTR, base64.StdEncoding.DecodeString, utils.NoUnpad)
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println(got == data)
```

备注：`NoPad` / `NoUnpad` 适合 `CTR/CFB/OFB` 等不依赖块补位的模式；GCM 接口不接收填充回调。`CTR/CFB/OFB`
不提供完整性校验，默认禁用，不建议用于新协议。

------

#### 加密模块性能验证命令

```bash
# 对称加密基准
go test -run '^$' -bench 'BenchmarkCipherAES(CBCEncrypt|CBCDecrypt|CTRNoPadEncrypt|CTRNoPadDecrypt|GCMEncrypt|GCMDecrypt)$' -benchmem ./...

# RSA 基准
go test -run '^$' -bench 'BenchmarkRSA(EncryptOAEP|DecryptOAEP|SignPSS|VerifyPSS)$' -benchmem ./...

# 全量竞态验证
go test -race ./...
```

备注：

- 基准数据受 CPU、系统负载、Go 版本影响较大，建议关注相对变化而不是绝对数值。
- `RSA` 私钥解密/签名天然比公钥加密/验签更重，属于算法特性，不是实现异常。
- 对称加密建议把 `CBC`、`CTR-NoPad`、`GCM` 分开观察，分别对应兼容分组模式、旧协议流模式和认证加密模式。

------

#### func [utils.PKCS7Pad](pkcs7.go)

```go
func PKCS7Pad(data []byte, blockSize int) []byte
```

备注：数据填充。

------

#### func [utils.PKCS7Unpad](pkcs7.go)

```go
func PKCS7Unpad(data []byte) ([]byte, error)
```

备注：校验并移除 PKCS7 填充，返回输入的子切片；填充不合法时返回错误。此校验不提供密文完整性认证。

------

### 5.7 0 填充与反填充

------

#### func [utils.ZeroPad](zero.go)

```go
func ZeroPad(data []byte, blockSize int) []byte
```

备注：数据填充。

------

#### func [utils.ZeroUnpad](zero.go)

```go
func ZeroUnpad(data []byte) ([]byte, error)
```

备注：数据反填充。

------

## 6. 整形、字符串转换

------

### 6.1 字符串转整形

------

#### func	strconv.ParseInt

```go
func ParseInt(s string, base int, bitSize int) (i int64, err error)
```

备注：string 转 int64。

------

#### func	strconv.Atoi

```go
func Atoi(s string) (int, error)
```

备注：string 转 int。

------

#### func [utils.ToInt64](strconv.go)

```go
func ToInt64(s string) (i int64)
```

备注：按十进制解析为 int64，接受正负号但不去除空白；语法错误返回 0，溢出返回相应整数边界值。

------

#### func [utils.ToInt](strconv.go)

```go
func ToInt(s string) (i int)
```

备注：按十进制解析为 int，接受正负号但不去除空白；语法错误返回 0，溢出返回当前平台的 int 边界值。

------

### 6.2 整形转字符串

------

#### func	strconv.FormatInt

```go
func FormatInt(i int64, base int) string
```

备注：int64 转 string。

------

#### func	strconv.Itoa

```go
func Itoa(i int) string
```

备注：int 转 string。

------

### 6.3 字符串转浮点数

------

#### func	strconv.ParseFloat

```go
func ParseFloat(s string, bitSize int) (float64, error)
```

备注：string 转 float64。

------

#### func [utils.ToFloat64](strconv.go)

```go
func ToFloat64(s string) (i float64)
```

备注：解析为 float64；语法错误返回 0，溢出返回相应符号的 Inf，NaN 和 Inf 沿用 `strconv.ParseFloat` 规则。

------

### 6.4 浮点数转字符串

------

#### func	strconv.FormatFloat

```go
func FormatFloat(f float64, fmt byte, prec, bitSize int) string
```

备注：float64 转 string。

------

### 6.5 以千位分隔符方式格式化一个数字

------

#### func [utils.FormatNumber](misce.go)

```go
func FormatNumber(number float64, decimals uint, decPoint, thousandsSep string) string
```

| 参数           | 描述        |
|--------------|-----------|
| *number*     | 需要格式化的数字。 |
| *decimals*   | 保留几位小数。   |
| *decPoint*   | 小数点[.]    |
| thousandsSep | 千位分隔符[,]  |

备注：以千位分隔符方式格式化一个数字。

------

### 6.6 进制转换

------

#### func [utils.BinOct](strconv.go)

```go
func BinOct(str string) (string, error)
```

备注：二进制转换为八进制。

------

#### func [utils.BinDec](strconv.go)

```go
func BinDec(str string) (int64, error)
```

备注：二进制转换为十进制。

------

#### func [utils.BinHex](strconv.go)

```go
func BinHex(str string) (string, error)
```

备注：二进制转换为十六进制。

------

#### func [utils.OctBin](strconv.go)

```go
func OctBin(data string) (string, error)
```

备注：八进制转换为二进制。

------

#### func [utils.OctDec](strconv.go)

```go
func OctDec(str string) (int64, error)
```

备注：八进制转换为十进制。

------

#### func [utils.OctHex](strconv.go)

```go
func OctHex(data string) (string, error)
```

备注：八进制转换为十六进制。

------

#### func [utils.DecBin](strconv.go)

```go
func DecBin(number int64) string
```

备注：十进制转换为二进制。

------

#### func [utils.DecOct](strconv.go)

```go
func DecOct(number int64) string
```

备注：十进制转换为八进制。

------

#### func [utils.DecHex](strconv.go)

```go
func DecHex(number int64) string
```

备注：十进制转换为十六进制。

------

#### func [utils.HexBin](strconv.go)

```go
func HexBin(data string) (string, error)
```

备注：十六进制转换为二进制。

------

#### func [utils.HexOct](strconv.go)

```go
func HexOct(str string) (string, error)
```

备注：十六进制转换为八进制。

------

#### func [utils.HexDec](strconv.go)

```go
func HexDec(str string) (int64, error)
```

备注：十六进制转换为十进制。

------

## 7. 数组/切片/链表

------

### 7.1 检查数组中是否存在某个值

------

#### func [utils.Contains](slices.go)

```go
func Contains[T comparable](v T, s []T) bool
```

备注：检查s中是否存在v。1.21版本以上推荐使用标准库 slices.Contains(s,v)

------

### 7.2 统计某个值在数组中出现次数

------

#### func [utils.HasCount](slices.go)

```go
func HasCount[T comparable](v T, s []T) (count int)
```

备注：统计v在s中出现次数。

------

### 7.3 反转数组

------

#### func [utils.Reverse](slices.go)

```go
func Reverse[T any](s []T) []T
```

备注：反转s。1.21版本以上推荐使用标准库 slices.Reverse(s)

------

### 7.4 去除数组中重复的值

------

#### func [utils.Unique](slices.go)

```go
func Unique[T comparable](s []T) []T
```

备注：按首次出现顺序去重，返回独立结果切片，不深拷贝元素；空结果为非 nil 切片。

------

### 7.5 计算两个数组的差集

------

#### func [utils.Diff](slices.go)

```go
func Diff[T comparable](s1, s2 []T) []T
```

备注：返回 s1 中不在 s2 中的元素，保留 s1 的顺序和重复值；结果使用独立切片，空结果为非 nil 切片。

------

### 7.6 计算两个数组的交集

------

#### func [utils.Intersect](slices.go)

```go
func Intersect[T comparable](s1, s2 []T) []T
```

备注：返回 s1 中也在 s2 中的元素，保留 s1 的顺序和重复值；结果使用独立切片，空结果为非 nil 切片。

------

### 7.7 切片求和

------

#### func [utils.SumSlice](slices.go)

```go
func SumSlice[T Number](nums []T) T
```

备注：计算切片中所有元素的和。

------

### 7.8 链表 container/list.List

移动、删除和合并小节各自使用输出中列出的初始列表，不依次累加前一节的修改。

------

#### 创建列表

```go
// 通过 list.New 创建列表
l := list.New()

// 添加元素
l.PushBack("5")
l.PushBack("6")

// 列表遍历
for i := l.Front(); i != nil; i = i.Next() {
	fmt.Println("Element =", i.Value)
}
```

输出：

```tex
Element = 5
Element = 6
```

------

#### 在列表头部插入元素

```go
// 在列表头部插入
ele4 := l.PushFront("4")
```

输出：

```tex
Element = 4
Element = 5
Element = 6
```

------

#### 在列表尾部插元素

```go
// 在列表尾部插
ele8 := l.PushBack("8")
```

输出：

```tex
Element = 4
Element = 5
Element = 6
Element = 8
```

------

#### 在列表指定元素前插入

```go
// 在指定元素ele4前插入元素"3"
ele3 := l.InsertBefore("3", ele4)
```

输出：

```tex
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
```

------

#### 在列表指定元素后插入

```go
// 在指定元素ele8后插入元素"9"
ele9 := l.InsertAfter("9", ele8)
```

输出：

```tex
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
```

------

#### 获取列表头结点

```go
// 获取列表头结点
front := l.Front()
fmt.Println("front =", front.Value)
```

输出：

```tex
front = 3
```

------

#### 获取列表尾结点

```go
// 获取列表尾结点
back := l.Back()
fmt.Println("back =", back.Value)
```

输出：

```tex
back = 9
```

------

#### 获取上一个结点

```go
// 获取ele4上一个结点
prev := ele4.Prev()
fmt.Println("ele4 prev =", prev.Value)
```

输出：

```tex
ele4 prev = 3
```

------

#### 获取下一个结点

```go
// 获取ele4下一个结点
next := ele4.Next()
fmt.Println("ele4 next =", next.Value)
```

输出：

```tex
ele4 next = 5
```

------

#### 移动到某元素的前面

```go
// 将ele4元素移动到back元素的前面
l.MoveBefore(ele4, back)
```

输出：

```tex
移动前：
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
--------------
移动后：
Element = 3
Element = 5
Element = 6
Element = 8
Element = 4
Element = 9
```

------

#### 移动到某元素的后面

```go
// 将ele4元素移动到back元素的后面
l.MoveAfter(ele4, back)
```

输出：

```tex
移动前：
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
--------------
移动后：
Element = 3
Element = 5
Element = 6
Element = 8
Element = 9
Element = 4
```

------

#### 移动到列表的最前面

```go
// 将ele9元素移动到列表的最前面
l.MoveToFront(ele9)
```

输出：

```tex
移动前：
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
--------------
移动后：
Element = 9
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
```

------

#### 移动到列表的最后面

```go
// 将ele3元素移动到列表的最后面
l.MoveToBack(ele3)
```

输出：

```tex
移动前：
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
--------------
移动后：
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
Element = 3
```

------

#### 列表正序遍历

```go
// 列表正序遍历
for i := l.Front(); i != nil; i = i.Next() {
	fmt.Println("Element =", i.Value)
}
```

输出：

```tex
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
```

------

#### 列表倒叙遍历

```go
// 列表倒叙遍历
for i := l.Back(); i != nil; i = i.Prev() {
	fmt.Println("Element =", i.Value)
}
```

输出：

```tex
Element = 9
Element = 8
Element = 6
Element = 5
Element = 4
Element = 3
```

------

#### 在列表中删除元素

```go
// 在列表中删除ele4元素
l.Remove(ele4)
// 在列表中删除ele8元素
l.Remove(ele8)
```

输出：

```tex
删除前：
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
--------------
删除后：
Element = 3
Element = 5
Element = 6
Element = 9
```

------

#### 在列表头部插入一个列表

```go
// 创建一个列表（头部列表）
frontList := list.New()
frontList.PushBack("f1")
frontList.PushBack("f2")

// 在列表头部插入一个列表
l.PushFrontList(frontList)
```

输出：

```tex
添加前：
Element = 3
Element = 4
--------------
添加后：
Element = f1
Element = f2
Element = 3
Element = 4
```

------

#### 在列表尾部插入一个列表

```go
// 创建一个列表（尾部列表）
backList := list.New()
backList.PushBack("b1")
backList.PushBack("b2")

// 在列表尾部插入一个列表
l.PushBackList(backList)
```

输出：

```tex
添加前：
Element = 3
Element = 4
--------------
添加后：
Element = 3
Element = 4
Element = b1
Element = b2
```

------

#### 获取列表长度

```go
l.Len()
```

------

#### 初始化或清除列表

```go
l.Init()
```

------

## 8. map/syncMap

------

### 8.1 获取map的所有key

------

#### func [utils.MapKeys](map.go)

```go
func MapKeys[K Ordered, V any](m map[K]V) []K
```

备注：返回所有 key，顺序不保证稳定。

------

### 8.2 获取map的所有value

------

#### func [utils.MapValues](map.go)

```go
func MapValues[K Ordered, V any](m map[K]V, isReverse ...bool) []V
```

| 参数          | 描述                       |
|-------------|--------------------------|
| *m*         | map。                     |
| *isReverse* | 是否降序排列：true 降序，false 升序。 |

备注：当传入 isReverse 参数时，会先按 key 排序后返回对应 value；未传入时直接返回 map 当前遍历到的所有 value，顺序不保证稳定。

------

### 8.3 有序遍历map元素

------

#### func [utils.MapRange](map.go)

```go
func MapRange[K Ordered, V any](m map[K]V, f func(key K, value V) bool, isReverse ...bool)
```

| 参数          | 描述                                          |
|-------------|---------------------------------------------|
| *m*         | map。                                        |
| *f*         | f 函数接收key与value，返回一个bool值，如果f函数返回false则终止遍历 |
| *isReverse* | 是否降序排列：true 降序，false 升序。                    |

备注：对map的key排序并按排序后的key遍历map的元素，如果f 函数返回 false则终止遍历。

------

### 8.4 过滤map的元素

------

#### func [utils.MapFilter](map.go)

```go
func MapFilter[K Ordered, V any](m map[K]V, f func(key K, value V) bool) map[K]V
```

| 参数  | 描述                                                   |
|-----|------------------------------------------------------|
| *m* | map。                                                 |
| *f* | f 函数接收key与value，返回一个bool值，如果f函数返回false则过滤掉该元素（删除该元素） |

备注：原地删除回调返回 false 的元素，返回同一个 map。

------

### 8.5 计算两个map的差集

------

#### func [utils.MapDiff](map.go)

```go
func MapDiff[K comparable, V comparable](m1, m2 map[K]V) []V
```

备注：计算 m1 与 m2 的值差集，返回结果保留 m1 当前遍历结果中的重复值。

------

#### func [utils.MapDiffKey](map.go)

```go
func MapDiffKey[K Ordered, V any](m1, m2 map[K]V) []K
```

备注：计算m1与m2的键差集。

------

### 8.6 计算两个map的交集

------

#### func [utils.MapIntersect](map.go)

```go
func MapIntersect[K comparable, V comparable](m1, m2 map[K]V) []V
```

备注：计算 m1 与 m2 的值交集，返回结果保留 m1 当前遍历结果中的重复值。

------

#### func [utils.MapIntersectKey](map.go)

```go
func MapIntersectKey[K Ordered, V any](m1, m2 map[K]V) []K
```

备注：计算m1与m2的键交集。

------

### 8.7 计算map的值和

------

#### func [utils.SumMap](map.go)

```go
func SumMap[K comparable, V Number](m map[K]V) V
```

备注：计算map中所有value的和。

------

### 8.8 sync.Map

------

#### 创建sync.Map

```go
//sync.Map 声明完之后，可以立即使用
var m sync.Map
m.Store("Server", "Golang")
m.Store("JavaScript", "Vue")
```

备注：创建map。

------

#### 添加元素 Store

```go
func (m *Map) Store(key, value any)
```

备注：存入键值对；key 的动态类型必须可比较，value 可为任意类型。

------

#### 获取元素 Load

```go
func (m *Map) Load(key any) (value any, ok bool)
```

备注：ok 表示键是否存在；value 为 any，需要调用具体类型的方法时再做类型断言。

------

#### 获取或添加 LoadOrStore

```go
func (m *Map) LoadOrStore(key, value any) (actual any, loaded bool)
```

备注：获取的 key 存在，返回 key 对应的元素，如果获取的 key 不存在，就返回设置的值，并且将设置的值，存入 map。

------

#### 删除元素 Delete

```go
func (m *Map) Delete(key any)
```

备注：删除元素，使用 sync.Map Delete 删除不存在的元素，不会报错。

------

#### 获取并删除 LoadAndDelete

```go
func (m *Map) LoadAndDelete(key any) (value any, loaded bool)
```

备注：如果key存在返回key的值并删除该key。

------

#### 遍历元素 Range

```go
func (m *Map) Range(f func(key, value any) bool)
```

备注：遍历元素，如果f 函数返回 false则终止遍历。

------

## 9. 时间

------


### 9.1 时区

------

#### func [utils.Local](time.go)

```go
func Local() *time.Location
```

备注：系统运行时区。

------

#### func [utils.CST](time.go)

```go
func CST() *time.Location
```

备注：东八时区。

------

#### func [utils.UTC](time.go)

```go
func UTC() *time.Location
```

备注：UTC时区。

------

### 9.2  验证日期：年、月、日

------

#### func [utils.CheckDate](time.go)

```go
func CheckDate(year, month, day int) bool
```

| 参数      | 描述  |
|---------|-----|
| *year*  | 年份。 |
| *month* | 月份。 |
| day     | 日期。 |

备注：验证日期：年、月、日；内部复用 `MonthDay` 判断闰年和大小月，边界为 `1 <= year <= 32767`。

------

### 9.3  获取指定月份有多少天

------

#### func [utils.MonthDay](time.go)

```go
func MonthDay(year int, month int) (days int)
```

| 参数      | 描述  |
|---------|-----|
| *year*  | 年份。 |
| *month* | 月份。 |

备注：获取指定月份有多少天。

------

### 9.4  增加时间

------

#### func [utils.AddTime](time.go)

```go
func AddTime(t time.Time, addTimes ...string) (time.Time, error)
```

| 参数         | 描述                                   |
|------------|--------------------------------------|
| *addTimes* | 增加时间（Y年，M月，D日，H时，I分，S秒，L毫秒，C微秒，N纳秒)。 |

备注：增加时间。

------

### 9.5  获取日期信息

------

#### func [utils.TimeDetails](time.go)

```go
func TimeDetails(t time.Time) map[string]any
```

备注：字段按 t 自身时区拆分；millisecond/microsecond/nanosecond 表示当前秒内的小数部分，weekDay 为 0–6（周日为 0），yearWeek 为 ISO 周数。

```
//	返回：year int - 年，
//		month int - 月，monthEn string - 英文月，
//		day int - 日，yearDay int - 一年中第几日，
//		hour int - 时，minute int - 分，second int - 秒，
//		millisecond int - 毫秒，microsecond int - 微秒，nanosecond int - 纳秒，
//		unix int64 - 时间戳-秒，unixNano int64 - 时间戳-纳秒，
//		weekDay int - 星期几，weekDayEn string - 星期几英文， yearWeek int - 一年中第几周，
//		date string - 格式化日期，dateNs string - 格式化日期（纳秒)
```

------

### 9.6  时间格式化为时间字符串

------

#### func [utils.TimeFormat](time.go)

```go
func TimeFormat(timeZone *time.Location, layout string, timestamp ...int64) string
```

| 参数          | 描述                  |
|-------------|---------------------|
| *timeZone*  | 时区。                 |
| layout      | 格式化。                |
| *timestamp* | 不传时使用当前时间；两项按 Unix 秒、纳秒解释；单项通常为秒，仅绝对值达到 1e18 时按纳秒解释。 |

备注：时间格式化为时间字符串。

------

### 9.7  解析时间字符串为time.Time

------

#### func [utils.TimeParse](time.go)

```go
func TimeParse(timeZone *time.Location, layout, timeStr string) (time.Time, error)
```

| 参数         | 描述     |
|------------|--------|
| *timeZone* | 时区。    |
| layout     | 格式化。   |
| *timeStr*  | 时间字符串。 |

备注：解析时间字符串为time.Time。

------

### 9.8  两个时间字符串判断

------

#### func [utils.Before](time.go)

```go
func Before(layout string, t1, t2 string) (bool, error)
```

| 参数     | 描述     |
|--------|--------|
| layout | 格式化。   |
| *t1*   | 时间字符串。 |
| t2     | 时间字符串。 |

备注：返回true: t1在t2之前（t1小于t2），返回false: t1大于等于t2。

------

#### func [utils.After](time.go)

```go
func After(layout string, t1, t2 string) (bool, error)
```

| 参数     | 描述     |
|--------|--------|
| layout | 格式化。   |
| *t1*   | 时间字符串。 |
| t2     | 时间字符串。 |

备注：返回true: t1在t2之后（t1大于t2），返回false: t1小于等于t2。

------

#### func [utils.Equal](time.go)

```go
func Equal(layout string, t1, t2 string) (bool, error)
```

| 参数     | 描述     |
|--------|--------|
| layout | 格式化。   |
| *t1*   | 时间字符串。 |
| t2     | 时间字符串。 |

备注：判断t1是否与t2相等。

------

### 9.9  求两个时间字符串的时间差

------

#### func [utils.Sub](time.go)

```go
func Sub(layout string, t1, t2 string) (time.Duration, error)
```

| 参数     | 描述     |
|--------|--------|
| layout | 格式化。   |
| *t1*   | 时间字符串。 |
| t2     | 时间字符串。 |

备注：t1与t2的时间差，t1>t2 结果大于0，否则结果小于等于0。

------

#### func	time.Since

```go
func Since(t Time) Duration
```

示例，程序运行时间：

```go
// 开始时间
start := time.Now()

// 业务逻辑
time.Sleep(time.Second)

// 求时间差
diff := time.Since(start)
fmt.Printf("运行时间：%v", diff) // 运行时间：1.001284412s
```

备注：求某个时间与现在的时间差。

------

## 10. 函数验证、正则验证

------

#### func [utils.Empty](regexp.go)

```go
func Empty(value string) bool
```

备注：空字符串验证。

------

#### func [utils.QQ](regexp.go)

```go
func QQ(value string) bool
```

备注：QQ号验证。

------

#### func [utils.Email](regexp.go)

```go
func Email(value string) bool
```

备注：电子邮件验证。

------

#### func [utils.Mobile](regexp.go)

```go
func Mobile(value string) bool
```

备注：中国大陆手机号码验证。

------

#### func [utils.Phone](regexp.go)

```go
func Phone(value string) bool
```

备注：中国大陆电话号码验证。

------

#### func [utils.Numeric](regexp.go)

```go
func Numeric(value string) bool
```

备注：有符号数字验证。

------

#### func [utils.UnNumeric](regexp.go)

```go
func UnNumeric(value string) bool
```

备注：无符号数字验证。

------

#### func [utils.UnInteger](regexp.go)

```go
func UnInteger(value string) bool
```

备注：无符号整数(正整数)验证。

------

#### func [utils.UnIntZero](regexp.go)

```go
func UnIntZero(value string) bool
```

备注：无符号整数(正整数+0)验证。

------

#### func [utils.Amount](regexp.go)

```go
func Amount(amount string, decimal uint8, signed ...bool) bool
```

| 参数        | 描述             |
|-----------|----------------|
| amount    | 金额字符串。         |
| *decimal* | 保留小数位长度。       |
| signed    | 带符号的金额: 默认无符号。 |

备注：金额验证。

------

#### func [utils.Alpha](regexp.go)

```go
func Alpha(value string) bool
```

备注：英文字母验证。

------

#### func [utils.Zh](regexp.go)

```go
func Zh(value string) bool
```

备注：中文字符验证。

------

#### func [utils.MixStr](regexp.go)

```go
func MixStr(value string) bool
```

备注：英文、数字、特殊字符(不包含换行符)。

------

#### func [utils.Alnum](regexp.go)

```go
func Alnum(value string) bool
```

备注：英文字母+数字验证。

------

#### func [utils.Domain](regexp.go)

```go
func Domain(value string) bool
```

备注：接受 ASCII 域名及可选的 http(s):// 前缀和末尾斜杠；顶级域为 2–6 个英文字母，不支持中文域名、端口或路径参数。

------

#### func [utils.TimeMonth](regexp.go)

```go
func TimeMonth(value string) bool
```

备注：校验年和月，年份限 1000–3999，月份可为一位或两位，分隔符支持 `-`、`/`、`.`。

------

#### func [utils.TimeDay](regexp.go)

```go
func TimeDay(value string) bool
```

备注：校验真实日期；年、月规则同 TimeMonth，日可为一位或两位，两个日期分隔符须相同。

------

#### func [utils.Timestamp](regexp.go)

```go
func Timestamp(value string) bool
```

备注：校验 TimeDay 日期、一个空格和 24 小时时分秒；时分秒可为一位或两位，以冒号分隔。

------

#### func [utils.Account](regexp.go)

```go
func Account(value string, min, max uint8) error
```

备注：帐号验证(字母开头，允许字母数字下划线，长度在min-max之间)。

------

#### func [utils.Password](regexp.go)

```go
func Password(value string, min, max uint8) error
```

备注：接受 ASCII 字母、数字和下划线，长度在 min–max 之间，不要求字母开头。

------

#### func [utils.StrongPassword](regexp.go)

```go
func StrongPassword(value string, min, max uint8) error
```

备注：强密码(必须包含大小写字母和数字的组合，不能使用特殊字符，长度在min-max之间)。

------

#### func [utils.StrongPasswordWithSymbols](regexp.go)

```go
func StrongPasswordWithSymbols(value string, min, max uint8) error
```

备注：必须同时包含 ASCII 大写、小写字母和数字，允许其他字符但不接受换行；长度按 Unicode 字符数计，范围为 min–max。

------

#### func [utils.HasSymbols](regexp.go)

```go
func HasSymbols(value string) bool
```

备注：是否包含符号。

------

#### 相关函数：[utils.Before](time.go) / [utils.After](time.go) / [utils.Equal](time.go)

备注：参考9.8 两个时间字符串判断。

------

## 11. http/curl

------

### 11.1 模拟curl请求

> 请求方式：GET、POST（form，file）、HEAD、PUT、PATCH、DELETE、OPTIONS

备注：

- `Curl` 适合作为“基础模板配置 + 按请求派生实例”使用。
- 共享基础配置时，生产环境建议使用 `Clone()` 或 `NewRequest()` 获取独立请求实例，再设置本次请求的 Header / Param / Body。
- `NewRequest()` 为新实例生成新的 `X-Request-Id`；仅共享模板已初始化的 Transport，初始化前派生的实例各自创建连接池。
- GET/POST/PUT/PATCH/DELETE/OPTIONS 追加查询参数时只拼接已经编码好的参数串，URL 合法性由发送阶段的 `http.NewRequest` 统一校验。
- 开启默认日志时，响应 body 只按 `SetLogBodyLimit` / dump 上限读取预览，并恢复 `Body` 供 `AfterResponse` / `AfterBody` 继续消费。
- 无法重放的请求体只发送一次；重试分支返回的错误仍保留首次发送原因，可用 `errors.Is` / `errors.As` 判断。
- dump 上限仅限制展示内容；Go 标准库生成出站头时仍会按 `ContentLength` 缓冲临时正文，大请求应留意这部分调试开销。

示例：

```go
baseCurl := utils.NewCurl().
	SetTimeout(10).
	SetHeader("Authorization", "Bearer xxx").
	SetUserAgent("go-utils-client/1.0")

reqCurl, err := baseCurl.NewRequest()
if err != nil {
	return err
}

err = reqCurl.
	SetParam("page", "1").
	SetBodyBytes([]byte(`{"name":"Lisa"}`)).
	AfterBody(func(body []byte) error {
		return json.Unmarshal(body, &result)
	}).
	Post("https://api.example.com/user")
```

------

#### func [(*Curl).Get](curl_client.go)

```go
func (c *Curl) Get(url string) error
```

备注：参考测试用例：[TestGet](curl_test.go)

------

#### func [(*Curl).Post](curl_client.go)

```go
func (c *Curl) Post(url string) error
```

备注：普通正文参考 [TestPost](curl_test.go)，multipart 文件上传参考 [TestPostFile](curl_test.go)。

------

#### func [(*Curl).PostForm](curl_client.go)

```go
func (c *Curl) PostForm(url string) error
```

备注：参考测试用例：[TestPostForm](curl_test.go)

------

#### func [(*Curl).Clone](curl_client.go) / [(*Curl).NewRequest](curl_client.go)

```go
func (c *Curl) Clone() (*Curl, error)
func (c *Curl) NewRequest() (*Curl, error)
```

备注：

- `Clone()` 隔离 Header、Params、Cookie、StatusCode、Body 等请求级配置；已初始化的 Transport、Cookie Jar、Logger 和回调仍共享。
- `NewRequest()` 基于 `Clone()` 派生实例并生成新的请求 ID；未初始化的连接池各自创建，代理或 TLS 配置变化时按原规则重建。
- 若当前 `Body` 是不可回放的流式 Reader，`Clone()` / `NewRequest()` 会返回错误，避免多个请求共享同一读取游标。
- 模板配置和内存请求体保持只读时可以并发派生；自定义 `io.ReadSeeker` 的访问同步与体积由调用方控制。
- 普通流式 `io.ReadCloser` 在发送结束或发送前失败时关闭；可定位的 `io.ReadSeekCloser` 由调用方负责关闭。

参考测试用例：

- [TestCurlCloneDeepCopiesRequestState](curl_test.go)
- [TestCurlNewRequestGeneratesIndependentRequestID](curl_test.go)
- [TestCurlTemplateReuseWithNewRequest](curl_test.go)

------

## 12. http/response

同一个 `Response` 只提交一次最终状态；`StatusCode(103).Write(nil)` 可先发送临时响应，再设置最终状态。1xx 除 101 外都允许后续状态，直接写正文时按 `net/http` 规则使用 200。使用 `ShowRequest` / `DownloadRequest` 传入原请求后，文件响应在最终状态未提交、且使用默认状态或显式 200 时支持 Range 和缓存条件；已有最终状态或显式非 200 时直接输出文件，HEAD 只发响应头。`Show` / `Download` 使用内部 GET 请求，不读取客户端的条件头或 HEAD 方法。

------

### 12.1 重定向

------

#### func [utils.Redirect](response.go)

```go
func Redirect(w http.ResponseWriter, url string, opts ...ResponseOption)
```

| 参数   | 描述                          |
|------|-----------------------------|
| url  | 重定向地址                       |
| opts | 响应配置项：如 WithStatusCode(301) |

```go
http.HandleFunc("/response/redirect", func(w http.ResponseWriter, r *http.Request) {
	utils.Redirect(w, "/response/json")
})
```

备注：重定向，默认响应302。

------

### 12.2 响应JSON

------

#### func [utils.JSON](response.go)

```go
// JSON 响应 JSON 数据
func JSON(w http.ResponseWriter, opts ...ResponseOption) *Response

// Success 成功响应返回 JSON 数据
func (r *Response) Success(code int, data any, message ...string)

// Fail 失败响应返回 JSON 数据
func (r *Response) Fail(code int, message string, data ...any)
```

示例：

```go
http.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {

	queryParam := r.URL.Query().Get("v")

	user := User{
		Name:      "张三",
		Age:       22,
		Sex:       "男",
		IsMarried: false,
		Address:   "北京市",
		phone:     "131188889999",
	}

	if queryParam == "fail" {
		// HTTP 状态与业务码分别设置。
		utils.JSON(w, utils.WithStatusCode(http.StatusNotAcceptable)).Fail(2000, "fail", user)
		return
	}
	utils.JSON(w).Success(1000, user)
})
```

备注：响应 JSON 数据，响应成功：JSON().Success()，响应失败：JSON().Fail()。

------

### 12.3 响应HTML

------

```go
http.HandleFunc("/response/html", func(w http.ResponseWriter, r *http.Request) {

	utils.View(w).HTML("<p>这是一个<b style=\"color: red\">段落!</b></p>")
})
```

备注：响应 HTML 文本 View().HTML()。

------

### 12.4 响应XML

------

```go
http.HandleFunc("/response/xml", func(w http.ResponseWriter, r *http.Request) {

	user := User{
		Name:      "张三",
		Age:       22,
		Sex:       "男",
		IsMarried: false,
		Address:   "北京市",
		phone:     "131188889999",
	}

	utils.View(w).XML(user)
})
```

备注：响应 XML 文本 View().XML()。

------

### 12.5 响应TEXT

------

```go
http.HandleFunc("/response/text", func(w http.ResponseWriter, r *http.Request) {
	utils.View(w).Text("<p>这是一个<b style=\"color: red\">段落!</b></p>")
})
```

备注：响应TEXT文本 View().Text()。

------

### 12.6 显示图片

------

```go
http.HandleFunc("/response/show", func(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if utils.IsExist(file) {
		utils.View(w).Show(file)
		return
	}
	utils.View(w, utils.WithStatusCode(http.StatusNotFound)).Text("不存在的文件：" + file)
})
```

备注：显示文件内容 View().Show()。

------

### 12.7 下载文件

------

```go
http.HandleFunc("/response/download", func(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if utils.IsExist(file) {
		utils.View(w).Download(file)
		return
	}
	utils.View(w, utils.WithStatusCode(http.StatusNotFound)).Text("不存在的文件：" + file)
})
```

备注：下载文件 View().Download()。

------

## 13. 打包压缩

------

### 13.1 zip

------

#### func [utils.Zip](zip.go)

```go
func Zip(zipFile string, files []string) error
```

| 参数      | 描述         |
|---------|------------|
| zipFile | 打包压缩后文件    |
| files   | 待打包压缩文件【夹】 |

备注：使用 zip 打包并压缩；只接受普通文件和目录，符号链接、命名管道及设备等条目返回错误。

------

#### func [utils.UnZip](zip.go)

```go
func UnZip(zipFile, destDir string) error
```

| 参数      | 描述     |
|---------|--------|
| zipFile | 待解压的文件 |
| destDir | 解压文件目录 |

备注：解压 zip 文件；格式判定忽略路径首尾空白，实际打开的路径保持原样，与打包端一致。默认仅允许解压到目标目录内，拒绝绝对路径、目录穿越和不支持的条目类型，并尽量恢复归档中的文件权限。

------

### 13.2 tar

------

#### func [utils.Tar](tar.go)

```go
func Tar(tarFile string, files []string) error
```

| 参数      | 描述       |
|---------|----------|
| tarFile | 打包后文件    |
| files   | 待打包文件【夹】 |

备注：使用 tar 打包；只接受普通文件和目录，符号链接、命名管道及设备等条目返回错误。

------

#### func [utils.TarGz](tar.go)

```go
func TarGz(tarGzFile string, files []string) error
```

| 参数        | 描述         |
|-----------|------------|
| tarGzFile | 打包压缩后文件    |
| files     | 待打包压缩文件【夹】 |

备注：使用 tar 打包并 gzip 压缩；输入条目限制与 Tar 相同。

------

#### func [utils.UnTar](tar.go)

```go
func UnTar(tarFile, destDir string) error
```

| 参数      | 描述     |
|---------|--------|
| tarFile | 待解压的文件 |
| destDir | 解压文件目录 |

备注：解压 tar 或 tar.gz 文件；格式判定忽略路径首尾空白，实际打开的路径保持原样。默认仅允许解压到目标目录内，拒绝绝对路径、目录穿越和不支持的条目类型，并尽量恢复归档中的文件权限。tar.gz 会校验 gzip 尾部，损坏或截断时返回错误；解包按文件写入，失败时可能已有文件落盘，调用方需要整体回滚时应使用独立临时目录。

------

## 14. 日志

------

### 14.1 默认日志（使用标准库 `log/slog`）

------

#### 设置日志等级和输出格式

```go
// 后续可修改级别，无须重建 Handler。
levelVar := &slog.LevelVar{}
levelVar.Set(slog.LevelDebug)

opts := &slog.HandlerOptions{
	AddSource: true,     // 输出日志的文件和行号
	Level:     levelVar, // 日志等级
}

file, err := os.OpenFile("sys.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
if err != nil {
	fmt.Printf("Failed to open error logger file: %v\n", err)
	return
}

// 所有使用该 handler 的写入结束后再关闭文件；此片段放在拥有其生命周期的函数中。
defer file.Close()

// 同一条 JSON 日志同时写入文件和标准错误输出。
handler := slog.NewJSONHandler(io.MultiWriter(file, os.Stderr), opts)

// 修改默认的日志输出方式
slog.SetDefault(slog.New(handler))
```

备注：设置日志级别，小于该级别的日志不会输出；若需禁用默认输出，可将 handler 输出到 io.Discard 或设置更高日志级别。

------


## 15. 杂项

------

### 15.1 环境变量

------

#### func	os.Getenv

```go
func Getenv(key string) string
```

备注：获取环境变量。

------

#### func [utils.GetEnv](env.go)

```go
func GetEnv(key string, defaultVal ...string) string
```

| 参数           | 描述           |
|--------------|--------------|
| key          | 变量名。         |
| *defaultVal* | 未获取到时默认返回的值。 |

备注：获取环境变量。

------

#### func	os.Setenv

```go
func Setenv(key, value string) error
```

备注：设置环境变量。

------

#### func	os.Unsetenv

```go
func Unsetenv(key string) error
```

备注：删除环境变量。

------


### 15.2 IP

------

#### func [utils.ServerIP](ip.go)

```go
func ServerIP() string
```

备注：优先返回缓存或本地网卡地址，可能是私网或回环地址，不保证为公网 IP。本地地址不可用时才回退到 UDP 探测；`ServerIPContext` 的取消与超时只控制 UDP 探测阶段。

------

#### func [utils.LocalIP](ip.go)

```go
func LocalIP() string
```

备注：服务器本地IP。

------

#### func [utils.ClientIP](ip.go)

```go
func ClientIP(r *http.Request) string
```

备注：获取客户端 IP。默认仅在请求来自回环地址时信任 `X-Forwarded-For` / `X-Real-IP`，否则回退到 `RemoteAddr`，避免把所有私网来源都视为
可信代理。生产环境建议优先使用 `ClientIPWithTrustedProxies` 显式配置白名单。

------

#### func [utils.NewTrustedProxies](ip.go)

```go
func NewTrustedProxies(values ...string) (*TrustedProxies, error)
```

备注：构造可信代理白名单，支持单个 IP 或 CIDR，例如 `10.0.0.10`、`10.0.0.0/8`、`fd00::/8`。IPv4 映射前缀按对应 IPv4 网段匹配，例如 `::ffff:192.0.2.0/120` 等价于 `192.0.2.0/24`；短于 96 位的 IPv6 前缀仍只匹配 IPv6。

------

#### func [utils.ClientIPWithTrustedProxies](ip.go)

```go
func ClientIPWithTrustedProxies(r *http.Request, trustedProxies *TrustedProxies) string
```

备注：使用显式可信代理白名单解析客户端 IP。生产环境建议按网关、负载均衡或 Ingress 固定地址配置可信代理；只有 `RemoteAddr`
命中白名单时才读取转发头。多个同名 `X-Forwarded-For` 按接收顺序合并，再从右向左跳过可信代理。
白名单只比较 IP，不按 IPv6 zone 区分网卡；匹配过程不移除最终 IPv6 地址中的 zone（如 `%en0`）。

------

### 15.3 三目运算

------

#### func [utils.Ternary](misce.go)

```go
func Ternary[T any](expr bool, trueVal, falseVal T) T
```

| 参数        | 描述              |
|-----------|-----------------|
| expr      | bool表达式。        |
| *trueVal* | expr为true时返回值。  |
| falseVal  | expr为false时返回值。 |

备注：类似于三目运算的函数。

### 15.4 获取当前行号、方法名、文件地址

------

#### func [utils.RuntimeInfo](runtime.go)

```go
func RuntimeInfo(skip int) *Frame
```

备注：获取当前行号、方法名、文件地址。

------

#### func [utils.GetFunctionName](runtime.go)

```go
func GetFunctionName(i any) string
```

备注：获取函数名（普通函数、结构体方法或匿名函数）。

------

### 15.5 重试机制

------

#### func [utils.Retry](misce.go)

```go
func Retry(maxRetries uint8, fn func(tries int) error) error
```

| 参数           | 描述                                 |
|--------------|------------------------------------|
| *maxRetries* | 最大尝试次数，包含首次执行。 |
| *fn*         | 要执行的函数，参数tries为当前第几次尝试，返回nil则停止重试。 |

备注：尝试执行 fn，如果 fn 返回错误则进行重试；当前退避从第一次失败后的 `100ms~200ms` 区间起步，按 2 倍指数退避并加入随机抖动，最大不超过
`3s`。
`maxRetries` 表示最大尝试次数，包含首次执行；当 `maxRetries == 0` 时会按 1 次处理。

------

### 15.6 带重试的一次性执行

------

#### struct [utils.Once](once.go)

```go
var o utils.Once

// 并发调用共享同一轮结果。
err := o.Do(func() error {
	// 失败返回错误以触发重试，成功返回 nil。
	return nil
}, 3) // 最多尝试 3 次，包含首次执行

// 等待当前轮结束，再允许下一轮执行。
o.Reset()
```

备注：零值可用，使用后不可复制。同一轮由一个调用方执行，其余调用方共享成功或最终失败的结果；Reset 等待当前轮结束后清空结果，不会中断函数。执行函数不能重入同一个 Once 的 Do、DoContext 或 Reset。

------

### 15.7 泛型对象池

------

#### struct [utils.Pool](pool.go)

```go
pool := utils.NewPool(func() *bytes.Buffer {
	return new(bytes.Buffer)
}, utils.WithPoolReset(func(b *bytes.Buffer) {
	b.Reset()
}))

// 池可丢弃闲置对象，Get 不保证取回上次归还的实例。
buf := pool.Get()

// 归还时执行重置回调，之后不再使用 buf。
pool.Put(buf)
```

备注：通过 NewPool 创建，nil 工厂使用 `new(T)`，`Put(nil)` 会被忽略。对象归还前执行重置回调，归还后不可继续使用；缓存可能随 GC 丢弃，不能用于持久保存资源。

------

### 15.8 全局配置

------

#### func [utils.Configure](config.go)

```go
func Configure(opts ...Option)
```

备注：仅第一次调用生效，包括无参数调用；多个选项应在启动时一起传入。自定义编解码器和日志实例会被共享调用，内部可变状态由实现方同步。

```go
// 编解码器和日志器在同一次 Configure 中安装。
utils.Configure(
	utils.WithJSON(customMarshal, customUnmarshal),
	utils.WithLogger(customLogger),
)
```
