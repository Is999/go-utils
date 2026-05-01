package utils_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Is999/go-utils"
)

var serveMux = http.NewServeMux()

func httpServer(addr string, header http.Handler, exit chan os.Signal) {
	//使用默认路由创建 http server
	srv := http.Server{
		Addr:    addr,
		Handler: header,
	}

	//监听 Ctrl+C 信号
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		select {
		case <-exit:
		case <-timer.C:
		}
		_ = srv.Shutdown(context.Background())
	}()

	// 启动 HTTP 服务器。部分测试复用固定端口，race 模式下前一个 server
	// 刚 Shutdown 时端口可能短暂未释放，这里做有限重试，避免测试偶发失败。
	for i := 0; i < 40; i++ {
		err := srv.ListenAndServe()
		if err == nil || err == http.ErrServerClosed {
			return
		}
		if !strings.Contains(err.Error(), "address already in use") {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

}

func waitHTTPServer(t *testing.T, addr string) {
	t.Helper()
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("HTTP server %s did not start", addr)
}

func TestResponse(t *testing.T) {
	// 退出
	exit := make(chan os.Signal)

	// 请求该路由退出
	// http://localhost:54333/response/exit
	serveMux.HandleFunc("/response/exit", func(w http.ResponseWriter, r *http.Request) {
		// 退出信号
		exit <- syscall.Signal(1)
	})

	// 响应html、xml、text、file、image
	// http://localhost:54333/response/html
	// http://localhost:54333/response/xml
	// http://localhost:54333/response/text
	// http://localhost:54333/response/show?file=go.mod
	// http://localhost:54333/response/show?file=resource/golang_icon.png
	// http://localhost:54333/response/download?file=go.mod
	// http://localhost:54333/response/download?file=resource/golang_icon.png
	ExampleView()

	// 响应json
	// http://localhost:54333/response/json
	ExampleJson()

	// 重定向
	// http://localhost:54333/response/redirect
	ExampleRedirect()

	httpServer(":54333", serveMux, exit)
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
