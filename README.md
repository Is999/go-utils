# go-utils

`go-utils` 是一组 Go 1.22+ 常用工具函数，覆盖加密、HTTP 请求、响应输出、归档、错误追踪、日志适配、集合、字符串、对象池和一次性执行等场景。

当前版本按“生产默认安全、历史兼容显式开启”的原则整理实现：新代码优先使用认证加密、显式 IV、可信代理白名单、有限重试、归档解压边界和并发安全的全局配置。

## 安装

```bash
go get -u github.com/Is999/go-utils
```

## 生产级结论

| 模块                    | 当前结论                                                                                                                   |
|-----------------------|------------------------------------------------------------------------------------------------------------------------|
| AES / DES             | AES 推荐 `GCM`；`CBC/CTR` 必须显式 `WithIV` 或 `WithRandIV(true)`；`ECB`、`CFB/OFB`、默认 key 派生 IV 默认禁用。DES/3DES 仅保留历史兼容，不建议新系统使用。 |
| RSA                   | 生成和导入密钥均要求至少 2048 位；推荐 `EncryptOAEP/DecryptOAEP` 与 `SignPSS/VerifyPSS`；`PKCS#1 v1.5` 仅用于旧协议兼容；签名拒绝 MD5/SHA1。           |
| PKCS7 / Zero          | `Pkcs7UnPadding` 严格校验填充字节；`ZeroPadding` 仅适合能接受尾部 0 歧义的旧协议。                                                             |
| Pool / Once           | `Pool[T]` 基于 `sync.Pool`，支持归还前 reset；`Once` 并发安全，失败会按有限指数退避重试，结果会缓存，需重新执行时调用 `Reset`。                                  |
| Retry                 | `Retry` 使用 capped exponential backoff + jitter：第一次失败后约 100ms~200ms，最多 3s；`maxRetries=0` 时只执行 1 次。                      |
| IP / Logger           | `ClientIP` 仅在远端为可信代理时读取转发头；生产建议使用 `ClientIPWithTrustedProxies`。默认 logger 基于 `log/slog` 并发安全，自定义 logger 也应保证并发安全。       |
| slices / string / map | 常用函数无共享可变全局状态；传入的 slice/map 若被外部并发修改，调用方需自行加锁。                                                                         |
| tar / zip             | 解压防 Zip Slip/Tar Slip，拒绝绝对路径、目录穿越、空字符、符号链接和特殊文件；限制条目数、单文件大小和总展开大小。                                                     |
| errors                | 支持栈追踪、错误码、上下文键值、`errors.Join`、文本/JSON/slog 输出；全局配置使用原子变量并发安全。                                                          |
| curl                  | 继承标准库默认 Transport 连接池/TLS 行为；Header 按请求克隆；可回放 body 才会重试；dump 有长度限制。`Curl` 实例配置阶段不建议并发修改。                               |
| response              | 状态码只写一次；Content-Type 自动规范化；下载文件名清洗 CR/LF 和路径；文件输出拒绝目录。                                                                 |

## 全局配置

`Configure` 只应在程序启动时调用一次。未配置时使用标准库 `encoding/json` 和 `log/slog`。

```go
package main

import (
	"encoding/json"

	utils "github.com/Is999/go-utils"
)

func main() {
	utils.Configure(
		utils.WithJSON(json.Marshal, json.Unmarshal),
		utils.WithLogger(utils.Log()),
	)
}
```

自定义 `Logger` 需要实现：

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

## 加密

### AES-GCM 推荐用法

```go
package main

import (
	"encoding/base64"

	utils "github.com/Is999/go-utils"
)

func aesGCM() (string, string, error) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		return "", "", err
	}

	aad := []byte("request-id=demo")
	encrypted, err := c.EncryptGCMString("hello", base64.StdEncoding.EncodeToString, aad)
	if err != nil {
		return "", "", err
	}

	plain, err := c.DecryptGCMString(encrypted, base64.StdEncoding.DecodeString, aad)
	return encrypted, plain, err
}
```

### CBC/CTR 兼容用法

```go
c, err := utils.AES("1234567812345678", utils.WithRandIV(true))
if err != nil {
	return err
}

encrypted, err := c.Encrypt("hello", utils.CBC, base64.StdEncoding.EncodeToString, utils.Pkcs7Padding)
plain, err := c.Decrypt(encrypted, utils.CBC, base64.StdEncoding.DecodeString, utils.Pkcs7UnPadding)
_ = plain
```

需要兼容旧系统时才显式开启：

```go
utils.WithAllowUnsafeECB(true)
utils.WithAllowUnsafeKeyIV(true)
utils.WithAllowUnsafeStreamMode(true)
```

### RSA 推荐用法

```go
files, err := utils.GenerateKeyRSA("./keys", 2048)
if err != nil {
	return err
}

r, err := utils.NewRSA(files[0], files[1], utils.WithRSAFilePath(true))
if err != nil {
	return err
}

encrypted, err := r.EncryptOAEP("secret", base64.StdEncoding.EncodeToString, sha256.New())
plain, err := r.DecryptOAEP(encrypted, base64.StdEncoding.DecodeString, sha256.New())

signature, err := r.SignPSS("payload", crypto.SHA256, base64.StdEncoding.EncodeToString, nil)
err = r.VerifyPSS("payload", signature, crypto.SHA256, base64.StdEncoding.DecodeString, nil)
_ = plain
```

## HTTP Curl

```go
err := utils.NewCurl(
	utils.WithCurlTimeout(10*time.Second),
	utils.WithCurlMaxRetry(2),
	utils.WithCurlDefLogOutput(true),
).
	SetHeader("Authorization", "Bearer token").
	SetParam("page", "1").
	SetBodyBytes([]byte(`{"name":"codex"}`)).
	AfterBody(func(body []byte) error {
		return nil
	}).
	Post("https://api.example.com/users")
```

`SetMaxRetry` / `WithCurlMaxRetry` 表示“最大请求尝试次数”，包含首次请求；默认 2 次，最大 5 次。不可回放请求体不会被盲目重试，避免发送空
body 或半截 body。

## Response

```go
func handler(w http.ResponseWriter, r *http.Request) {
	utils.Json(w).Success(0, map[string]string{"id": "1"})
}

func download(w http.ResponseWriter, r *http.Request) {
	utils.View(w).DownloadRequest(r, "./report.pdf", "report.pdf")
}
```

`Json` 默认 `application/json; charset=utf-8`；`Download` 会清洗下载文件名并设置 `X-Content-Type-Options: nosniff`。

## 归档

```go
err := utils.Zip("./dist.zip", []string{"./dist"})
err = utils.UnZip("./dist.zip", "./out")

err = utils.TarGz("./dist.tar.gz", []string{"./dist"})
err = utils.UnTar("./dist.tar.gz", "./out")
```

解压限制：

| 限制项       | 默认值     |
|-----------|---------|
| 最大条目数     | 10000   |
| 单文件最大展开大小 | 256 MiB |
| 总展开大小     | 1 GiB   |

## 错误追踪

```go
import apperrors "github.com/Is999/go-utils/errors"

err := apperrors.WithCode(apperrors.Wrap(io.EOF, "读取配置失败"), 10001)
slog.Error(err.Error(), "trace", apperrors.Trace(err))

text := apperrors.TraceString(err)
jsonText := apperrors.TraceJSON(err)
_, _ = text, jsonText
```

可调配置：

```go
apperrors.SetStackDepth(16)
apperrors.SetTraceEnabled(true)
```

## IP

```go
trusted, err := utils.NewTrustedProxies("10.0.0.0/8", "127.0.0.1")
if err != nil {
	return err
}

clientIP := utils.ClientIPWithTrustedProxies(r, trusted)
_ = clientIP
```

生产环境建议显式配置可信代理，不要无条件信任 `X-Forwarded-For`。

## Pool / Once / Retry

```go
pool := utils.NewPool(
	func() *bytes.Buffer { return bytes.NewBuffer(nil) },
	utils.WithPoolReset(func(b *bytes.Buffer) { b.Reset() }),
)
buf := pool.Get()
pool.Put(buf)

var once utils.Once
err := once.Do(func() error {
	return nil
}, 3)

err = utils.Retry(3, func(tries int) error {
	return nil
})
```

## 集合与字符串

常用函数包括：

- slice：`IsHas`、`HasCount`、`Reverse`、`Unique`、`Diff`、`Intersect`、`SumSlice`
- map：`MapKeys`、`MapValues`、`MapRange`、`MapFilter`、`MapDiff`、`MapIntersect`、`MapDiffKey`、`MapIntersectKey`、`SumMap`
- string：`Replace`、`Substr`、`StrRev`、`RandStr`、`RandStr2`、`RandStr3`、`UniqId`

这些函数不维护共享业务状态；对同一个 map/slice 的并发读写仍需调用方自行同步。

## 代码风格

- 命名遵循 Go 约定，保留已有公开 API 的兼容名称。
- 结构体字段使用同一行中文说明，公开方法和关键内部逻辑使用中文注释。
- 日志统一走 `Logger` 接口，默认实现为 `log/slog`。
- 安全风险开关均以 `WithAllowUnsafe...` 命名，调用方需要显式表达兼容意图。

## 验证

本次交付通过以下命令验证：

```bash
go test ./...
go test -race ./...
go vet ./...
```
