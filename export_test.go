package utils

import (
	"io"
	"net/http"
	"net/url"
)

const (
	// DefaultJSONContentType 是测试用默认 JSON Content-Type 常量。
	DefaultJSONContentType = defaultJSONContentType
	// HeaderContentType 是测试用 Content-Type 响应头常量。
	HeaderContentType = headerContentType
	// HeaderContentDisposition 是测试用 Content-Disposition 响应头常量。
	HeaderContentDisposition = headerContentDisposition
	// HeaderLocation 是测试用 Location 响应头常量。
	HeaderLocation = headerLocation
)

// GenerateUniqID 是测试用请求 ID 生成函数包装。
func GenerateUniqID(length int) string {
	return generateUniqID(length)
}

// GenerateUniqId 是测试用请求 ID 生成函数包装。
//
// Deprecated: 请使用 GenerateUniqID。
func GenerateUniqId(length int) string {
	return GenerateUniqID(length)
}

// BuildURL 是测试用 URL 查询参数合并函数包装。
func BuildURL(baseURL string, params url.Values) (string, error) {
	return buildURL(baseURL, params)
}

// BuildUrl 是测试用 URL 查询参数合并函数包装。
//
// Deprecated: 请使用 BuildURL。
func BuildUrl(baseURL string, params url.Values) (string, error) {
	return BuildURL(baseURL, params)
}

// ReadBodyPreviewAndRestore 是测试用响应体预览与恢复函数包装。
func ReadBodyPreviewAndRestore(body io.ReadCloser, limit int64) ([]byte, bool, io.ReadCloser, error) {
	return readBodyPreviewAndRestore(body, limit)
}

// NormalizeContentType 是测试用 Content-Type 规范化函数包装。
func NormalizeContentType(contentType string) string {
	return normalizeContentType(contentType)
}

// NewResponse 是测试用响应构造函数包装。
func NewResponse(w http.ResponseWriter, statusCode int, opts ...ResponseOption) *Response {
	return newResponse(w, statusCode, opts...)
}
