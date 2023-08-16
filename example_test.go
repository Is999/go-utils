package utils_test

import (
	"encoding/xml"
	"github.com/Is999/go-utils"
	"net/http"
)

type User struct {
	Name      string `json:"name" xml:"name"`
	Age       int    `json:"age" xml:"age"`
	Sex       string `json:"sex" xml:"sex"`
	IsMarried bool   `json:"is_married" xml:"isMarried"`
	Address   string `json:"address" xml:"address"`
	phone     string
}

func ExampleRedirect() {
	serveMux.HandleFunc("/response/redirect", func(w http.ResponseWriter, r *http.Request) {
		// 重定向
		utils.Redirect(w, "/response/json")
	})
}

func ExampleJsonResp() {
	// 响应json数据
	serveMux.HandleFunc("/response/json", func(w http.ResponseWriter, r *http.Request) {
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
			utils.JsonResp[any](w, http.StatusNotAcceptable).Fail(20000, "fail")
			return
		}

		// 成功响应
		utils.JsonResp[User](w).Success(10000, user)
	})
}

func ExampleView() {
	// 响应html
	serveMux.HandleFunc("/response/html", func(w http.ResponseWriter, r *http.Request) {

		// 响应html数据
		utils.View(w).Html("<p>这是一个<b style=\"color: red\">段落!</b></p>")
	})

	// 响应xml
	serveMux.HandleFunc("/response/xml", func(w http.ResponseWriter, r *http.Request) {

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
		utils.View(w).Xml(string(xmlData))
	})

	// 响应text
	serveMux.HandleFunc("/response/text", func(w http.ResponseWriter, r *http.Request) {
		// 响应text数据
		utils.View(w).Text("<p>这是一个<b style=\"color: red\">段落!</b></p>")
	})

	// 显示image
	serveMux.HandleFunc("/response/show", func(w http.ResponseWriter, r *http.Request) {
		// 获取URL查询字符串参数
		file := r.URL.Query().Get("file")
		if utils.IsExist(file) {
			// 显示文件内容

			utils.View(w).Show(file)
			return
		}
		// 处理错误
		utils.View(w, http.StatusNotFound).Text("不存在的文件：" + file)
	})

	// 下载文件
	serveMux.HandleFunc("/response/download", func(w http.ResponseWriter, r *http.Request) {
		// 获取URL查询字符串参数
		file := r.URL.Query().Get("file")
		if utils.IsExist(file) {
			// 下载文件数据
			utils.View(w).Download(file)
			return
		}
		// 处理错误
		utils.View(w, http.StatusNotFound).Text("不存在的文件：" + file)
	})
}
