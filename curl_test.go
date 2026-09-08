package utils_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Is999/go-utils"
	"github.com/Is999/go-utils/errors"
)

// setLogConfig 只在当前测试期间开启日志，结束后恢复其他用例使用的默认实例。
func setLogConfig(t *testing.T) {
	t.Helper()
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	levelVar := &slog.LevelVar{}
	// 允许 Debug，使请求初始化、重试和关闭日志都能进入测试输出。
	levelVar.Set(slog.LevelDebug)

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     levelVar,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
}

// TestGet 核对查询参数回显，以及成功和已允许失败状态下的完整 JSON 响应。
func TestGet(t *testing.T) {
	setLogConfig(t)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/curl/get", func(w http.ResponseWriter, r *http.Request) {
		// user 按查询参数回显，未导出的 phone 不会写入 JSON。
		user := User{
			Name:      r.URL.Query().Get("Name"),
			Age:       utils.ToInt(r.URL.Query().Get("Age")),
			Sex:       r.URL.Query().Get("Sex"),
			IsMarried: r.URL.Query().Get("IsMarried") == "true",
			Address:   r.URL.Query().Get("Address"),
			phone:     r.URL.Query().Get("phone"),
		}
		if r.URL.Query().Get("success") == "false" {
			utils.JSON(w, utils.WithStatusCode(http.StatusNotAcceptable)).Fail(20000, "fail", user)
			return
		}
		utils.JSON(w).Success(10000, user)
	})
	server := httptest.NewServer(serveMux)
	defer server.Close()

	curl := utils.NewCurl().SetDefaultLogOutput(true).
		SetStatusCode(http.StatusUnauthorized, http.StatusNotAcceptable)
	defer curl.CloseIdleConnections()
	tests := []struct {
		name        string
		user        User // 原始输入，覆盖数字、布尔值和中文参数。
		wantSuccess bool // 控制服务端业务分支；失败场景的 406 已加入允许列表。
	}{
		{name: "001", user: User{
			Name: "Andy", Age: 18, Sex: "男", IsMarried: false,
			Address: "火星", phone: "18899995555",
		}, wantSuccess: true},
		{name: "002", user: User{
			Name: "Lisa", Age: 28, Sex: "女", IsMarried: true,
			Address: "月星", phone: "18899996666",
		}, wantSuccess: true},
		{name: "003", user: User{
			Name: "Jack", Age: 38, Sex: "男", IsMarried: false,
			Address: "金星", phone: "18899998888",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			curl.SetRequestID().SetParam("success", fmt.Sprint(tt.wantSuccess))
			userType := reflect.TypeFor[User]()
			userValue := reflect.ValueOf(tt.user)
			for field := range userType.Fields() {
				curl.SetParam(field.Name, fmt.Sprint(userValue.FieldByName(field.Name)))
			}

			var got RespBody[User] // 回调未执行或响应字段丢失时，也会与完整期望值不符。
			curl.AfterBody(func(body []byte) error { return utils.Unmarshal(body, &got) })
			if err := curl.Get(server.URL + "/curl/get"); err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			want := RespBody[User]{Success: tt.wantSuccess, Code: 10000, Message: "SUCCESS", Data: tt.user}
			want.Data.phone = "" // JSON 不传输未导出字段。
			if !tt.wantSuccess {
				want.Code, want.Message = 20000, "fail"
			}
			if got != want {
				t.Fatalf("response = %+v, want %+v", got, want)
			}
		})
	}
}

// TestPost 核对 JSON 请求体回显，并区分 HTTP 发送成功与业务失败响应。
func TestPost(t *testing.T) {
	setLogConfig(t)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/curl/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			utils.JSON(w, utils.WithStatusCode(http.StatusMethodNotAllowed)).Fail(2000, "Method not allowed")
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			utils.JSON(w, utils.WithStatusCode(http.StatusInternalServerError)).Fail(2000, "Failed to read request body")
			return
		}
		user := new(User)
		if err := utils.Unmarshal(body, user); err != nil {
			utils.JSON(w, utils.WithStatusCode(http.StatusBadRequest)).Fail(2000, "Invalid JSON request body")
			return
		}
		if r.URL.Query().Get("success") == "false" {
			utils.JSON(w, utils.WithStatusCode(http.StatusNotAcceptable)).Fail(2000, "fail", user)
			return
		}
		utils.JSON(w).Success(1000, user)
	})
	server := httptest.NewServer(serveMux)
	defer server.Close()

	curl := utils.NewCurl().SetDump(true).SetMaxRetry(3).SetContentType("application/json").
		SetStatusCode(http.StatusUnauthorized, http.StatusNotAcceptable)
	defer curl.CloseIdleConnections()
	tests := []struct {
		name        string
		user        User // 序列化到请求体，覆盖中文、数字和布尔字段。
		wantSuccess bool // 控制 URL 参数选定的服务端业务分支。
	}{
		{name: "001", user: User{
			Name: "Andy", Age: 18, Sex: "男", IsMarried: false,
			Address: "火星", phone: "18899995555",
		}, wantSuccess: true},
		{name: "002", user: User{
			Name: "Lisa", Age: 28, Sex: "女", IsMarried: true,
			Address: "月星", phone: "18899996666",
		}, wantSuccess: true},
		{name: "003", user: User{
			Name: "Jack", Age: 38, Sex: "男", IsMarried: false,
			Address: "金星", phone: "18899998888",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := utils.Marshal(tt.user)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			defer curl.ResetParams(nil)
			curl.SetRequestID().SetParam("page", "2").AddParam("limit", "10").
				SetParam("success", fmt.Sprint(tt.wantSuccess)).SetBodyBytes(body)

			var got RespBody[User] // 直接断言完整响应，日志输出不能代表业务结果正确。
			curl.AfterBody(func(body []byte) error { return utils.Unmarshal(body, &got) })
			if err := curl.Post(server.URL + "/curl/post"); err != nil {
				t.Fatalf("Post() error = %v", err)
			}
			want := RespBody[User]{Success: tt.wantSuccess, Code: 1000, Message: "SUCCESS", Data: tt.user}
			want.Data.phone = "" // JSON 不传输未导出字段。
			if !tt.wantSuccess {
				want.Code, want.Message = 2000, "fail"
			}
			if got != want {
				t.Fatalf("response = %+v, want %+v", got, want)
			}
		})
	}
}

// TestPostForm 核对表单字段回显和同名值的追加顺序，发送成功还必须得到预期业务响应。
func TestPostForm(t *testing.T) {
	setLogConfig(t)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/curl/form", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			utils.JSON(w, utils.WithStatusCode(http.StatusMethodNotAllowed)).Fail(2000, "Method not allowed")
			return
		}
		if err := r.ParseForm(); err != nil {
			utils.JSON(w, utils.WithStatusCode(http.StatusBadRequest)).Fail(2000, "Error parsing form")
			return
		}
		info := map[string]any{ // 回显实际解析值，避免仅验证本地 Params 配置。
			"name":     r.FormValue("name"),
			"age":      r.FormValue("age"),
			"language": r.FormValue("language"),
			"friends":  r.Form["friends"],
			"hobby":    r.Form["hobby"],
		}
		if r.URL.Query().Get("success") == "false" {
			utils.JSON(w, utils.WithStatusCode(http.StatusNotAcceptable)).Fail(2000, "fail", info)
			return
		}
		utils.JSON(w).Success(1000, info)
	})
	server := httptest.NewServer(serveMux)
	defer server.Close()

	curl := utils.NewCurl().SetDefaultLogOutput(true).SetRequestID().SetMaxRetry(3).
		SetStatusCode(http.StatusUnauthorized, http.StatusNotAcceptable)
	defer curl.CloseIdleConnections()
	curl.SetParams(map[string]string{"name": "Lisa", "age": "22"}).
		AddParams(map[string][]string{
			"hobby":   {"读书", "游泳", "旅游"},
			"friends": {"Kelly", "Shirley"},
		}).
		SetParam("language", "English,中文,Français").
		AddParam("hobby", "骑行").AddParam("hobby", "冒险")

	var got RespBody[map[string]any] // 没有执行回调时，零值也无法通过下面的业务字段断言。
	curl.AfterBody(func(body []byte) error { return utils.Unmarshal(body, &got) })
	if err := curl.PostForm(server.URL + "/curl/form"); err != nil {
		t.Fatalf("PostForm() error = %v", err)
	}
	want := RespBody[map[string]any]{
		Success: true, Code: 1000, Message: "SUCCESS",
		Data: map[string]any{
			"name":     "Lisa",
			"age":      "22",
			"language": "English,中文,Français",
			// JSON 数组解码到 []any；保留多值字段在请求中的添加顺序。
			"friends": []any{"Kelly", "Shirley"},
			"hobby":   []any{"读书", "游泳", "旅游", "骑行", "冒险"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("response = %#v, want %#v", got, want)
	}
}

// TestPostFile 验证 multipart 上传在服务端的实际解析结果。
func TestPostFile(t *testing.T) {
	wantParams := map[string][]string{
		"name": {"Lisa"}, "age": {"22"}, "language": {"English,中文,Français"},
		"hobby": {"读书", "游泳", "旅游", "骑行"}, "friends": {"Kelly", "Shirley"},
	}
	// 同一字段的文件名按上传顺序排列。
	wantFiles := map[string][]string{
		"json_file": {"data.json"}, "env_file": {"settings.env"}, "files": {"one.txt", "two.txt"},
	}
	form := utils.NewForm().AddParams(wantParams)
	// 独立夹具固定文件名和正文，避免测试依赖仓库源码的大小与内容。
	dir := t.TempDir()
	for field, names := range wantFiles {
		for _, name := range names {
			path := filepath.Join(dir, name)
			if err := os.WriteFile(path, []byte("body:"+name), 0o600); err != nil {
				t.Fatal(err)
			}
			form.AddFile(field, path)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("ParseMultipartForm() error = %v", err)
			http.Error(w, "invalid multipart body", http.StatusBadRequest)
			return
		}
		defer r.MultipartForm.RemoveAll()
		if !reflect.DeepEqual(r.MultipartForm.Value, wantParams) {
			t.Errorf("form fields = %v, want %v", r.MultipartForm.Value, wantParams)
		}
		if len(r.MultipartForm.File) != len(wantFiles) {
			t.Errorf("file fields = %d, want %d", len(r.MultipartForm.File), len(wantFiles))
		}
		for field, names := range wantFiles {
			headers := r.MultipartForm.File[field]
			if len(headers) != len(names) {
				t.Errorf("file field %q count = %d, want %d", field, len(headers), len(names))
				continue
			}
			for i, header := range headers {
				file, err := header.Open()
				if err != nil {
					t.Errorf("Open(%q) error = %v", header.Filename, err)
					continue
				}
				data, readErr := io.ReadAll(file)
				closeErr := file.Close()
				if readErr != nil || closeErr != nil {
					t.Errorf("file %s[%d]: read=%v close=%v", field, i, readErr, closeErr)
					continue
				}
				wantBody := "body:" + names[i]
				if header.Filename != names[i] || header.Size != int64(len(wantBody)) {
					t.Errorf("file %s[%d]: name=%q size=%d", field, i, header.Filename, header.Size)
				}
				if string(data) != wantBody {
					t.Errorf("file %s[%d]: body=%q, want %q", field, i, data, wantBody)
				}
			}
		}
		utils.JSON(w).Success(1000, "uploaded")
	}))
	defer server.Close()

	body, contentType, err := form.Reader()
	if err != nil {
		t.Fatal(err)
	}
	var got RespBody[string]
	// boundary 必须使用本次 Reader 返回的值。
	curl := utils.NewCurl().SetContentType(contentType).SetBody(body).
		AfterBody(func(body []byte) error { return utils.Unmarshal(body, &got) })
	defer curl.CloseIdleConnections()
	if err := curl.PostContext(t.Context(), server.URL); err != nil {
		t.Fatal(err)
	}
	want := RespBody[string]{Success: true, Code: 1000, Message: "SUCCESS", Data: "uploaded"}
	if got != want {
		t.Fatalf("response = %+v, want %+v", got, want)
	}
}

type RespBody[T any] struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}
type curlTestLogger struct{}

func (curlTestLogger) Debug(string, ...any)     {}
func (curlTestLogger) Info(string, ...any)      {}
func (curlTestLogger) Warn(string, ...any)      {}
func (curlTestLogger) Error(string, ...any)     {}
func (curlTestLogger) With(...any) utils.Logger { return curlTestLogger{} }
func (curlTestLogger) Enabled(context.Context, utils.LogLevel) bool {
	return true
}

func TestDebugLoggingPreservesRequestAndResponseBody(t *testing.T) {
	const payload = `{"name":"alice"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("server ReadAll() error = %v", err)
			return
		}
		if string(body) != payload {
			t.Errorf("request body = %q, want %q", body, payload)
			return
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	var gotBody string
	err := utils.NewCurl(utils.WithCurlLogger(curlTestLogger{}), utils.WithCurlDefLogOutput(true)).
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

// TestDebugLoggingDoesNotConsumeCustomReadSeekCloser 验证日志预览不会消耗自定义可 Seek 请求体。
func TestDebugLoggingDoesNotConsumeCustomReadSeekCloser(t *testing.T) {
	const payload = "seekable-body"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("server ReadAll() error = %v", err)
			return
		}
		if string(body) != payload {
			t.Errorf("server body = %q, want %q", body, payload)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	body := &trackingReadSeekCloser{Reader: bytes.NewReader([]byte(payload))}
	defer body.Close()
	err := utils.NewCurl(
		utils.WithCurlLogger(curlTestLogger{}),
		utils.WithCurlDefLogOutput(true),
	).
		SetBody(body).
		Post(srv.URL)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	if body.closed.Load() {
		t.Fatal("Post() closed the caller-owned seekable body")
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
	err := utils.NewCurl(utils.WithCurlLogger(curlTestLogger{}), utils.WithCurlMaxRetry(2)).
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

func TestCurlHeadIncludesParams(t *testing.T) {
	query := make(chan url.Values, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query <- r.URL.Query()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := utils.NewCurl(utils.WithCurlLogger(curlTestLogger{})).
		SetParams(map[string]string{"page": "2"}).
		Head(server.URL)
	if err != nil {
		t.Fatalf("Head() error = %v", err)
	}
	if got := (<-query).Get("page"); got != "2" {
		t.Fatalf("HEAD query page = %q, want %q", got, "2")
	}
}

func TestCurlDisabledDefaultLogsSuppressInternalErrors(t *testing.T) {
	t.Run("retry", func(t *testing.T) {
		logger := &countLogger{}
		rt := roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("temporary transport error")
		})

		err := utils.NewCurl(
			utils.WithCurlLogger(logger),
			utils.WithCurlDefLogOutput(false),
			utils.WithCurlMaxRetry(2),
		).BeforeClient(func(client *http.Client) error {
			client.Transport = rt
			return nil
		}).Get("http://example.test/retry")
		if err == nil {
			t.Fatal("Get() error = nil, want transport error")
		}
		if got := logger.calls.Load(); got != 0 {
			t.Fatalf("logger calls = %d, want 0", got)
		}
	})

	t.Run("body close", func(t *testing.T) {
		logger := &countLogger{}
		rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       closeErrorBody{Reader: strings.NewReader("ok")},
				Request:    req,
			}, nil
		})

		err := utils.NewCurl(
			utils.WithCurlLogger(logger),
			utils.WithCurlDefLogOutput(false),
		).BeforeClient(func(client *http.Client) error {
			client.Transport = rt
			return nil
		}).Get("http://example.test/close")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got := logger.calls.Load(); got != 0 {
			t.Fatalf("logger calls = %d, want 0", got)
		}
	})
}

func TestCurlBytesBufferBodyCanBeSentRepeatedly(t *testing.T) {
	const payload = "buffer-body"

	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("server ReadAll() error = %v", err)
			return
		}
		bodies = append(bodies, string(body))
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	curl := utils.NewCurl().SetBody(bytes.NewBufferString(payload))
	for i := range 2 {
		if err := curl.Post(srv.URL); err != nil {
			t.Fatalf("Post(%d) error = %v", i, err)
		}
	}
	if len(bodies) != 2 || bodies[0] != payload || bodies[1] != payload {
		t.Fatalf("request bodies = %#v, want two %q bodies", bodies, payload)
	}
}

func TestCurlGetPreservesExistingQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("lang") != "go" || query.Get("page") != "2" || query.Get("q") != "alice" {
			t.Errorf("query = %v", query)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	err := utils.NewCurl().
		SetParam("page", "2").
		SetParam("q", "alice").
		Get(srv.URL + "?lang=go#top")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestCurlParamsReflectExternalMutation(t *testing.T) {
	// srv 回显 GET 查询串或 POST Form body，用于验证请求始终读取当前参数。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("server ReadAll() error = %v", err)
				return
			}
			_, _ = w.Write(body)
			return
		}
		_, _ = w.Write([]byte(r.URL.Query().Encode()))
	}))
	defer srv.Close()

	// gotBody 是客户端 AfterBody 捕获的服务端回显内容，用于断言每次请求的实际参数。
	var gotBody string
	curl := utils.NewCurl().
		SetParam("page", "1").
		AfterBody(func(body []byte) error {
			gotBody = string(body)
			return nil
		})

	if err := curl.Get(srv.URL); err != nil {
		t.Fatalf("first Get() error = %v", err)
	}
	if values, err := url.ParseQuery(gotBody); err != nil || values.Get("page") != "1" {
		t.Fatalf("first query = %q, values=%v, err=%v", gotBody, values, err)
	}

	curl.SetParam("page", "2")
	if err := curl.Get(srv.URL); err != nil {
		t.Fatalf("second Get() error = %v", err)
	}
	if values, err := url.ParseQuery(gotBody); err != nil || values.Get("page") != "2" {
		t.Fatalf("second query = %q, values=%v, err=%v", gotBody, values, err)
	}

	// params 是 Params 暴露给调用方的可变 map，直接修改后必须用于下一次请求。
	params := curl.Params()
	params.Set("page", "3")
	params.Set("q", "中文 空格")
	if err := curl.PostForm(srv.URL); err != nil {
		t.Fatalf("PostForm() error = %v", err)
	}
	if values, err := url.ParseQuery(gotBody); err != nil || values.Get("page") != "3" || values.Get("q") != "中文 空格" {
		t.Fatalf("form body = %q, values=%v, err=%v", gotBody, values, err)
	}

	// 调用方可长期持有 Params 返回的 map，后续修改也必须反映到下一次请求。
	params.Set("page", "4")
	if err := curl.Get(srv.URL); err != nil {
		t.Fatalf("third Get() error = %v", err)
	}
	if values, err := url.ParseQuery(gotBody); err != nil || values.Get("page") != "4" {
		t.Fatalf("third query = %q, values=%v, err=%v", gotBody, values, err)
	}
}

func TestDebugLoggingRestoresResponseBody(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader("abcdef")}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.Copy(w, body)
	}))
	defer srv.Close()

	var gotBody string
	err := utils.NewCurl(
		utils.WithCurlLogger(curlTestLogger{}),
		utils.WithCurlDefLogOutput(true),
	).
		AfterBody(func(body []byte) error {
			gotBody = string(body)
			return nil
		}).
		Get(srv.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if gotBody != "abcdef" {
		t.Fatalf("AfterBody body = %q, want abcdef", gotBody)
	}
}

func TestNewBindsRequestIDToCustomLogger(t *testing.T) {
	logger := &captureLogger{}

	c := utils.NewCurl(utils.WithCurlLogger(logger), utils.WithCurlRequestID("req-123"))

	if c.GetRequestID() != "req-123" {
		t.Fatalf("request id = %q, want req-123", c.GetRequestID())
	}
	if !logger.hasRequestID("req-123") {
		t.Fatalf("custom logger should receive X-Request-Id, got %#v", logger.args())
	}
}

func TestSetRequestIDDoesNotStackLoggerFields(t *testing.T) {
	c := utils.NewCurl(utils.WithCurlLogger(fieldLogger{}), utils.WithCurlRequestID("req-1"))
	c.SetRequestID("req-2")

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

func TestSetStatusCodeOverridesPreviousValues(t *testing.T) {
	c := utils.NewCurl().
		SetStatusCode(http.StatusCreated).
		SetStatusCode(http.StatusAccepted)

	if len(c.GetStatusCode()) != 1 {
		t.Fatalf("status code count = %d, want 1", len(c.GetStatusCode()))
	}
	if got := c.GetStatusCode()[0]; got != http.StatusAccepted {
		t.Fatalf("status code = %d, want %d", got, http.StatusAccepted)
	}
	// 无参数调用清空上一次配置，恢复只接受默认成功状态码。
	c.SetStatusCode()
	if got := c.GetStatusCode(); len(got) != 0 {
		t.Fatalf("status codes after reset = %v, want empty", got)
	}
}

func TestCurlCloneDeepCopiesRequestState(t *testing.T) {
	base := utils.NewCurl().
		SetHeader("X-Base", "base").
		SetParam("page", "1").
		SetBodyBytes([]byte("base-body")).
		SetCookies(&http.Cookie{Name: "sid", Value: "base"}).
		SetStatusCode(http.StatusCreated).
		SetRequestID("req-base")

	cloned, err := base.Clone()
	if err != nil {
		t.Fatalf("Clone() error = %v", err)
	}

	cloned.SetHeader("X-Base", "clone").
		SetParam("page", "2").
		SetBodyBytes([]byte("clone-body")).
		SetCookies(&http.Cookie{Name: "sid", Value: "clone"}).
		SetStatusCode(http.StatusAccepted).
		SetRequestID("req-clone")

	if got := base.Header().Get("X-Base"); got != "base" {
		t.Fatalf("base header = %q, want base", got)
	}
	if got := base.Params().Get("page"); got != "1" {
		t.Fatalf("base param = %q, want 1", got)
	}
	if got := base.GetCookie("sid").Value; got != "base" {
		t.Fatalf("base cookie = %q, want base", got)
	}
	if got := base.GetStatusCode()[0]; got != http.StatusCreated {
		t.Fatalf("base status code = %d, want %d", got, http.StatusCreated)
	}
	if got := base.GetRequestID(); got != "req-base" {
		t.Fatalf("base request id = %q, want req-base", got)
	}
}

// TestCurlNewRequestGeneratesIndependentRequestID 防止派生请求沿用模板 ID，混淆请求日志。
func TestCurlNewRequestGeneratesIndependentRequestID(t *testing.T) {
	base := utils.NewCurl().SetRequestID("req-base")

	requestCurl, err := base.NewRequest()
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	if requestCurl == base {
		t.Fatal("NewRequest() should return a new instance")
	}
	if got := requestCurl.GetRequestID(); len(got) != 16 || got == base.GetRequestID() {
		t.Fatalf("request id = %q, want independent 16-character id", got)
	}
	if got := base.GetRequestID(); got != "req-base" {
		t.Fatalf("base request id = %q, want req-base", got)
	}
}

func TestCurlCloneRejectsNonReplayableBody(t *testing.T) {
	reader, _ := io.Pipe()
	base := utils.NewCurl().SetBody(reader)

	_, err := base.Clone()
	if err == nil {
		t.Fatal("Clone() expected error for non-replayable body")
	}
}

func TestCurlTemplateReuseWithNewRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("server ReadAll() error = %v", err)
			return
		}
		_, _ = w.Write([]byte(r.Header.Get("X-Req") + "|" + r.URL.Query().Get("id") + "|" + string(body)))
	}))
	defer srv.Close()

	base := utils.NewCurl().
		SetHeader("X-Base", "template").
		SetParam("base", "1")

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("req-%d", i)
			requestCurl, err := base.NewRequest()
			if err != nil {
				t.Errorf("NewRequest() error = %v", err)
				return
			}

			var got string
			err = requestCurl.
				SetHeader("X-Req", id).
				SetParam("id", id).
				SetBodyBytes([]byte(id)).
				AfterBody(func(body []byte) error {
					got = string(body)
					return nil
				}).
				Post(srv.URL)
			if err != nil {
				t.Errorf("Post() error = %v", err)
				return
			}
			want := id + "|" + id + "|" + id
			if got != want {
				t.Errorf("response = %q, want %q", got, want)
			}
		}(i)
	}
	wg.Wait()

	if got := base.Header().Get("X-Req"); got != "" {
		t.Fatalf("base X-Req header = %q, want empty", got)
	}
	if got := base.Params().Get("id"); got != "" {
		t.Fatalf("base id param = %q, want empty", got)
	}
}

func TestCurlRebuildsTransportOnlyWhenTransportConfigChanges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	curl := utils.NewCurl().SetTimeout(1)

	if err := curl.Get(server.URL); err != nil {
		t.Fatalf("first Get() error = %v", err)
	}

	// 代理/TLS 配置变更后，下一次请求应基于新配置重建 Transport。
	curl.SetProxyURL("http://127.0.0.1:1")
	err := curl.Get(server.URL)
	if err == nil {
		t.Fatal("Get() expected proxy error after transport config changed")
	}
}

func TestCurlGetContext_CancelStopsRetryBackoff(t *testing.T) {
	curl := utils.NewCurl().
		SetTimeout(1).
		SetMaxRetry(5).
		SetProxyURL("http://127.0.0.1:1")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := curl.GetContext(ctx, "http://example.com")
	if err == nil {
		t.Fatal("GetContext() error = nil, want error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("GetContext() error = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed >= 200*time.Millisecond {
		t.Fatalf("GetContext() elapsed = %v, want < 200ms", elapsed)
	}
}

func TestCurlContextCallbacksReceiveRequestContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	type ctxKey string
	const key ctxKey = "trace-id"
	ctx := context.WithValue(context.Background(), key, "ctx-123")

	curl := utils.NewCurl()

	var (
		beforeRequestCalled bool
		beforeClientCalled  bool
		afterResponseCalled bool
		afterBodyCalled     bool
		afterDoneCalled     bool
	)

	curl.
		BeforeRequestContext(func(ctx context.Context, request *http.Request) error {
			beforeRequestCalled = true
			if got := ctx.Value(key); got != "ctx-123" {
				t.Fatalf("BeforeRequestContext ctx value = %v, want ctx-123", got)
			}
			if got := request.Context().Value(key); got != "ctx-123" {
				t.Fatalf("request.Context() value = %v, want ctx-123", got)
			}
			return nil
		}).
		BeforeClientContext(func(ctx context.Context, client *http.Client) error {
			beforeClientCalled = true
			if got := ctx.Value(key); got != "ctx-123" {
				t.Fatalf("BeforeClientContext ctx value = %v, want ctx-123", got)
			}
			if client == nil {
				t.Fatal("BeforeClientContext client = nil")
			}
			return nil
		}).
		AfterResponseContext(func(ctx context.Context, response *http.Response) (bool, error) {
			afterResponseCalled = true
			if got := ctx.Value(key); got != "ctx-123" {
				t.Fatalf("AfterResponseContext ctx value = %v, want ctx-123", got)
			}
			if response == nil {
				t.Fatal("AfterResponseContext response = nil")
			}
			return false, nil
		}).
		AfterBodyContext(func(ctx context.Context, body []byte) error {
			afterBodyCalled = true
			if got := ctx.Value(key); got != "ctx-123" {
				t.Fatalf("AfterBodyContext ctx value = %v, want ctx-123", got)
			}
			if string(body) != "ok" {
				t.Fatalf("AfterBodyContext body = %q, want ok", body)
			}
			return nil
		}).
		AfterDoneContext(func(ctx context.Context, client *http.Client, request *http.Request, response *http.Response) {
			afterDoneCalled = true
			if got := ctx.Value(key); got != "ctx-123" {
				t.Fatalf("AfterDoneContext ctx value = %v, want ctx-123", got)
			}
			if client == nil || request == nil || response == nil {
				t.Fatalf("AfterDoneContext received nil argument: client=%v request=%v response=%v", client, request, response)
			}
			if got := request.Context().Value(key); got != "ctx-123" {
				t.Fatalf("AfterDoneContext request ctx value = %v, want ctx-123", got)
			}
		})

	if err := curl.GetContext(ctx, server.URL); err != nil {
		t.Fatalf("GetContext() error = %v", err)
	}
	if !beforeRequestCalled || !beforeClientCalled || !afterResponseCalled || !afterBodyCalled || !afterDoneCalled {
		t.Fatalf("callback called flags = beforeRequest:%v beforeClient:%v afterResponse:%v afterBody:%v afterDone:%v",
			beforeRequestCalled, beforeClientCalled, afterResponseCalled, afterBodyCalled, afterDoneCalled)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type countLogger struct {
	calls atomic.Int64
}

func (l *countLogger) Debug(string, ...any) { l.calls.Add(1) }
func (l *countLogger) Info(string, ...any)  { l.calls.Add(1) }
func (l *countLogger) Warn(string, ...any)  { l.calls.Add(1) }
func (l *countLogger) Error(string, ...any) { l.calls.Add(1) }
func (l *countLogger) With(...any) utils.Logger {
	return l
}
func (l *countLogger) Enabled(context.Context, utils.LogLevel) bool {
	return true
}

type closeErrorBody struct {
	io.Reader
}

func (closeErrorBody) Close() error {
	return errors.New("close failed")
}

type trackingReadCloser struct {
	io.Reader
	closed atomic.Bool
}

func (r *trackingReadCloser) Close() error {
	r.closed.Store(true)
	return nil
}

// trackingReadSeekCloser 是测试用可读、可 Seek、可关闭请求体，用于覆盖日志预览对读取游标的影响。
type trackingReadSeekCloser struct {
	*bytes.Reader             // 内存请求体数据源，模拟业务侧传入的可回放 body。
	closed        atomic.Bool // 验证发送结束后仍由调用方持有关闭责任。
}

// Close 记录关闭状态，模拟真实请求体资源释放。
func (r *trackingReadSeekCloser) Close() error {
	r.closed.Store(true)
	return nil
}

type captureLogger struct {
	mu       sync.Mutex
	withArgs []any
}

func (l *captureLogger) Debug(string, ...any) {}
func (l *captureLogger) Info(string, ...any)  {}
func (l *captureLogger) Warn(string, ...any)  {}
func (l *captureLogger) Error(string, ...any) {}
func (l *captureLogger) Enabled(context.Context, utils.LogLevel) bool {
	return true
}

func (l *captureLogger) With(args ...any) utils.Logger {
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
func (l fieldLogger) Enabled(context.Context, utils.LogLevel) bool {
	return true
}

func (l fieldLogger) With(args ...any) utils.Logger {
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
