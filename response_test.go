package utils_test

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/textproto"
	"os"
	"path/filepath"
	"slices"
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
		{name: "json success", path: "/response/json", statusCode: http.StatusOK, contentType: "application/json; charset=utf-8"},
		{name: "json fail", path: "/response/json?v=fail", statusCode: http.StatusNotAcceptable, contentType: "application/json; charset=utf-8"},
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
				if got := res.Header.Get("Content-Type"); got != tt.contentType {
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
	if got := res.Header.Get("Location"); got != "/response/json" {
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

// TestResponseInformationalStatus 使用真实 HTTP 服务验证临时响应、最终状态和重复提交。
// httptest.ResponseRecorder 只保留首次状态码，无法表达多次 1xx 后再提交最终响应。
func TestResponseInformationalStatus(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "body.txt")
	if err := os.WriteFile(filePath, []byte("created"), 0o600); err != nil {
		t.Fatal(err)
	}
	emptyPath := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(emptyPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		method     string // 默认 GET；HEAD 验证临时响应后的文件头不提交正文。
		write      func(*utils.Response)
		hints      []int
		status     int
		body       string
		bodyPrefix bool // 错误响应包含运行时生成的追踪 ID，只核对固定前缀。
	}{
		{
			name: "explicit final status",
			write: func(r *utils.Response) {
				r.Write(nil)
				r.StatusCode(http.StatusProcessing).Write(nil)
				r.StatusCode(http.StatusCreated).Text("created")
			},
			hints: []int{http.StatusEarlyHints, http.StatusProcessing}, status: http.StatusCreated, body: "created",
		},
		{
			name:  "bytes commit implicit OK",
			write: func(r *utils.Response) { r.Write([]byte("created")) },
			hints: []int{http.StatusEarlyHints}, status: http.StatusOK, body: "created",
		},
		{
			name:  "text commits implicit OK",
			write: func(r *utils.Response) { r.Text("created") },
			hints: []int{http.StatusEarlyHints}, status: http.StatusOK, body: "created",
		},
		{
			name:  "file commits implicit OK",
			write: func(r *utils.Response) { r.Show(filePath) },
			hints: []int{http.StatusEarlyHints}, status: http.StatusOK, body: "created",
		},
		{
			name:  "empty file commits OK",
			write: func(r *utils.Response) { r.Show(emptyPath) },
			hints: []int{http.StatusEarlyHints}, status: http.StatusOK,
		},
		{
			name: "encoding failure after hints",
			write: func(r *utils.Response) {
				r.Write(nil)
				r.XML(func() {})
			},
			hints: []int{http.StatusEarlyHints}, status: http.StatusInternalServerError,
			body: "Response error, code-", bodyPrefix: true,
		},
		{
			name:   "switching protocols is final",
			write:  func(r *utils.Response) { r.StatusCode(http.StatusSwitchingProtocols).Write(nil) },
			status: http.StatusSwitchingProtocols,
		},
		{
			name: "HEAD permits final status", method: http.MethodHead,
			write: func(r *utils.Response) {
				r.ShowRequest(httptest.NewRequest(http.MethodHead, "/file", nil), filePath)
				r.StatusCode(http.StatusCreated).Write(nil)
			},
			hints: []int{http.StatusEarlyHints}, status: http.StatusCreated,
		},
		{
			name: "file after hints and text",
			write: func(r *utils.Response) {
				r.Text("first")
				r.Show(filePath)
			},
			hints: []int{http.StatusEarlyHints}, status: http.StatusOK, body: "firstcreated",
		},
		{
			name: "file after final status",
			write: func(r *utils.Response) {
				r.StatusCode(http.StatusOK).Text("first")
				r.Show(filePath)
			},
			status: http.StatusOK, body: "firstcreated",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverLog bytes.Buffer // 额外的最终 WriteHeader 会被 net/http 记入服务日志。
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				r := utils.View(w).StatusCode(http.StatusEarlyHints)
				tt.write(r)
				r.StatusCode(http.StatusTeapot).Write(nil)
			}))
			server.Config.ErrorLog = log.New(&serverLog, "", 0)
			server.Start()
			defer server.Close()

			var hints []int // 客户端按接收顺序记录全部临时响应。
			ctx := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{
				Got1xxResponse: func(code int, _ textproto.MIMEHeader) error {
					hints = append(hints, code)
					return nil
				},
			})
			method := tt.method
			if method == "" {
				method = http.MethodGet
			}
			req, err := http.NewRequestWithContext(ctx, method, server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Close = true // 101 响应升级后的连接也随 handler 返回而关闭。
			res, err := server.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(res.Body)
			_ = res.Body.Close()
			server.Close()
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(hints, tt.hints) || res.StatusCode != tt.status {
				t.Fatalf("statuses = %v then %d, want %v then %d", hints, res.StatusCode, tt.hints, tt.status)
			}
			if tt.bodyPrefix {
				if !strings.HasPrefix(string(body), tt.body) {
					t.Fatalf("body = %q, want prefix %q", body, tt.body)
				}
			} else if string(body) != tt.body {
				t.Fatalf("body = %q, want %q", body, tt.body)
			}
			if serverLog.Len() != 0 {
				t.Fatalf("unexpected server log: %s", serverLog.String())
			}
		})
	}
}

func TestResponseSuccessRespectsStatusCode(t *testing.T) {
	w := httptest.NewRecorder()

	utils.JSON(w, utils.WithStatusCode(http.StatusCreated)).Success(1000, map[string]string{"id": "1"})

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", res.StatusCode, http.StatusCreated)
	}
	if got := res.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json; charset=utf-8")
	}
}

func TestJSONRespectsExplicitContentType(t *testing.T) {
	w := httptest.NewRecorder()

	utils.JSON(w, utils.WithContentType("application/problem+json")).Fail(4000, "bad request")

	res := w.Result()
	defer res.Body.Close()
	if got, want := res.Header.Get("Content-Type"), "application/problem+json; charset=utf-8"; got != want {
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
			w := httptest.NewRecorder()
			utils.JSON(w, utils.WithContentType(tt.in)).Success(1000, nil)
			if got := w.Result().Header.Get("Content-Type"); got != tt.want {
				t.Fatalf("Content-Type for %q = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResponseTextHTMLXMLAndHeader(t *testing.T) {
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
	utils.View(html).HTML("<b>ok</b>")
	if got := html.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("html content type = %q", got)
	}

	xmlResp := httptest.NewRecorder()
	type node struct {
		XMLName xml.Name `xml:"node"`
		Name    string   `xml:"name"`
	}
	utils.View(xmlResp).XML(node{Name: "alice"})
	if !strings.Contains(xmlResp.Body.String(), "<name>alice</name>") {
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
	disposition := res.Header.Get("Content-Disposition")
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

func TestResponseHeadWithExplicitStatusHasNoBody(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodHead, "/file", nil)
	utils.View(w).StatusCode(http.StatusCreated).ShowRequest(req, file.Name())

	if w.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", w.Code, http.StatusCreated)
	}
	if got := w.Header().Get("Content-Length"); got != "5" {
		t.Fatalf("Content-Length = %q, want %q", got, "5")
	}
	if w.Body.Len() != 0 {
		t.Fatalf("HEAD body length = %d, want 0", w.Body.Len())
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
	if got := res.Header.Get("Location"); got != "/nextbad" {
		t.Fatalf("Location = %q, want /nextbad", got)
	}
}

func TestResponseFailUsesBadRequestByDefault(t *testing.T) {
	w := httptest.NewRecorder()

	utils.JSON(w).Fail(4001, "bad request")

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", res.StatusCode, http.StatusBadRequest)
	}
}

func TestResponseJSONMatchesStandardEscaping(t *testing.T) {
	// 消息覆盖 HTML 敏感字符、JS 行分隔符和非法 UTF-8，作为公开响应格式的兼容边界。
	message := "bad <>&\u2028" + string([]byte{0xff})
	data := map[string]string{"name": "<example>&"}
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
	utils.JSON(w).Success(1000, data, message)

	if got := w.Body.String(); got != string(expectedBody) {
		t.Fatalf("response body = %q, want %q", got, string(expectedBody))
	}
}

func BenchmarkResponseText(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		utils.View(newDiscardResponseWriter()).Text("hello")
	}
}

func BenchmarkResponseJSONSuccess(b *testing.B) {
	data := map[string]string{"id": "1", "name": "alice"}
	b.ResetTimer()
	for range b.N {
		utils.JSON(newDiscardResponseWriter()).Success(1000, data)
	}
}

// BenchmarkResponseEncode 覆盖空数据、小对象和大文本，便于评估编码器的整体成本。
func BenchmarkResponseEncode(b *testing.B) {
	for name, data := range map[string]any{
		"nil":  nil,
		"map":  map[string]string{"id": "1", "name": "example"},
		"text": strings.Repeat("example <>&", 1024),
	} {
		b.Run(name, func(b *testing.B) {
			response := utils.Response{Body: utils.Body{
				Success: true,
				Code:    1000,
				Message: "SUCCESS",
				Data:    data,
			}}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := response.Encode(); err != nil {
					b.Fatal(err)
				}
			}
		})
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
