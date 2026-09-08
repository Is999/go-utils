package utils_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/Is999/go-utils"
)

// User 供响应示例和 HTTP 测试共用，未导出的 phone 不参与 JSON/XML 编码。
type User struct {
	Name      string `json:"name" xml:"name"`
	Age       int    `json:"age" xml:"age"`
	Sex       string `json:"sex" xml:"sex"`
	IsMarried bool   `json:"is_married" xml:"isMarried"`
	Address   string `json:"address" xml:"address"`
	phone     string
}

// ExampleRedirect 核对默认 302 状态和 Location；实际处理器传入自己的 ResponseWriter。
func ExampleRedirect() {
	w := httptest.NewRecorder()
	utils.Redirect(w, "/response/json")
	fmt.Println(w.Code, w.Header().Get("Location"))

	// Output: 302 /response/json
}

// registerRedirectExample 为集成测试注册独立路由，不修改默认 ServeMux。
func registerRedirectExample(mux *http.ServeMux) {
	mux.HandleFunc("/response/redirect", func(w http.ResponseWriter, r *http.Request) {
		utils.Redirect(w, "/response/json")
	})
}

// ExampleJSON 展示成功响应的固定信封，业务数据保存在 data 中。
func ExampleJSON() {
	w := httptest.NewRecorder()
	utils.JSON(w).Success(10000, map[string]string{"name": "张三"})
	fmt.Println(w.Body.String())

	// Output: {"success":true,"code":10000,"message":"SUCCESS","data":{"name":"张三"}}
}

// registerJSONExample 通过查询参数切换成功与失败响应，供 Response 集成测试使用。
func registerJSONExample(mux *http.ServeMux) {
	mux.HandleFunc("/response/json", func(w http.ResponseWriter, r *http.Request) {
		queryParam := r.URL.Query().Get("v")
		user := User{
			Name:      "张三",
			Age:       22,
			Sex:       "男",
			IsMarried: false,
			Address:   "北京市",
			phone:     "131188889999",
		}

		if queryParam == "fail" {
			utils.JSON(w, utils.WithStatusCode(http.StatusNotAcceptable)).Fail(20000, "fail")
			return
		}

		utils.JSON(w).Success(10000, user)
	})
}

// ExampleView 直接写入 HTML，内容由调用方准备，不经过模板转义。
func ExampleView() {
	w := httptest.NewRecorder()
	utils.View(w).HTML("<p>hello</p>")
	fmt.Println(w.Header().Get("Content-Type"))
	fmt.Println(w.Body.String())

	// Output:
	// text/html; charset=utf-8
	// <p>hello</p>
}

// registerViewExample 复用不同格式的响应入口；文件路径由测试准备，访问授权属于调用方。
func registerViewExample(mux *http.ServeMux) {
	mux.HandleFunc("/response/html", func(w http.ResponseWriter, r *http.Request) {
		utils.View(w).HTML("<p>这是一个<b style=\"color: red\">段落!</b></p>")
	})

	mux.HandleFunc("/response/xml", func(w http.ResponseWriter, r *http.Request) {
		user := User{
			Name:      "张三",
			Age:       22,
			Sex:       "男",
			IsMarried: false,
			Address:   "北京市",
			phone:     "131188889999",
		}

		utils.View(w).XML(user)
	})

	mux.HandleFunc("/response/text", func(w http.ResponseWriter, r *http.Request) {
		utils.View(w).Text("<p>这是一个<b style=\"color: red\">段落!</b></p>")
	})

	mux.HandleFunc("/response/show", func(w http.ResponseWriter, r *http.Request) {
		file := r.URL.Query().Get("file")
		if utils.IsExist(file) {
			utils.View(w).Show(file)
			return
		}
		utils.View(w, utils.WithStatusCode(http.StatusNotFound)).Text("不存在的文件：" + file)
	})

	mux.HandleFunc("/response/download", func(w http.ResponseWriter, r *http.Request) {
		file := r.URL.Query().Get("file")
		if utils.IsExist(file) {
			utils.View(w).Download(file)
			return
		}
		utils.View(w, utils.WithStatusCode(http.StatusNotFound)).Text("不存在的文件：" + file)
	})
}
