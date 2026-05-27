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
//
// 返回值：http.Header，可直接写入请求头键值。
func (c *Curl) ensureHeader() http.Header {
	if c.header == nil {
		c.header = make(http.Header, 2)
	}
	return c.header
}

// ensureDefaultContentType 返回已补齐默认 Content-Type 的请求头映射。
// 默认值只在读取 Header 或发送请求前写入，避免只创建 Curl 模板时分配 Header map。
//
// 返回值：http.Header，至少包含默认或用户自定义的 Content-Type。
func (c *Curl) ensureDefaultContentType() http.Header {
	header := c.ensureHeader()
	if header.Get(curlHeaderContentType) == "" {
		header.Set(curlHeaderContentType, defaultCurlContentType)
	}
	return header
}

// ensureParams 返回可写查询参数映射。
// 参数数据来源于 SetParam/AddParam/PostForm 等调用；未使用参数能力时保持 nil，避免空 map 分配。
//
// 返回值：url.Values，可直接写入查询参数。
func (c *Curl) ensureParams() url.Values {
	if c.params == nil {
		c.params = make(url.Values)
	}
	return c.params
}

// markParamsDirty 标记参数编码缓存失效。
// 业务意图：params 是可变 map，任意 Set/Add/Del 或暴露给调用方修改后，都必须让下次请求重新 Encode。
func (c *Curl) markParamsDirty() {
	c.paramsDirty = true
	c.encodedParams = ""
}

// encodedQueryParams 返回当前参数的 URL 编码结果。
// 缓存数据来源于 url.Values.Encode；参数未变更时复用上次结果，避免重复请求时反复排序与拼接。
func (c *Curl) encodedQueryParams() string {
	if len(c.params) == 0 {
		c.encodedParams = ""
		c.paramsDirty = false
		return ""
	}
	if !c.paramsDirty {
		return c.encodedParams
	}

	// encoded 是本次请求使用的稳定查询串；Encode 内部会按 key 排序，缓存后可保护重复请求热路径。
	encoded := c.params.Encode()
	c.encodedParams = encoded
	c.paramsDirty = false
	return encoded
}

// ensureCookies 返回可写 Cookie 映射。
// Cookie 数据来源于 SetCookies/AddCookies；capacity 用于批量设置时预留容量，减少扩容。
//
// 参数说明：
//   - capacity：预估 Cookie 数量，小于等于 0 时按默认容量创建。
//
// 返回值：Cookie 名称到 Cookie 对象的映射。
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

// GetHeader 获取当前请求头配置。
//
// 返回值：http.Header 请求头映射
func (c *Curl) GetHeader() http.Header {
	// 调用方读取完整 Header 时补齐懒生成的请求 ID，保持构造后可观察到链路头的兼容行为。
	if c.requestID == "" {
		c.SetRequestID()
	}
	return c.ensureDefaultContentType()
}

// GetHeaderValues 获取请求头中指定键的所有值。
//
// 参数说明：
//   - key：请求头键名
//
// 返回值：值列表
func (c *Curl) GetHeaderValues(key string) []string {
	return c.header.Values(key)
}

// HasHeader 检查请求头中是否设置了指定的键。
//
// 参数说明：
//   - key：请求头键名
//
// 返回值：true 表示已设置
func (c *Curl) HasHeader(key string) bool {
	return c.header.Values(key) != nil
}

// SetHeader 设置单个请求头键值对。
//
// 参数说明：
//   - key：请求头键名
//   - value：请求头键值
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetHeader(key, value string) *Curl {
	c.ensureHeader().Set(key, value)
	return c
}

// SetHeaders 批量设置请求头。
//
// 参数说明：
//   - headers：请求头键值对映射
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetHeaders(headers map[string]string) *Curl {
	for key, value := range headers {
		c.SetHeader(key, value)
	}
	return c
}

// AddHeader 对请求头键添加多个值。
//
// 参数说明：
//   - key：请求头键名
//   - values：多个值
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) AddHeader(key string, values ...string) *Curl {
	header := c.ensureHeader()
	for _, value := range values {
		header.Add(key, value)
	}
	return c
}

// AddHeaders 批量添加请求头值。
//
// 参数说明：
//   - headers：请求头映射，值为切片
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) AddHeaders(headers map[string][]string) *Curl {
	for key, values := range headers {
		if len(values) > 0 {
			c.AddHeader(key, values...)
		}
	}
	return c
}

// DelHeaders 删除指定的请求头。
//
// 参数说明：
//   - keys：可变数量的键名
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) DelHeaders(keys ...string) *Curl {
	if c.header == nil {
		return c
	}
	for _, key := range keys {
		c.header.Del(key)
	}
	return c
}

// ReSetHeader 重置请求头。
// 如果 header 为 nil，清空所有现有请求头；否则替换为新的请求头。
//
// 参数说明：
//   - header：新的请求头（可选）
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) ReSetHeader(header http.Header) *Curl {
	if header == nil {
		if c.header == nil {
			return c
		}
		for key := range c.header {
			c.header.Del(key)
		}
		return c
	}
	c.header = header
	return c
}

// GetParams 获取当前 URL 查询参数。
//
// 返回值：url.Values 查询参数映射
func (c *Curl) GetParams() url.Values {
	// 返回内部 map 后调用方可能直接修改，提前标脏以保证下一次发送请求不会复用旧编码结果。
	c.markParamsDirty()
	return c.ensureParams()
}

// GetParamValues 获取查询参数中指定键的值。
//
// 参数说明：
//   - key：参数键名
//
// 返回值：值列表
func (c *Curl) GetParamValues(key string) []string {
	return c.params[key]
}

// HasParam 检查查询参数中是否设置了给定的键。
//
// 参数说明：
//   - key：参数键名
//
// 返回值：true 表示已设置
func (c *Curl) HasParam(key string) bool {
	return c.params.Has(key)
}

// SetParam 设置单个查询参数。
//
// 参数说明：
//   - key：参数键名
//   - value：参数值
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetParam(key, value string) *Curl {
	c.ensureParams().Set(key, value)
	c.markParamsDirty()
	return c
}

// SetParams 批量设置查询参数。
//
// 参数说明：
//   - params：查询参数字典
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetParams(params map[string]string) *Curl {
	if len(params) == 0 {
		return c
	}
	values := c.ensureParams()
	for key, value := range params {
		values.Set(key, value)
	}
	c.markParamsDirty()
	return c
}

// AddParam 对查询参数键添加多个值。
//
// 参数说明：
//   - key：参数键名
//   - values：多个值
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) AddParam(key string, values ...string) *Curl {
	params := c.ensureParams()
	for _, value := range values {
		params.Add(key, value)
	}
	if len(values) > 0 {
		c.markParamsDirty()
	}
	return c
}

// AddParams 批量添加查询参数值。
//
// 参数说明：
//   - params：查询参数映射，值为切片
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) AddParams(params map[string][]string) *Curl {
	if len(params) == 0 {
		return c
	}
	paramsValues := c.ensureParams()
	changed := false
	for key, list := range params {
		if len(list) > 0 {
			for _, value := range list {
				paramsValues.Add(key, value)
			}
			changed = true
		}
	}
	if changed {
		c.markParamsDirty()
	}
	return c
}

// DelParams 删除指定的查询参数。
//
// 参数说明：
//   - keys：可变数量的键名
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) DelParams(keys ...string) *Curl {
	if len(c.params) == 0 || len(keys) == 0 {
		return c
	}
	for _, key := range keys {
		c.params.Del(key)
	}
	c.markParamsDirty()
	return c
}

// ReSetParams 重置查询参数。
// 如果 params 为 nil，清空所有现有参数；否则替换为新的参数。
//
// 参数说明：
//   - params：新的查询参数（可选）
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) ReSetParams(params url.Values) *Curl {
	if params == nil {
		if c.params == nil {
			return c
		}
		for key := range c.params {
			c.params.Del(key)
		}
		c.markParamsDirty()
		return c
	}
	c.params = params
	c.markParamsDirty()
	return c
}

// SetBody 设置请求体。
//
// 参数说明：
//   - body：io.Reader 类型的请求体
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetBody(body io.Reader) *Curl {
	c.body = body
	return c
}

// SetBodyBytes 设置请求体字节。
// 内部会将字节数组包装为 bytes.Reader。
//
// 参数说明：
//   - body：字节数组
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetBodyBytes(body []byte) *Curl {
	c.body = bytes.NewReader(body)
	return c
}

// GetCookie 获取指定名称的 Cookie。
//
// 参数说明：
//   - cookieName：Cookie 名称
//
// 返回值：*http.Cookie，不存在时返回 nil
func (c *Curl) GetCookie(cookieName string) *http.Cookie {
	return c.cookies[cookieName]
}

// HasCookie 检查是否设置了给定名称的 Cookie。
//
// 参数说明：
//   - cookieName：Cookie 名称
//
// 返回值：true 表示已设置
func (c *Curl) HasCookie(cookieName string) bool {
	_, ok := c.cookies[cookieName]
	return ok
}

// SetCookies 设置 Cookie 列表。
// 会先清空现有 Cookie，再添加新的 Cookie。
//
// 参数说明：
//   - cookies：可变数量的 Cookie
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetCookies(cookies ...*http.Cookie) *Curl {
	c.cookies = make(map[string]*http.Cookie, len(cookies))
	c.AddCookies(cookies...)
	return c
}

// AddCookies 添加 Cookie 列表。
// 不会清空现有 Cookie，而是增量添加。
//
// 参数说明：
//   - cookies：可变数量的 Cookie
//
// 返回值：Curl 指针，支持链式调用
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

// DelCookies 删除指定的 Cookie。
//
// 参数说明：
//   - cookieName：可变数量的 Cookie 名称
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) DelCookies(cookieName ...string) *Curl {
	for _, name := range cookieName {
		delete(c.cookies, name)
	}
	return c
}

// ClearCookies 清空所有 Cookie。
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) ClearCookies() *Curl {
	for name := range c.cookies {
		delete(c.cookies, name)
	}
	return c
}

// SetTimeout 设置超时时间（秒）。
//
// 参数说明：
//   - timeout：超时秒数
//
// 返回值：Curl 指针，支持链式调用
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
//
// 参数说明：
//   - contentType：Content-Type 值
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetContentType(contentType string) *Curl {
	c.ensureHeader().Set(curlHeaderContentType, contentType)
	return c
}

// SetUserAgent 设置请求头 User-Agent。
//
// 参数说明：
//   - userAgent：User-Agent 字符串
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetUserAgent(userAgent string) *Curl {
	c.ensureHeader().Set("User-Agent", userAgent)
	return c
}

// SetBasicAuth 设置 Basic 认证的账号及密码。
//
// 参数说明：
//   - username：账号
//   - password：密码
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetBasicAuth(username, password string) *Curl {
	c.username = username
	c.password = password
	return c
}

// SetProxyURL 设置代理地址。
//
// 参数说明：
//   - proxyURL：代理服务器 URL
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetProxyURL(proxyURL string) *Curl {
	c.proxyURL = proxyURL
	c.markTransportDirty()
	return c
}

// InsecureSkipVerify 设置是否跳过 HTTPS 不安全验证。
// 注意：生产环境禁止使用，存在安全风险。
//
// 参数说明：
//   - isSkip：true 跳过验证，false 进行验证
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) InsecureSkipVerify(isSkip bool) *Curl {
	c.insecureSkipVerify = isSkip
	c.markTransportDirty()
	return c
}

// SetRootCAs 设置根证书。
//
// 参数说明：
//   - rootCAs：根证书文件路径
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetRootCAs(rootCAs string) *Curl {
	c.rootCAs = rootCAs
	c.markTransportDirty()
	return c
}

// SetCertKey 设置客户端证书和私钥。
//
// 参数说明：
//   - cert：证书文件路径
//   - key：私钥文件路径
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetCertKey(cert, key string) *Curl {
	c.cert = cert
	c.key = key
	c.markTransportDirty()
	return c
}

// SetStatusCode 设置可接受的状态码列表。
// 当响应状态码不在列表中且不是 200 时，将返回错误。
//
// 参数说明：
//   - statusCode：可变数量的状态码
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetStatusCode(statusCode ...int) *Curl {
	if len(statusCode) == 0 {
		c.statusCode = c.statusCode[:0]
		return c
	}
	c.statusCode = append(c.statusCode[:0], statusCode...)
	return c
}

// GetStatusCode 获取当前允许通过校验的状态码列表副本。
//
// 返回值：状态码切片副本，调用方修改不会影响 Curl 内部状态。
func (c *Curl) GetStatusCode() []int {
	if len(c.statusCode) == 0 {
		return nil
	}
	statusCode := make([]int, len(c.statusCode))
	copy(statusCode, c.statusCode)
	return statusCode
}

// SetMaxRetry 设置最大请求尝试次数。
// max 包含首次请求；max=0 或 max=1 表示不额外重试，最大不超过 5。
//
// 参数说明：
//   - max：最大请求尝试次数
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetMaxRetry(max uint8) *Curl {
	c.maxRetry = max
	return c
}

// SetDump 设置是否开启 dump 模式。
// dump 模式会详细打印请求和响应的信息。
//
// 参数说明：
//   - dump：true 开启，false 关闭
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) SetDump(dump bool) *Curl {
	c.dump = dump
	return c
}
