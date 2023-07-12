package utils

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

var apiUrl = "http://localhost:8080/curl"

type User struct {
	Name      string `json:"name" xml:"name"`
	Age       int    `json:"age" xml:"age"`
	Sex       string `json:"sex" xml:"sex"`
	IsMarried bool   `json:"is_married" xml:"isMarried"`
	Address   string `json:"address" xml:"address"`
	phone     string
}

func TestGet(t *testing.T) {
	// 设置日志格式级等级
	SetLevel(DEBUG)

	//http.HandleFunc("/curl", func(w http.ResponseWriter, r *http.Request) {
	//	Info(r.URL.Query())
	//
	//	// 响应的数据
	//	user := User{
	//		Name:      r.URL.Query().Get("Name"),
	//		Age:       Str2Int(r.URL.Query().Get("Age")),
	//		Sex:       r.URL.Query().Get("Sex"),
	//		IsMarried: r.URL.Query().Get("IsMarried") == "true",
	//		Address:   r.URL.Query().Get("Address"),
	//		phone:     r.URL.Query().Get("phone"),
	//	}
	//
	//	if r.URL.Query().Get("success") == "false" {
	//		// 写入响应数据
	//		JsonResp[User](w, http.StatusNotAcceptable).Fail(2000, "fail", user)
	//		return
	//	}
	//
	//	// 写入响应数据
	//	JsonResp[User](w).Success(1000, user)
	//})

	// 创建一个curl
	curl := NewCurl()

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
			url: apiUrl,
			resolve: func(body []byte) error {
				res := &Body[User]{}
				if err := Unmarshal(body, res); err != nil {
					return err
				}
				if !res.Success {
					// 错误处理
					curl.Printf(ERROR, "resolve res error Status: %#v", res)
				} else {
					// 正常处理
					curl.Printf(INFO, "resolve res: %#v", res.Data)
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
			url: apiUrl,
			resolve: func(body []byte) error {
				res := &Body[User]{}
				if err := Unmarshal(body, res); err != nil {
					return err
				}
				if !res.Success {
					// 错误处理
					curl.Printf(ERROR, "resolve res error Status: %#v", res)
				} else {
					// 正常处理
					curl.Printf(INFO, "resolve res: %#v", res.Data)
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
			url: apiUrl,
			resolve: func(body []byte) error {
				res := &Body[User]{}
				if err := Unmarshal(body, res); err != nil {
					return err
				}
				if !res.Success {
					// 错误处理
					curl.Printf(ERROR, "resolve res error Status: %#v", res)
				} else {
					// 正常处理
					curl.Printf(INFO, "resolve res: %#v", res.Data)
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
			//	curl.CloseIdleConnections()
			//}()

			// 设置请求ID
			curl.SetRequestId()

			// 设置记录日志模式
			//curl.SetDump(true)

			// 设置日志等级
			//curl.SetLogLevel(INFO)

			// 设置重试次数
			//curl.SetMaxBadRetry(5)

			// 设置ContentType
			//curl.SetContentType("application/json")

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
	// 设置日志格式级等级
	SetLevel(DEBUG)

	//func handlePost(w http.ResponseWriter, r *http.Request) {
	//	Info(r.URL.Query())
	//	if r.Method == http.MethodPost {
	//		body, err := ioutil.ReadAll(r.Body)
	//		if err != nil {
	//			// 写入响应数据
	//			JsonResp[string](w, http.StatusInternalServerError).Fail(2000, "Failed to read request body")
	//			return
	//		}
	//
	//		// 处理接收到的 POST 数据
	//		Info("Received POST data:", string(body))
	//
	//		// 解析body
	//		user := new(User)
	//		Unmarshal(body, user)
	//
	//		// 返回响应
	//		if r.URL.Query().Get("success") == "false" {
	//			// 写入响应数据
	//			JsonResp[*User](w, http.StatusNotAcceptable).Fail(2000, "fail", user)
	//			return
	//		}
	//
	//		// 写入响应数据
	//		JsonResp[*User](w).Success(1000, user)
	//	} else {
	//		JsonResp[string](w, http.StatusMethodNotAllowed).Fail(2000, "Method not allowed")
	//		return
	//	}
	//}

	// 创建一个curl
	curl := NewCurl()

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
			url: apiUrl,
			resolve: func(body []byte) error {
				res := &Body[User]{}
				if err := Unmarshal(body, res); err != nil {
					return err
				}
				if !res.Success {
					// 错误处理
					curl.Printf(ERROR, "resolve res error Status: %#v", res)
				} else {
					// 正常处理
					curl.Printf(INFO, "resolve res: %#v", res.Data)
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
			url: apiUrl,
			resolve: func(body []byte) error {
				res := &Body[User]{}
				if err := Unmarshal(body, res); err != nil {
					return err
				}
				if !res.Success {
					// 错误处理
					curl.Printf(ERROR, "resolve res error Status: %#v", res)
				} else {
					// 正常处理
					curl.Printf(INFO, "resolve res: %#v", res.Data)
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
			url: apiUrl,
			resolve: func(body []byte) error {
				res := &Body[User]{}
				if err := Unmarshal(body, res); err != nil {
					return err
				}
				if !res.Success {
					// 错误处理
					curl.Printf(ERROR, "resolve res error Status: %#v", res)
				} else {
					// 正常处理
					curl.Printf(INFO, "resolve res: %#v", res.Data)
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
			marshal, err := Marshal(tt.args.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestPost Marshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			defer func() {
				// 关闭连接
				// curl.CloseIdleConnections()

				// 清空params
				curl.ReSetParams(nil) // 清空params
			}()

			// 设置请求ID
			curl.SetRequestId()

			// 设置记录日志模式
			curl.SetDump(true)

			// 设置日志等级
			curl.SetLogLevel(INFO)

			// 设置重试次数
			curl.SetMaxBadRetry(3)

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
	// 设置日志格式级等级
	SetLevel(DEBUG)

	// 创建一个curl
	curl := NewCurl()

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
			url: apiUrl,
			resolve: func(body []byte) error {
				res := &Body[string]{}
				if err := Unmarshal(body, res); err != nil {
					return err
				}
				if !res.Success {
					// 错误处理
					curl.Printf(INFO, "resolve res error Status: %#v", res)
				} else {
					// 正常处理
					curl.Printf(INFO, "resolve res: %#v", res)
				}
				return nil
			},
		}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置请求ID
			curl.SetRequestId()

			// 设置日志等级
			curl.SetLogLevel(INFO)

			// 设置重试次数
			curl.SetMaxBadRetry(3)

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
	// 设置日志格式级等级
	SetLevel(DEBUG)

	// 创建一个curl
	curl := NewCurl()

	// 设置curl日志等级
	curl.SetLogLevel(ERROR)

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
			url: apiUrl,
			resolve: func(body []byte) error {
				res := &Body[string]{}
				if err := Unmarshal(body, res); err != nil {
					return err
				}
				if !res.Success {
					// 错误处理
					curl.Printf(INFO, "resolve res error Status: %#v", res)
				} else {
					// 正常处理
					curl.Printf(INFO, "resolve res: %#v", res)
				}
				return nil
			},
		}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建一个form
			form := Form{
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
				t.Errorf("form.Reade() err %v", err)
			}

			// 设置请求ID
			curl.SetRequestId()

			// 设置日志等级
			curl.SetLogLevel(INFO)

			// 设置重试次数
			curl.SetMaxBadRetry(3)

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
