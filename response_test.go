package utils_test

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Is999/go-utils"
)

func TestResponse(t *testing.T) {
	mux := http.NewServeMux()
	registerViewExample(mux)
	registerJSONExample(mux)
	registerRedirectExample(mux)

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name        string
		path        string
		statusCode  int
		contentType string
	}{
		{name: "json success", path: "/response/json", statusCode: http.StatusOK, contentType: utils.DefaultJSONContentType},
		{name: "json fail", path: "/response/json?v=fail", statusCode: http.StatusNotAcceptable, contentType: utils.DefaultJSONContentType},
		{name: "html", path: "/response/html", statusCode: http.StatusOK, contentType: "text/html; charset=utf-8"},
		{name: "xml", path: "/response/xml", statusCode: http.StatusOK, contentType: "application/xml; charset=utf-8"},
		{name: "text", path: "/response/text", statusCode: http.StatusOK, contentType: "text/plain; charset=utf-8"},
		{name: "show file", path: "/response/show?file=go.mod", statusCode: http.StatusOK},
		{name: "show missing", path: "/response/show?file=missing.txt", statusCode: http.StatusNotFound, contentType: "text/plain; charset=utf-8"},
		{name: "download file", path: "/response/download?file=go.mod", statusCode: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := server.Client().Get(server.URL + tt.path)
			if err != nil {
				t.Fatalf("GET %s error = %v", tt.path, err)
			}
			defer res.Body.Close()

			if res.StatusCode != tt.statusCode {
				t.Fatalf("status code = %d, want %d", res.StatusCode, tt.statusCode)
			}
			if tt.contentType != "" {
				if got := res.Header.Get(utils.HeaderContentType); got != tt.contentType {
					t.Fatalf("Content-Type = %q, want %q", got, tt.contentType)
				}
			}
		})
	}

	noRedirectClient := server.Client()
	noRedirectClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	res, err := noRedirectClient.Get(server.URL + "/response/redirect")
	if err != nil {
		t.Fatalf("GET /response/redirect error = %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusFound {
		t.Fatalf("redirect status code = %d, want %d", res.StatusCode, http.StatusFound)
	}
	if got := res.Header.Get(utils.HeaderLocation); got != "/response/json" {
		t.Fatalf("Location = %q, want /response/json", got)
	}
}

func TestResponseWriteHeaderOnce(t *testing.T) {
	w := &countingResponseWriter{header: make(http.Header)}

	resp := utils.View(w, utils.WithStatusCode(http.StatusCreated))
	resp.Write([]byte("hello"))
	resp.Write([]byte(" world"))

	if w.statusCode != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", w.statusCode, http.StatusCreated)
	}
	if w.writeHeaderCount != 1 {
		t.Fatalf("WriteHeader count = %d, want 1", w.writeHeaderCount)
	}
	if got := w.body.String(); got != "hello world" {
		t.Fatalf("body = %q, want hello world", got)
	}
}

func TestResponseSuccessRespectsStatusCode(t *testing.T) {
	w := httptest.NewRecorder()

	utils.Json(w, utils.WithStatusCode(http.StatusCreated)).Success(1000, map[string]string{"id": "1"})

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", res.StatusCode, http.StatusCreated)
	}
	if got := res.Header.Get(utils.HeaderContentType); got != utils.DefaultJSONContentType {
		t.Fatalf("Content-Type = %q, want %q", got, utils.DefaultJSONContentType)
	}
}

func TestJsonRespectsExplicitContentType(t *testing.T) {
	w := httptest.NewRecorder()

	utils.Json(w, utils.WithContentType("application/problem+json")).Fail(4000, "bad request")

	res := w.Result()
	defer res.Body.Close()
	if got, want := res.Header.Get(utils.HeaderContentType), "application/problem+json; charset=utf-8"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
}

func TestResponseContentTypeNormalization(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "json", in: "application/json", want: "application/json; charset=utf-8"},
		{name: "text", in: "text/plain", want: "text/plain; charset=utf-8"},
		{name: "binary", in: "application/octet-stream", want: "application/octet-stream"},
		{name: "explicit", in: "application/json; charset=gbk", want: "application/json; charset=gbk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.NormalizeContentType(tt.in); got != tt.want {
				t.Fatalf("utils.NormalizeContentType(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResponseTextHtmlXMLAndHeader(t *testing.T) {
	text := httptest.NewRecorder()
	utils.View(text, utils.WithHeader(func(header http.Header) {
		header.Set("X-Test", "yes")
	})).StatusCode(http.StatusAccepted).Text("hello")
	if text.Code != http.StatusAccepted {
		t.Fatalf("text status = %d", text.Code)
	}
	if text.Body.String() != "hello" {
		t.Fatalf("text body = %q", text.Body.String())
	}
	if got := text.Header().Get("X-Test"); got != "yes" {
		t.Fatalf("header = %q", got)
	}

	html := httptest.NewRecorder()
	utils.View(html).Html("<b>ok</b>")
	if got := html.Header().Get(utils.HeaderContentType); got != "text/html; charset=utf-8" {
		t.Fatalf("html content type = %q", got)
	}

	xmlResp := httptest.NewRecorder()
	type node struct {
		XMLName xml.Name `xml:"node"`
		Name    string   `xml:"name"`
	}
	utils.View(xmlResp).Xml(node{Name: "codex"})
	if !strings.Contains(xmlResp.Body.String(), "<name>codex</name>") {
		t.Fatalf("xml body = %q", xmlResp.Body.String())
	}
}

func TestResponseDownloadSanitizesFilename(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "response-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("hello"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	utils.View(w).Download(file.Name(), "../bad\r\nname.txt")

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", res.StatusCode, http.StatusOK)
	}
	disposition := res.Header.Get(utils.HeaderContentDisposition)
	if strings.ContainsAny(disposition, "\r\n") {
		t.Fatalf("Content-Disposition contains CR/LF: %q", disposition)
	}
	if !strings.Contains(disposition, "attachment") {
		t.Fatalf("Content-Disposition = %q, want attachment", disposition)
	}
}

func TestResponseFileRequestMethods(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "response-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("hello"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/file", nil)

	download := httptest.NewRecorder()
	utils.View(download).DownloadRequest(req, file.Name(), "")
	if download.Code != http.StatusOK {
		t.Fatalf("download status = %d", download.Code)
	}

	show := httptest.NewRecorder()
	utils.View(show).ShowRequest(req, file.Name())
	if show.Code != http.StatusOK {
		t.Fatalf("show status = %d", show.Code)
	}
	if show.Body.String() != "hello" {
		t.Fatalf("show body = %q", show.Body.String())
	}
}

func TestResponseShowRejectsDirectory(t *testing.T) {
	w := httptest.NewRecorder()

	utils.View(w).Show(t.TempDir())

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
}

func TestRedirectFallsBackToFoundForNonRedirectStatus(t *testing.T) {
	w := httptest.NewRecorder()

	utils.Redirect(w, "/next\r\nbad", utils.WithStatusCode(http.StatusOK))

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusFound {
		t.Fatalf("status code = %d, want %d", res.StatusCode, http.StatusFound)
	}
	if got := res.Header.Get(utils.HeaderLocation); got != "/nextbad" {
		t.Fatalf("Location = %q, want /nextbad", got)
	}
}

func TestResponseFailUsesBadRequestByDefault(t *testing.T) {
	w := httptest.NewRecorder()

	utils.Json(w).Fail(4001, "bad request")

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", res.StatusCode, http.StatusBadRequest)
	}
}

func TestResponseJSONFastPathMatchesStandardEscaping(t *testing.T) {
	// message 是包含 HTML 敏感字符、JS 行分隔符和非法 UTF-8 的业务消息，用于校验快路径与标准库完全一致。
	message := "bad <>&\u2028" + string([]byte{0xff})
	// data 是响应业务数据源，map 可覆盖 data 片段仍由 encoding/json 负责编码的边界。
	data := map[string]string{"name": "<codex>&"}
	// expectedBody 是标准库对完整 Body 的编码结果，作为兼容性基准。
	expectedBody, err := json.Marshal(utils.Body{
		Success: true,
		Code:    1000,
		Message: message,
		Data:    data,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	w := httptest.NewRecorder()
	utils.Json(w).Success(1000, data, message)

	if got := w.Body.String(); got != string(expectedBody) {
		t.Fatalf("response body = %q, want %q", got, string(expectedBody))
	}
}

func BenchmarkResponseText(b *testing.B) {
	for i := 0; i < b.N; i++ {
		utils.View(newDiscardResponseWriter()).Text("hello")
	}
}

func BenchmarkResponseJSONSuccess(b *testing.B) {
	data := map[string]string{"id": "1", "name": "codex"}
	for i := 0; i < b.N; i++ {
		utils.Json(newDiscardResponseWriter()).Success(1000, data)
	}
}

type countingResponseWriter struct {
	header           http.Header
	body             strings.Builder
	statusCode       int
	writeHeaderCount int
}

func (w *countingResponseWriter) Header() http.Header {
	return w.header
}

func (w *countingResponseWriter) Write(body []byte) (int, error) {
	return w.body.WriteString(string(body))
}

func (w *countingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.writeHeaderCount++
}

type discardResponseWriter struct {
	header http.Header
}

func newDiscardResponseWriter() *discardResponseWriter {
	return &discardResponseWriter{header: make(http.Header)}
}

func (w *discardResponseWriter) Header() http.Header {
	return w.header
}

func (w *discardResponseWriter) Write(body []byte) (int, error) {
	return len(body), nil
}

func (w *discardResponseWriter) WriteHeader(int) {}
