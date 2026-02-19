package utils_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	utils "github.com/Is999/go-utils"
)

func TestCurlLoggerCompatibility(t *testing.T) {
	defaultLogger := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(defaultLogger)
	})

	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})))

	var buf bytes.Buffer
	curl := utils.NewCurl().SetDefLogOutput(true)
	curl.Logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With("X-Request-Id", curl.GetRequestId())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	if err := curl.Get(srv.URL); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if !strings.Contains(buf.String(), "DrainBody(resp.Body)") {
		t.Fatalf("expected response info log, got: %s", buf.String())
	}
}
