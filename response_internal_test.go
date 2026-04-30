package utils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestResponseWriteHeaderOnce(t *testing.T) {
	w := &countingResponseWriter{header: make(http.Header)}

	resp := View(w, WithStatusCode(http.StatusCreated))
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

	Json(w, WithStatusCode(http.StatusCreated)).Success(1000, map[string]string{"id": "1"})

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", res.StatusCode, http.StatusCreated)
	}
	if got := res.Header.Get(headerContentType); got != defaultJSONContentType {
		t.Fatalf("Content-Type = %q, want %q", got, defaultJSONContentType)
	}
}

func TestJsonRespectsExplicitContentType(t *testing.T) {
	w := httptest.NewRecorder()

	Json(w, WithContentType("application/problem+json")).Fail(4000, "bad request")

	res := w.Result()
	defer res.Body.Close()
	if got, want := res.Header.Get(headerContentType), "application/problem+json; charset=utf-8"; got != want {
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
			if got := normalizeContentType(tt.in); got != tt.want {
				t.Fatalf("normalizeContentType(%q) = %q, want %q", tt.in, got, tt.want)
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
	View(w).Download(file.Name(), "../bad\r\nname.txt")

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", res.StatusCode, http.StatusOK)
	}
	disposition := res.Header.Get(headerContentDisposition)
	if strings.ContainsAny(disposition, "\r\n") {
		t.Fatalf("Content-Disposition contains CR/LF: %q", disposition)
	}
	if !strings.Contains(disposition, "attachment") {
		t.Fatalf("Content-Disposition = %q, want attachment", disposition)
	}
}

func TestResponseShowRejectsDirectory(t *testing.T) {
	w := httptest.NewRecorder()

	View(w).Show(t.TempDir())

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
}

func TestRedirectFallsBackToFoundForNonRedirectStatus(t *testing.T) {
	w := httptest.NewRecorder()

	Redirect(w, "/next\r\nbad", WithStatusCode(http.StatusOK))

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusFound {
		t.Fatalf("status code = %d, want %d", res.StatusCode, http.StatusFound)
	}
	if got := res.Header.Get(headerLocation); got != "/nextbad" {
		t.Fatalf("Location = %q, want /nextbad", got)
	}
}

func BenchmarkResponseText(b *testing.B) {
	for i := 0; i < b.N; i++ {
		View(newDiscardResponseWriter()).Text("hello")
	}
}

func BenchmarkResponseJSONSuccess(b *testing.B) {
	data := map[string]string{"id": "1", "name": "codex"}
	for i := 0; i < b.N; i++ {
		Json(newDiscardResponseWriter()).Success(1000, data)
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
