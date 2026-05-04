package utils_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/Is999/go-utils"
	"github.com/Is999/go-utils/errors"
)

var apiUrl = "http://127.0.0.1:54334"

func setLogConfig() {
	// 日志等级
	levelVar := &slog.LevelVar{}
	levelVar.Set(slog.LevelDebug)

	opts := &slog.HandlerOptions{
		AddSource: true,     // 输出日志的文件和行号
		Level:     levelVar, // 日志等级
	}

	// 日志输出格式
	handler := slog.NewTextHandler(os.Stdout, opts)
	//handler := slog.NewJSONHandler(os.Stdout, opts)

	// 修改默认的日志输出方式
	slog.SetDefault(slog.New(handler))
}

func TestGet(t *testing.T) {
	// 日志配置
	setLogConfig()

	// 退出
	exit := make(chan os.Signal)

	// 启动http服务器
	go func() {
		serveMux := http.NewServeMux()

		// GET 请求
		serveMux.HandleFunc("/curl/get", func(w http.ResponseWriter, r *http.Request) {
			slog.Info(fmt.Sprintf("%v", r.URL.Query()))

			// 响应的数据
			user := User{
				Name:      r.URL.Query().Get("Name"),
				Age:       utils.Str2Int(r.URL.Query().Get("Age")),
				Sex:       r.URL.Query().Get("Sex"),
				IsMarried: r.URL.Query().Get("IsMarried") == "true",
				Address:   r.URL.Query().Get("Address"),
				phone:     r.URL.Query().Get("phone"),
			}

			if r.URL.Query().Get("success") == "false" {
				// 写入响应数据
				utils.Json(w, utils.WithStatusCode(http.StatusNotAcceptable)).Fail(20000, "fail", user)
				return
			}

			// 写入响应数据
			utils.Json(w).Success(10000, user)
		})

		httpServer(":54334", serveMux, exit)
	}()
	waitHTTPServer(t, ":54334")

	// 关闭启动的http服务
	defer func() {
		// 退出信号
		exit <- syscall.Signal(1)
	}()

	// 创建一个curl，开启默认日志
	curl := utils.NewCurl().SetDefLogOutput(true)

	type args struct {
		url         string
		user        User
		wantSuccess bool
		resolve     func(body []byte) error
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{
			url: apiUrl + "/curl/get",
			resolve: func(body []byte) error {
				res := &RespBody[User]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return errors.Tag(err)
				}
				if !res.Success {
					// 错误处理
					curl.Logger.Error("失败", "body", res)
				} else {
					// 正常处理
					curl.Logger.Info("成功", "body", res)
				}
				return nil
			},
			user: User{
				Name:      "Andy",
				Age:       18,
				Sex:       "男",
				IsMarried: false,
				Address:   "火星",
				phone:     "18899995555",
			},
			wantSuccess: true,
		}, wantErr: false},
		{name: "002", args: args{
			url: apiUrl + "/curl/get",
			resolve: func(body []byte) error {
				res := &RespBody[User]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return errors.Tag(err)
				}
				if !res.Success {
					// 错误处理
					curl.Logger.Error("失败", "body", res)
				} else {
					// 正常处理
					curl.Logger.Info("成功", "body", res)
				}
				return nil
			},
			user: User{
				Name:      "Lisa",
				Age:       28,
				Sex:       "女",
				IsMarried: true,
				Address:   "月星",
				phone:     "18899996666",
			},
			wantSuccess: true,
		}, wantErr: false},
		{name: "003", args: args{
			url: apiUrl + "/curl/get",
			resolve: func(body []byte) error {
				res := &RespBody[User]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return errors.Tag(err)
				}
				if !res.Success {
					// 错误处理
					curl.Logger.Error("失败", "body", res)
				} else {
					// 正常处理
					curl.Logger.Info("成功", "body", res)
				}
				return nil
			},
			user: User{
				Name:      "Jack",
				Age:       38,
				Sex:       "男",
				IsMarried: false,
				Address:   "金星",
				phone:     "18899998888",
			},
		}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//defer func() {
			//	// 关闭连接
			//	Curl.CloseIdleConnections()
			//}()

			// 设置请求ID
			curl.SetRequestId()

			// 设置记录日志模式
			//Curl.SetDump(true)

			// 设置重试次数
			//Curl.SetMaxRetry(5)

			// 设置ContentType
			//Curl.SetContentType("application/json")

			// 添加请求参数
			curl.SetParam("success", fmt.Sprint(tt.args.wantSuccess))

			userType := reflect.TypeOf(tt.args.user)
			userValue := reflect.ValueOf(tt.args.user)
			for i := 0; i < userType.NumField(); i++ {
				// 获取每个成员的结构体字段类型
				field := userType.Field(i)
				value := userValue.FieldByName(field.Name)
				curl.SetParam(field.Name, fmt.Sprint(value))
			}

			// 解析响应数据
			curl.AfterBody(tt.args.resolve)

			// 设置响应状态码
			curl.SetStatusCode(http.StatusUnauthorized, http.StatusNotAcceptable)

			if err := curl.Get(tt.args.url); (err != nil) != tt.wantErr {
				slog.Error(err.Error(), "trace", errors.Trace(err))
				t.Errorf("TestGet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

func TestPost(t *testing.T) {
	const postAPIURL = "http://127.0.0.1:54335"

	// 日志配置
	setLogConfig()

	// 退出
	exit := make(chan os.Signal)

	// 启动http服务器
	go func() {
		serveMux := http.NewServeMux()

		// POST 请求
		serveMux.HandleFunc("/curl/post", func(w http.ResponseWriter, r *http.Request) {
			slog.Info(fmt.Sprintf("%v", r.URL.Query()))
			if r.Method == http.MethodPost {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					utils.Json(w, utils.WithStatusCode(http.StatusInternalServerError)).Fail(2000, "Failed to read request body")
					return
				}

				// 处理接收到的 POST 数据
				slog.Info("Received POST", "body", string(body))

				// 解析body
				user := new(User)
				utils.Unmarshal(body, user)

				// 返回响应
				if r.URL.Query().Get("success") == "false" {
					// 写入响应数据
					utils.Json(w, utils.WithStatusCode(http.StatusNotAcceptable)).Fail(2000, "fail", user)
					return
				}

				// 写入响应数据
				utils.Json(w).Success(1000, user)
			} else {
				utils.Json(w, utils.WithStatusCode(http.StatusMethodNotAllowed)).Fail(2000, "Method not allowed")
				return
			}
		})

		httpServer(":54335", serveMux, exit)
	}()
	waitHTTPServer(t, ":54335")

	// 关闭启动的http服务
	defer func() {
		// 退出信号
		exit <- syscall.Signal(1)
	}()

	// 创建一个curl
	curl := utils.NewCurl()

	type args struct {
		url         string
		user        User
		wantSuccess bool
		resolve     func(body []byte) error
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{
			url: postAPIURL + "/curl/post",
			resolve: func(body []byte) error {
				res := &RespBody[User]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return errors.Tag(err)
				}
				if !res.Success {
					// 错误处理
					curl.Logger.Error("失败", "body", res)
				} else {
					// 正常处理
					curl.Logger.Info("成功", "body", res)
				}
				return nil
			},
			user: User{
				Name:      "Andy",
				Age:       18,
				Sex:       "男",
				IsMarried: false,
				Address:   "火星",
				phone:     "18899995555",
			},
			wantSuccess: true,
		}, wantErr: false},
		{name: "002", args: args{
			url: postAPIURL + "/curl/post",
			resolve: func(body []byte) error {
				res := &RespBody[User]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return errors.Tag(err)
				}
				if !res.Success {
					// 错误处理
					curl.Logger.Error("失败", "body", res)
				} else {
					// 正常处理
					curl.Logger.Info("成功", "body", res)
				}
				return nil
			},
			user: User{
				Name:      "Lisa",
				Age:       28,
				Sex:       "女",
				IsMarried: true,
				Address:   "月星",
				phone:     "18899996666",
			},
			wantSuccess: true,
		}, wantErr: false},
		{name: "003", args: args{
			url: postAPIURL + "/curl/post",
			resolve: func(body []byte) error {
				res := &RespBody[User]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return errors.Tag(err)
				}
				if !res.Success {
					// 错误处理
					curl.Logger.Error("失败", "body", res)
				} else {
					// 正常处理
					curl.Logger.Info("成功", "body", res)
				}
				return nil
			},
			user: User{
				Name:      "Jack",
				Age:       38,
				Sex:       "男",
				IsMarried: false,
				Address:   "金星",
				phone:     "18899998888",
			},
		}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marshal, err := utils.Marshal(tt.args.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestPost Marshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			defer func() {
				// 关闭连接
				// Curl.CloseIdleConnections()

				// 清空params
				curl.ReSetParams(nil) // 清空params
			}()

			// 设置请求ID
			curl.SetRequestId()

			// 设置记录日志模式
			curl.SetDump(true)

			// 设置重试次数
			curl.SetMaxRetry(3)

			// 设置ContentType
			curl.SetContentType("application/json")

			// 添加参数url pathinfo模式参数
			curl.SetParam("page", "2").AddParam("limit", "10")
			curl.SetParam("success", fmt.Sprint(tt.args.wantSuccess))

			// 添加post参数
			curl.SetBodyBytes(marshal)

			// 解析响应数据
			curl.AfterBody(tt.args.resolve)

			// 设置响应状态码
			curl.SetStatusCode(http.StatusUnauthorized, http.StatusNotAcceptable)

			if err := curl.Post(tt.args.url); (err != nil) != tt.wantErr {
				slog.Error(err.Error(), "trace", errors.Trace(err))
				t.Errorf("TestPost() error = %v, wantErr %v", err, tt.wantErr)
			}

		})
	}
}

func TestPostForm(t *testing.T) {
	// 日志配置
	setLogConfig()

	// 退出
	exit := make(chan os.Signal)

	// 启动http服务器
	go func() {
		serveMux := http.NewServeMux()

		// POST FORM 请求
		serveMux.HandleFunc("/curl/form", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				// 解析表单数据
				err := r.ParseForm()
				if err != nil {
					utils.Json(w, utils.WithStatusCode(http.StatusBadRequest)).Fail(2000, "Error parsing form")
					return
				}
				// 处理接收到的 POST 数据
				slog.Info("Received POST FORM", "form", r.Form)

				info := make(map[string]any)
				// 获取表单字段的值
				info["name"] = r.FormValue("name")
				info["age"] = r.FormValue("age")
				info["language"] = r.FormValue("language")

				// 获取复选框字段的值
				info["friends"] = r.Form["friends"]
				info["hobby"] = r.Form["hobby"]

				// 返回响应
				if r.URL.Query().Get("success") == "false" {
					// 写入响应数据
					utils.Json(w, utils.WithStatusCode(http.StatusNotAcceptable)).Fail(2000, "fail", info)
					return
				}

				// 写入响应数据
				utils.Json(w).Success(1000, info)
			} else {
				utils.Json(w, utils.WithStatusCode(http.StatusMethodNotAllowed)).Fail(2000, "Method not allowed")
				return
			}
		})

		httpServer(":54334", serveMux, exit)
	}()
	waitHTTPServer(t, ":54334")

	// 关闭启动的http服务
	defer func() {
		// 退出信号
		exit <- syscall.Signal(1)
	}()

	// 创建一个curl
	curl := utils.NewCurl()

	type args struct {
		url     string
		resolve func(body []byte) error
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{
			url: apiUrl + "/curl/form",
			resolve: func(body []byte) error {
				res := &RespBody[map[string]any]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return errors.Tag(err)
				}
				if !res.Success {
					// 错误处理
					curl.Logger.Error("失败", "body", res)
				} else {
					// 正常处理
					curl.Logger.Info("成功", "body", res)
				}
				return nil
			},
		}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 开启默认日志
			curl.SetDefLogOutput(true)

			// 设置请求ID
			curl.SetRequestId()

			// 设置重试次数
			curl.SetMaxRetry(3)

			// 添加参数url pathinfo模式参数
			curl.SetParams(map[string]string{
				// 批量设置请求参数
				"name": "Lisa",
				"age":  "22",
			}).
				AddParams(map[string][]string{
					// 批量设置checkbox类型请求参数,与其他语言通信参数名后面或许需加上`[]`, 如：`hobby[]`
					"hobby":   {"读书", "游泳", "旅游"},
					"friends": {"Kelly", "Shirley"},
				}).
				SetParam("language", "English,中文,Français").
				AddParam("hobby", "骑行"). // hobby 追加值
				AddParam("hobby", "冒险")  // hobby 追加值

			// 解析响应数据
			curl.AfterBody(tt.args.resolve)

			// 设置响应状态码
			curl.SetStatusCode(http.StatusUnauthorized, http.StatusNotAcceptable)

			if err := curl.PostForm(tt.args.url); (err != nil) != tt.wantErr {
				slog.Error(err.Error(), "trace", errors.Trace(err))
				t.Errorf("TestPostForm() error = %v, wantErr %v", err, tt.wantErr)
			}

		})
	}
}

func TestPostFile(t *testing.T) {
	// 日志配置
	setLogConfig()

	// 退出
	exit := make(chan os.Signal)

	// 启动http服务器
	go func() {
		serveMux := http.NewServeMux()

		// POST FILE 请求
		serveMux.HandleFunc("/curl/file", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				// 解析表单数据
				err := r.ParseMultipartForm(10 << 20) // 10 MB limit for file upload
				if err != nil {
					utils.Json(w, utils.WithStatusCode(http.StatusBadRequest)).Fail(2000, "Error parsing form")
					return
				}

				// 处理接收到的 POST 数据
				slog.Info("Received POST FORM", "form", r.Form)

				info := make(map[string]any)
				// 获取表单字段的值
				info["name"] = r.FormValue("name")
				info["age"] = r.FormValue("age")
				info["language"] = r.FormValue("language")

				// 获取复选框字段的值
				info["friends"] = r.Form["friends"]
				info["hobby"] = r.Form["hobby"]

				// 处理接收到的 POST 数据
				slog.Info("Received POST File", "File", r.MultipartForm.File)

				// 处理上传的文件
				_, fileHeader, err := r.FormFile("json_file")
				if err != nil {
					utils.Json(w, utils.WithStatusCode(http.StatusInternalServerError)).Fail(2000, "Error retrieving file")
					return
				}
				info["json_file"] = map[string]any{"name": fileHeader.Filename, "size": fileHeader.Size, "type": mime.TypeByExtension(filepath.Ext(fileHeader.Filename))}

				_, fileHeader, err = r.FormFile("env_file")
				if err != nil {
					utils.Json(w, utils.WithStatusCode(http.StatusInternalServerError)).Fail(2000, "Error retrieving file")
					return
				}
				info["env_file"] = map[string]any{"name": fileHeader.Filename, "size": fileHeader.Size, "type": mime.TypeByExtension(filepath.Ext(fileHeader.Filename))}

				// 处理上传的文件
				files := r.MultipartForm.File["files"]
				filesInfo := make([]map[string]any, len(files))
				for i, fileHeader := range files {
					filesInfo[i] = map[string]any{"name": fileHeader.Filename, "size": fileHeader.Size, "type": mime.TypeByExtension(filepath.Ext(fileHeader.Filename))}
				}
				info["files"] = filesInfo

				// 创建本地文件

				// 拷贝上传文件内容到本地文件

				// 返回响应
				if r.URL.Query().Get("success") == "false" {
					// 写入响应数据
					utils.Json(w, utils.WithStatusCode(http.StatusNotAcceptable)).Fail(2000, "fail", info)
					return
				}

				// 写入响应数据
				utils.Json(w).Success(1000, info)
			} else {
				utils.Json(w, utils.WithStatusCode(http.StatusMethodNotAllowed)).Fail(2000, "Method not allowed")
				return
			}
		})

		httpServer(":54334", serveMux, exit)
	}()
	waitHTTPServer(t, ":54334")

	// 关闭启动的http服务
	defer func() {
		// 退出信号
		exit <- syscall.Signal(1)
	}()

	// 创建一个curl
	curl := utils.NewCurl()

	type args struct {
		url     string
		resolve func(body []byte) error
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{
			url: apiUrl + "/curl/file",
			resolve: func(body []byte) error {
				res := &RespBody[map[string]any]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return errors.Tag(err)
				}
				if !res.Success {
					// 错误处理
					curl.Logger.Error("失败", "body", res)
				} else {
					// 正常处理
					curl.Logger.Info("成功", "body", res)
				}
				return nil
			},
		}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建一个form
			form := utils.Form{
				Params: map[string][]string{},
				Files:  map[string][]string{},
			}

			// 设置请求参数
			form.SetParams(map[string]string{
				// 批量设置请求参数
				"name": "Lisa",
				"age":  "22",
			}).AddParams(map[string][]string{
				// 批量设置checkbox类型请求参数,与其他语言通信参数名后面或许需加上`[]`, 如：`hobby[]`
				"hobby":   {"读书", "游泳", "旅游"},
				"friends": {"Kelly", "Shirley"},
			}).SetFiles(map[string]string{
				// 批量上传文件
				"json_file": "./json.go",
				"env_file":  "./env.go",
			}).AddFiles(map[string][]string{
				// 上传多个文件
				"files": {"./html.go", "./aes.go"},
			}).AddParam("hobby", "骑行") // 对参数追加值（checkbox类型追加值才有意义，否则接收到的参数可能是非期望值）

			// 获取 body 和 contentType
			body, contentType, err := form.Reader()
			if err != nil {
				t.Errorf("form.Reade() err=%v", err)
			}

			// 设置请求ID
			curl.SetRequestId()

			// 设置重试次数
			curl.SetMaxRetry(3)

			// 设置响应状态码
			curl.SetStatusCode(http.StatusUnauthorized, http.StatusNotAcceptable)

			// 设置contentType
			curl.SetContentType(contentType)

			// 解析响应数据
			curl.AfterBody(tt.args.resolve)

			// 设置传输的body
			curl.SetBody(body)

			// 发送请求
			if err = curl.Post(tt.args.url); (err != nil) != tt.wantErr {
				slog.Error(err.Error(), "trace", errors.Trace(err))
				t.Errorf("TestPostFile() error = %#v, wantErr %v", err, tt.wantErr)
			}
		})
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

func TestBuildURLPreservesExistingQuery(t *testing.T) {
	params := mapValues("page", "2", "q", "codex")
	got, err := utils.BuildUrl("https://example.com/search?lang=go", params)
	if err != nil {
		t.Fatalf("utils.BuildUrl() error = %v", err)
	}
	for _, want := range []string{"lang=go", "page=2", "q=codex"} {
		if !strings.Contains(got, want) {
			t.Fatalf("utils.BuildUrl() = %q, missing %q", got, want)
		}
	}
}

func TestReadBodyPreviewAndRestoreClosesOriginal(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader("abcdef")}

	preview, truncated, restored, err := utils.ReadBodyPreviewAndRestore(body, 3)
	if err != nil {
		t.Fatalf("utils.ReadBodyPreviewAndRestore() error = %v", err)
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

	c := utils.NewCurl(utils.WithCurlLogger(logger), utils.WithCurlRequestId("req-123"))

	if c.GetRequestId() != "req-123" {
		t.Fatalf("request id = %q, want req-123", c.GetRequestId())
	}
	if !logger.hasRequestID("req-123") {
		t.Fatalf("custom logger should receive X-Request-Id, got %#v", logger.args())
	}
}

func TestSetRequestIDDoesNotStackLoggerFields(t *testing.T) {
	c := utils.NewCurl(utils.WithCurlLogger(fieldLogger{}), utils.WithCurlRequestId("req-1"))
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
