package utils_test

import (
	"context"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Is999/go-utils"
)

func TestFormReaderStreamsMultipartBody(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(filePath, []byte("stream-body"), 0o644); err != nil {
		t.Fatal(err)
	}

	form := utils.NewForm().
		SetParam("name", "codex").
		SetFile("file", filePath)

	body, contentType, err := form.Reader()
	if err != nil {
		t.Fatalf("Form.Reader() error = %v", err)
	}
	if !strings.Contains(contentType, "multipart/form-data") {
		t.Fatalf("contentType = %q, want multipart/form-data", contentType)
	}

	readAll, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(readAll), "stream-body") {
		t.Fatalf("multipart body = %q, want file content", readAll)
	}
	if !strings.Contains(string(readAll), `name="name"`) || !strings.Contains(string(readAll), "codex") {
		t.Fatalf("multipart body = %q, want form field", readAll)
	}
}

func TestFormReaderRejectsOversizeFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "upload.txt")
	content := []byte("oversize")
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	form := utils.NewForm().
		SetFile("file", filePath).
		SetMaxSingleFileSize(int64(len(content) - 1))

	if _, _, err := form.Reader(); err == nil {
		t.Fatal("Form.Reader() expected single file size limit error")
	}
}

func TestFormReaderRejectsOversizeTotalFiles(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "one.txt")
	file2 := filepath.Join(dir, "two.txt")
	if err := os.WriteFile(file1, []byte("1234"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("5678"), 0o644); err != nil {
		t.Fatal(err)
	}

	form := utils.NewForm().
		AddFile("files", file1, file2).
		SetMaxTotalFileSize(7)

	if _, _, err := form.Reader(); err == nil {
		t.Fatal("Form.Reader() expected total file size limit error")
	}
}

func TestCurlDefaultLogUsesBodyPreviewLimit(t *testing.T) {
	const requestBody = "request-body-preview"
	const responseBody = "response-body-preview"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer srv.Close()

	logger := &recordLogger{}
	var gotResponseBody string
	err := utils.NewCurl(
		utils.WithCurlLogger(logger),
		utils.WithCurlDefLogOutput(true),
		utils.WithCurlLogBodyLimit(8),
	).
		SetBodyBytes([]byte(requestBody)).
		AfterBody(func(body []byte) error {
			gotResponseBody = string(body)
			return nil
		}).
		Post(srv.URL)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	if gotResponseBody != responseBody {
		t.Fatalf("AfterBody body = %q, want %q", gotResponseBody, responseBody)
	}

	requestLog := logger.joinMessage("Request")
	if !strings.Contains(requestLog, "Request Body Preview:\nrequest-") {
		t.Fatalf("request log = %q, want truncated preview", requestLog)
	}
	if strings.Contains(requestLog, requestBody) {
		t.Fatalf("request log = %q, should not contain full body", requestLog)
	}

	responseLog := logger.joinMessage("Response")
	if !strings.Contains(responseLog, "Response Body Preview:\nresponse") || !strings.Contains(responseLog, "...[truncated]") {
		t.Fatalf("response log = %q, want truncated preview", responseLog)
	}
	if strings.Contains(responseLog, responseBody) {
		t.Fatalf("response log = %q, should not contain full body", responseLog)
	}
}

func TestRootCAsAppendsSystemPool(t *testing.T) {
	systemPool, err := x509.SystemCertPool()
	if err != nil || systemPool == nil {
		t.Skipf("SystemCertPool() unavailable: %v", err)
	}
	before := len(systemPool.Subjects())
	if before == 0 {
		t.Skip("system cert pool is empty")
	}

	certPath := filepath.Join(t.TempDir(), "root-ca.pem")
	if err := os.WriteFile(certPath, buildTestCAPEM(t), 0o644); err != nil {
		t.Fatal(err)
	}

	config := &tls.Config{}
	if err := utils.RootCAs(config, certPath); err != nil {
		t.Fatalf("RootCAs() error = %v", err)
	}
	after := len(config.RootCAs.Subjects())
	if after <= before {
		t.Fatalf("RootCAs() subjects = %d, want > %d", after, before)
	}
}

func TestTransportHelpersRejectNilInputs(t *testing.T) {
	if err := utils.ProxyURL(nil, "http://127.0.0.1:8080"); err == nil {
		t.Fatal("ProxyURL() expected nil transport error")
	}
	if err := utils.RootCAs(nil, "root-ca.pem"); err == nil {
		t.Fatal("RootCAs() expected nil TLS config error")
	}
	if err := utils.Certificate(nil, "client.crt", "client.key"); err == nil {
		t.Fatal("Certificate() expected nil TLS config error")
	}
}

type recordLogger struct {
	mu      sync.Mutex
	records map[string][]string
}

func (l *recordLogger) Debug(string, ...any) {}
func (l *recordLogger) Warn(string, ...any)  {}
func (l *recordLogger) Error(string, ...any) {}
func (l *recordLogger) Enabled(context.Context, utils.LogLevel) bool {
	return true
}

func (l *recordLogger) With(...any) utils.Logger {
	return l
}

func (l *recordLogger) Info(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.records == nil {
		l.records = make(map[string][]string)
	}
	for i := 0; i+1 < len(args); i += 2 {
		if key, ok := args[i].(string); ok {
			l.records[msg] = append(l.records[msg], key+"="+toString(args[i+1]))
		}
	}
}

func (l *recordLogger) joinMessage(msg string) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.records[msg], "\n")
}

func buildTestCAPEM(t *testing.T) []byte {
	t.Helper()

	privateKey, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: "go-utils-test-root-ca",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(crand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
