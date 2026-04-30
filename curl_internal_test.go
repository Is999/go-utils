package utils

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

type testLogger struct{}

func (testLogger) Debug(string, ...any) {}
func (testLogger) Info(string, ...any)  {}
func (testLogger) Warn(string, ...any)  {}
func (testLogger) Error(string, ...any) {}
func (testLogger) With(...any) Logger   { return testLogger{} }
func (testLogger) Enabled(context.Context, LogLevel) bool {
	return true
}

func TestDebugLoggingPreservesRequestAndResponseBody(t *testing.T) {
	const payload = `{"name":"codex"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("server ReadAll() error = %v", err)
		}
		if string(body) != payload {
			t.Fatalf("request body = %q, want %q", body, payload)
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	var gotBody string
	err := NewCurl(WithCurlLogger(testLogger{}), WithCurlDefLogOutput(true)).
		SetBodyBytes([]byte(payload)).
		AfterBody(func(body []byte) error {
			gotBody = string(body)
			return nil
		}).
		Post(srv.URL)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	if gotBody != payload {
		t.Fatalf("AfterBody body = %q, want %q", gotBody, payload)
	}
}

func TestRetryRewindsRequestBody(t *testing.T) {
	const payload = "retry-body"

	var attempts atomic.Int32
	var bodies []string
	rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		bodies = append(bodies, string(body))
		if attempts.Add(1) == 1 {
			return nil, errors.New("temporary transport error")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(body)),
			Request:    req,
		}, nil
	})

	var gotBody string
	err := NewCurl(WithCurlLogger(testLogger{}), WithCurlMaxRetry(2)).
		SetBodyBytes([]byte(payload)).
		BeforeClient(func(client *http.Client) error {
			client.Transport = rt
			return nil
		}).
		AfterBody(func(body []byte) error {
			gotBody = string(body)
			return nil
		}).
		Post("http://example.test/retry")
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d, want 2", got)
	}
	if len(bodies) != 2 || bodies[0] != payload || bodies[1] != payload {
		t.Fatalf("request bodies = %#v, want two %q bodies", bodies, payload)
	}
	if gotBody != payload {
		t.Fatalf("AfterBody body = %q, want %q", gotBody, payload)
	}
}

func TestBuildURLPreservesExistingQuery(t *testing.T) {
	params := mapValues("page", "2", "q", "codex")
	got, err := buildUrl("https://example.com/search?lang=go", params)
	if err != nil {
		t.Fatalf("buildUrl() error = %v", err)
	}
	for _, want := range []string{"lang=go", "page=2", "q=codex"} {
		if !strings.Contains(got, want) {
			t.Fatalf("buildUrl() = %q, missing %q", got, want)
		}
	}
}

func TestReadBodyPreviewAndRestoreClosesOriginal(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader("abcdef")}

	preview, truncated, restored, err := readBodyPreviewAndRestore(body, 3)
	if err != nil {
		t.Fatalf("readBodyPreviewAndRestore() error = %v", err)
	}
	if string(preview) != "abc" || !truncated {
		t.Fatalf("preview = %q, truncated = %v; want abc, true", preview, truncated)
	}
	restoredBody, err := io.ReadAll(restored)
	if err != nil {
		t.Fatalf("restored ReadAll() error = %v", err)
	}
	if string(restoredBody) != "abcdef" {
		t.Fatalf("restored body = %q, want abcdef", restoredBody)
	}
	if err := restored.Close(); err != nil {
		t.Fatalf("restored Close() error = %v", err)
	}
	if !body.closed.Load() {
		t.Fatal("restored Close() should close original body")
	}
}

func TestNewBindsRequestIDToCustomLogger(t *testing.T) {
	logger := &captureLogger{}

	c := NewCurl(WithCurlLogger(logger), WithCurlRequestId("req-123"))

	if c.GetRequestId() != "req-123" {
		t.Fatalf("request id = %q, want req-123", c.GetRequestId())
	}
	if !logger.hasRequestID("req-123") {
		t.Fatalf("custom logger should receive X-Request-Id, got %#v", logger.args())
	}
}

func TestSetRequestIDDoesNotStackLoggerFields(t *testing.T) {
	c := NewCurl(WithCurlLogger(fieldLogger{}), WithCurlRequestId("req-1"))
	c.SetRequestId("req-2")

	logger, ok := c.Logger.(fieldLogger)
	if !ok {
		t.Fatalf("logger type = %T, want fieldLogger", c.Logger)
	}
	if count := logger.count("X-Request-Id"); count != 1 {
		t.Fatalf("X-Request-Id field count = %d, want 1; fields=%#v", count, logger.fields)
	}
	if got := logger.value("X-Request-Id"); got != "req-2" {
		t.Fatalf("X-Request-Id = %v, want req-2", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type trackingReadCloser struct {
	io.Reader
	closed atomic.Bool
}

func (r *trackingReadCloser) Close() error {
	r.closed.Store(true)
	return nil
}

func mapValues(kv ...string) url.Values {
	m := make(url.Values, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		m.Set(kv[i], kv[i+1])
	}
	return m
}

type captureLogger struct {
	mu       sync.Mutex
	withArgs []any
}

func (l *captureLogger) Debug(string, ...any) {}
func (l *captureLogger) Info(string, ...any)  {}
func (l *captureLogger) Warn(string, ...any)  {}
func (l *captureLogger) Error(string, ...any) {}
func (l *captureLogger) Enabled(context.Context, LogLevel) bool {
	return true
}

func (l *captureLogger) With(args ...any) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.withArgs = append(l.withArgs, args...)
	return l
}

func (l *captureLogger) args() []any {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]any, len(l.withArgs))
	copy(out, l.withArgs)
	return out
}

func (l *captureLogger) hasRequestID(id string) bool {
	args := l.args()
	for i := 0; i+1 < len(args); i += 2 {
		if args[i] == "X-Request-Id" && args[i+1] == id {
			return true
		}
	}
	return false
}

type fieldLogger struct {
	fields []any
}

func (l fieldLogger) Debug(string, ...any) {}
func (l fieldLogger) Info(string, ...any)  {}
func (l fieldLogger) Warn(string, ...any)  {}
func (l fieldLogger) Error(string, ...any) {}
func (l fieldLogger) Enabled(context.Context, LogLevel) bool {
	return true
}

func (l fieldLogger) With(args ...any) Logger {
	fields := make([]any, 0, len(l.fields)+len(args))
	fields = append(fields, l.fields...)
	fields = append(fields, args...)
	return fieldLogger{fields: fields}
}

func (l fieldLogger) count(key string) int {
	var n int
	for i := 0; i+1 < len(l.fields); i += 2 {
		if l.fields[i] == key {
			n++
		}
	}
	return n
}

func (l fieldLogger) value(key string) any {
	for i := 0; i+1 < len(l.fields); i += 2 {
		if l.fields[i] == key {
			return l.fields[i+1]
		}
	}
	return nil
}
