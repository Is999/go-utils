package utils

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ensureHeader 延迟分配请求头，构造只读模板时可保持 nil。
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

// ensureParams 延迟分配参数映射，未配置参数时保持 nil。
func (c *Curl) ensureParams() url.Values {
	if c.params == nil {
		c.params = make(url.Values)
	}
	return c.params
}

// Header 返回可直接修改的请求头映射，并补齐默认 Content-Type。
// 请求 ID 尚未生成时会创建；已生成后不重新添加被调用方删除的 ID 头。
func (c *Curl) Header() http.Header {
	if c.requestID == "" {
		c.SetRequestID()
	}
	return c.ensureDefaultContentType()
}

// GetHeaderValues 返回指定请求头的值切片，与内部配置共享底层数据。
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

// SetHeaders 覆盖给定请求头的值，不清空其他键。
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

// ResetHeaders 用传入映射替换请求头，不做复制；nil 清空现有映射。
// 默认 Content-Type 仍会在 Header 或发送请求时补齐。
func (c *Curl) ResetHeaders(header http.Header) *Curl {
	if header == nil {
		clear(c.header)
		return c
	}
	c.header = header
	return c
}

// Params 返回可直接修改的参数映射；请求发送时读取当前值，不缓存编码结果。
func (c *Curl) Params() url.Values {
	return c.ensureParams()
}

// GetParamValues 返回指定参数的值切片，与内部配置共享底层数据。
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

// SetParams 覆盖给定参数的值，不清空其他键。
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
		for _, value := range list {
			paramsValues.Add(key, value)
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

// ResetParams 用传入映射替换参数，不做复制；nil 清空现有映射。
func (c *Curl) ResetParams(params url.Values) *Curl {
	if params == nil {
		clear(c.params)
		return c
	}
	c.params = params
	return c
}

// SetBody 保存请求体引用，不立即读取；重放与关闭归属见 SendContext。
func (c *Curl) SetBody(body io.Reader) *Curl {
	c.body = body
	return c
}

// SetBodyBytes 使用传入切片作为可重放请求体，不复制数据；发送期间不得修改该切片。
func (c *Curl) SetBodyBytes(body []byte) *Curl {
	c.body = bytes.NewReader(body)
	return c
}

// GetCookie 返回配置中的 Cookie 指针；不存在时返回 nil，修改会影响后续请求。
func (c *Curl) GetCookie(cookieName string) *http.Cookie {
	return c.cookies[cookieName]
}

// HasCookie 检查是否设置了给定名称的 Cookie。
func (c *Curl) HasCookie(cookieName string) bool {
	_, ok := c.cookies[cookieName]
	return ok
}

// SetCookies 替换全部 Cookie；按名称保存传入指针，忽略 nil。
func (c *Curl) SetCookies(cookies ...*http.Cookie) *Curl {
	c.cookies = make(map[string]*http.Cookie, len(cookies))
	c.AddCookies(cookies...)
	return c
}

// AddCookies 按名称添加 Cookie 指针，同名项覆盖原值，nil 项被忽略。
func (c *Curl) AddCookies(cookies ...*http.Cookie) *Curl {
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		// 首个有效项才初始化映射，空参数和全 nil 输入保持原状态。
		if c.cookies == nil {
			c.cookies = make(map[string]*http.Cookie, len(cookies))
		}
		c.cookies[cookie.Name] = cookie
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

// SetTimeout 设置单次 Client.Do 超时，单位：秒；0 在发送时回退为 30 秒。
func (c *Curl) SetTimeout(timeout uint16) *Curl {
	c.timeout = time.Duration(timeout) * time.Second
	return c
}

// SetContentType 设置请求媒体类型；multipart 的 boundary 应使用 Form.Reader 返回的值。
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
// 用户名或密码可以为空；两者均为空时不自动设置认证头。
func (c *Curl) SetBasicAuth(username, password string) *Curl {
	c.username = username
	c.password = password
	return c
}

// SetProxyURL 设置代理地址，在下一次发送时重建 Transport；空值沿用默认 Transport 的代理规则。
func (c *Curl) SetProxyURL(proxyURL string) *Curl {
	c.proxyURL = proxyURL
	c.transportDirty = true
	return c
}

// InsecureSkipVerify 设置是否跳过服务端证书链和主机名校验，下一次发送时应用。
func (c *Curl) InsecureSkipVerify(isSkip bool) *Curl {
	c.insecureSkipVerify = isSkip
	c.transportDirty = true
	return c
}

// SetRootCAs 设置 PEM 根证书文件路径，下一次发送时加载；证书池规则见 RootCAs。
func (c *Curl) SetRootCAs(rootCAs string) *Curl {
	c.rootCAs = rootCAs
	c.transportDirty = true
	return c
}

// SetCertKey 设置客户端证书和私钥路径，两者均非空时在下一次发送中加载。
func (c *Curl) SetCertKey(cert, key string) *Curl {
	c.cert = cert
	c.key = key
	c.transportDirty = true
	return c
}

// SetStatusCode 替换 200 之外允许通过校验的状态码；空参数清空列表。
// 状态校验失败会直接返回错误，不执行响应回调，也不触发重试。
func (c *Curl) SetStatusCode(statusCode ...int) *Curl {
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

// SetDump 选择包含请求/响应头的日志格式，正文仍按 dumpBodyLimit 截断。
// 仅在默认日志已开启且 Logger 允许 Info 时生效。
func (c *Curl) SetDump(dump bool) *Curl {
	c.dump = dump
	return c
}
