package utils

import (
	"encoding/xml"
	"net/http"
	"testing"
)

func TestResponse(t *testing.T) {
	type User struct {
		Name      string `json:"name" xml:"name"`
		Age       int    `json:"age" xml:"age"`
		Sex       string `json:"sex" xml:"sex"`
		IsMarried bool   `json:"is_married" xml:"isMarried"`
		Address   string `json:"address" xml:"address"`
		phone     string
	}

	http.HandleFunc("/response/redirect", func(w http.ResponseWriter, r *http.Request) {
		// 重定向
		Redirect(w, "/response/json")
	})

	// 响应json数据
	http.HandleFunc("/response/json", func(w http.ResponseWriter, r *http.Request) {

		// 获取URL查询字符串参数
		queryParam := r.URL.Query().Get("v")

		// 响应的数据
		user := User{
			Name:      "张三",
			Age:       22,
			Sex:       "男",
			IsMarried: false,
			Address:   "北京市",
			phone:     "131188889999",
		}

		if queryParam == "fail" {
			// 错误响应
			JsonResp[User](w, http.StatusNotAcceptable).Fail(2000, "fail", user)
			return
		}
		// 成功响应
		JsonResp[User](w).Success(1000, user)
	})

	// 响应html
	http.HandleFunc("/response/html", func(w http.ResponseWriter, r *http.Request) {

		// 响应html数据
		View(w).Html("<p>这是一个<b style=\"color: red\">段落!</b></p>")
	})

	// 响应xml
	http.HandleFunc("/response/xml", func(w http.ResponseWriter, r *http.Request) {

		// 响应的数据
		user := User{
			Name:      "张三",
			Age:       22,
			Sex:       "男",
			IsMarried: false,
			Address:   "北京市",
			phone:     "131188889999",
		}

		// 将Person对象转换为XML格式数据
		xmlData, err := xml.MarshalIndent(user, "", "  ")
		if err != nil {
			// 处理错误
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 响应xml数据
		View(w).Xml(string(xmlData))
	})

	// 响应text
	http.HandleFunc("/response/text", func(w http.ResponseWriter, r *http.Request) {
		// 响应text数据
		View(w).Text("<p>这是一个<b style=\"color: red\">段落!</b></p>")
	})

	// 显示image
	http.HandleFunc("/response/show", func(w http.ResponseWriter, r *http.Request) {
		// 获取URL查询字符串参数
		file := r.URL.Query().Get("file")
		if IsExist(file) {
			// 显示文件内容
			View(w).Show(file)
			return
		}
		// 处理错误
		View(w, http.StatusNotFound).Text("不存在的文件：" + file)
	})

	// 下载文件
	http.HandleFunc("/response/download", func(w http.ResponseWriter, r *http.Request) {
		// 获取URL查询字符串参数
		file := r.URL.Query().Get("file")
		if IsExist(file) {
			// 下载文件数据
			View(w).Download(file)
			return
		}
		// 处理错误
		View(w, http.StatusNotFound).Text("不存在的文件：" + file)
	})

	// 启动HTTP服务器，监听在指定端口
	//err := http.ListenAndServe(":8080", nil)
	//if err != nil {
	//	fmt.Println("HTTP server failed to start:", err)
	//}
}
