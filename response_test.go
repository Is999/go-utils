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
			// 写入响应数据
			JsonResp[User](w, http.StatusNotAcceptable).Fail(2000, "fail", user)
			return
		}
		// 写入响应数据
		JsonResp[User](w).Success(1000, user)
	})

	// 响应html
	http.HandleFunc("/response/html", func(w http.ResponseWriter, r *http.Request) {

		// 写入响应数据
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

		// 写入响应数据
		View(w).Xml(string(xmlData))
	})

	// 响应text
	http.HandleFunc("/response/text", func(w http.ResponseWriter, r *http.Request) {
		// 写入响应数据
		View(w).Text("<p>这是一个<b style=\"color: red\">段落!</b></p>")
	})

	// 响应image
	http.HandleFunc("/response/show", func(w http.ResponseWriter, r *http.Request) {
		// 获取URL查询字符串参数
		queryParam := r.URL.Query().Get("v")
		if IsExist(queryParam) {
			// 写入响应数据
			View(w).
				//Herder(func(header http.Header) {
				//	header.Set("X-Lang", "ZH-CN")
				//}).
				//ContentType("image/png").
				Show(queryParam)
			return
		}
		// 处理错误
		http.Error(w, "不存在的文件："+queryParam, http.StatusNotFound)
	})

	// 响应文件
	http.HandleFunc("/response/download", func(w http.ResponseWriter, r *http.Request) {
		// 获取URL查询字符串参数
		queryParam := r.URL.Query().Get("v")
		if IsExist(queryParam) {
			// 写入响应数据
			View(w).Download(queryParam)
			return
		}
		// 处理错误
		http.Error(w, "不存在的文件："+queryParam, http.StatusNotFound)
	})

	// 启动HTTP服务器，监听在指定端口
	//err := http.ListenAndServe(":8080", nil)
	//if err != nil {
	//	fmt.Println("HTTP server failed to start:", err)
	//}
}
