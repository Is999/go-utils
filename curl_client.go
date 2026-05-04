package utils

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Is999/go-utils/errors"
)

// ============================ 常量定义 ============================

const (
	defaultTimeout    = 30 * time.Second // 默认超时时间
	defaultMaxRetry   = 2                // 默认最大请求尝试次数，包含首次请求
	defaultMaxRetries = 5                // 允许的最大请求尝试次数上限
	defaultDumpLimit  = 4096             // dump 预览默认长度上限
)

// requestIDCounter 是请求 ID 自增计数器，保证同纳秒内生成值仍可区分。
var requestIDCounter atomic.Uint64

// ============================ CurlOption 配置项 ============================

// CurlOption Curl 配置项函数类型。
// 用于通过选项模式配置 Curl 客户端实例。
type CurlOption func(*Curl)

// ============================ Curl 核心结构体 ============================

// Curl HTTP 请求客户端。
// 支持链式调用，配置丰富，功能完善：
//   - GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS 七种 HTTP 方法
//   - Header/Params/Cookie/Body 配置
//   - BasicAuth/Proxy/TLS 认证配置
//   - 请求重试/超时控制
//   - 请求响应日志追踪
//
// 使用示例：
//
//	utils.NewCurl().
//	    SetTimeout(10).
//	    SetHeader("Authorization", "Bearer xxx").
//	    SetParam("page", "1").
//	    SetBodyBytes(userData).
//	    AfterBody(func(body []byte) error {
//	        return json.Unmarshal(body, &result)
//	    }).
//	    Post("https://api.example.com/user")
type Curl struct {
	cli                *http.Client                                                              // HTTP 客户端实例，懒加载
	header             http.Header                                                               // 请求头配置
	timeout            time.Duration                                                             // 请求超时时间
	username, password string                                                                    // Basic 认证账号密码
	proxyURL           string                                                                    // 代理地址
	insecureSkipVerify bool                                                                      // 是否跳过 HTTPS 不安全验证（生产环境禁止使用）
	rootCAs            string                                                                    // TLS 根证书路径
	cert, key          string                                                                    // TLS 客户端证书和私钥路径
	cookies            map[string]*http.Cookie                                                   // Cookie 配置
	params             url.Values                                                                // URL 查询参数或 POST Form 参数
	body               io.Reader                                                                 // 请求体
	statusCode         []int                                                                     // 可接受的状态码列表（除 200 以外需要特殊处理的状态码）
	beforeRequest      func(request *http.Request) error                                         // 请求发送前的回调，可对 Request 进行自定义处理
	beforeClient       func(client *http.Client) error                                           // 请求发送前的回调，可对 Client 进行自定义处理
	afterResponse      func(response *http.Response) (isDone bool, err error)                    // 请求发送后的回调
	afterBody          func(body []byte) error                                                   // 请求发送后对 Response.Body 的处理回调
	afterDone          func(client *http.Client, request *http.Request, response *http.Response) // 请求完成后的回调
	requestID          string                                                                    // 请求唯一标识
	maxRetry           uint8                                                                     // 最大请求尝试次数（默认 2 次，最大 5 次，包含首次请求）
	dump               bool                                                                      // 是否开启 dump 模式：输出完整的请求和响应详情
	dumpBodyLimit      int64                                                                     // dump 预览内容长度上限
	defLogOutput       bool                                                                      // 是否启用默认日志输出（INFO 及以下级别）
	baseLogger         Logger                                                                    // 原始日志实例，用于重新绑定请求 ID 时避免字段叠加
	Logger             Logger                                                                    // 日志实例
}

// ============================ 构造函数 ============================

// New 创建一个新的 Curl 客户端实例。
// 默认配置：
//   - 超时时间：30 秒
//   - 最大重试：2 次
//   - Content-Type：application/json
//   - 自动生成 X-Request-Id
//
// 参数说明：
//   - opts：可选的配置项列表
//
// 返回值：配置好的 Curl 客户端指针
func NewCurl(opts ...CurlOption) *Curl {
	logger := Log()
	c := &Curl{
		cli:                &http.Client{},
		header:             make(http.Header),
		timeout:            defaultTimeout,
		username:           "",
		password:           "",
		proxyURL:           "",
		insecureSkipVerify: false,
		rootCAs:            "",
		cert:               "",
		key:                "",
		cookies:            make(map[string]*http.Cookie),
		params:             make(url.Values),
		body:               nil,
		statusCode:         make([]int, 0),
		beforeRequest:      nil,
		beforeClient:       nil,
		afterResponse:      nil,
		afterBody:          nil,
		requestID:          "",
		maxRetry:           defaultMaxRetry,
		dump:               false,
		dumpBodyLimit:      defaultDumpLimit,
		defLogOutput:       false,
		baseLogger:         logger,
		Logger:             logger,
	}

	// 设置默认 Content-Type
	c.SetContentType("application/json")

	// 应用配置选项
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	// 统一绑定请求 ID，确保 WithLogger/WithRequestID 任意顺序都能生效。
	c.SetRequestID(c.requestID)

	return c
}

// ============================ 配置选项 ============================

// WithTimeout 设置默认超时时间。
func WithCurlTimeout(timeout time.Duration) CurlOption {
	return func(c *Curl) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// WithLogger 设置日志实例。
func WithCurlLogger(logger Logger) CurlOption {
	return func(c *Curl) {
		if logger != nil {
			c.baseLogger = logger
			c.Logger = logger
		}
	}
}

// WithDefLogOutput 设置默认日志输出开关。
func WithCurlDefLogOutput(enable bool) CurlOption {
	return func(c *Curl) {
		c.defLogOutput = enable
	}
}

// WithCurlRequestID 设置请求 ID。
func WithCurlRequestID(requestID string) CurlOption {
	return func(c *Curl) {
		c.requestID = strings.TrimSpace(requestID)
	}
}

// WithCurlRequestId 设置请求 ID。
//
// Deprecated: 请使用 WithCurlRequestID。
func WithCurlRequestId(requestId string) CurlOption {
	return WithCurlRequestID(requestId)
}

// WithContentType 设置请求头 Content-Type。
func WithCurlContentType(contentType string) CurlOption {
	return func(c *Curl) {
		c.SetContentType(contentType)
	}
}

// WithHeader 设置请求头键值。
func WithCurlHeader(key, value string) CurlOption {
	return func(c *Curl) {
		c.SetHeader(key, value)
	}
}

// WithHeaders 批量设置请求头。
func WithCurlHeaders(headers map[string]string) CurlOption {
	return func(c *Curl) {
		c.SetHeaders(headers)
	}
}

// WithParams 批量设置请求参数。
func WithCurlParams(params map[string]string) CurlOption {
	return func(c *Curl) {
		c.SetParams(params)
	}
}

// WithBody 设置请求体。
func WithCurlBody(body io.Reader) CurlOption {
	return func(c *Curl) {
		c.body = body
	}
}

// WithBodyBytes 设置请求体字节。
func WithCurlBodyBytes(body []byte) CurlOption {
	return func(c *Curl) {
		c.body = bytes.NewReader(body)
	}
}

// WithCookies 设置请求 Cookie。
func WithCurlCookies(cookies ...*http.Cookie) CurlOption {
	return func(c *Curl) {
		c.SetCookies(cookies...)
	}
}

// WithBasicAuth 设置 BasicAuth 账号密码。
func WithCurlBasicAuth(username, password string) CurlOption {
	return func(c *Curl) {
		c.username = username
		c.password = password
	}
}

// WithProxyURL 设置代理地址。
func WithCurlProxyURL(proxyURL string) CurlOption {
	return func(c *Curl) {
		c.proxyURL = proxyURL
	}
}

// WithInsecureSkipVerify 设置是否跳过 HTTPS 验证（生产环境禁止使用）。
func WithCurlInsecureSkipVerify(isSkip bool) CurlOption {
	return func(c *Curl) {
		c.insecureSkipVerify = isSkip
	}
}

// WithRootCAs 设置根证书路径。
func WithCurlRootCAs(rootCAs string) CurlOption {
	return func(c *Curl) {
		c.rootCAs = rootCAs
	}
}

// WithCertKey 设置客户端证书和私钥。
func WithCurlCertKey(cert, key string) CurlOption {
	return func(c *Curl) {
		c.cert = cert
		c.key = key
	}
}

// WithStatusCode 设置可接受的状态码。
func WithCurlStatusCode(statusCode ...int) CurlOption {
	return func(c *Curl) {
		if len(statusCode) > 0 {
			c.statusCode = append(c.statusCode[:0], statusCode...)
		}
	}
}

// WithMaxRetry 设置最大请求尝试次数。
//
// max 包含首次请求；max=0 或 max=1 表示不额外重试，最大不超过 5。
func WithCurlMaxRetry(max uint8) CurlOption {
	return func(c *Curl) {
		c.maxRetry = max
	}
}

// WithDump 设置是否开启 dump 模式。
func WithCurlDump(dump bool) CurlOption {
	return func(c *Curl) {
		c.dump = dump
	}
}

// WithDumpBodyLimit 设置 dump 预览内容长度上限。
func WithCurlDumpBodyLimit(limit int64) CurlOption {
	return func(c *Curl) {
		if limit > 0 {
			c.dumpBodyLimit = limit
		}
	}
}

// ============================ 生命周期方法 ============================

// SetDefLogOutput 设置默认日志输出开关。
// true：打印 INFO 及以下级别日志；false：禁止打印默认日志。
func (c *Curl) SetDefLogOutput(enable bool) *Curl {
	c.defLogOutput = enable
	return c
}

// CloseIdleConnections 关闭所有空闲连接并释放 Transport。
func (c *Curl) CloseIdleConnections() {
	if c.cli != nil {
		c.cli.CloseIdleConnections()
		c.cli.Transport = nil
	}
}

// SetRequestID 设置请求唯一标识。
// 如果未指定或为空字符串，自动生成 16 位唯一 ID。
// 设置后会自动更新 Logger 和 Header 中的 X-Request-Id。
func (c *Curl) SetRequestID(requestID ...string) *Curl {
	if len(requestID) == 0 || strings.TrimSpace(requestID[0]) == "" {
		c.requestID = generateUniqID(16)
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

	// 更新日志实例的请求 ID 字段
	c.Logger = logger.With("X-Request-Id", c.requestID)

	// 更新请求头
	c.header.Set("X-Request-Id", c.requestID)
	return c
}

// SetRequestId 设置请求唯一标识。
//
// Deprecated: 请使用 SetRequestID。
func (c *Curl) SetRequestId(requestId ...string) *Curl {
	return c.SetRequestID(requestId...)
}

// GetRequestID 获取当前请求 ID。
func (c *Curl) GetRequestID() string {
	return c.requestID
}

// GetRequestId 获取当前请求 ID。
//
// Deprecated: 请使用 GetRequestID。
func (c *Curl) GetRequestId() string {
	return c.GetRequestID()
}

// ============================ HTTP 请求方法 ============================

// Get 发起 GET 请求。
func (c *Curl) Get(url string) (err error) {
	url, err = buildURL(url, c.params)
	if err != nil {
		return errors.Tag(err)
	}
	return c.Send(http.MethodGet, url, c.body)
}

// Post 发起 POST 请求。
func (c *Curl) Post(url string) (err error) {
	url, err = buildURL(url, c.params)
	if err != nil {
		return errors.Tag(err)
	}
	return c.Send(http.MethodPost, url, c.body)
}

// PostForm 发起 POST Form 请求。
func (c *Curl) PostForm(url string) error {
	return c.SetContentType("application/x-www-form-urlencoded").
		Send(http.MethodPost, url, strings.NewReader(c.params.Encode()))
}

// Put 发起 PUT 请求。
func (c *Curl) Put(url string) (err error) {
	url, err = buildURL(url, c.params)
	if err != nil {
		return errors.Tag(err)
	}
	return c.Send(http.MethodPut, url, c.body)
}

// Patch 发起 PATCH 请求。
func (c *Curl) Patch(url string) (err error) {
	url, err = buildURL(url, c.params)
	if err != nil {
		return errors.Tag(err)
	}
	return c.Send(http.MethodPatch, url, c.body)
}

// Head 发起 HEAD 请求。
func (c *Curl) Head(url string) error {
	return c.Send(http.MethodHead, url, nil)
}

// Delete 发起 DELETE 请求。
func (c *Curl) Delete(url string) (err error) {
	url, err = buildURL(url, c.params)
	if err != nil {
		return errors.Tag(err)
	}
	return c.Send(http.MethodDelete, url, c.body)
}

// Options 发起 OPTIONS 请求。
func (c *Curl) Options(url string) (err error) {
	url, err = buildURL(url, c.params)
	if err != nil {
		return errors.Tag(err)
	}
	return c.Send(http.MethodOptions, url, c.body)
}

// ============================ 内部辅助函数 ============================

// generateUniqID 生成指定长度的请求唯一 ID。
func generateUniqID(length int) string {
	if length <= 0 {
		return ""
	}
	var buf [64]byte
	out := buf[:0]
	out = strconv.AppendUint(out, uint64(time.Now().UnixNano()), 36)
	out = strconv.AppendUint(out, requestIDCounter.Add(1), 36)
	for len(out) < length {
		out = strconv.AppendUint(out, requestIDCounter.Add(1), 36)
	}
	return string(out[:length])
}

// generateUniqId 生成指定长度的请求唯一 ID。
//
// Deprecated: 请使用 generateUniqID。
func generateUniqId(length int) string {
	return generateUniqID(length)
}

// buildURL 构建完整的 URL，将 params 追加为查询参数。
func buildURL(baseURL string, params url.Values) (string, error) {
	if params == nil || len(params) == 0 {
		return baseURL, nil
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", errors.Tag(err)
	}
	q := u.Query()
	for key, values := range params {
		for _, value := range values {
			q.Add(key, value)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// buildUrl 构建完整的 URL，将 params 追加为查询参数。
//
// Deprecated: 请使用 buildURL。
func buildUrl(baseUrl string, params url.Values) (string, error) {
	return buildURL(baseUrl, params)
}
