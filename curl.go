package utils

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/Is999/go-utils/errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

type Curl struct {
	cli *http.Client

	// 请求头
	header http.Header

	// 请求超时时间：秒
	timeout time.Duration

	// Basic认证: 账号, 密码
	username, password string

	// 代理地址
	proxyURL string

	// 跳过https不安全验证
	insecureSkipVerify bool

	// 证书: rootCAs 根证书， cert 证书， key 秘钥
	rootCAs, cert, key string

	// Cookie
	cookies map[string]*http.Cookie

	// 请求参数：url地址后的参数或postForm参数，其它类型参数请使用body
	params url.Values

	// 请求body
	body io.Reader

	// 返回状态码，除200以外需特殊处理的状态码
	statusCode []int

	// 在发送请求之前可以对Request处理方法
	request func(request *http.Request) error

	// 在发送请求之前可以对Client处理方法
	client func(client *http.Client) error

	// 在发送请求之后可以对Response处理方法
	//	isDone 返回true 终止执行调用该方法之后的代码; 返回false 继续执行后续代码
	response func(response *http.Response) (isDone bool, err error)

	// 在发送请求之后可以对Response.Body处理方法
	resolve func(body []byte) error

	// 请求完成后的处理方法, 如关闭连接等操作。注意：client、request、response 有可能为nil
	done func(client *http.Client, request *http.Request, response *http.Response)

	// 请求标识
	requestId string

	// 失败重连次数: 默认2次，最大5次
	maxRetry uint8

	// dump 模式：使用httputil包下的 DumpRequestOut, DumpResponse 记录请求和响应的详细信息
	dump bool

	// 打印默认日志（INFO及以下级别日志: true 打印， false 禁止打印
	defLogOutput bool

	// 日志
	Logger *slog.Logger
}

func NewCurl() *Curl {
	c := &Curl{
		cli:                &http.Client{},
		header:             make(http.Header),
		timeout:            30 * time.Second,
		username:           "",
		password:           "",
		proxyURL:           "",
		insecureSkipVerify: false,
		rootCAs:            "",
		cert:               "",
		key:                "",
		cookies:            make(map[string]*http.Cookie, 0),
		params:             make(url.Values),
		body:               nil,
		statusCode:         make([]int, 0),
		request:            nil,
		client:             nil,
		response:           nil,
		resolve:            nil,
		requestId:          "",
		maxRetry:           2,
		dump:               false,
		defLogOutput:       false,
	}
	// set Content-Type
	c.SetContentType("application/json")

	// set X-Request-Id Or Logger
	c.SetRequestId()

	return c
}

// SetDefLogOutput 打印默认日志（INFO及以下级别日志: true 打印， false 禁止打印
func (c *Curl) SetDefLogOutput(enable bool) *Curl {
	c.defLogOutput = enable
	return c
}

// GetHeader 获取header
func (c *Curl) GetHeader() http.Header {
	return c.header
}

// GetHeaderValues 获取header内指定键的值
func (c *Curl) GetHeaderValues(key string) []string {
	return c.header.Values(key)
}

// HasHeader 检查header内是否设置了指定的键
func (c *Curl) HasHeader(key string) bool {
	return c.header.Values(key) != nil
}

// SetHeader 设置header键值对
func (c *Curl) SetHeader(key, value string) *Curl {
	c.header.Set(key, value)
	return c
}

// SetHeaders 设置header键值对
func (c *Curl) SetHeaders(headers map[string]string) *Curl {
	for key, value := range headers {
		c.SetHeader(key, value)
	}
	return c
}

// AddHeader 对header键添加多个值
func (c *Curl) AddHeader(key string, values ...string) *Curl {
	for _, value := range values {
		c.header.Add(key, value)
	}
	return c
}

// AddHeaders 对header键添加多个值
func (c *Curl) AddHeaders(Headers map[string][]string) *Curl {
	for key, values := range Headers {
		if values != nil && len(values) > 0 {
			c.AddHeader(key, values...)
		}
	}
	return c
}

// DelHeaders 删除header键值
func (c *Curl) DelHeaders(keys ...string) {
	for _, key := range keys {
		c.header.Del(key)
	}
}

// ReSetHeader 重置header
func (c *Curl) ReSetHeader(header http.Header) *Curl {
	if header == nil {
		for key := range c.header {
			c.header.Del(key)
		}
		return c
	}
	c.header = header
	return c
}

// GetParams 获取params
func (c *Curl) GetParams() url.Values {
	return c.params
}

// GetParamValues 获取params指定键的值
func (c *Curl) GetParamValues(key string) []string {
	return c.params[key]
}

// HasParam 检查params是否设置了给定的key
func (c *Curl) HasParam(key string) bool {
	return c.params.Has(key)
}

// SetParam 设置params键值对
func (c *Curl) SetParam(key, value string) *Curl {
	c.params.Set(key, value)
	return c
}

// SetParams 设置params键值对
func (c *Curl) SetParams(params map[string]string) *Curl {
	for key, value := range params {
		c.SetParam(key, value)
	}
	return c
}

// AddParam 对params键添加多个值
func (c *Curl) AddParam(key string, values ...string) *Curl {
	for _, value := range values {
		c.params.Add(key, value)
	}
	return c
}

// AddParams 对params键添加多个值
func (c *Curl) AddParams(params map[string][]string) *Curl {
	for key, values := range params {
		if values != nil && len(values) > 0 {
			c.AddParam(key, values...)
		}
	}
	return c
}

// DelParams 删除params键值
func (c *Curl) DelParams(keys ...string) {
	for _, key := range keys {
		c.params.Del(key)
	}
}

// ReSetParams 重置params
func (c *Curl) ReSetParams(params url.Values) *Curl {
	if params == nil {
		for key := range c.params {
			c.params.Del(key)
		}
		return c
	}
	c.params = params
	return c
}

// SetBody 设置请求体
func (c *Curl) SetBody(body io.Reader) *Curl {
	c.body = body
	return c
}

// SetBodyBytes 设置请求体
func (c *Curl) SetBodyBytes(body []byte) *Curl {
	c.body = bytes.NewReader(body)
	return c
}

// GetCookie 获取cookies指定键的值
func (c *Curl) GetCookie(cookieName string) *http.Cookie {
	return c.cookies[cookieName]
}

// HasCookie 检查cookies是否设置了给定的cookie名
func (c *Curl) HasCookie(cookieName string) bool {
	_, ok := c.cookies[cookieName]
	return ok
}

// SetCookies 设置cookies
func (c *Curl) SetCookies(cookies ...*http.Cookie) *Curl {
	c.cookies = make(map[string]*http.Cookie, len(cookies))
	c.AddCookies(cookies...)
	return c
}

// AddCookies 设置cookies
func (c *Curl) AddCookies(cookies ...*http.Cookie) *Curl {
	for _, cookie := range cookies {
		c.cookies[cookie.Name] = cookie
	}
	return c
}

// DelCookies 删除cookie
func (c *Curl) DelCookies(cookieName ...string) {
	for _, name := range cookieName {
		delete(c.cookies, name)
	}
}

// ClearCookies 清空cookie
func (c *Curl) ClearCookies() {
	for name := range c.cookies {
		delete(c.cookies, name)
	}
}

// SetTimeout 设置超时时间
func (c *Curl) SetTimeout(timeout uint8) *Curl {
	c.timeout = time.Duration(timeout) * time.Second
	return c
}

// SetContentType 设置请求头 Content-Type
//
//	contentType 类型："application/x-www-form-urlencoded", "application/json", "multipart/form-data", "text/plain"
func (c *Curl) SetContentType(contentType string) *Curl {
	c.header.Set("Content-Type", contentType)
	return c
}

// SetUserAgent 设置请求头 User-Agent
func (c *Curl) SetUserAgent(userAgent string) *Curl {
	c.header.Set("User-Agent", userAgent)
	return c
}

// SetBasicAuth 设置Basic认证的账号及密码
func (c *Curl) SetBasicAuth(username, password string) *Curl {
	c.username = username
	c.password = password
	return c
}

// SetProxyURL 设置代理地址
func (c *Curl) SetProxyURL(proxyURL string) *Curl {
	c.proxyURL = proxyURL
	return c
}

// InsecureSkipVerify 设置是否跳过https不安全验证
func (c *Curl) InsecureSkipVerify(isSkip bool) *Curl {
	c.insecureSkipVerify = isSkip
	return c
}

// SetRootCAs 设置根证书
func (c *Curl) SetRootCAs(rootCAs string) *Curl {
	c.rootCAs = rootCAs
	return c
}

// SetCertKey 设置证书和秘钥
func (c *Curl) SetCertKey(cert, key string) *Curl {
	c.cert = cert
	c.key = key
	return c
}

// SetStatusCode 设置返回状态码，除200以外需特殊处理的状态码
func (c *Curl) SetStatusCode(statusCode ...int) *Curl {
	if len(statusCode) > 0 {
		c.statusCode = append(c.statusCode, statusCode...)
	}
	return c
}

// SetRequestId 设置RequestId
func (c *Curl) SetRequestId(requestId ...string) *Curl {
	if len(requestId) == 0 || strings.TrimSpace(requestId[0]) == "" {
		c.requestId = UniqId(16)
	} else {
		c.requestId = strings.TrimSpace(requestId[0])
	}

	// 设置日志X-Request-Id
	c.Logger = slog.With("X-Request-Id", c.requestId)

	// 设置 header X-Request-Id
	c.SetHeader("X-Request-Id", c.requestId)
	return c
}

// GetRequestId 获取请求ID
func (c *Curl) GetRequestId() string {
	return c.requestId
}

// SetMaxRetry 设置失败重连次数
func (c *Curl) SetMaxRetry(max uint8) *Curl {
	c.maxRetry = max
	return c
}

// SetDump dump模式会详细打印请求和响应的信息，否则只记录关键信息
func (c *Curl) SetDump(dump bool) *Curl {
	c.dump = dump
	return c
}

// Request 在发送请求之前可以对Request处理方法
func (c *Curl) Request(f func(request *http.Request) error) *Curl {
	c.request = f
	return c
}

// Client 在发送请求之前可以对Client处理方法
func (c *Curl) Client(f func(request *http.Client) error) *Curl {
	c.client = f
	return c
}

// Response 在发送请求之后可以对Response处理方法
//
//	isDone 返回true 终止调用该方法之后的代码; 返回false 继续执行后续代码
func (c *Curl) Response(f func(response *http.Response) (isDone bool, err error)) *Curl {
	c.response = f
	return c
}

// Resolve 在发送请求之后可以对Response.Body处理方法
func (c *Curl) Resolve(f func(body []byte) error) *Curl {
	c.resolve = f
	return c
}

// Done 请求完成后的处理方法, 如关闭连接等操作。 注意：client、request、response 有可能为nil
func (c *Curl) Done(f func(client *http.Client, request *http.Request, response *http.Response)) *Curl {
	c.done = f
	return c
}

// Send 发起请求
//
//	method 请求方式：GET, POST, PUT, DELETE, PATCH, HEAD
//	url 请求地址
//	body 请求体
func (c *Curl) Send(method, url string, body io.Reader) (err error) {
	t := time.Now()

	// 设置 requestId
	if c.requestId == "" {
		c.SetRequestId()
	}

	// Debug 日志
	if c.defLogOutput {
		c.Logger.Debug("HTTP START", "time", t.Format(MicrosecondDash))
	}

	var (
		req  *http.Request
		resp *http.Response
	)

	// 请求完成后的处理方法, 如关闭连接等操作。注意：Client、Request、Response 有可能为nil
	defer func() {
		// 关闭 Response.Body
		defer func() {
			if resp == nil || resp.Body == nil || resp.Body == http.NoBody {
				return
			}

			// Debug 日志
			if c.defLogOutput {
				c.Logger.Debug("Close Response Body")
			}

			if err := resp.Body.Close(); err != nil {
				c.Logger.Error("Body.Close()", "err", err.Error()) // Error 日志
			}
		}()

		// 执行 done
		if c.done != nil {
			// Debug 日志
			if c.defLogOutput {
				c.Logger.Debug("done()")
			}

			c.done(c.cli, req, resp)
		}
	}()

	// 实例 Request
	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return errors.Wrap(err)
	}

	// 设置 header
	if c.header != nil && len(c.header) > 0 {
		// Debug 日志
		if c.defLogOutput {
			c.Logger.Debug("set header")
		}

		req.Header = c.header
	}

	// 设置 Cookie
	if c.cookies != nil && len(c.cookies) > 0 {
		// Debug 日志
		if c.defLogOutput {
			c.Logger.Debug("AddCookie()")
		}

		for _, cookie := range c.cookies {
			req.AddCookie(cookie)
		}
	}

	// 设置 BasicAuth 认证
	if c.username != "" && c.password != "" {
		// Debug 日志
		if c.defLogOutput {
			c.Logger.Debug("SetBasicAuth()")
		}

		req.SetBasicAuth(c.username, c.password)
	}

	// 在发送请求之前对Request处理方法
	if c.request != nil {
		// Debug 日志
		if c.defLogOutput {
			c.Logger.Debug("request()")
		}

		if err = c.request(req); err != nil {
			return errors.Wrap(err)
		}
	}

	// 记录请求日志
	if c.defLogOutput && slog.Default().Enabled(context.Background(), slog.LevelInfo) {
		if c.dump {
			// 使用httputil.DumpRequestOut记录日志
			dump, err := httputil.DumpRequestOut(req, true)
			if err != nil {
				return errors.Wrap(err)
			}
			c.Logger.Info("httputil.DumpRequestOut()", "request", string(dump)) // Info 日志
		} else {
			// 只记录关键性日志
			var b strings.Builder
			b.WriteString(method + ": " + url + "\n")

			if req.Body != nil {
				b.WriteString("Request Body:\n")
				var reqBody []byte
				reqBody, req.Body, err = DrainBody(req.Body)
				if err != nil {
					return errors.Wrap(err)
				}
				b.Write(reqBody)
			}
			c.Logger.Info("DrainBody(req.Body)", "body", b.String()) // Info 日志
		}
	}

	// 如果Client未初始化进行初始化
	if c.cli == nil {
		// Debug 日志
		if c.defLogOutput {
			c.Logger.Debug("Init Client")
		}

		c.cli = &http.Client{}
	}

	// 设置超时时间
	c.cli.Timeout = Ternary(c.timeout > 0, c.timeout, 30*time.Second)

	// 初始化Transport
	if c.cli.Transport == nil {
		// Debug 日志
		if c.defLogOutput {
			c.Logger.Debug("Init Transport")
		}

		tr := &http.Transport{}

		// 设置代理
		if len(c.proxyURL) > 0 {
			// Debug 日志
			if c.defLogOutput {
				c.Logger.Debug("ProxyURL()")
			}

			if err = ProxyURL(tr, c.proxyURL); err != nil {
				return errors.Wrap(err)
			}
		}

		// 跳过https不安全验证
		if c.insecureSkipVerify {
			// Debug 日志
			if c.defLogOutput {
				c.Logger.Debug("InsecureSkipVerify")
			}

			if tr.TLSClientConfig == nil {
				tr.TLSClientConfig = &tls.Config{}
			}
			tr.TLSClientConfig.InsecureSkipVerify = true
		}

		// 根证书
		if len(c.rootCAs) > 0 {
			// Debug 日志
			if c.defLogOutput {
				c.Logger.Debug("RootCAs()")
			}

			if tr.TLSClientConfig == nil {
				tr.TLSClientConfig = &tls.Config{}
			}

			if err = RootCAs(tr.TLSClientConfig, c.rootCAs); err != nil {
				return errors.Wrap(err)
			}
		}

		// 证书
		if len(c.cert) > 0 && len(c.key) > 0 {
			// Debug 日志
			if c.defLogOutput {
				c.Logger.Debug("Certificate()")
			}

			if tr.TLSClientConfig == nil {
				tr.TLSClientConfig = &tls.Config{}
			}

			if err = Certificate(tr.TLSClientConfig, c.cert, c.key); err != nil {
				return errors.Wrap(err)
			}
		}

		c.cli.Transport = tr
	}

	// 在发送请求之前对Client处理方法
	if c.client != nil {
		// Debug 日志
		if c.defLogOutput {
			c.Logger.Debug("client()")
		}

		err = c.client(c.cli)
		if err != nil {
			return errors.Wrap(err)
		}
	}

	// 失败重连次数: 默认2次, 最大5次
	maxRetry := Ternary(c.maxRetry > 0, int(c.maxRetry), 2)
	maxRetry = Ternary(maxRetry > 5, 5, maxRetry)

	t1 := time.Now()
	// Debug 日志
	if c.defLogOutput {
		c.Logger.Debug("client start", "time", t1.Format(MicrosecondDash))
	}

	// 发送请求
	for i := 1; i <= maxRetry; i++ {
		resp, err = c.cli.Do(req)
		if err == nil {
			// 请求成功后终止
			break
		}

		if i < maxRetry {
			c.Logger.Warn("client.Do()", "maxRetry", maxRetry, "currentRetry", i, "err", err.Error()) // Warn 日志
			time.Sleep(time.Millisecond * time.Duration(2<<(2*i)))                                    // 间隔 8, 32, 128, 512 毫秒
		}
	}

	// Debug 日志
	if c.defLogOutput {
		c.Logger.Debug("client end", " time spent", time.Since(t1).String())
	}

	if err != nil {
		return errors.Errorf("client.Do() Retry %d times err: %v", maxRetry, err.Error())
	}

	// 返回body内容
	var respBody []byte

	// 记录返回日志
	if c.defLogOutput && slog.Default().Enabled(context.Background(), slog.LevelInfo) {
		if c.dump {
			// 使用httputil.DumpResponse记录返回信息
			dump, err := httputil.DumpResponse(resp, true)
			if err != nil {
				return errors.Wrap(err)
			}
			c.Logger.Info("httputil.DumpResponse()", "response", string(dump)) // Info 日志
		} else {
			// 只记录返回的关键信息
			var b strings.Builder
			b.WriteString(fmt.Sprintf("Response Status: %d %s\n", resp.StatusCode, resp.Status))
			b.WriteString("Response Body:\n")

			// 读取body内容
			respBody, resp.Body, err = DrainBody(resp.Body)
			if err != nil {
				return errors.Wrap(err)
			}
			b.Write(respBody)
			c.Logger.Info("DrainBody(resp.Body)", "Body", b.String()) // Info 日志
		}
	}

	// 判断状态码是否是200正常状态及已标记的状态码
	if resp.StatusCode != 200 && !IsHas(resp.StatusCode, c.statusCode) {
		return errors.Errorf("response error StatusCode: statusCode=%d, Status=%s", resp.StatusCode, resp.Status)
	}

	// 在发送请求之后可以对Response处理方法
	if c.response != nil {
		// Debug 日志
		if c.defLogOutput {
			c.Logger.Debug("response()")
		}

		isDone, err := c.response(resp)
		if err != nil {
			return errors.Wrap(err)
		}

		// isDone 返回true 终止执行后续代码; 返回false 继续执行后续代码
		if isDone {
			return nil
		}
	}

	// 在发送请求之后对Response.Body处理方法
	if c.resolve != nil {
		// Debug 日志
		if c.defLogOutput {
			c.Logger.Debug("resolve()")
		}

		if respBody == nil {
			// 读取body内容
			var buf bytes.Buffer
			//respBody, err := io.ReadAll(resp.Body)
			_, err = buf.ReadFrom(resp.Body)
			if err != nil {
				return errors.Wrap(err)
			}
			respBody = buf.Bytes()
		}

		if err = c.resolve(respBody); err != nil {
			return errors.Wrap(err)
		}
	}

	// Debug 日志
	if c.defLogOutput {
		c.Logger.Debug("HTTP END", "total time spent", time.Since(t).String())
	}

	return nil
}

// CloseIdleConnections 关闭连接
func (c *Curl) CloseIdleConnections() {
	if c.cli != nil {
		c.cli.CloseIdleConnections()
		c.cli.Transport = nil
	}
}

// Get 请求方式
func (c *Curl) Get(url string) (err error) {
	url, err = UrlPath(url, c.params)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.Send(http.MethodGet, url, c.body)
}

// Post 请求方式
func (c *Curl) Post(url string) (err error) {
	url, err = UrlPath(url, c.params)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.Send(http.MethodPost, url, c.body)
}

// PostForm 请求方式
func (c *Curl) PostForm(url string) error {
	return c.SetContentType("application/x-www-form-urlencoded").
		Send(http.MethodPost, url, strings.NewReader(c.params.Encode()))
}

// Put 请求方式
func (c *Curl) Put(url string) (err error) {
	url, err = UrlPath(url, c.params)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.Send(http.MethodPut, url, c.body)
}

// Patch 请求方式
func (c *Curl) Patch(url string) (err error) {
	url, err = UrlPath(url, c.params)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.Send(http.MethodPatch, url, c.body)
}

// Head 请求方式
func (c *Curl) Head(url string) error {
	return c.Send(http.MethodHead, url, nil)
}

// Delete 请求方式
func (c *Curl) Delete(url string) (err error) {
	url, err = UrlPath(url, c.params)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.Send(http.MethodDelete, url, c.body)
}

// Options 请求方式
func (c *Curl) Options(url string) (err error) {
	url, err = UrlPath(url, c.params)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.Send(http.MethodOptions, url, c.body)
}

// DrainBody 读取read内容并返回其内容和一个新的ReadCloser，
func DrainBody(b io.ReadCloser) ([]byte, io.ReadCloser, error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return nil, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(b); err != nil {
		return nil, b, errors.Wrap(err)
	}
	if err := b.Close(); err != nil {
		return nil, b, errors.Wrap(err)
	}
	return buf.Bytes(), io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

// RootCAs HTTPS请求证书
//
//	caCertPath 根证书
func RootCAs(config *tls.Config, rootCAs string) error {
	// 根证书
	cert, err := os.ReadFile(rootCAs)
	if err != nil {
		return errors.Wrap(err)
	}
	// 证书池
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(cert)

	config.RootCAs = certPool
	return nil
}

// Certificate HTTPS请求证书
//
//	certFile 证书
//	keyFile 秘钥
func Certificate(config *tls.Config, certFile, keyFile string) error {
	// 加载证书
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return errors.Wrap(err)
	}

	config.Certificates = []tls.Certificate{certificate}
	return nil
}

// ProxyURL 设置代理地址
func ProxyURL(transport *http.Transport, proxyURL string) error {
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return errors.Wrap(err)
	}
	transport.Proxy = http.ProxyURL(proxy)
	return nil
}

// Form 表单
type Form struct {
	Params url.Values
	Files  url.Values
}

// SetParam 设置Params键值对
func (f *Form) SetParam(key, value string) *Form {
	f.Params.Set(key, value)
	return f
}

// SetParams 设置Params键值对
func (f *Form) SetParams(params map[string]string) *Form {
	for key, value := range params {
		f.SetParam(key, value)
	}
	return f
}

// AddParam 对Params键添加多个值
func (f *Form) AddParam(key string, values ...string) *Form {
	for _, value := range values {
		f.Params.Add(key, value)
	}
	return f
}

// AddParams 对Params键添加多个值
func (f *Form) AddParams(params map[string][]string) *Form {
	for key, values := range params {
		if values != nil && len(values) > 0 {
			f.AddParam(key, values...)
		}
	}
	return f
}

// DelParams 删除Params键值
func (f *Form) DelParams(keys ...string) {
	for _, key := range keys {
		f.Params.Del(key)
	}
}

// SetFile 设置Files键值对
func (f *Form) SetFile(fileName, filePath string) *Form {
	f.Files.Set(fileName, filePath)
	return f
}

// SetFiles 设置Files键值对
func (f *Form) SetFiles(files map[string]string) *Form {
	for name, path := range files {
		f.SetFile(name, path)
	}
	return f
}

// AddFile 对Files键添加多个值
func (f *Form) AddFile(fileName string, filePath ...string) *Form {
	for _, path := range filePath {
		f.Files.Add(fileName, path)
	}
	return f
}

// AddFiles 对Files键添加多个值
func (f *Form) AddFiles(files map[string][]string) *Form {
	for name, paths := range files {
		if paths != nil && len(paths) > 0 {
			f.AddFile(name, paths...)
		}
	}
	return f
}

// DelFiles 删除Files键值
func (f *Form) DelFiles(fileNames ...string) {
	for _, name := range fileNames {
		f.Files.Del(name)
	}
}

// Reader 读取Form内容，转换为以键值对上传文件和表单的body及content-type
func (f *Form) Reader() (body io.Reader, contentType string, err error) {
	if f.Files == nil || len(f.Files) == 0 {
		return strings.NewReader(f.Params.Encode()), "application/x-www-form-urlencoded", nil
	}

	b := &bytes.Buffer{}

	// 创建一个multipart类型的写文件
	writer := multipart.NewWriter(b)

	// 处理表单
	if f.Params != nil {
		for key, values := range f.Params {
			if len(values) > 1 {
				for _, value := range values {
					if err := writer.WriteField(key, value); err != nil {
						return nil, "", errors.Wrap(err)
					}
				}
			} else {
				if err := writer.WriteField(key, values[0]); err != nil {
					return nil, "", errors.Wrap(err)
				}
			}
		}
	}

	// 创建要上传的文件
	createFormFile := func(writer *multipart.Writer, fieldName string, filePath string) error {
		//打开要上传的文件
		file, err := os.Open(filePath)
		if err != nil {
			return errors.Wrap(err)
		}
		defer file.Close()

		// 使用给出的属性名fieldName和文件名filePath创建一个新的form-data头
		part, err := writer.CreateFormFile(fieldName, filePath)
		if err != nil {
			return errors.Wrap(err)
		}

		_, err = io.Copy(part, file)
		return nil
	}

	// 处理文件
	for fieldName, files := range f.Files {
		if len(files) > 1 {
			for _, file := range files {
				if err := createFormFile(writer, fieldName, file); err != nil {
					return nil, "", errors.Wrap(err)
				}
			}
		} else {
			if err := createFormFile(writer, fieldName, files[0]); err != nil {
				return nil, "", errors.Wrap(err)
			}
		}
	}

	// 关闭
	if err := writer.Close(); err != nil {
		return nil, "", errors.Wrap(err)
	}

	return b, writer.FormDataContentType(), nil
}
