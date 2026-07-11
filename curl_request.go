package utils

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ============================ 请求构建方法 ============================

// ensureHeader 返回可写请求头映射。
// Header 只有在构造默认头、用户设置头或发送请求时才需要分配；懒初始化可降低 NewCurl 热路径开销。
func (c *Curl) ensureHeader() http.Header {
	if c.header == nil {
		c.header = make(http.Header, 2)
	}
	return c.header
}

// ensureDefaultContentType 返回已补齐默认 Content-Type 的请求头映射。
// 默认值只在读取 Header 或发送请求前写入，避免只创建 Curl 模板时分配 Header map。
func (c *Curl) ensureDefaultContentType() http.Header {
	header := c.ensureHeader()
	if header.Get(curlHeaderContentType) == "" {
		header.Set(curlHeaderContentType, defaultCurlContentType)
	}
	return header
}

// ensureParams 返回可写查询参数映射。
// 参数数据来源于 SetParam/AddParam/PostForm 等调用；未使用参数能力时保持 nil，避免空 map 分配。
func (c *Curl) ensureParams() url.Values {
	if c.params == nil {
		c.params = make(url.Values)
	}
	return c.params
}

// ensureCookies 返回可写 Cookie 映射。
// Cookie 数据来源于 SetCookies/AddCookies；capacity 用于批量设置时预留容量，减少扩容。
func (c *Curl) ensureCookies(capacity int) map[string]*http.Cookie {
	if c.cookies == nil {
		if capacity > 0 {
			c.cookies = make(map[string]*http.Cookie, capacity)
		} else {
			c.cookies = make(map[string]*http.Cookie)
		}
	}
	return c.cookies
}

// Header 获取当前请求头配置。
func (c *Curl) Header() http.Header {
	// 调用方读取完整 Header 时补齐懒生成的请求 ID，保持构造后可观察到链路头的兼容行为。
	if c.requestID == "" {
		c.SetRequestID()
	}
	return c.ensureDefaultContentType()
}

// GetHeaderValues 获取请求头中指定键的所有值。
func (c *Curl) GetHeaderValues(key string) []string {
	return c.header.Values(key)
}

// HasHeader 检查请求头中是否设置了指定的键。
func (c *Curl) HasHeader(key string) bool {
	return c.header.Values(key) != nil
}

// SetHeader 设置单个请求头键值对。
func (c *Curl) SetHeader(key, value string) *Curl {
	c.ensureHeader().Set(key, value)
	return c
}

// SetHeaders 批量设置请求头。
func (c *Curl) SetHeaders(headers map[string]string) *Curl {
	if len(headers) == 0 {
		return c
	}
	header := c.ensureHeader()
	for key, value := range headers {
		header.Set(key, value)
	}
	return c
}

// AddHeader 对请求头键添加多个值。
func (c *Curl) AddHeader(key string, values ...string) *Curl {
	header := c.ensureHeader()
	for _, value := range values {
		header.Add(key, value)
	}
	return c
}

// AddHeaders 批量添加请求头值。
func (c *Curl) AddHeaders(headers map[string][]string) *Curl {
	for key, values := range headers {
		if len(values) > 0 {
			c.AddHeader(key, values...)
		}
	}
	return c
}

// DeleteHeaders 删除指定的请求头。
func (c *Curl) DeleteHeaders(keys ...string) *Curl {
	for _, key := range keys {
		c.header.Del(key)
	}
	return c
}

// ResetHeaders 重置请求头。
// 如果 header 为 nil，清空所有现有请求头；否则替换为新的请求头。
func (c *Curl) ResetHeaders(header http.Header) *Curl {
	if header == nil {
		clear(c.header)
		return c
	}
	c.header = header
	return c
}

// Params 获取当前 URL 查询参数。
func (c *Curl) Params() url.Values {
	return c.ensureParams()
}

// GetParamValues 获取查询参数中指定键的值。
func (c *Curl) GetParamValues(key string) []string {
	return c.params[key]
}

// HasParam 检查查询参数中是否设置了给定的键。
func (c *Curl) HasParam(key string) bool {
	return c.params.Has(key)
}

// SetParam 设置单个查询参数。
func (c *Curl) SetParam(key, value string) *Curl {
	c.ensureParams().Set(key, value)
	return c
}

// SetParams 批量设置查询参数。
func (c *Curl) SetParams(params map[string]string) *Curl {
	if len(params) == 0 {
		return c
	}
	values := c.ensureParams()
	for key, value := range params {
		values.Set(key, value)
	}
	return c
}

// AddParam 对查询参数键添加多个值。
func (c *Curl) AddParam(key string, values ...string) *Curl {
	params := c.ensureParams()
	for _, value := range values {
		params.Add(key, value)
	}
	return c
}

// AddParams 批量添加查询参数值。
func (c *Curl) AddParams(params map[string][]string) *Curl {
	if len(params) == 0 {
		return c
	}
	paramsValues := c.ensureParams()
	for key, list := range params {
		if len(list) > 0 {
			for _, value := range list {
				paramsValues.Add(key, value)
			}
		}
	}
	return c
}

// DeleteParams 删除指定的查询参数。
func (c *Curl) DeleteParams(keys ...string) *Curl {
	for _, key := range keys {
		c.params.Del(key)
	}
	return c
}

// ResetParams 重置查询参数。
// 如果 params 为 nil，清空所有现有参数；否则替换为新的参数。
func (c *Curl) ResetParams(params url.Values) *Curl {
	if params == nil {
		clear(c.params)
		return c
	}
	c.params = params
	return c
}

// SetBody 设置请求体。
func (c *Curl) SetBody(body io.Reader) *Curl {
	c.body = body
	return c
}

// SetBodyBytes 设置请求体字节。
// 内部会将字节数组包装为 bytes.Reader。
func (c *Curl) SetBodyBytes(body []byte) *Curl {
	c.body = bytes.NewReader(body)
	return c
}

// GetCookie 获取指定名称的 Cookie。
func (c *Curl) GetCookie(cookieName string) *http.Cookie {
	return c.cookies[cookieName]
}

// HasCookie 检查是否设置了给定名称的 Cookie。
func (c *Curl) HasCookie(cookieName string) bool {
	_, ok := c.cookies[cookieName]
	return ok
}

// SetCookies 设置 Cookie 列表。
// 会先清空现有 Cookie，再添加新的 Cookie。
func (c *Curl) SetCookies(cookies ...*http.Cookie) *Curl {
	c.cookies = make(map[string]*http.Cookie, len(cookies))
	c.AddCookies(cookies...)
	return c
}

// AddCookies 添加 Cookie 列表。
// 不会清空现有 Cookie，而是增量添加。
func (c *Curl) AddCookies(cookies ...*http.Cookie) *Curl {
	var cookieMap map[string]*http.Cookie
	for _, cookie := range cookies {
		if cookie != nil {
			if cookieMap == nil {
				cookieMap = c.ensureCookies(len(cookies))
			}
			cookieMap[cookie.Name] = cookie
		}
	}
	return c
}

// DeleteCookies 删除指定的 Cookie。
func (c *Curl) DeleteCookies(cookieName ...string) *Curl {
	for _, name := range cookieName {
		delete(c.cookies, name)
	}
	return c
}

// ClearCookies 清空所有 Cookie。
func (c *Curl) ClearCookies() *Curl {
	clear(c.cookies)
	return c
}

// SetTimeout 设置超时时间（秒）。
func (c *Curl) SetTimeout(timeout uint16) *Curl {
	c.timeout = time.Duration(timeout) * time.Second
	return c
}

// SetContentType 设置请求头 Content-Type。
// 常见类型：
//   - "application/x-www-form-urlencoded"
//   - "application/json"
//   - "multipart/form-data"
//   - "text/plain"
func (c *Curl) SetContentType(contentType string) *Curl {
	c.ensureHeader().Set(curlHeaderContentType, contentType)
	return c
}

// SetUserAgent 设置请求头 User-Agent。
func (c *Curl) SetUserAgent(userAgent string) *Curl {
	c.ensureHeader().Set("User-Agent", userAgent)
	return c
}

// SetBasicAuth 设置 Basic 认证的账号及密码。
func (c *Curl) SetBasicAuth(username, password string) *Curl {
	c.username = username
	c.password = password
	return c
}

// SetProxyURL 设置代理地址。
func (c *Curl) SetProxyURL(proxyURL string) *Curl {
	c.proxyURL = proxyURL
	c.transportDirty = true
	return c
}

// InsecureSkipVerify 设置是否跳过 HTTPS 不安全验证。
// 注意：生产环境禁止使用，存在安全风险。
func (c *Curl) InsecureSkipVerify(isSkip bool) *Curl {
	c.insecureSkipVerify = isSkip
	c.transportDirty = true
	return c
}

// SetRootCAs 设置根证书。
func (c *Curl) SetRootCAs(rootCAs string) *Curl {
	c.rootCAs = rootCAs
	c.transportDirty = true
	return c
}

// SetCertKey 设置客户端证书和私钥。
func (c *Curl) SetCertKey(cert, key string) *Curl {
	c.cert = cert
	c.key = key
	c.transportDirty = true
	return c
}

// SetStatusCode 设置可接受的状态码列表。
// 当响应状态码不在列表中且不是 200 时，将返回错误。
func (c *Curl) SetStatusCode(statusCode ...int) *Curl {
	if len(statusCode) == 0 {
		c.statusCode = c.statusCode[:0]
		return c
	}
	c.statusCode = append(c.statusCode[:0], statusCode...)
	return c
}

// GetStatusCode 获取当前允许通过校验的状态码列表副本。
func (c *Curl) GetStatusCode() []int {
	return append([]int(nil), c.statusCode...)
}

// SetMaxRetry 设置最大请求尝试次数。
// max 包含首次请求；max=0 或 max=1 表示不额外重试，最大不超过 5。
func (c *Curl) SetMaxRetry(max uint8) *Curl {
	c.maxRetry = max
	return c
}

// SetDump 设置是否开启 dump 模式。
// dump 模式会详细打印请求和响应的信息。
func (c *Curl) SetDump(dump bool) *Curl {
	c.dump = dump
	return c
}
