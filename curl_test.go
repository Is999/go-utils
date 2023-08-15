package utils_test

import (
	"context"
	"fmt"
	"github.com/Is999/go-utils"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"testing"
	"time"
)

var apiUrl = "http://localhost:54334"

func curlResponse() {
	// 退出
	exit := make(chan os.Signal)

	// 请求该路由退出
	// http://localhost:54333/response/exit
	http.HandleFunc("/curl/exit", func(w http.ResponseWriter, r *http.Request) {
		// 退出信号
		exit <- syscall.Signal(1)
	})

	http.HandleFunc("/curl/get", func(w http.ResponseWriter, r *http.Request) {
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
			utils.JsonResp[User](w, http.StatusNotAcceptable).Fail(20000, "fail", user)
			return
		}

		// 写入响应数据
		utils.JsonResp[User](w).Success(10000, user)
	})

	http.HandleFunc("/curl/post", func(w http.ResponseWriter, r *http.Request) {
		slog.Info(fmt.Sprintf("%v", r.URL.Query()))
		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				// 写入响应数据
				utils.JsonResp[string](w, http.StatusInternalServerError).Fail(2000, "Failed to read request body")
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
				utils.JsonResp[*User](w, http.StatusNotAcceptable).Fail(2000, "fail", user)
				return
			}

			// 写入响应数据
			utils.JsonResp[*User](w).Success(1000, user)
		} else {
			utils.JsonResp[string](w, http.StatusMethodNotAllowed).Fail(2000, "Method not allowed")
			return
		}
	})

	//使用默认路由创建 http server
	srv := http.Server{
		Addr:    ":54334",
		Handler: http.DefaultServeMux,
	}

	//监听 Ctrl+C 信号
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		timer := time.NewTimer(180 * time.Second)
		for {
			select {
			case <-exit:
				fmt.Println("Exit...")
				srv.Shutdown(context.Background())
			case <-timer.C:
				fmt.Println("Delayed 5s Exit...")
				//使用context控制srv.Shutdown的超时时间
				//ctx, _ := context.WithTimeout(context.Background(), time.Second)
				srv.Shutdown(context.Background())
			default:
				time.Sleep(time.Second)
				fmt.Println("default 1s...")
			}
		}
	}()

	// 启动HTTP服务器，监听在指定端口
	err := srv.ListenAndServe()
	if err != nil {
		fmt.Println("HTTP server failed to start:", err)
	}

}

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

	// 启动http服务器
	go curlResponse()

	// 关闭启动的http服务
	defer func() {
		utils.NewCurl().Head(apiUrl + "/curl/exit")
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
			url: apiUrl + "/curl/get",
			resolve: func(body []byte) error {
				res := &utils.Body[User]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return utils.Wrap(err)
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
				res := &utils.Body[User]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return utils.Wrap(err)
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
				res := &utils.Body[User]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return utils.Wrap(err)
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
			curl.Resolve(tt.args.resolve)

			// 设置响应状态码
			curl.SetStatusCode(http.StatusUnauthorized, http.StatusNotAcceptable)

			if err := curl.Get(tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("TestGet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

func TestPost(t *testing.T) {
	// 启动http服务器
	go curlResponse()

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
			url: apiUrl + "/curl/post",
			resolve: func(body []byte) error {
				res := &utils.Body[User]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return utils.Wrap(err)
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
			url: apiUrl + "/curl/post",
			resolve: func(body []byte) error {
				res := &utils.Body[User]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return utils.Wrap(err)
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
			url: apiUrl + "/curl/post",
			resolve: func(body []byte) error {
				res := &utils.Body[User]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return utils.Wrap(err)
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
			curl.Resolve(tt.args.resolve)

			// 设置响应状态码
			curl.SetStatusCode(http.StatusUnauthorized, http.StatusNotAcceptable)

			if err := curl.Post(tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("TestPost() error = %v, wantErr %v", err, tt.wantErr)
			}

		})
	}
}

func TestPostForm(t *testing.T) {
	// 启动http服务器
	go curlResponse()

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
				res := &utils.Body[string]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return utils.Wrap(err)
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
			curl.Resolve(tt.args.resolve)

			// 设置响应状态码
			curl.SetStatusCode(http.StatusUnauthorized, http.StatusNotAcceptable)

			if err := curl.PostForm(tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("TestPostForm() error = %v, wantErr %v", err, tt.wantErr)
			}

		})
	}
}

func TestPostFile(t *testing.T) {
	// 启动http服务器
	go curlResponse()

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
				res := &utils.Body[string]{}
				if err := utils.Unmarshal(body, res); err != nil {
					return utils.Wrap(err)
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
				t.Errorf("form.Reade() WrapError %v", err)
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
			curl.Resolve(tt.args.resolve)

			// 设置传输的body
			curl.SetBody(body)

			// 发送请求
			if err = curl.Post(tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("TestPostFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
