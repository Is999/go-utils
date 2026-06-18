# go-utils

#### 介绍

golang 帮助函数

#### 安装教程

1. github安装 go get -u github.com/Is999/go-utils

2. gitee安装 go get -u gitee.com/Is999/go-utils

### 使用说明

1. utils 包已按“生产默认安全、历史兼容显式开启”的原则整理；落地生产前仍需结合业务协议、密钥管理、日志合规和调用方并发模型做最终评审。
2. 版本要求 Go 1.26。
3. 本版本按 Go 命名规范做破坏性改名，不保留旧名 wrapper；调用方请按下方迁移表替换。

------

### 破坏性 API 迁移表

| 旧 API | 新 API | 说明 |
| --- | --- | --- |
| `errors.Type[T]` | `errors.AsType[T]` | 对齐 Go 1.26 `errors.AsType` 泛型语义。 |
| `PassWord` | `Password` | 基础密码格式校验。 |
| `PassWord2` | `StrongPassword` | 强密码校验，不允许特殊符号。 |
| `PassWord3` | `StrongPasswordWithSymbols` | 强密码校验，允许特殊符号。 |
| `UniqID` / `UniqId` | `UniqueID` | 使用完整单词，保留 ID initialism。 |
| `SecureUniqID` | `SecureUniqueID` | 密码学安全随机 ID。 |
| `RandStr` | `RandomLetters` | 使用英文字母字符集。 |
| `RandStr2` | `RandomID` | 首字符为字母，后续为字母数字。 |
| `RandStr3` | `RandomString` | 自定义字符集随机字符串。 |
| `SecureRandStr*` | `SecureRandomLetters` / `SecureRandomID` / `SecureRandomString` | 密码学安全随机字符串。 |
| `McryptMode` | `CipherMode` | 加密模式类型。 |
| `Padding` / `UnPadding` | `Pad` / `Unpad` | 填充和去填充函数类型。 |
| `NoPadding` / `NoUnPadding` | `NoPad` / `NoUnpad` | 不填充策略。 |
| `Pkcs7Padding` / `Pkcs7UnPadding` | `PKCS7Pad` / `PKCS7Unpad` | Go initialism 规范。 |
| `ZeroPadding` / `ZeroUnPadding` | `ZeroPad` / `ZeroUnpad` | 零填充策略。 |
| `DES3` | `TripleDES` | 语义更明确。 |
| `UrlPath` | `URLPath` | URL initialism 规范；旧名已删除。 |
| `*RequestId` | `*RequestID` | ID initialism 规范；旧名已删除。 |
| `ReSetHeader` / `ReSetParams` | `ResetHeader` / `ResetParams` | 动词拼写规范。 |
| `DelHeaders` / `DelParams` / `DelCookies` / `DelFiles` | `DeleteHeaders` / `DeleteParams` / `DeleteCookies` / `DeleteFiles` | 动词完整清晰。 |
| `Md5` / `Sha1` / `Sha256` / `Sha512` | `MD5` / `SHA1` / `SHA256` / `SHA512` | 摘要算法 initialism 规范。 |
| `Str2Int` / `Str2Int64` / `Str2Float` | `ToInt` / `ToInt64` / `ToFloat64` | 简洁转换命名。 |
| `StrRev` | `ReverseString` | 语义明确。 |
| `Strtotime` | `ParseTime` | 与标准库 parse 语境一致。 |

### 模块结论

本版本保留官方标准库快捷概览，并在工具函数侧按安全性、并发性、性能边界和文档一致性整理。核心结论如下：

| 模块                    | 生产级结论                                                                                                                       |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------|
| AES / DES             | AES 推荐 `GCM`；`CBC` 必须显式 `WithIV(...)` 或 `WithRandIV(true)`；`CTR/CFB/OFB`、`ECB`、默认 key 派生 IV 默认禁用。DES/3DES 仅保留历史兼容，不建议新系统使用。 |
| RSA                   | 生成和导入密钥均要求至少 2048 位；推荐 `EncryptOAEP/DecryptOAEP` 与 `SignPSS/VerifyPSS`；`PKCS#1 v1.5` 仅用于旧协议兼容；签名拒绝 MD5/SHA1。                |
| PKCS7 / Zero          | `PKCS7Unpad` 严格校验填充字节；`ZeroPad` 仅适合能接受尾部 0 歧义的旧协议。                                                                  |
| Pool / Once           | `Pool[T]` 基于 `sync.Pool`，支持归还前 reset；`Once` 并发安全，失败按有限指数退避重试，结果会缓存，需重新执行时调用 `Reset`。                                        |
| Retry                 | `Retry` 使用 capped exponential backoff + jitter：第一次失败后约 100ms~200ms，最多 3s；`maxRetries=0` 时只执行 1 次。                           |
| IP / Logger           | `ClientIP` 仅在远端为可信代理时读取转发头；生产建议使用 `ClientIPWithTrustedProxies`。默认 logger 基于 `log/slog` 并发安全，自定义 logger 也应保证并发安全。            |
| slices / string / map | 常用函数无共享可变全局状态；传入的 slice/map 若被外部并发修改，调用方需自行加锁。循环热路径优先使用 `UniqueInto/DiffInto/IntersectInto` 和 `NewReplacer` 复用分配。              |
| tar / zip             | 解压防 Zip Slip/Tar Slip，拒绝绝对路径、目录穿越、空字符、符号链接和特殊文件；限制条目数、单文件大小和总展开大小。                                                          |
| errors                | 支持栈追踪、错误码、上下文键值、`errors.Join`、文本/JSON/slog 输出；全局配置使用原子变量并发安全。                                                               |
| curl                  | 继承标准库默认 Transport 连接池/TLS 行为；Header 按请求克隆；可回放 body 才会重试；默认日志 body 为预览上限并读取后恢复；支持 `Clone/NewRequest` 派生独立请求实例并复用连接池。         |
| response              | 状态码只写一次；Content-Type 自动规范化；下载文件名清洗 CR/LF 和路径；文件输出拒绝目录。                                                                      |

### 全局配置 Configure

`Configure` 是全局配置入口，只需在程序入口处（如 `main` 函数）调用一次。支持自定义 JSON 编解码器和自定义 Logger。

```go
import utils "github.com/Is999/go-utils"

func main() {
	utils.Configure(
		utils.WithJSON(json.Marshal, json.Unmarshal), // 可选：自定义 JSON 编解码器
		utils.WithLogger(yourLogger),                // 可选：自定义 Logger
	)
}
```

#### JSON 编解码配置

通过 `WithJSON` 设置自定义的 JSON
编解码方法（如 [sonic](https://github.com/bytedance/sonic)、[go-json](https://github.com/goccy/go-json)
等高性能库），若未设置则默认使用标准库 `encoding/json`。

```go
import "github.com/bytedance/sonic"

utils.Configure(utils.WithJSON(sonic.Marshal, sonic.Unmarshal))
```

设置后，使用 `utils.Marshal()` 和 `utils.Unmarshal()` 即会调用自定义的编解码器。

#### Logger 配置

通过 `WithLogger` 设置自定义 Logger，若未设置则默认使用标准库 `log/slog`。

Logger 接口定义如下，实现 6 个方法即可集成任何第三方日志库（如 zap、logrus 等）：

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
"go.uber.org/zap"
"go.uber.org/zap/zapcore"
utils "github.com/Is999/go-utils"
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
zapLogger, _ := zap.NewProduction()
utils.Configure(utils.WithLogger(&zapLoggerAdapter{logger: zapLogger.Sugar()}))
}
```

##### 集成 Logrus 示例

```go
import (
"context"
"github.com/sirupsen/logrus"
utils "github.com/Is999/go-utils"
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

```go
import "github.com/Is999/go-utils/errors"

err := errors.Wrap(originalErr, "操作失败")

// 方式1: 使用 slog（默认日志库），Trace 返回 slog.LogValuer 接口
slog.Error(err.Error(), "trace", errors.Trace(err))

// 方式2: 使用第三方日志库（zap/logrus 等），TraceString 返回简洁字符串
logger.Error("操作失败", "error", err.Error(), "trace", errors.TraceString(err))

// 方式3: 获取完整 JSON 格式追踪（含嵌套 wrap 信息）
logger.Error("操作失败", "error", err.Error(), "trace", errors.TraceJSON(err))

// 方式4: 使用 fmt 格式化
fmt.Sprintf("%+v", err) // 等同于 TraceJSON
fmt.Sprintf("%#v", err) // 等同于 TraceJSON
```

------

### 历史变更

1. 版本要求 Go 1.26
2. 移除了1.21 版本前的Max、Min 两个函数，推荐使用golang 内置函数 max、min
3. 移除了1.21 版本前的Logger 文件，使用标准库中 log/slog
4. Curl 和 Response 记录日志方式使用了标准库 log/slog记录日志
5. 根据1.21版本 log/slog 增加了errors文件，实现了LogValuer 接口，对error的日志追踪
6. utils中返回的error 统一使用了WrapError, 支持error记录追踪
7. RSA加密解密增加了对长文本的支持，增加了对PEM key 去除头尾标记和还原头尾标记方法
8. math/rand 改1.22版本 math/rand/v2, 部分函数形参 rand.Source 改为*rand.Rand
9. ParseTime函数增强，支持更多时间格式自动解析（毫秒、微秒、纳秒格式）
10. Response模块使用 Header 方法（修复 Herder 拼写错误，当前仅保留 Header 方法）
11. 修复types.go中"有符合"拼写错误，更正为"有符号"
12. 新增 Retry 函数，支持带指数退避的重试机制
13. 新增 Once 结构体，线程安全的带重试机制的一次性执行
14. 新增泛型对象池 Pool[T]，基于 sync.Pool 封装，支持自定义重置函数
15. 新增 SumSlice 切片求和、SumMap Map值求和函数
16. 新增 Configure 全局配置入口，支持自定义 JSON 编解码器和 Logger
17. 新增 GetFunctionName 获取函数名函数
18. 修复 IsFile 对不存在路径错误返回 true 的bug
19. 修复 tar.go/zip.go 中 strings.TrimRight 误用为 strings.TrimSuffix
20. 修复 cipher.go 中 Decrypt 注释错误（"加密"更正为"解密"）
21. 修复 response.go 中 Content-Disposition filename 未加引号导致含空格文件名异常
22. 修复 file.go 中路径分隔符硬编码问题，改用 filepath.Separator
23. 优化 rsa.go 中 strings.Index 为更符合语义的 strings.Contains，strings.Replace 为 strings.ReplaceAll

# Go常用标准库方法及utils包帮助函数

>
开发中使用频率较高的Go标准库中的方法及utils包中的帮助方法。utils包中的方法都可以在单元测试中找到使用方法示例。版本要求 >=
1.26版本。

> 注意：utils包中代码仅供参考，不建议用于商业生产，造成损失概不负责。

## 1. 字符串

​        **strings** 和 **bytes** 两个包对字符串的操作基本相同拥有**相同的方法名称和参数**，只是参数类型的不同。

------

### 1.1 截取字符串

> **推荐**：包含中文（宽字符）时，使用 string转rune切片截取字符串。

------

#### type	string

```go
string[start: end]
```

| 参数       | 描述                                                   |
|----------|------------------------------------------------------|
| *string* | 原字符串。                                                |
| *start*  | 表示要截取的第一个字符所在的索引（截取时包含该字符）。如果不指定，默认为 0，也就是从字符串的开头截取。 |
| *end*    | 表示要截取的最后一个字符所在的索引（截取时不包含该字符）。如果不指定，默认为字符串的长度。        |

------

#### func [utils.Substr](https://github.com/Is999/go-utils/blob/master/string.go#L116)

```go
func Substr(str string, start, length int) string
```

| 参数       | 描述                                                                                                                                                                                                                                    |
|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| *str*    | 原字符串。                                                                                                                                                                                                                                 |
| *start*  | 截取的起始位置，即截取的第一个字符所在的索引：<br />start小于0时，start = len(str) + start                                                                                                                                                                       |
| *length* | 截取的截止位置，即截取的最后一个字符所在的索引：<br />length大于0时，length表示为截取子字符串的**长度**，截取的最后一个字符所在的索引，值为：**start + length** 。<br />length小于0时，length表示为截取的最后一个字符所在的**索引**，值为：**len(str) + length + 1** 。例如：等于 **-1** 时，表示截取到最后一个字符；等于 **-2** 时，表示截取到倒数第二个字符。 |

备注：Substr 内部实现string转rune切片。

```go
// string转rune切片
runes := []rune(str)
string(runes[start: end])
```

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
s3 := b.String() // 拼接后的字符串
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

> **推荐**：包含宽字符（宽字符算一个长度）时，使用utf8.RuneCount、 utf8.RuneCountInString获取长度。

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

备注：按**空格分割**字符串

------

#### func	strings.FieldsFunc

```go
func FieldsFunc(s string, f func (rune) bool) []string 
```

备注：按**字符分割**字符串

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
// 按空格(空字符串)分割
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

#### func	stringsLastIndex

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

备注：返回**第一次出现字符序列的索引**；反之，则返回 **-1**。

------

#### func	strings.LastIndexAny

```go
func LastIndexAny(s, chars string) int
```

| 参数      | 描述        |
|---------|-----------|
| *s*     | 原字符串。     |
| *chars* | 要检索的字符序列。 |

备注：返回**最后一次出现字符序列的索引**；反之，则返回 **-1**。

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
| *r* | 表示要检索的字符。 |

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

#### func	strings.LastIndexRune

```go
func LastIndexRune(s string, r rune) int
```

| 参数  | 描述        |
|-----|-----------|
| *s* | 原字符串。     |
| *c* | 表示要检索的字符。 |

备注：返回**最后一次出现字符的索引**；反之，则返回 **-1**。

------

#### func	strings.IndexFunc

```go
func IndexFunc(s string, f func (rune) bool) int
```

| 参数  | 描述               |
|-----|------------------|
| *s* | 原字符串。            |
| *f* | 表示要检索的字符的条件判断函数。 |

备注：返回**第一次出现字符的索引**；反之，则返回 **-1**。

------

#### func	strings.LastIndexFunc

```go
func LastIndexFunc(s string, f func (rune) bool) int
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

备注：检索**字符串s**是否包含**字符串substr**。函数内部实现 **strings.Index >= 0** 。

------

#### func	strings.ContainsRune

```go
func ContainsRune(s string, r rune) bool
```

| 参数  | 描述        |
|-----|-----------|
| *s* | 原字符串。     |
| *r* | 表示要检索的字符。 |

备注：检索**字符串s**是否包含**字符r**。函数内部实现 **strings.IndexRune >= 0** 。

------

#### func	strings.ContainsAny

```go
func ContainsAny(s, chars string) bool
```

| 参数      | 描述         |
|---------|------------|
| *s*     | 原字符串。      |
| *chars* | 表示要检索的字符串。 |

备注：检索**字符串s**是否包含**字符串chars**。函数内部实现 **strings.IndexAny >= 0** 。

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

备注：将字符串**首字母转成大写**。

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

备注：将**字符串左右两边的空格去除**。

------

#### func	strings.Trim

```go
func Trim(s string, cutset string) string
```

| 参数       | 描述        |
|----------|-----------|
| *s*      | 原字符串。     |
| *cutset* | 需要去除的字符串。 |

备注：将**字符串左右两边的指定字符串 cutset 去除**。

------

#### func	strings.TrimLeft

```go
func TrimLeft(s, cutset string) string
```

| 参数       | 描述        |
|----------|-----------|
| *s*      | 原字符串。     |
| *cutset* | 需要去除的字符串。 |

备注：将**字符串左边的指定字符串 cutset 去除**。

------

#### func	strings.TrimRight

```go
func TrimRight(s, cutset string) string
```

| 参数       | 描述        |
|----------|-----------|
| *s*      | 原字符串。     |
| *cutset* | 需要去除的字符串。 |

备注：将**字符串右边的指定字符串 cutset 去除**。

------

#### func	strings.TrimPrefix

```go
TrimPrefix(s, prefix string) string
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
func TrimFunc(s string, f func (rune) bool) string 
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
func TrimLeftFunc(s string, f func (rune) bool) string
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
func TrimRightFunc(s string, f func (rune) bool) string
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
func Map(mapping func (rune) rune, s string) string
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

#### func [utils.Replace](https://github.com/Is999/go-utils/blob/master/string.go#L99)

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

#### func [utils.ReverseString](https://github.com/Is999/go-utils/blob/master/string.go#L197)

```go
func ReverseString(str string) string
```

备注：将字符串 str 反转。

------

### 1.16 生成随机字符串

------

#### func [utils.UniqueID](https://github.com/Is999/go-utils/blob/master/string.go#L303)

```go
func UniqueID(l uint8, r ...*rand.Rand) string
```

| 参数  | 描述                                          |
|-----|---------------------------------------------|
| *l* | 生成字符串的长度。                                   |
| *r* | 随机种子 utils.RandSource：批量生成时传入r参数可提升生成随机数效率。 |

备注：生成一个长度范围16-32位的唯一ID字符串(可排序的字符串)。UniqueID 生成字符串并不保证全局强唯一性；该函数基于 `math/rand`，仅适用于非安全场景；token、验证码、重置链接等安全用途请使用 `SecureUniqueID`。

------

#### func [utils.RandomLetters](https://github.com/Is999/go-utils/blob/master/string.go#L207)

```go
func RandomLetters(n int, r ...*rand.Rand) string
```

| 参数  | 描述                                          |
|-----|---------------------------------------------|
| *n* | 生成字符串的长度。                                   |
| *r* | 随机种子 utils.RandSource：批量生成时传入r参数可提升生成随机数效率。 |

备注：随机生成字符串 ALPHA。ALPHA 值为：A-Za-z。该函数基于 `math/rand`，仅适用于测试数据、临时标识等非安全场景；安全用途请使用
`SecureRandomLetters`。

------

#### func [utils.RandomID](https://github.com/Is999/go-utils/blob/master/string.go#L218)

```go
func RandomID(n int, r ...*rand.Rand) string
```

| 参数  | 描述                                          |
|-----|---------------------------------------------|
| *n* | 生成字符串的长度。                                   |
| *r* | 随机种子 utils.RandSource：批量生成时传入r参数可提升生成随机数效率。 |

备注：随机生成字符串 ALNUM。ALNUM 值为：A-Za-z0-9；为兼容旧行为，首字符固定为字母，不会以数字开头。该函数基于 `math/rand`，
仅适用于非安全场景；安全用途请使用 `SecureRandomID`。

------

#### func [utils.RandomString](https://github.com/Is999/go-utils/blob/master/string.go#L247)

```go
func RandomString(n int, alpha string, r ...*rand.Rand) string
```

| 参数      | 描述                                          |
|---------|---------------------------------------------|
| *n*     | 生成字符串的长度。                                   |
| *alpha* | 生成随机字符串的种子。                                 |
| *r*     | 随机种子 utils.RandSource：批量生成时传入r参数可提升生成随机数效率。 |

备注：随机生成字符串。alpha 指定生成随机字符串的种子。该函数基于 `math/rand`，仅适用于非安全场景；安全用途请使用
`SecureRandomString`。

------

### 1.17 字符串 Read

------

#### struct	Reader

```go
NewReader(s string).Read((b []byte)
```

备注：实现了read接口。

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

备注：将实体字符转换为可编译的html字符。

------

#### func	url.QueryEscape

```go
func QueryEscape(s string) string
```

备注：将URL中的字符进行转义。

------

#### func	url.QueryUnescape

```go
func QueryUnescape(s string) (string, error)
```

备注：将URL中的转义字符转换为对应的字符。

------

#### func [utils.URLPath](https://github.com/Is999/go-utils/blob/master/url.go#L15)

```go
func URLPath(urlPath string, params url.Values) (string, error)
```

| 参数      | 描述         |
|---------|------------|
| urlPath | 基础 URL 路径。 |
| params  | 查询参数键值对。   |

备注：将 params 合并到 urlPath 的查询字符串中，并保留原有查询参数。

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

#### func [utils.Rand](https://github.com/Is999/go-utils/blob/master/math.go#L13)

```go
func Rand(minInt, maxInt int64, r ...*rand.Rand) int64
```

| 参数       | 描述                                          |
|----------|---------------------------------------------|
| *minInt* | 最小值。                                        |
| *maxInt* | 最大值。                                        |
| *r*      | 随机种子 utils.RandSource：批量生成时传入r参数可提升生成随机数效率。 |

备注：返回 minInt~maxInt 之间的随机数，值可能包含 minInt 和 maxInt；当 minInt 大于 maxInt 时会自动交换。

------

#### func [utils.Round](https://github.com/Is999/go-utils/blob/master/math.go#L54)

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

备注：打开名为name的文件。

------

#### func	os.Create

```go
func Create(name string) (*File, error)
```

备注：创建名为name的文件。

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

备注：取得当前工作目录的根路径。

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

备注：返回相path的最短路径名。

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

#### func [utils.IsDir](https://github.com/Is999/go-utils/blob/master/file.go#L23)

```go
func IsDir(path string) bool
```

备注：判断给定路径是否是一个目录。

------

#### func [utils.IsFile](https://github.com/Is999/go-utils/blob/master/file.go#L32)

```go
func IsFile(filepath string) bool
```

备注：判断给定的文件路径名是否是一个文件。

------

#### func [utils.IsExist](https://github.com/Is999/go-utils/blob/master/file.go#L41)

```go
func IsExist(path string) bool
```

备注：判断一个文件（夹）是否存在。

------

### 4.11 获取文件大小

------

#### func [utils.Size](https://github.com/Is999/go-utils/blob/master/file.go#L47)

```go
func Size(filepath string) (int64, error) 
```

备注：取得文件大小。

------

#### func [utils.SizeFormat](https://github.com/Is999/go-utils/blob/master/file.go#L584)

```go
func SizeFormat(size int64, decimals uint) string 
```

| 参数         | 描述            |
|------------|---------------|
| *size*     | 文件实际大小(Byte)。 |
| *decimals* | 保留几位小数。       |

备注：文件大小格式化已可读式显示文件大小。

------

### 4.12 复制文件

------

#### func [utils.Copy](https://github.com/Is999/go-utils/blob/master/file.go#L59)

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

#### func [utils.FindFiles](https://github.com/Is999/go-utils/blob/master/file.go#L175)

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

#### func [utils.Scan](https://github.com/Is999/go-utils/blob/master/file.go#L313)

```go
func Scan(r io.Reader, handle ReadScan, size ...int) error
```

| 参数       | 描述                                                                                                                                                                    |
|----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| *r*      | 实现io.Reader接口。                                                                                                                                                        |
| *handle* | func(num int, line []byte, err error) error 函数。<br /> num 行号: 当前扫描到第几行<br /> line 行数据: 当前扫描的行数据<br /> err 扫描错误信息<br /> error 处理错误信息: 返回的 error == DONE 代表正确处理完数据并终止扫描 |
| *size*   | 设置Scanner.maxTokenSize 的大小(默认值: 64*1024): 单行内容大于该值则无法读取                                                                                                               |

备注：使用scan扫描文件每一行数据。

------

#### func [utils.Line](https://github.com/Is999/go-utils/blob/master/file.go#L339)

```go
func Line(r io.Reader, handle ReadLine) error
```

| 参数       | 描述                                                                                                                                                                                                                                 |
|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| *r*      | 实现io.Reader接口。                                                                                                                                                                                                                     |
| *handle* | func(num int, line []byte, lineDone bool) error 函数。<br /> num 行号: 当前扫描到第几行<br /> line 行数据: 当前扫描的行数据<br /> lineDone 当前行(num)数据是否读取完毕: true 当前行(num)数据读取完毕; false 当前行(num)数据未读完<br /> error 处理错误信息: 返回的 error == DONE 代表正确处理完数据并终止扫描 |

备注：读取一行数据，读取大文件大行数据性能略优于Scan。

------

#### func [utils.Read](https://github.com/Is999/go-utils/blob/master/file.go#L369)

```go
func Read(r io.Reader, handle ReadBlock) error
```

| 参数       | 描述                                                                                                                                 |
|----------|------------------------------------------------------------------------------------------------------------------------------------|
| *r*      | 实现io.Reader接口。                                                                                                                     |
| *handle* | func(size int, block []byte) error 函数。<br /> size 读取的数据块大小<br /> block 读取的数据块<br /> error 处理错误信息: 返回的 error == DONE 代表正确处理完数据并终止扫描 |

备注：使用分块读取文件数据，适用于读取大文件或无换行的文件。

------

### 4.15 写入内容到文件

------

#### func [utils.NewWrite](https://github.com/Is999/go-utils/blob/master/file.go#L439)

```go
func NewWrite(fileName string, opts ...WriteOption) (*WriteFile, error)
```

| 参数         | 描述                                                            |
|------------|---------------------------------------------------------------|
| *fileName* | 文件路径名。                                                        |
| *opts*     | 写入配置项: WithWriteAppend(true) 追加写入; WithWritePerm(0644) 设置文件权限 |

备注：返回一个WriteFile实例。

补充说明：

- `NewWrite` 适合日志、顺序输出等持续写入场景。
- 若是配置文件、密钥文件、状态文件等“整文件覆盖”场景，生产环境更建议使用 `WriteFileAtomic` / `WriteStringAtomic`，避免直接
  `O_TRUNC` 导致半写文件。
- `NewWrite` 会拒绝目标文件为符号链接，降低通过通用写入口写穿到其它路径的风险。

```go
// 实例化一个 WriteFile（追加写入）
w, err := utils.NewWrite(fileName, utils.WithWriteAppend(true))
if err != nil {
	fmt.Errorf("NewWrite() error = %v", err)
	return
}

// 关闭文件
defer func() {
	if err := w.Close(); err != nil {
		fmt.Errorf("Close() err %v", err)
	}
}()
```

------

#### func [utils.WriteFileAtomic](https://github.com/Is999/go-utils/blob/master/file.go#L420) / [utils.WriteStringAtomic](https://github.com/Is999/go-utils/blob/master/file.go#L431)

```go
func WriteFileAtomic(fileName string, data []byte, perm os.FileMode) error
func WriteStringAtomic(fileName, data string, perm os.FileMode) error
```

备注：使用同目录临时文件 + `Sync` + `Close` + `Rename` 原子覆盖目标文件，适用于配置、密钥、状态等要求“覆盖即完整替换”的场景。

```go
err := utils.WriteFileAtomic(fileName, []byte("hello world"), 0644)
if err != nil {
	fmt.Errorf("WriteFileAtomic() err %v", err)
	return
}

err = utils.WriteStringAtomic(fileName, "hello world", 0644)
if err != nil {
	fmt.Errorf("WriteStringAtomic() err %v", err)
	return
}
```

备注：适合一次性覆盖完整文件内容；若需要持续写入、追加写入或 `bufio.Writer` 批量写入，继续使用 `NewWrite`。

------

### 4.16 获取文件类型

------

#### func [utils.FileType](https://github.com/Is999/go-utils/blob/master/file.go#L604)

```go
func FileType(f *os.File) (string, error)
```

备注：获取文件类型

------

## 5. 加密、解密与摘要

------

### 5.1 MD5 摘要

------

#### func [utils.MD5](https://github.com/Is999/go-utils/blob/master/md5.go#L12)

```go
func MD5(str string) string
```

备注：返回 MD5 十六进制摘要。MD5 已不适合密码存储、签名、完整性安全校验等安全场景，仅用于历史协议兼容或非安全哈希标识。

------

### 5.2 SHA 摘要

------

#### func [utils.SHA1](https://github.com/Is999/go-utils/blob/master/sha.go#L14)

```go
func SHA1(str string) string
```

备注：返回 SHA-1 十六进制摘要。SHA-1 已不适合签名、证书、密码存储等安全场景，仅用于历史协议兼容或非安全哈希标识。

------

#### func [utils.SHA256](https://github.com/Is999/go-utils/blob/master/sha.go#L21)

```go
func SHA256(str string) string
```

备注：返回 SHA-256 十六进制摘要。普通摘要场景可使用；密码存储仍应使用 bcrypt/argon2/scrypt 等专用算法。

------

#### func [utils.SHA512](https://github.com/Is999/go-utils/blob/master/sha.go#L28)

```go
func SHA512(str string) string
```

备注：返回 SHA-512 十六进制摘要。普通摘要场景可使用；密码存储仍应使用 bcrypt/argon2/scrypt 等专用算法。

------

### 5.3 RSA 非对称加密与解密

------

#### func [utils.GenerateKeyRSA](https://github.com/Is999/go-utils/blob/master/rsa.go#L413)

```go
func GenerateKeyRSA(path string, bits int, pkcs ...bool) ([]string, error)
```

| 参数     | 描述                                                                                                                                      |
|--------|-----------------------------------------------------------------------------------------------------------------------------------------|
| *path* | 文件名路径。                                                                                                                                  |
| *bits* | 生成秘钥位大小，生产环境要求至少 2048。                                                                                                                  |
| *pkcs* | 秘钥格式, 默认格式(公钥PKCS8格式 私钥PKCS1格式):<br />   - pkcs[0] isPubPKCS8 公钥是否是PKCS8格式: 默认 true <br />   - pkcs[1] isPriPKCS1 私钥是否是PKCS1格式: 默认 true |

备注：生成秘钥，默认格式(公钥PKCS8格式 私钥PKCS1格式)。返回两个文件名, 第一个公钥文件名, 第二个私钥文件名；为保证生产安全，密钥位数不能低于
2048。

------

#### RSA 加密与解密：[utils.NewRSA](https://github.com/Is999/go-utils/blob/master/rsa.go#L54) / [(*RSA).Encrypt](https://github.com/Is999/go-utils/blob/master/rsa.go#L156) / [(*RSA).Decrypt](https://github.com/Is999/go-utils/blob/master/rsa.go#L176) / [(*RSA).EncryptOAEP](https://github.com/Is999/go-utils/blob/master/rsa.go#L256) / [(*RSA).DecryptOAEP](https://github.com/Is999/go-utils/blob/master/rsa.go#L282)

```go
// 实例化RSA，并设置key
r, err := NewRSA(publicKey, privateKey, WithRSAFilePath(true))
if err != nil {
fmt.Errorf("NewRSA() err = %v", err)
return
}

// 源数据
marshal, err := json.Marshal(map[string]interface{}{
"Title":   tt.name,
"Content": strings.Repeat("测试内容8282@334&-", 1024) + tt.name,
})

// 公钥加密 PKCS1v15
encodeString, err := r.Encrypt(string(marshal), base64.StdEncoding.EncodeToString)
if err != nil {
fmt.Errorf("Encrypt() err = %v", err)
return
}

// 私钥解密 PKCS1v15
decryptString, err := r.Decrypt(encodeString, base64.StdEncoding.DecodeString)
if err != nil {
fmt.Errorf("Decrypt() err = %v", err)
return
}

// 公钥加密 OAEP
encodeString, err = r.EncryptOAEP(string(marshal), base64.StdEncoding.EncodeToString, sha256.New())
if err != nil {
fmt.Errorf("Encrypt() err = %v", err)
return
}

// 私钥解密 OAEP
decryptString, err = r.DecryptOAEP(encodeString, base64.StdEncoding.DecodeString, sha256.New())
if err != nil {
fmt.Errorf("Decrypt() err = %v", err)
return
}
```

备注：先实例化RSA 设置公钥私钥，使用公钥加密数据， 私钥解密数据。

------

#### RSA 签名与验签：[(*RSA).Sign](https://github.com/Is999/go-utils/blob/master/rsa.go#L210) / [(*RSA).Verify](https://github.com/Is999/go-utils/blob/master/rsa.go#L232) / [(*RSA).SignPSS](https://github.com/Is999/go-utils/blob/master/rsa.go#L366) / [(*RSA).VerifyPSS](https://github.com/Is999/go-utils/blob/master/rsa.go#L388)

```go
// 实例化RSA，并设置key
r, err := NewRSA(publicKey, privateKey, WithRSAFilePath(true))
if err != nil {
fmt.Errorf("NewRSA() err = %v", err)
return
}

// 源数据
marshal, err := json.Marshal(map[string]interface{}{
"Title":   tt.name,
"Content": strings.Repeat("测试内容8282@334&-", 1024) + tt.name,
})

// 私钥签名 PKCS1v15
sign, err := r.Sign(string(marshal), crypto.SHA256, base64.StdEncoding.EncodeToString)
if err != nil {
fmt.Errorf("Sign() err = %v", err)
return
}

// 公钥验签 PKCS1v15
if err := r.Verify(string(marshal), sign, crypto.SHA256, base64.StdEncoding.DecodeString); err != nil {
fmt.Errorf("Verify() err = %v", err)
return
} else {
fmt.Log("Verify() = 验证成功")
}

// 私钥签名 PSS
sign, err = privRsa.SignPSS(string(marshal), crypto.SHA256, base64.StdEncoding.EncodeToString, nil)
if err != nil {
fmt.Errorf("Sign() err = %v", err)
return
}

// 公钥验签 PSS
if err := pubRsa.VerifyPSS(string(marshal), sign, crypto.SHA256, base64.StdEncoding.DecodeString, nil); err != nil {
fmt.Errorf("Verify() err = %v", err)
return
} else {
fmt.Log("Verify() = 验证成功")
}
```

备注：先实例化RSA 设置公钥私钥，使用私钥签名，公钥验签。

------

#### 加密模式选型建议

| 场景         | 推荐方案                                            | 说明                                     |
|------------|-------------------------------------------------|----------------------------------------|
| 新系统对称加密    | `AES + GCM`                                     | 默认优先方案，带机密性和完整性校验，支持 `additionalData`。 |
| 兼容分组协议     | `AES/DES + CBC + WithIV/WithRandIV`             | 需要补位，适合和旧系统的固定协议对接。                    |
| 旧系统流模式兼容   | `CTR/CFB/OFB + WithAllowUnsafeStreamMode(true)` | 默认禁用；无需块补位，但不提供完整性校验，仅用于兼容历史协议。        |
| 旧系统 ECB 兼容 | `ECB + WithAllowUnsafeECB(true)`                | 默认禁用，不建议新系统使用。                         |
| RSA 加密     | `EncryptOAEP/DecryptOAEP`                       | 新协议优先，适合加密小数据或对称密钥。                    |
| RSA 签名     | `SignPSS/VerifyPSS`                             | 新协议优先，推荐配合 `SHA256` 或更强摘要。             |

备注：

- RSA 适合加密小块数据或密钥封装，不建议直接承载大体积正文。
- `PKCS#1 v1.5` 加解密与签名接口主要保留给旧系统兼容；新系统优先使用 OAEP/PSS。
- 生产环境推荐 `2048` 位及以上 RSA 密钥，摘要算法推荐 `SHA256` / `SHA384` / `SHA512`。

------

#### RSA 密钥格式转换：[utils.RemovePEMHeaders](https://github.com/Is999/go-utils/blob/master/rsa.go#L495) / [utils.AddPEMHeaders](https://github.com/Is999/go-utils/blob/master/rsa.go#L512)

```go
// 读取公钥文件内容
pub, err := os.ReadFile(pubFile)
if err != nil {
t.Errorf("ReadFile() WrapError = %v", err)
}

//fmt.Println("公钥 %s", string(pub))
rPub := utils.RemovePEMHeaders(string(pub))
//fmt.Println("remove 公钥 %s", rPub)
aPub := utils.AddPEMHeaders(rPub, "public")
//fmt.Println("add 公钥 %s %v", aPub, strings.EqualFold(aPub, strings.TrimSpace(string(pub))))
if !strings.EqualFold(aPub, strings.TrimSpace(string(pub))) {
fmt.Errorf("转换后的公钥与原始公钥不相等")
}

// 读取私钥文件内容
pri, err := os.ReadFile(priFile)
if err != nil {
t.Errorf("ReadFile() WrapError = %v", err)
}
//fmt.Println("私钥 %s", string(pri))
rPri := utils.RemovePEMHeaders(string(pri))
//fmt.Println("remove 私钥 %s", rPri)
aPri := utils.AddPEMHeaders(rPri, "private")
//fmt.Println("add 私钥 %s %v", aPri, strings.EqualFold(aPri, strings.TrimSpace(string(pri))))
if !strings.EqualFold(aPri, strings.TrimSpace(string(pri))) {
fmt.Errorf("转换后的私钥与原始私钥不相等")
}
```

------

### 5.4 AES 加密与解密

------

#### AES [加密与解密](https://github.com/Is999/go-utils/blob/master/aes.go#L12)

```go
// 实例化AES，并设置key和iv
a, err := AES(key, WithIV(iv))
if err != nil {
fmt.Errorf("AES() error = %v", err)
return
}

// 加密数据
encryptStr, err := a.Encrypt(data, CBC, base64.StdEncoding.EncodeToString, PKCS7Pad)
if err != nil {
fmt.Errorf("Encrypt() mode = %v error = %v", CBC, err)
return
}

// 解密数据
got, err := a.Decrypt(encryptStr, CBC, base64.StdEncoding.DecodeString, PKCS7Unpad)
if err != nil {
fmt.Errorf("Decrypt() mode = %v error = %v", CBC, err)
return
}
```

备注：先实例化 AES 并设置 key；CBC/CTR/CFB/OFB 等需要显式通过 `WithIV` 设置固定 IV，或通过 `WithRandIV(true)` 使用随机 IV。
`CTR/CFB/OFB` 属于非认证流模式，默认禁用，仅在兼容旧系统时通过 `WithAllowUnsafeStreamMode(true)` 显式开启。`ECB`
默认禁用，仅在兼容旧系统时通过 `WithAllowUnsafeECB(true)` 显式开启；历史上“未设置 IV 时使用 key 派生
IV”的兼容行为也默认禁用，如需兼容旧密文可显式开启 `WithAllowUnsafeKeyIV(true)`。

补充：`CTR/CFB/OFB` 若用于旧协议兼容且无需补位，建议配合 `NoPad` / `NoUnpad` 使用，避免多余填充和块长度约束。

------

### 5.5 DES 加密与解密

------

#### DES 加密与解密：[utils.DES](https://github.com/Is999/go-utils/blob/master/des.go#L15) / [utils.TripleDES](https://github.com/Is999/go-utils/blob/master/des.go#L32)

```go
// 实例化DES，并设置key和iv
a, err := DES(key, WithIV(iv))
if err != nil {
fmt.Errorf("DES() error = %v", err)
return
}

// 加密数据
encryptStr, err := a.Encrypt(data, CBC, base64.StdEncoding.EncodeToString, PKCS7Pad)
if err != nil {
fmt.Errorf("Encrypt() mode = %v error = %v", CBC, err)
return
}

// 解密数据
got, err := a.Decrypt(encryptStr, CBC, base64.StdEncoding.DecodeString, PKCS7Unpad)
if err != nil {
fmt.Errorf("Decrypt() mode = %v error = %v", CBC, err)
return
}
```

备注：DES/3DES 仅建议用于旧系统兼容，新系统应优先使用 AES-GCM。先实例化 DES 并设置 key；CBC/CTR/CFB/OFB 等需要显式通过
`WithIV` 设置固定 IV，或通过 `WithRandIV(true)` 使用随机 IV。`CTR/CFB/OFB` 属于非认证流模式，默认禁用，仅在兼容旧系统时通过
`WithAllowUnsafeStreamMode(true)` 显式开启。
`ECB` 默认禁用，仅在兼容旧系统时通过 `WithAllowUnsafeECB(true)` 显式开启；历史上“未设置 IV 时使用 key 派生
IV”的兼容行为也默认禁用，如需兼容旧密文可显式开启 `WithAllowUnsafeKeyIV(true)`。

补充：`CTR/CFB/OFB` 若用于旧协议兼容且无需补位，建议配合 `NoPad` / `NoUnpad` 使用，避免多余填充和块长度约束。

------

### 5.6 pkcs7 填充与反填充

------

#### AES GCM（推荐）

```go
// 实例化 AES
a, err := AES(key)
if err != nil {
fmt.Errorf("AES() error = %v", err)
return
}

// GCM 加密，additionalData 可为空
encryptStr, err := a.EncryptGCMString(data, base64.StdEncoding.EncodeToString, []byte("request-id=r-1"))
if err != nil {
fmt.Errorf("EncryptGCMString() error = %v", err)
return
}

// GCM 解密，additionalData 必须与加密时一致
got, err := a.DecryptGCMString(encryptStr, base64.StdEncoding.DecodeString, []byte("request-id=r-1"))
if err != nil {
fmt.Errorf("DecryptGCMString() error = %v", err)
return
}
```

备注：GCM 属于 AEAD 认证加密模式，优先级高于 CBC/CTR/CFB/OFB；每次加密都会自动生成随机 nonce 并写入密文头部。

------

#### NoPad / NoUnpad

```go
// CTR 模式下直接关闭补位。CTR 不提供完整性校验，仅用于旧协议兼容。
a, err := AES(key, WithRandIV(true), WithAllowUnsafeStreamMode(true))
if err != nil {
return
}

encryptStr, err := a.Encrypt(data, CTR, base64.StdEncoding.EncodeToString, NoPad)
if err != nil {
return
}

got, err := a.Decrypt(encryptStr, CTR, base64.StdEncoding.DecodeString, NoUnpad)
if err != nil {
return
}
```

备注：`NoPad` / `NoUnpad` 适合 `CTR/CFB/OFB/GCM` 这类不依赖块补位的模式；其中 `CTR/CFB/OFB`
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

#### func [utils.PKCS7Pad](https://github.com/Is999/go-utils/blob/master/pkcs7.go#L8)

```go
func PKCS7Pad(data []byte, blockSize int) []byte
```

备注：数据填充。

------

#### func [utils.PKCS7Unpad](https://github.com/Is999/go-utils/blob/master/pkcs7.go#L24)

```go
func PKCS7Unpad(data []byte) ([]byte, error)
```

备注：数据反填充。

------

### 5.7 0 填充与反填充

------

#### func [utils.ZeroPad](https://github.com/Is999/go-utils/blob/master/zero.go#L8)

```go
func ZeroPad(data []byte, blockSize int) []byte
```

备注：数据填充。

------

#### func [utils.ZeroUnpad](https://github.com/Is999/go-utils/blob/master/zero.go#L19)

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

#### func [utils.ToInt64](https://github.com/Is999/go-utils/blob/master/strconv.go#L16)

```go
func ToInt64(s string) (i int64)
```

备注：string 转 int64，转换失败返回零值。

------

#### func [utils.ToInt](https://github.com/Is999/go-utils/blob/master/strconv.go#L10)

```go
func ToInt(s string) (i int)
```

备注：string 转 int，转换失败返回零值。

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

#### func [utils.ToFloat64](https://github.com/Is999/go-utils/blob/master/strconv.go#L22)

```go
func ToFloat64(s string) (i float64)
```

备注：string 转 float64，失败返回零值。

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

#### func [utils.NumberFormat](https://github.com/Is999/go-utils/blob/master/misce.go#L39)

```go
func NumberFormat(number float64, decimals uint, decPoint, thousandsSep string) string 
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

#### func [utils.BinOct](https://github.com/Is999/go-utils/blob/master/strconv.go#L28)

```go
func BinOct(str string) (string, error)
```

备注：二进制转换为八进制。

------

#### func [utils.BinDec](https://github.com/Is999/go-utils/blob/master/strconv.go#L33)

```go
func BinDec(str string) (int64, error)
```

备注：二进制转换为十进制。

------

#### func [utils.BinHex](https://github.com/Is999/go-utils/blob/master/strconv.go#L38)

```go
func BinHex(str string) (string, error)
```

备注：二进制转换为十六进制。

------

#### func [utils.OctBin](https://github.com/Is999/go-utils/blob/master/strconv.go#L43)

```go
func OctBin(data string) (string, error)
```

备注：八进制转换为二进制。

------

#### func [utils.OctDec](https://github.com/Is999/go-utils/blob/master/strconv.go#L48)

```go
func OctDec(str string) (int64, error)
```

备注：八进制转换为十进制。

------

#### func [utils.OctHex](https://github.com/Is999/go-utils/blob/master/strconv.go#L53)

```go
func OctHex(data string) (string, error)
```

备注：八进制转换为十六进制。

------

#### func [utils.DecBin](https://github.com/Is999/go-utils/blob/master/strconv.go#L58)

```go
func DecBin(number int64) string
```

备注：十进制转换为二进制。

------

#### func [utils.DecOct](https://github.com/Is999/go-utils/blob/master/strconv.go#L63)

```go
func DecOct(number int64) string
```

备注：十进制转换为八进制。

------

#### func [utils.DecHex](https://github.com/Is999/go-utils/blob/master/strconv.go#L68)

```go
func DecHex(number int64) string
```

备注：十进制转换为十六进制。

------

#### func [utils.HexBin](https://github.com/Is999/go-utils/blob/master/strconv.go#L73)

```go
func HexBin(data string) (string, error) 
```

备注：十六进制转换为二进制。

------

#### func [utils.HexOct](https://github.com/Is999/go-utils/blob/master/strconv.go#L78)

```go
func HexOct(str string) (string, error)
```

备注：十六进制转换为八进制。

------

#### func [utils.HexDec](https://github.com/Is999/go-utils/blob/master/strconv.go#L83)

```go
func HexDec(str string) (int64, error)
```

备注：十六进制转换为十进制。

------

## 7. 数组/切片/链表

------

### 7.1 检查数组中是否存在某个值

------

#### func [utils.IsHas](https://github.com/Is999/go-utils/blob/master/slices.go#L10)

```go
func IsHas[T comparable](v T, s []T) bool
```

备注：检查s中是否存在v。1.21版本以上推荐使用标准库 slices.Contains(s,v)

------

### 7.2 统计某个值在数组中出现次数

------

#### func [utils.HasCount](https://github.com/Is999/go-utils/blob/master/slices.go#L20)

```go
func HasCount[T comparable](v T, s []T) (count int)
```

备注：统计v在s中出现次数。

------

### 7.3 反转数组

------

#### func [utils.Reverse](https://github.com/Is999/go-utils/blob/master/slices.go#L30)

```go
func Reverse[T any](s []T) []T
```

备注：反转s。1.21版本以上推荐使用标准库 slices.Reverse(s)

------

### 7.4 去除数组中重复的值

------

#### func [utils.Unique](https://github.com/Is999/go-utils/blob/master/slices.go#L39)

```go
func Unique[T comparable](s []T) []T
```

备注：去除s中重复的值。

------

### 7.5 计算两个数组的差集

------

#### func [utils.Diff](https://github.com/Is999/go-utils/blob/master/slices.go#L83)

```go
func Diff[T comparable](s1, s2 []T) []T
```

备注：计算s1与s2的差集，返回结果保持 s1 原有顺序，并保留 s1 中原本存在的重复值。

------

### 7.6 计算两个数组的交集

------

#### func [utils.Intersect](https://github.com/Is999/go-utils/blob/master/slices.go#L125)

```go
func Intersect[T comparable](s1, s2 []T) []T
```

备注：计算s1与s2的交集，返回结果保持 s1 原有顺序，并保留 s1 中原本存在的重复值。

------

### 7.7 切片求和

------

#### func [utils.SumSlice](https://github.com/Is999/go-utils/blob/master/slices.go#L174)

```go
func SumSlice[T Number](nums []T) T
```

备注：计算切片中所有元素的和。

------

### 7.8 列表 status container/list.List

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

#### 移动到列表的最面

```go
// 将ele3元素移动到列表的最后面
l.MoveToFront(ele3)
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

#### func [utils.MapKeys](https://github.com/Is999/go-utils/blob/master/map.go#L8)

```go
func MapKeys[K Ordered, V any](m map[K]V) []K 
```

备注：获取map所有的key

------

### 8.2 获取map的所有value

------

#### func [utils.MapValues](https://github.com/Is999/go-utils/blob/master/map.go#L19)

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

#### func [utils.MapRange](https://github.com/Is999/go-utils/blob/master/map.go#L49)

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

#### func [utils.MapFilter](https://github.com/Is999/go-utils/blob/master/map.go#L69)

```go
func MapFilter[K Ordered, V any](m map[K]V, f func(key K, value V) bool) map[K]V
```

| 参数  | 描述                                                   |
|-----|------------------------------------------------------|
| *m* | map。                                                 |
| *f* | f 函数接收key与value，返回一个bool值，如果f函数返回false则过滤掉该元素（删除该元素） |

备注：使用回调函数过滤map的元素，如果f 函数返回 false则过滤掉该元素（删除该元素）。

------

### 8.5 计算两个map的差集

------

#### func [utils.MapDiff](https://github.com/Is999/go-utils/blob/master/map.go#L80)

```go
func MapDiff[K comparable, V comparable](m1, m2 map[K]V) []V
```

备注：计算 m1 与 m2 的值差集，返回结果保留 m1 当前遍历结果中的重复值。

------

#### func [utils.MapDiffKey](https://github.com/Is999/go-utils/blob/master/map.go#L113)

```go
func MapDiffKey[K Ordered, V any](m1, m2 map[K]V) []K 
```

备注：计算m1与m2的键差集。

------

### 8.6 计算两个map的交集

------

#### func [utils.MapIntersect](https://github.com/Is999/go-utils/blob/master/map.go#L97)

```go
func MapIntersect[K comparable, V comparable](m1, m2 map[K]V) []V
```

备注：计算 m1 与 m2 的值交集，返回结果保留 m1 当前遍历结果中的重复值。

------

#### func [utils.MapIntersectKey](https://github.com/Is999/go-utils/blob/master/map.go#L124)

```go
func MapIntersectKey[K Ordered, V any](m1, m2 map[K]V) []K
```

备注：计算m1与m2的键交集。

------

### 8.7 计算map的值和

------

#### func [utils.SumMap](https://github.com/Is999/go-utils/blob/master/map.go#L135)

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
func (m *Map) Store(key, value interface{})
```

备注：向 map 中存入键为 key，值为 value 的键值对，这里的 key 和 value 都是 *
*[interface](https://haicoder.net/golang/golang-interface.html)** 类型的，因此 key 和 value 可以存入任意的类型。

------

#### 获取元素 Load

```go
func (m *Map) Load(key interface{}) (value interface{}, ok bool)
```

备注：返回的 value 是 interface 类型的，因此 value 我们不可以直接使用，而必须要转换之后才可以使用，返回的ok是 bool
值，表明获取是否成功。

------

#### 获取或添加 LoadOrStore

```go
func (m *Map) LoadOrStore(key, value interface{}) (actual interface{}, loaded bool) 
```

备注：获取的 key 存在，返回 key 对应的元素，如果获取的 key 不存在，就返回设置的值，并且将设置的值，存入 map。

------

#### 删除元素 Delete

```go
func (m *Map) Delete(key interface{})
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
func (m *Map) Range(f func (key, value interface{}) bool)
```

备注：遍历元素，如果f 函数返回 false则终止遍历。

------

## 9. 时间

------


### 9.1 时区

------

#### func [utils.Local](https://github.com/Is999/go-utils/blob/master/time.go#L14)

```go
func Local() *time.Location
```

备注：系统运行时区。

------

#### func [utils.CST](https://github.com/Is999/go-utils/blob/master/time.go#L19)

```go
func CST() *time.Location 
```

备注：东八时区。

------

#### func [utils.UTC](https://github.com/Is999/go-utils/blob/master/time.go#L24)

```go
func UTC() *time.Location 
```

备注：UTC时区。

------

### 9.2  验证日期：年、月、日

------

#### func [utils.CheckDate](https://github.com/Is999/go-utils/blob/master/time.go#L95)

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

#### func [utils.MonthDay](https://github.com/Is999/go-utils/blob/master/time.go#L80)

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

#### func [utils.AddTime](https://github.com/Is999/go-utils/blob/master/time.go#L111)

```go
func AddTime(t time.Time, addTimes ...string) (time.Time, error)
```

| 参数         | 描述                                   |
|------------|--------------------------------------|
| *addTimes* | 增加时间（Y年，M月，D日，H时，I分，S秒，L毫秒，C微妙，N纳秒)。 |

备注：增加时间。

------

### 9.5  获取日期信息

------

#### func [utils.DateInfo](https://github.com/Is999/go-utils/blob/master/time.go#L185)

```go
func DateInfo(t time.Time) map[string]interface{}
```

备注：获取日期信息。

```
//	返回：year int - 年，
//		month int - 月，monthEn string - 英文月，
//		day int - 日，yearDay int - 一年中第几日， weekDay int - 一周中第几日，
//		hour int - 时，hour int - 分，second int - 秒，
//		millisecond int - 毫秒，microsecond int - 微妙，nanosecond int - 纳秒，
//		unix int64 - 时间戳-秒，unixNano int64 - 时间戳-纳秒，
//		weekDay int - 星期几，weekDayEn string - 星期几英文， yearWeek int - 一年中第几周，
//		date string - 格式化日期，dateNs string - 格式化日期（纳秒)
```

------

### 9.6  时间格式化为时间字符串

------

#### func [utils.TimeFormat](https://github.com/Is999/go-utils/blob/master/time.go#L217)

```go
func TimeFormat(timeZone *time.Location, layout string, timestamp ...int64) string
```

| 参数          | 描述                  |
|-------------|---------------------|
| *timeZone*  | 时区。                 |
| layout      | 格式化。                |
| *timestamp* | Unix 时间sec秒和nsec纳秒。 |

备注：时间格式化为时间字符串。

------

### 9.7  解析时间字符串为time.Time

------

#### func [utils.TimeParse](https://github.com/Is999/go-utils/blob/master/time.go#L236)

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

#### func [utils.Before](https://github.com/Is999/go-utils/blob/master/time.go#L326)

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

#### func [utils.After](https://github.com/Is999/go-utils/blob/master/time.go#L335)

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

#### func [utils.Equal](https://github.com/Is999/go-utils/blob/master/time.go#L344)

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

#### func [utils.Sub](https://github.com/Is999/go-utils/blob/master/time.go#L353)

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

#### func [utils.Empty](https://github.com/Is999/go-utils/blob/master/regexp.go#L142)

```go
func Empty(value string) bool 
```

备注：空字符串验证。

------

#### func [utils.QQ](https://github.com/Is999/go-utils/blob/master/regexp.go#L147)

```go
func QQ(value string) bool
```

备注：QQ号验证。

------

#### func [utils.Email](https://github.com/Is999/go-utils/blob/master/regexp.go#L152)

```go
func Email(value string) bool
```

备注：电子邮件验证。

------

#### func [utils.Mobile](https://github.com/Is999/go-utils/blob/master/regexp.go#L157)

```go
func Mobile(value string) bool
```

备注：中国大陆手机号码验证。

------

#### func [utils.Phone](https://github.com/Is999/go-utils/blob/master/regexp.go#L162)

```go
func Phone(value string) bool
```

备注：中国大陆电话号码验证。

------

#### func [utils.Numeric](https://github.com/Is999/go-utils/blob/master/regexp.go#L167)

```go
func Numeric(value string) bool
```

备注：有符号数字验证。

------

#### func [utils.UnNumeric](https://github.com/Is999/go-utils/blob/master/regexp.go#L172)

```go
func UnNumeric(value string) bool
```

备注：无符号数字验证。

------

#### func [utils.UnInteger](https://github.com/Is999/go-utils/blob/master/regexp.go#L177)

```go
func UnInteger(value string) bool
```

备注：无符号整数(正整数)验证。

------

#### func [utils.UnIntZero](https://github.com/Is999/go-utils/blob/master/regexp.go#L182)

```go
func UnIntZero(value string) bool
```

备注：无符号整数(正整数+0)验证。

------

#### func [utils.Amount](https://github.com/Is999/go-utils/blob/master/regexp.go#L191)

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

#### func [utils.Alpha](https://github.com/Is999/go-utils/blob/master/regexp.go#L201)

```go
func Alpha(value string) bool
```

备注：英文字母验证。

------

#### func [utils.Zh](https://github.com/Is999/go-utils/blob/master/regexp.go#L206)

```go
func Zh(value string) bool
```

备注：中文字符验证。

------

#### func [utils.MixStr](https://github.com/Is999/go-utils/blob/master/regexp.go#L211)

```go
func MixStr(value string) bool 
```

备注：英文、数字、特殊字符(不包含换行符)。

------

#### func [utils.Alnum](https://github.com/Is999/go-utils/blob/master/regexp.go#L216)

```go
func Alnum(value string) bool
```

备注：英文字母+数字验证。

------

#### func [utils.Domain](https://github.com/Is999/go-utils/blob/master/regexp.go#L221)

```go
func Domain(value string) bool
```

备注：域名(64位内正确的域名，可包含中文、字母、数字和.-)。

------

#### func [utils.TimeMonth](https://github.com/Is999/go-utils/blob/master/regexp.go#L226)

```go
func TimeMonth(value string) bool
```

备注：时间格式验证 yyyy-MM yyyy/MM。

------

#### func [utils.TimeDay](https://github.com/Is999/go-utils/blob/master/regexp.go#L231)

```go
func TimeDay(value string) bool
```

备注：时间格式验证 yyyy-MM-dd。

------

#### func [utils.Timestamp](https://github.com/Is999/go-utils/blob/master/regexp.go#L253)

```go
func Timestamp(value string) bool
```

备注：Timestamp 时间格式验证 yyyy-MM-dd hh:mm:ss。

------

#### func [utils.Account](https://github.com/Is999/go-utils/blob/master/regexp.go#L274)

```go
func Account(value string, min, max uint8) error
```

备注：帐号验证(字母开头，允许字母数字下划线，长度在min-max之间)。

------

#### func [utils.Password](https://github.com/Is999/go-utils/blob/master/regexp.go#L292)

```go
func Password(value string, min, max uint8) error
```

备注：密码(字母开头，允许字母数字下划线，长度在 min - max之间)。

------

#### func [utils.StrongPassword](https://github.com/Is999/go-utils/blob/master/regexp.go#L306)

```go
func StrongPassword(value string, min, max uint8) error
```

备注：强密码(必须包含大小写字母和数字的组合，不能使用特殊字符，长度在min-max之间)。

------

#### func [utils.StrongPasswordWithSymbols](https://github.com/Is999/go-utils/blob/master/regexp.go#L331)

```go
func StrongPasswordWithSymbols(value string, min, max uint8) error
```

备注：强密码(必须包含大小写字母和数字的组合，可以使用特殊字符，长度在min-max之间)。

------

#### func [utils.HasSymbols](https://github.com/Is999/go-utils/blob/master/regexp.go#L355)

```go
func HasSymbols(value string) bool
```

备注：是否包含符号。

------

#### 相关函数：[utils.Before](https://github.com/Is999/go-utils/blob/master/time.go#L326) / [utils.After](https://github.com/Is999/go-utils/blob/master/time.go#L335) / [utils.Equal](https://github.com/Is999/go-utils/blob/master/time.go#L344)

备注：参考9.7 两个时间字符串判断。

------

## 11. http/curl

------

### 11.1 模拟curl请求

> 请求方式：GET、POST（form，file）、HEAD、PUT、PATCH、DELETE、OPTIONS

备注：

- `Curl` 适合作为“基础模板配置 + 按请求派生实例”使用。
- 共享基础配置时，生产环境建议使用 `Clone()` 或 `NewRequest()` 获取独立请求实例，再设置本次请求的 Header / Param / Body。
- `NewRequest()` 会复用底层 Transport 连接池，同时为新实例生成新的 `X-Request-Id`，更适合并发场景。
- GET/POST/PUT/PATCH/DELETE/OPTIONS 追加查询参数时只拼接已经编码好的参数串，URL 合法性由发送阶段的 `http.NewRequest` 统一校验。
- 开启默认日志时，响应 body 只按 `SetLogBodyLimit` / dump 上限读取预览，并恢复 `Body` 供 `AfterResponse` / `AfterBody` 继续消费。

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

#### func [(*Curl).Get](https://github.com/Is999/go-utils/blob/master/curl_client.go#L416)

```go
func (c *Curl) Get(url string) (err error)
```

备注：参考测试用例：[TestGet](https://github.com/Is999/go-utils/blob/master/curl_test.go#L44)

------

#### func [(*Curl).Post](https://github.com/Is999/go-utils/blob/master/curl_client.go#L427)

```go
func (c *Curl) Post(url string) (err error) 
```

备注：参考测试用例：[TestPost](https://github.com/Is999/go-utils/blob/master/curl_test.go#L210)

------

#### func [(*Curl).PostForm](https://github.com/Is999/go-utils/blob/master/curl_client.go#L438)

```go
func (c *Curl) PostForm(url string) error 
```

备注：参考测试用例：[TestPostForm](https://github.com/Is999/go-utils/blob/master/curl_test.go#L382)

------

#### func [(*Curl).Post](https://github.com/Is999/go-utils/blob/master/curl_client.go#L427)

```go
func (c *Curl) Post(url string) (err error) 
```

备注：参考测试用例：[TestPostFile](https://github.com/Is999/go-utils/blob/master/curl_test.go#L487)

------

#### func [(*Curl).Clone](https://github.com/Is999/go-utils/blob/master/curl_client.go#L358) / [(*Curl).NewRequest](https://github.com/Is999/go-utils/blob/master/curl_client.go#L405)

```go
func (c *Curl) Clone() (*Curl, error)
func (c *Curl) NewRequest() (*Curl, error)
```

备注：

- `Clone()` 会深拷贝 Header、Params、Cookie、StatusCode、Body 等请求级配置，并复用底层连接池。
- `NewRequest()` 基于 `Clone()` 派生新的请求实例，并自动生成新的请求 ID。
- 若当前 `Body` 是不可回放的流式 Reader，`Clone()` / `NewRequest()` 会返回错误，避免多个请求共享同一读取游标。

参考测试用例：

- [TestCurlCloneDeepCopiesRequestState](https://github.com/Is999/go-utils/blob/master/curl_test.go#L934)
- [TestCurlNewRequestGeneratesIndependentRequestID](https://github.com/Is999/go-utils/blob/master/curl_test.go#L972)
- [TestCurlTemplateReuseWithNewRequest](https://github.com/Is999/go-utils/blob/master/curl_test.go#L1000)

------

## 12. http/response

------

### 12.1 重定向

------

#### func [utils.Redirect](https://github.com/Is999/go-utils/blob/master/response.go#L311)

```go
func Redirect(w http.ResponseWriter, url string, opts ...ResponseOption)
```

| 参数   | 描述                          |
|------|-----------------------------|
| url  | 重定向地址                       |
| opts | 响应配置项：如 WithStatusCode(301) |

```go
http.HandleFunc("/response/redirect", func(w http.ResponseWriter, r *http.Request) {
		// 重定向
		utils.Redirect(w, "/response/json")
	})
```

备注：重定向，默认响应302。

------

### 12.2 响应JSON

------

#### func [utils.Json](https://github.com/Is999/go-utils/blob/master/response.go#L298)

```go
// Json 响应Json数据
func Json(w http.ResponseWriter, opts ...ResponseOption) *Response

// Success 成功响应返回Json数据
func (r *Response) Success(code int, data any, message ...string)

// Fail 失败响应返回Json数据
func (r *Response) Fail(code int, message string, data ...any)
```

示例：

```go
// 响应json数据
http.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {

  // 获取URL查询字符串参数
  queryParam := r.URL.Query().Get("v")

  // 响应的数据
  user := User{
    Name:      "张三",
    Age:       22,
    Sex:       "男",
    IsMarried: false,
    Address:   "北京市",
    phone:     "131188889999",
  }

  if queryParam == "fail" {
    // 错误响应
	utils.Json(w, utils.WithStatusCode(http.StatusNotAcceptable)).Fail(2000, "fail", user)
    return
  }
  // 成功响应
  utils.Json(w).Success(1000, user)
})
```

备注：响应JSON数据，响应成功：Json().Success()，响应失败：Json().Fail()。

------

### 12.3 响应HTML

------

```go
// 响应html
http.HandleFunc("/response/html", func(w http.ResponseWriter, r *http.Request) {

  // 响应html数据
  utils.View(w).Html("<p>这是一个<b style=\"color: red\">段落!</b></p>")
})
```

备注：响应HTML文本 View().Html()。

------

### 12.4 响应XML

------

```go
// 响应xml
http.HandleFunc("/response/xml", func(w http.ResponseWriter, r *http.Request) {

  // 响应的数据
  user := User{
    Name:      "张三",
    Age:       22,
    Sex:       "男",
    IsMarried: false,
    Address:   "北京市",
    phone:     "131188889999",
  }

  // 响应xml数据
  utils.View(w).Xml(user)
})
```

备注：响应XML文本 View().Xml()。

------

### 12.5 响应TEXT

------

```go
// 响应text
http.HandleFunc("/response/text", func(w http.ResponseWriter, r *http.Request) {
  // 响应text数据
  utils.View(w).Text("<p>这是一个<b style=\"color: red\">段落!</b></p>")
})
```

备注：响应TEXT文本 View().Text()。

------

### 12.6 显示图片

------

```go
// 响应image
http.HandleFunc("/response/show", func(w http.ResponseWriter, r *http.Request) {
  // 获取URL查询字符串参数
  file := r.URL.Query().Get("file")
  if utils.IsExist(file) {
    // 显示文件内容

    utils.View(w).Show(file)
    return
  }
  // 处理错误
  utils.View(w, utils.WithStatusCode(http.StatusNotFound)).Text("不存在的文件：" + file)
})
```

备注：显示文件内容 View().Show()。

------

### 12.7 下载文件

------

```go
// 下载文件
http.HandleFunc("/response/download", func(w http.ResponseWriter, r *http.Request) {
		// 获取URL查询字符串参数
		file := r.URL.Query().Get("file")
		if utils.IsExist(file) {
			// 下载文件数据
			utils.View(w).Download(file)
			return
		}
		// 处理错误
		utils.View(w, utils.WithStatusCode(http.StatusNotFound)).Text("不存在的文件：" + file)
})
```

备注：下载文件 View().Download()。

------

## 13. 打包压缩

------

### 13.1 zip

------

#### func [utils.Zip](https://github.com/Is999/go-utils/blob/master/zip.go#L17)

```go
func Zip(zipFile string, files []string) error
```

| 参数      | 描述         |
|---------|------------|
| zipFile | 打包压缩后文件    |
| files   | 待打包压缩文件【夹】 |

备注：使用 zip 打包并压缩；为保证安全，默认不支持符号链接打包。

------

#### func [utils.UnZip](https://github.com/Is999/go-utils/blob/master/zip.go#L157)

```go
func UnZip(zipFile, destDir string) error
```

| 参数      | 描述     |
|---------|--------|
| zipFile | 代解压的文件 |
| destDir | 解压文件目录 |

备注：解压 zip 文件；默认仅允许解压到目标目录内，拒绝绝对路径、目录穿越和不支持的条目类型，并尽量恢复归档中的文件权限。

------

### 13.2 tar

------

#### func [utils.Tar](https://github.com/Is999/go-utils/blob/master/tar.go#L18)

```go
func Tar(tarFile string, files []string) error 
```

| 参数      | 描述       |
|---------|----------|
| tarFile | 打包后文件    |
| files   | 待打包文件【夹】 |

备注：使用 tar 打包；为保证安全，默认不支持符号链接打包。

------

#### func [utils.TarGz](https://github.com/Is999/go-utils/blob/master/tar.go#L47)

```go
func TarGz(tarGzFile string, files []string) error
```

| 参数        | 描述         |
|-----------|------------|
| tarGzFile | 打包压缩后文件    |
| files     | 待打包压缩文件【夹】 |

备注：使用 tar 打包并 gzip 压缩；为保证安全，默认不支持符号链接打包。

------

#### func [utils.UnTar](https://github.com/Is999/go-utils/blob/master/tar.go#L193)

```go
func UnTar(tarFile, destDir string) error 
```

| 参数      | 描述     |
|---------|--------|
| tarFile | 代解压的文件 |
| destDir | 解压文件目录 |

备注：解压 tar 或 tar.gz 文件；默认仅允许解压到目标目录内，拒绝绝对路径、目录穿越和不支持的条目类型，并尽量恢复归档中的文件权限。

------

## 14. 日志

------

### 14.1 默认日志（使用标准库的 `log` 包来记录日志）

------

#### 设置日志等级和输出格式

```go
// 日志等级
levelVar := &slog.LevelVar{}
levelVar.Set(slog.LevelDebug)

opts := &slog.HandlerOptions{
  AddSource: true,     // 输出日志的文件和行号
  Level:     levelVar, // 日志等级
}

//日志输出文件
file, err := os.OpenFile("sys.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
if err != nil {
  fmt.Printf("Failed to open error logger file: %v\n", err)
  return
}

// 日志输出格式
//handler := slog.NewTextHandler(os.Stdout, opts)
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

#### func [utils.GetEnv](https://github.com/Is999/go-utils/blob/master/env.go#L13)

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

#### func [utils.ServerIP](https://github.com/Is999/go-utils/blob/master/ip.go#L34)

```go
func ServerIP() string
```

备注：服务器对外IP。默认优先返回缓存值或本地网卡 IP，仅在本地 IP 不可用时才回退到 UDP 探测；如需控制超时可使用
`ServerIPContext`。

------

#### func [utils.LocalIP](https://github.com/Is999/go-utils/blob/master/ip.go#L59)

```go
func LocalIP() string
```

备注：服务器本地IP。

------

#### func [utils.ClientIP](https://github.com/Is999/go-utils/blob/master/ip.go#L87)

```go
func ClientIP(r *http.Request) string
```

备注：获取客户端 IP。默认仅在请求来自回环地址时信任 `X-Forwarded-For` / `X-Real-IP`，否则回退到 `RemoteAddr`，避免把所有私网来源都视为
可信代理。生产环境建议优先使用 `ClientIPWithTrustedProxies` 显式配置白名单。

------

#### func [utils.NewTrustedProxies](https://github.com/Is999/go-utils/blob/master/ip.go#L93)

```go
func NewTrustedProxies(values ...string) (*TrustedProxies, error)
```

备注：构造可信代理白名单，支持单个 IP 或 CIDR，例如 `10.0.0.10`、`10.0.0.0/8`、`fd00::/8`。

------

#### func [utils.ClientIPWithTrustedProxies](https://github.com/Is999/go-utils/blob/master/ip.go#L137)

```go
func ClientIPWithTrustedProxies(r *http.Request, trustedProxies *TrustedProxies) string
```

备注：使用显式可信代理白名单解析客户端 IP。生产环境建议按网关、负载均衡或 Ingress 固定地址配置可信代理；只有 `RemoteAddr`
命中白名单时才读取转发头。

------

### 15.3 三目运算

------

#### func [utils.Ternary](https://github.com/Is999/go-utils/blob/master/misce.go#L26)

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

#### func [utils.RuntimeInfo](https://github.com/Is999/go-utils/blob/master/runtime.go#L16)

```go
func RuntimeInfo(skip int) *Frame
```

备注：获取当前行号、方法名、文件地址。

------

#### func [utils.GetFunctionName](https://github.com/Is999/go-utils/blob/master/runtime.go#L38)

```go
func GetFunctionName(i interface{}) string
```

备注：获取函数名（普通函数、结构体方法或匿名函数）。

------

### 15.5 重试机制

------

#### func [utils.Retry](https://github.com/Is999/go-utils/blob/master/misce.go#L101)

```go
func Retry(maxRetries uint8, fn func(tries int) error) error
```

| 参数           | 描述                                 |
|--------------|------------------------------------|
| *maxRetries* | 最大重试次数。                            |
| *fn*         | 要执行的函数，参数tries为当前第几次尝试，返回nil则停止重试。 |

备注：尝试执行 fn，如果 fn 返回错误则进行重试；当前退避从第一次失败后的 `100ms~200ms` 区间起步，按 2 倍指数退避并加入随机抖动，最大不超过
`3s`。
`maxRetries` 表示最大尝试次数，包含首次执行；当 `maxRetries == 0` 时会按 1 次处理。

------

### 15.6 带重试的一次性执行

------

#### struct [utils.Once](https://github.com/Is999/go-utils/blob/master/once.go#L12)

```go
var o utils.Once

// 执行带重试机制的函数调用（线程安全）
err := o.Do(func () error {
// 业务逻辑
return nil
}, 3) // 最大重试3次

// 重置状态，使其可再次执行
o.Reset()
```

备注：线程安全的带重试机制的一次性执行器；同一轮只会有一个 goroutine 真正执行目标函数，其余调用方等待最终结果。成功或最终失败后会缓存结果，需重新执行时调用
`Reset()`。

------

### 15.7 泛型对象池

------

#### struct [utils.Pool](https://github.com/Is999/go-utils/blob/master/pool.go#L10)

```go
// 创建对象池
pool := utils.NewPool(func () *bytes.Buffer {
return new(bytes.Buffer)
}, utils.WithPoolReset(func (b *bytes.Buffer) {
b.Reset()
}))

// 获取对象
buf := pool.Get()

// 归还对象（自动执行重置逻辑）
pool.Put(buf)
```

备注：基于 sync.Pool 封装的泛型对象池，支持通过 `WithPoolReset` 设置对象归还时的重置函数；未传工厂函数时会回退为 `new(T)`，
`Put(nil)` 会被安全忽略。

------

### 15.8 全局配置

------

#### func [utils.Configure](https://github.com/Is999/go-utils/blob/master/config.go#L95)

```go
func Configure(opts ...Option)
```

备注：设置全局参数入口，只需在程序入口处设置一次。目前支持通过 WithJSON 设置自定义 JSON 编解码器，通过 WithLogger 设置自定义日志实现。

```go
// 使用自定义 JSON 编解码器
utils.Configure(utils.WithJSON(customMarshal, customUnmarshal))

// 使用自定义 Logger
utils.Configure(utils.WithLogger(customLogger))
```
