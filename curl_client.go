package utils

import (
	"bytes"
	"context"
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
	defaultTimeout         = 30 * time.Second   // 默认超时时间
	defaultMaxRetry        = 2                  // 默认最大请求尝试次数，包含首次请求
	defaultMaxRetries      = 5                  // 允许的最大请求尝试次数上限
	defaultDumpLimit       = 4096               // dump 预览默认长度上限
	defaultLogLimit        = 4096               // 默认日志 body 预览长度上限
	defaultCurlContentType = "application/json" // Curl 默认请求 Content-Type，发送 JSON 请求体时无需调用方重复设置
	curlHeaderContentType  = "Content-Type"     // HTTP Content-Type 请求头名，用于默认值与用户显式设置的边界判断
	curlHeaderRequestID    = "X-Request-Id"     // 请求链路 ID 头名，发送前写入 Header 并同步到 Logger 字段
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
// 注意：Curl 实例的请求配置阶段不是并发安全的；多个 goroutine 并发请求时，应为每个请求分别创建 Curl 实例。
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
	cli                *http.Client                                                                                   // HTTP 客户端实例，懒加载
	header             http.Header                                                                                    // 请求头配置
	timeout            time.Duration                                                                                  // 请求超时时间
	username, password string                                                                                         // Basic 认证账号密码
	proxyURL           string                                                                                         // 代理地址
	insecureSkipVerify bool                                                                                           // 是否跳过 HTTPS 不安全验证（生产环境禁止使用）
	rootCAs            string                                                                                         // TLS 根证书路径
	cert, key          string                                                                                         // TLS 客户端证书和私钥路径
	transportDirty     bool                                                                                           // 传输层配置是否已变更；仅在代理/TLS 配置变化时重建 Transport
	cookies            map[string]*http.Cookie                                                                        // Cookie 配置
	params             url.Values                                                                                     // URL 查询参数或 POST Form 参数
	encodedParams      string                                                                                         // 已编码参数缓存，数据来源于 params.Encode，供重复请求复用
	paramsDirty        bool                                                                                           // 参数缓存是否失效；Set/Add/Del/GetParams 暴露可变 map 后置为 true
	body               io.Reader                                                                                      // 请求体
	statusCode         []int                                                                                          // 可接受的状态码列表（除 200 以外需要特殊处理的状态码）
	beforeRequest      func(ctx context.Context, request *http.Request) error                                         // 请求发送前的回调，可对 Request 进行自定义处理
	beforeClient       func(ctx context.Context, client *http.Client) error                                           // 请求发送前的回调，可对 Client 进行自定义处理
	afterResponse      func(ctx context.Context, response *http.Response) (isDone bool, err error)                    // 请求发送后的回调
	afterBody          func(ctx context.Context, body []byte) error                                                   // 请求发送后对 Response.Body 的处理回调
	afterDone          func(ctx context.Context, client *http.Client, request *http.Request, response *http.Response) // 请求完成后的回调
	requestID          string                                                                                         // 请求唯一标识
	maxRetry           uint8                                                                                          // 最大请求尝试次数（默认 2 次，最大 5 次，包含首次请求）
	dump               bool                                                                                           // 是否开启 dump 模式：输出完整的请求和响应详情
	dumpBodyLimit      int64                                                                                          // dump 预览内容长度上限
	logBodyLimit       int64                                                                                          // 默认日志 body 预览长度上限
	defLogOutput       bool                                                                                           // 是否启用默认日志输出（INFO 及以下级别）
	baseLogger         Logger                                                                                         // 原始日志实例，用于重新绑定请求 ID 时避免字段叠加
	Logger             Logger                                                                                         // 日志实例
}

// ============================ 构造函数 ============================

// NewCurl 创建一个新的 Curl 客户端实例。
// 默认配置：
//   - 超时时间：30 秒
//   - 最大重试：2 次
//   - Content-Type：application/json
//   - 显式传入 X-Request-Id 时立即绑定；未传入时首次发送或读取请求 ID 时自动生成
func NewCurl(opts ...CurlOption) *Curl {
	c := &Curl{
		timeout:        defaultTimeout,
		transportDirty: true,
		maxRetry:       defaultMaxRetry,
		dumpBodyLimit:  defaultDumpLimit,
		logBodyLimit:   defaultLogLimit,
	}

	// 应用配置选项。Header/Params/Cookies 等可变容器由对应 setter 懒初始化，避免只构造不配置时产生无用分配。
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

	// 显式请求 ID 立即绑定 Logger/Header；默认 ID 延迟到真正发送请求，降低只构造模板对象时的开销。
	if c.requestID != "" {
		c.SetRequestID(c.requestID)
	}

	return c
}

// ============================ 配置选项 ============================

// WithCurlTimeout 设置默认超时时间。
func WithCurlTimeout(timeout time.Duration) CurlOption {
	return func(c *Curl) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// WithCurlLogger 设置日志实例。
func WithCurlLogger(logger Logger) CurlOption {
	return func(c *Curl) {
		if logger != nil {
			c.baseLogger = logger
			c.Logger = logger
		}
	}
}

// WithCurlDefLogOutput 设置默认日志输出开关。
func WithCurlDefLogOutput(enable bool) CurlOption {
	return func(c *Curl) {
		c.SetDefLogOutput(enable)
	}
}

// WithCurlRequestID 设置请求 ID。
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

// WithCurlHeaders 批量设置请求头。
func WithCurlHeaders(headers map[string]string) CurlOption {
	return func(c *Curl) {
		c.SetHeaders(headers)
	}
}

// WithCurlParams 批量设置请求参数。
func WithCurlParams(params map[string]string) CurlOption {
	return func(c *Curl) {
		c.SetParams(params)
	}
}

// WithCurlBody 设置请求体。
func WithCurlBody(body io.Reader) CurlOption {
	return func(c *Curl) {
		c.SetBody(body)
	}
}

// WithCurlBodyBytes 设置请求体字节。
func WithCurlBodyBytes(body []byte) CurlOption {
	return func(c *Curl) {
		c.SetBodyBytes(body)
	}
}

// WithCurlCookies 设置请求 Cookie。
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

// WithCurlInsecureSkipVerify 设置是否跳过 HTTPS 验证（生产环境禁止使用）。
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

// WithCurlStatusCode 设置可接受的状态码。
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

// WithCurlDumpBodyLimit 设置 dump 预览内容长度上限。
func WithCurlDumpBodyLimit(limit int64) CurlOption {
	return func(c *Curl) {
		if limit > 0 {
			c.dumpBodyLimit = limit
		}
	}
}

// WithCurlLogBodyLimit 设置默认日志 body 预览长度上限。
// 小于 0 的值会被忽略；等于 0 表示不记录 body 预览。
func WithCurlLogBodyLimit(limit int64) CurlOption {
	return func(c *Curl) {
		c.SetLogBodyLimit(limit)
	}
}

// ============================ 生命周期方法 ============================

// SetDefLogOutput 设置默认日志输出开关。
// true：打印 INFO 及以下级别日志；false：禁止打印默认日志。
func (c *Curl) SetDefLogOutput(enable bool) *Curl {
	c.defLogOutput = enable
	return c
}

// SetLogBodyLimit 设置默认日志 body 预览长度上限。
// 等于 0 表示不记录 body 预览。
func (c *Curl) SetLogBodyLimit(limit int64) *Curl {
	if limit >= 0 {
		c.logBodyLimit = limit
	}
	return c
}

// CloseIdleConnections 关闭所有空闲连接并释放 Transport。
func (c *Curl) CloseIdleConnections() {
	if c.cli != nil {
		c.cli.CloseIdleConnections()
		c.cli.Transport = nil
	}
	c.markTransportDirty()
}

// SetRequestID 设置请求唯一标识。
// 如果未指定或为空字符串，自动生成 16 位唯一 ID。
// 设置后会自动更新 Logger 和 Header 中的 X-Request-Id。
func (c *Curl) SetRequestID(requestID ...string) *Curl {
	if len(requestID) == 0 || strings.TrimSpace(requestID[0]) == "" {
		c.requestID = generateUniqueID(16)
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

// Clone 深拷贝当前 Curl 配置。
// 适用于以当前 Curl 为模板派生新的请求实例，避免多个 goroutine 共享可变请求状态。
// Clone 会复用底层 Transport 连接池，但会复制 Header/Params/Cookie/Body 等请求级配置。
func (c *Curl) Clone() (*Curl, error) {
	if c == nil {
		return nil, errors.New("Clone() Curl 不能为空")
	}

	body, err := cloneCurlBody(c.body)
	if err != nil {
		return nil, errors.Tag(err)
	}

	cloned := &Curl{
		cli:                cloneHTTPClient(c.cli),
		header:             cloneHeader(c.header),
		timeout:            c.timeout,
		username:           c.username,
		password:           c.password,
		proxyURL:           c.proxyURL,
		insecureSkipVerify: c.insecureSkipVerify,
		rootCAs:            c.rootCAs,
		cert:               c.cert,
		key:                c.key,
		transportDirty:     c.transportDirty,
		cookies:            cloneCookies(c.cookies),
		params:             cloneURLValues(c.params),
		encodedParams:      c.encodedParams,
		paramsDirty:        c.paramsDirty,
		body:               body,
		statusCode:         append([]int(nil), c.statusCode...),
		beforeRequest:      c.beforeRequest,
		beforeClient:       c.beforeClient,
		afterResponse:      c.afterResponse,
		afterBody:          c.afterBody,
		afterDone:          c.afterDone,
		requestID:          c.requestID,
		maxRetry:           c.maxRetry,
		dump:               c.dump,
		dumpBodyLimit:      c.dumpBodyLimit,
		logBodyLimit:       c.logBodyLimit,
		defLogOutput:       c.defLogOutput,
		baseLogger:         c.baseLogger,
		Logger:             c.Logger,
	}
	return cloned, nil
}

// NewRequest 基于当前 Curl 配置派生新的请求实例。
// 与 Clone 不同，NewRequest 会为新实例生成新的请求 ID，适合把当前 Curl 作为并发安全的模板复用。
func (c *Curl) NewRequest() (*Curl, error) {
	cloned, err := c.Clone()
	if err != nil {
		return nil, errors.Tag(err)
	}
	return cloned.SetRequestID(), nil
}

// ============================ HTTP 请求方法 ============================

// Get 发起 GET 请求。
func (c *Curl) Get(url string) (err error) {
	return c.GetContext(context.Background(), url)
}

// GetContext 发起带 context 的 GET 请求。
func (c *Curl) GetContext(ctx context.Context, url string) (err error) {
	url = buildURLWithEncodedParams(url, c.encodedQueryParams())
	return c.SendContext(ctx, http.MethodGet, url, c.body)
}

// Post 发起 POST 请求。
func (c *Curl) Post(url string) (err error) {
	return c.PostContext(context.Background(), url)
}

// PostContext 发起带 context 的 POST 请求。
func (c *Curl) PostContext(ctx context.Context, url string) (err error) {
	url = buildURLWithEncodedParams(url, c.encodedQueryParams())
	return c.SendContext(ctx, http.MethodPost, url, c.body)
}

// PostForm 发起 POST Form 请求。
func (c *Curl) PostForm(url string) error {
	return c.PostFormContext(context.Background(), url)
}

// PostFormContext 发起带 context 的 POST Form 请求。
func (c *Curl) PostFormContext(ctx context.Context, url string) error {
	return c.SetContentType("application/x-www-form-urlencoded").
		SendContext(ctx, http.MethodPost, url, strings.NewReader(c.encodedQueryParams()))
}

// Put 发起 PUT 请求。
func (c *Curl) Put(url string) (err error) {
	return c.PutContext(context.Background(), url)
}

// PutContext 发起带 context 的 PUT 请求。
func (c *Curl) PutContext(ctx context.Context, url string) (err error) {
	url = buildURLWithEncodedParams(url, c.encodedQueryParams())
	return c.SendContext(ctx, http.MethodPut, url, c.body)
}

// Patch 发起 PATCH 请求。
func (c *Curl) Patch(url string) (err error) {
	return c.PatchContext(context.Background(), url)
}

// PatchContext 发起带 context 的 PATCH 请求。
func (c *Curl) PatchContext(ctx context.Context, url string) (err error) {
	url = buildURLWithEncodedParams(url, c.encodedQueryParams())
	return c.SendContext(ctx, http.MethodPatch, url, c.body)
}

// Head 发起 HEAD 请求。
func (c *Curl) Head(url string) error {
	return c.HeadContext(context.Background(), url)
}

// HeadContext 发起带 context 的 HEAD 请求。
func (c *Curl) HeadContext(ctx context.Context, url string) error {
	return c.SendContext(ctx, http.MethodHead, url, nil)
}

// Delete 发起 DELETE 请求。
func (c *Curl) Delete(url string) (err error) {
	return c.DeleteContext(context.Background(), url)
}

// DeleteContext 发起带 context 的 DELETE 请求。
func (c *Curl) DeleteContext(ctx context.Context, url string) (err error) {
	url = buildURLWithEncodedParams(url, c.encodedQueryParams())
	return c.SendContext(ctx, http.MethodDelete, url, c.body)
}

// Options 发起 OPTIONS 请求。
func (c *Curl) Options(url string) (err error) {
	return c.OptionsContext(context.Background(), url)
}

// OptionsContext 发起带 context 的 OPTIONS 请求。
func (c *Curl) OptionsContext(ctx context.Context, url string) (err error) {
	url = buildURLWithEncodedParams(url, c.encodedQueryParams())
	return c.SendContext(ctx, http.MethodOptions, url, c.body)
}

// ============================ 内部辅助函数 ============================

// generateUniqueID 生成指定长度的请求唯一 ID。
func generateUniqueID(length int) string {
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

// cloneHTTPClient 复制 http.Client，并复用已有 Transport 连接池。
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

// cloneHeader 复制请求头，避免克隆实例之间共享可变 map。
func cloneHeader(header http.Header) http.Header {
	if header == nil {
		return make(http.Header)
	}
	return header.Clone()
}

// cloneURLValues 复制 URL 参数，避免克隆实例之间共享可变 map。
func cloneURLValues(values url.Values) url.Values {
	if values == nil {
		return make(url.Values)
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
		return make(map[string]*http.Cookie)
	}
	cloned := make(map[string]*http.Cookie, len(cookies))
	for name, cookie := range cookies {
		if cookie == nil {
			continue
		}
		cp := *cookie
		cloned[name] = &cp
	}
	return cloned
}

// cloneCurlBody 复制可安全重放的请求体。
// 对于流式或不可回放的请求体，返回错误，避免多个实例共享同一读取游标。
func cloneCurlBody(body io.Reader) (io.Reader, error) {
	switch v := body.(type) {
	case nil:
		return nil, nil
	case *bytes.Buffer:
		return bytes.NewReader(append([]byte(nil), v.Bytes()...)), nil
	case *bytes.Reader:
		return cloneReadSeeker(v)
	case io.ReadSeeker:
		return cloneReadSeeker(v)
	default:
		return nil, errors.Errorf("Clone() 不支持复制不可重放的请求体类型: %T", body)
	}
}

// cloneReadSeeker 将可回放 Reader 复制为独立的 bytes.Reader。
func cloneReadSeeker(reader io.ReadSeeker) (io.Reader, error) {
	if reader == nil {
		return nil, nil
	}
	data, err := snapshotReadSeeker(reader)
	if err != nil {
		return nil, errors.Tag(err)
	}
	return bytes.NewReader(data), nil
}

// buildURL 构建完整的 URL，将 params 追加为查询参数。
func buildURL(baseURL string, params url.Values) string {
	if len(params) == 0 {
		return baseURL
	}
	return buildURLWithEncodedParams(baseURL, params.Encode())
}

// buildURLWithEncodedParams 构建完整 URL，并复用调用方已经编码好的查询串。
// URL 合法性由 http.NewRequest 统一校验，避免每次追加参数时重复解析。
func buildURLWithEncodedParams(baseURL, encodedParams string) string {
	if encodedParams == "" {
		return baseURL
	}
	return appendEncodedQuery(baseURL, encodedParams)
}

// appendEncodedQuery 将已经编码好的查询参数追加到 URL 中。
//
// buildURL 的新增参数来自 url.Values.Encode，已经完成转义。
// 原 URL 的 query 不重新解析，避免在每次请求前重复分配 map 和切片。
// fragment 必须保留在 URL 最末尾，因此追加点位于 # 之前。
func appendEncodedQuery(baseURL, encodedParams string) string {
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
