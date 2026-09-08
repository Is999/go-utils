package utils

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Is999/go-utils/errors"
)

const (
	defaultTimeout         = 30 * time.Second   // 单次 Client.Do 的默认超时
	defaultMaxRetry        = 2                  // 默认最大请求尝试次数，包含首次请求
	defaultMaxRetries      = 5                  // 允许的最大请求尝试次数上限
	defaultDumpLimit       = 4096               // dump 正文默认预览字节数
	defaultLogLimit        = 4096               // 普通日志正文默认预览字节数
	defaultCurlContentType = "application/json" // Curl 默认请求 Content-Type，发送 JSON 请求体时无需调用方重复设置
	curlHeaderContentType  = "Content-Type"     // HTTP Content-Type 请求头名，用于默认值与用户显式设置的边界判断
	curlHeaderRequestID    = "X-Request-Id"     // 请求链路 ID 头名，发送前写入 Header 并同步到 Logger 字段
)

// requestIDCounter 提供进程内序号，与时间一起生成日志跟踪 ID。
var requestIDCounter atomic.Uint64

// CurlOption 在 NewCurl 构造期间按传入顺序修改配置。
type CurlOption func(*Curl)

// Curl 保存可串行复用的 HTTP 请求配置和客户端。
// 配置、访问可变容器和发送请求均不能在同一实例上并发进行；
// 并发请求可从只读模板调用 NewRequest 派生，具体共享边界见 Clone。
type Curl struct {
	cli                *http.Client            // HTTP 客户端实例，懒加载
	header             http.Header             // 请求头配置
	timeout            time.Duration           // 单次 Client.Do 的超时，包含重定向及响应体读取
	username, password string                  // Basic 认证账号密码
	proxyURL           string                  // 代理地址
	insecureSkipVerify bool                    // 是否跳过服务端证书链和主机名校验
	rootCAs            string                  // TLS 根证书路径
	cert, key          string                  // TLS 客户端证书和私钥路径
	transportDirty     bool                    // 传输层配置是否已变更；仅在代理/TLS 配置变化时重建 Transport
	cookies            map[string]*http.Cookie // 按名称保存调用方提供的 Cookie 指针
	params             url.Values              // URL 查询参数或 POST Form 参数
	body               io.Reader               // 待发送的读取流，发送时的关闭归属见 SendContext
	statusCode         []int                   // 200 之外允许通过校验的状态码

	// 应用请求配置后、日志预览前执行，重试不重复调用
	beforeRequest func(ctx context.Context, request *http.Request) error

	// 初始化 Client/Transport 后执行，重试不重复调用
	beforeClient func(ctx context.Context, client *http.Client) error

	// 响应日志与状态校验后执行，true 跳过 afterBody
	afterResponse func(ctx context.Context, response *http.Response) (isDone bool, err error)

	// 接收 afterResponse 处理后剩余的完整正文
	afterBody func(ctx context.Context, body []byte) error

	// 成功或失败时执行，先于响应体关闭
	afterDone func(ctx context.Context, client *http.Client, request *http.Request, response *http.Response)

	requestID     string // 请求头与日志共用的跟踪 ID
	maxRetry      uint8  // 最大请求尝试次数（默认 2 次，最大 5 次，包含首次请求）
	dump          bool   // 是否用包含头部的 dump 替代普通日志，正文仍受预览限额约束
	dumpBodyLimit int64  // dump 正文预览上限，单位：字节
	logBodyLimit  int64  // 普通日志正文预览上限，单位：字节；0 不预览
	defLogOutput  bool   // Curl 内部各级日志总开关，默认关闭
	baseLogger    Logger // 原始日志实例，用于重新绑定请求 ID 时避免字段叠加
	Logger        Logger // 日志实例
}

// NewCurl 创建一个新的 Curl 客户端实例。
// 默认配置：
//   - 超时时间：30 秒
//   - 总尝试次数：2 次，包含首次请求
//   - Content-Type：application/json；内部日志默认关闭
//   - WithCurlRequestID 非空时立即绑定；否则首次发送或读取请求 ID 时生成
func NewCurl(opts ...CurlOption) *Curl {
	c := &Curl{
		timeout:        defaultTimeout,
		transportDirty: true,
		maxRetry:       defaultMaxRetry,
		dumpBodyLimit:  defaultDumpLimit,
		logBodyLimit:   defaultLogLimit,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	if c.baseLogger == nil {
		c.baseLogger = Log()
	}
	if c.Logger == nil {
		c.Logger = c.baseLogger
	}

	// 默认 ID 延迟生成，显式 ID 则在返回前绑定到日志和请求头。
	if c.requestID != "" {
		c.SetRequestID(c.requestID)
	}

	return c
}

// WithCurlTimeout 设置单次请求超时；小于等于 0 时保留原配置。
func WithCurlTimeout(timeout time.Duration) CurlOption {
	return func(c *Curl) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// WithCurlLogger 设置日志实例；nil 保留原配置。
func WithCurlLogger(logger Logger) CurlOption {
	return func(c *Curl) {
		if logger != nil {
			c.baseLogger = logger
			c.Logger = logger
		}
	}
}

// WithCurlDefLogOutput 控制 Curl 内部日志，开启后仍受 Logger 等级限制。
func WithCurlDefLogOutput(enable bool) CurlOption {
	return func(c *Curl) {
		c.SetDefaultLogOutput(enable)
	}
}

// WithCurlRequestID 设置固定跟踪 ID；裁剪首尾空白，空值在首次使用时生成。
func WithCurlRequestID(requestID string) CurlOption {
	return func(c *Curl) {
		c.requestID = strings.TrimSpace(requestID)
	}
}

// WithCurlContentType 设置请求头 Content-Type。
func WithCurlContentType(contentType string) CurlOption {
	return func(c *Curl) {
		c.SetContentType(contentType)
	}
}

// WithCurlHeader 设置请求头键值。
func WithCurlHeader(key, value string) CurlOption {
	return func(c *Curl) {
		c.SetHeader(key, value)
	}
}

// WithCurlHeaders 应用选项时逐项覆盖请求头，保留未指定的键。
func WithCurlHeaders(headers map[string]string) CurlOption {
	return func(c *Curl) {
		c.SetHeaders(headers)
	}
}

// WithCurlParams 应用选项时逐项覆盖请求参数，保留未指定的键。
func WithCurlParams(params map[string]string) CurlOption {
	return func(c *Curl) {
		c.SetParams(params)
	}
}

// WithCurlBody 保存请求体引用；发送与关闭规则见 SendContext。
func WithCurlBody(body io.Reader) CurlOption {
	return func(c *Curl) {
		c.SetBody(body)
	}
}

// WithCurlBodyBytes 使用调用方的字节切片作为请求体，不复制数据。
func WithCurlBodyBytes(body []byte) CurlOption {
	return func(c *Curl) {
		c.SetBodyBytes(body)
	}
}

// WithCurlCookies 替换全部 Cookie，按名称保存传入指针并忽略 nil。
func WithCurlCookies(cookies ...*http.Cookie) CurlOption {
	return func(c *Curl) {
		c.SetCookies(cookies...)
	}
}

// WithCurlBasicAuth 设置 BasicAuth 账号密码。
func WithCurlBasicAuth(username, password string) CurlOption {
	return func(c *Curl) {
		c.SetBasicAuth(username, password)
	}
}

// WithCurlProxyURL 设置代理地址。
func WithCurlProxyURL(proxyURL string) CurlOption {
	return func(c *Curl) {
		c.SetProxyURL(proxyURL)
	}
}

// WithCurlInsecureSkipVerify 设置是否跳过服务端证书链和主机名校验。
func WithCurlInsecureSkipVerify(isSkip bool) CurlOption {
	return func(c *Curl) {
		c.InsecureSkipVerify(isSkip)
	}
}

// WithCurlRootCAs 设置根证书路径。
func WithCurlRootCAs(rootCAs string) CurlOption {
	return func(c *Curl) {
		c.SetRootCAs(rootCAs)
	}
}

// WithCurlCertKey 设置客户端证书和私钥。
func WithCurlCertKey(cert, key string) CurlOption {
	return func(c *Curl) {
		c.SetCertKey(cert, key)
	}
}

// WithCurlStatusCode 替换 200 之外允许通过校验的状态码；空参数保留原配置。
func WithCurlStatusCode(statusCode ...int) CurlOption {
	return func(c *Curl) {
		if len(statusCode) > 0 {
			c.SetStatusCode(statusCode...)
		}
	}
}

// WithCurlMaxRetry 设置最大请求尝试次数。
//
// max 包含首次请求；max=0 或 max=1 表示不额外重试，最大不超过 5。
func WithCurlMaxRetry(max uint8) CurlOption {
	return func(c *Curl) {
		c.SetMaxRetry(max)
	}
}

// WithCurlDump 设置是否开启 dump 模式。
func WithCurlDump(dump bool) CurlOption {
	return func(c *Curl) {
		c.SetDump(dump)
	}
}

// WithCurlDumpBodyLimit 设置 dump 正文预览字节数；小于等于 0 时保留原配置。
func WithCurlDumpBodyLimit(limit int64) CurlOption {
	return func(c *Curl) {
		if limit > 0 {
			c.dumpBodyLimit = limit
		}
	}
}

// WithCurlLogBodyLimit 设置普通日志正文预览字节数。
// 小于 0 的值会被忽略；等于 0 表示不记录 body 预览。
func WithCurlLogBodyLimit(limit int64) CurlOption {
	return func(c *Curl) {
		c.SetLogBodyLimit(limit)
	}
}

// SetDefaultLogOutput 控制 Curl 内部各级日志，包括重试和关闭错误日志。
// 默认关闭；启用后仍遵循 Logger 的等级配置。
func (c *Curl) SetDefaultLogOutput(enable bool) *Curl {
	c.defLogOutput = enable
	return c
}

// SetLogBodyLimit 设置普通日志正文预览字节数；0 不预览，负值保留原配置。
// 该值只影响日志，不限制响应读取大小。
func (c *Curl) SetLogBodyLimit(limit int64) *Curl {
	if limit >= 0 {
		c.logBodyLimit = limit
	}
	return c
}

// CloseIdleConnections 关闭当前连接池的空闲连接，下一次发送时重建 Transport。
// 不中断正在使用的连接；已共享该 Transport 的克隆实例也会失去这些空闲连接。
func (c *Curl) CloseIdleConnections() {
	if c.cli != nil {
		c.cli.CloseIdleConnections()
		c.cli.Transport = nil
	}
	c.transportDirty = true
}

// SetRequestID 设置请求跟踪 ID。
// 未传值或首个值仅含空白时，使用时间与序号生成 16 位 ID。
// 设置后会自动更新 Logger 和 Header 中的 X-Request-Id。
func (c *Curl) SetRequestID(requestID ...string) *Curl {
	if len(requestID) == 0 || strings.TrimSpace(requestID[0]) == "" {
		c.requestID = generateRequestID()
	} else {
		c.requestID = strings.TrimSpace(requestID[0])
	}

	logger := c.baseLogger
	if logger == nil {
		logger = c.Logger
	}
	if logger == nil {
		logger = Log()
	}

	c.Logger = logger.With("X-Request-Id", c.requestID)

	c.ensureHeader().Set(curlHeaderRequestID, c.requestID)
	return c
}

// GetRequestID 获取当前请求 ID。
// 未显式设置时会懒生成默认 ID，确保读取行为与发送请求时的链路 ID 一致。
func (c *Curl) GetRequestID() string {
	if c.requestID == "" {
		c.SetRequestID()
	}
	return c.requestID
}

// Clone 复制请求配置并保留当前请求 ID，Header/Params/Cookie/Body 与状态码列表分别隔离。
// 已初始化的 Transport、Cookie Jar、Logger 和回调仍共享；尚未初始化的连接池各自创建。
// 模板及其内存请求体保持只读时可并发克隆；自定义 io.ReadSeeker 的克隆须串行进行。
func (c *Curl) Clone() (*Curl, error) {
	if c == nil {
		return nil, errors.New("Clone() Curl 不能为空")
	}

	body, err := cloneCurlBody(c.body)
	if err != nil {
		return nil, errors.Tag(err)
	}

	// Curl 不含锁等同步状态；值配置和回调沿用模板，只隔离请求可变数据。
	cloned := *c
	cloned.cli = cloneHTTPClient(c.cli)
	cloned.header = c.header.Clone()
	cloned.cookies = cloneCookies(c.cookies)
	cloned.params = cloneURLValues(c.params)
	cloned.body = body
	cloned.statusCode = slices.Clone(c.statusCode)
	return &cloned, nil
}

// NewRequest 基于当前 Curl 配置派生新的请求实例。
// 与 Clone 不同，NewRequest 会生成新的请求 ID；模板复用遵循 Clone 的并发边界。
func (c *Curl) NewRequest() (*Curl, error) {
	cloned, err := c.Clone()
	if err != nil {
		return nil, errors.Tag(err)
	}
	return cloned.SetRequestID(), nil
}

// Get 发起 GET 请求。
func (c *Curl) Get(url string) error {
	return c.GetContext(context.Background(), url)
}

// GetContext 发起带 context 的 GET 请求。
func (c *Curl) GetContext(ctx context.Context, url string) error {
	url = buildURL(url, c.params.Encode())
	return c.SendContext(ctx, http.MethodGet, url, c.body)
}

// Post 发起 POST 请求。
func (c *Curl) Post(url string) error {
	return c.PostContext(context.Background(), url)
}

// PostContext 发起带 context 的 POST 请求。
func (c *Curl) PostContext(ctx context.Context, url string) error {
	url = buildURL(url, c.params.Encode())
	return c.SendContext(ctx, http.MethodPost, url, c.body)
}

// PostForm 发起 POST Form 请求。
func (c *Curl) PostForm(url string) error {
	return c.PostFormContext(context.Background(), url)
}

// PostFormContext 将当前 Params 编码为表单正文，并设置 application/x-www-form-urlencoded。
// 该入口使用表单正文，不使用 SetBody 保存的请求体。
func (c *Curl) PostFormContext(ctx context.Context, url string) error {
	return c.SetContentType("application/x-www-form-urlencoded").
		SendContext(ctx, http.MethodPost, url, strings.NewReader(c.params.Encode()))
}

// Put 发起 PUT 请求。
func (c *Curl) Put(url string) error {
	return c.PutContext(context.Background(), url)
}

// PutContext 发起带 context 的 PUT 请求。
func (c *Curl) PutContext(ctx context.Context, url string) error {
	url = buildURL(url, c.params.Encode())
	return c.SendContext(ctx, http.MethodPut, url, c.body)
}

// Patch 发起 PATCH 请求。
func (c *Curl) Patch(url string) error {
	return c.PatchContext(context.Background(), url)
}

// PatchContext 发起带 context 的 PATCH 请求。
func (c *Curl) PatchContext(ctx context.Context, url string) error {
	url = buildURL(url, c.params.Encode())
	return c.SendContext(ctx, http.MethodPatch, url, c.body)
}

// Head 发起 HEAD 请求。
func (c *Curl) Head(url string) error {
	return c.HeadContext(context.Background(), url)
}

// HeadContext 将当前 Params 追加到 URL，发送不含正文的 HEAD 请求。
func (c *Curl) HeadContext(ctx context.Context, url string) error {
	url = buildURL(url, c.params.Encode())
	return c.SendContext(ctx, http.MethodHead, url, nil)
}

// Delete 发起 DELETE 请求。
func (c *Curl) Delete(url string) error {
	return c.DeleteContext(context.Background(), url)
}

// DeleteContext 发起带 context 的 DELETE 请求。
func (c *Curl) DeleteContext(ctx context.Context, url string) error {
	url = buildURL(url, c.params.Encode())
	return c.SendContext(ctx, http.MethodDelete, url, c.body)
}

// Options 发起 OPTIONS 请求。
func (c *Curl) Options(url string) error {
	return c.OptionsContext(context.Background(), url)
}

// OptionsContext 发起带 context 的 OPTIONS 请求。
func (c *Curl) OptionsContext(ctx context.Context, url string) error {
	url = buildURL(url, c.params.Encode())
	return c.SendContext(ctx, http.MethodOptions, url, c.body)
}

// generateRequestID 将时间与进程内序号拼接为固定长度的日志跟踪 ID。
func generateRequestID() string {
	const length = 16 // 默认请求头和日志共用 16 位跟踪 ID。
	var buf [64]byte
	out := buf[:0]
	out = strconv.AppendUint(out, uint64(time.Now().UnixNano()), 36)
	out = strconv.AppendUint(out, requestIDCounter.Add(1), 36)
	for len(out) < length {
		out = strconv.AppendUint(out, requestIDCounter.Add(1), 36)
	}
	return string(out[:length])
}

// cloneHTTPClient 复制 Client 的公开配置，Transport、Jar 和重定向回调仍共享。
func cloneHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		return &http.Client{}
	}
	return &http.Client{
		Transport:     client.Transport,
		CheckRedirect: client.CheckRedirect,
		Jar:           client.Jar,
		Timeout:       client.Timeout,
	}
}

// cloneURLValues 复制 URL 参数，避免克隆实例之间共享可变 map。
func cloneURLValues(values url.Values) url.Values {
	if values == nil {
		return nil
	}
	cloned := make(url.Values, len(values))
	for key, list := range values {
		cloned[key] = append([]string(nil), list...)
	}
	return cloned
}

// cloneCookies 深拷贝 Cookie 映射，避免克隆实例之间共享指针。
func cloneCookies(cookies map[string]*http.Cookie) map[string]*http.Cookie {
	if cookies == nil {
		return nil
	}
	cloned := make(map[string]*http.Cookie, len(cookies))
	for name, cookie := range cookies {
		if cookie == nil {
			continue
		}
		cp := *cookie
		cp.Unparsed = slices.Clone(cookie.Unparsed)
		cloned[name] = &cp
	}
	return cloned
}

// cloneCurlBody 为克隆实例建立独立读取位置；无法重放的流返回错误。
func cloneCurlBody(body io.Reader) (io.Reader, error) {
	switch v := body.(type) {
	case nil:
		return nil, nil
	case *bytes.Buffer:
		// Buffer 只保留未读部分，复制后不再依赖调用方的可变缓冲区。
		return bytes.NewReader(bytes.Clone(v.Bytes())), nil
	case *bytes.Reader:
		// ReadAt 不移动模板游标，已知总长度也避免快照读取时反复扩容。
		data := make([]byte, v.Size())
		_, _ = v.ReadAt(data, 0)
		return bytes.NewReader(data), nil
	case *strings.Reader:
		// 字符串不可变，只需为派生请求保留独立游标。
		cloned := *v
		_, _ = cloned.Seek(0, io.SeekStart)
		return &cloned, nil
	case io.ReadSeeker:
		// 自定义正文从头建立快照，读取完成后恢复模板游标。
		return cloneReadSeeker(v)
	default:
		return nil, errors.Errorf("Clone() 不支持复制不可重放的请求体类型: %T", body)
	}
}

// cloneReadSeeker 复制自定义可定位请求体并恢复原始游标。
// 调用方负责限制体积并串行访问底层 reader。
func cloneReadSeeker(reader io.ReadSeeker) (io.Reader, error) {
	pos, err := reader.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, errors.Tag(err)
	}
	if _, err = reader.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Tag(err)
	}
	data, err := io.ReadAll(reader)
	// 即使读取失败也恢复游标，避免失败的 Clone 改变调用方的后续读取位置。
	_, restoreErr := reader.Seek(pos, io.SeekStart)
	if err != nil {
		return nil, errors.Tag(err)
	}
	if restoreErr != nil {
		return nil, errors.Tag(restoreErr)
	}
	return bytes.NewReader(data), nil
}

// buildURL 将已编码的查询参数追加到 URL，并保留 fragment。
func buildURL(baseURL, encodedParams string) string {
	if encodedParams == "" {
		return baseURL
	}

	fragment := ""
	queryTarget := baseURL
	if idx := strings.IndexByte(baseURL, '#'); idx >= 0 {
		queryTarget = baseURL[:idx]
		fragment = baseURL[idx:]
	}

	separator := "?"
	if strings.Contains(queryTarget, "?") {
		switch {
		case strings.HasSuffix(queryTarget, "?"), strings.HasSuffix(queryTarget, "&"):
			separator = ""
		default:
			separator = "&"
		}
	}
	return queryTarget + separator + encodedParams + fragment
}
