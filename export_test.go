package utils

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
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

// GenerateUniqueID 是测试用请求 ID 生成函数包装。
func GenerateUniqueID(length int) string {
	return generateUniqueID(length)
}

// BuildURL 是测试用 URL 查询参数合并函数包装。
func BuildURL(baseURL string, params url.Values) (string, error) {
	return buildURL(baseURL, params.Encode()), nil
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

// ResetConfigForTest 重置全局配置，供外部测试包隔离 Configure 单次设置语义。
func ResetConfigForTest() {
	setOptionsOnce = sync.Once{}
	configValue.Store(defaultOptions())
}

// TestDumpRequestSafePreservesBody 验证请求 dump 不会替换原始请求体。
func TestDumpRequestSafePreservesBody(t *testing.T) {
	original := io.NopCloser(strings.NewReader("payload"))
	req, err := http.NewRequest(http.MethodPost, "http://example.com", original)
	if err != nil {
		t.Fatal(err)
	}

	getBodyCalls := 0
	req.GetBody = func() (io.ReadCloser, error) {
		getBodyCalls++
		return io.NopCloser(strings.NewReader("payload")), nil
	}

	if _, err = dumpRequestSafe(req, 1024); err != nil {
		t.Fatal(err)
	}
	if req.Body != original {
		t.Fatal("dumpRequestSafe replaced the original request body")
	}
	if getBodyCalls != 1 {
		t.Fatalf("GetBody called %d times, want 1", getBodyCalls)
	}
}
