package utils

import (
	"io"
	"net/http"
	"os"
)

type Response[T any] struct {
	Body[T]
	statusCode int
	writer     http.ResponseWriter
}

type Body[T any] struct {
	Success bool   `json:"success"` // 响应状态：true 成功, false 失败
	Code    int    `json:"code"`    // 响应识别码（常用于识别响应信息来源，便于开发人员排查）
	Message string `json:"message"` // 响应信息
	Data    T      `json:"data"`    // 响应数据
}

// Success 成功响应返回Json数据
func (r *Response[T]) Success(code int, data T, message ...string) {
	r.Body.Success = true // 成功状态
	r.Code = code
	if len(message) > 0 {
		r.Message = message[0]
	} else {
		r.Message = "SUCCESS"
	}
	r.Data = data
	r.statusCode = http.StatusOK

	body, err := r.Encode()
	if err != nil {
		uid := UniqId(16)
		Errorf("%s Response Encode() Error: %v", uid, err.Error())
		http.Error(r.writer, "Encode error: "+uid, http.StatusInternalServerError)
		return
	}
	r.Write(body)
}

// Fail 失败响应返回Json数据
func (r *Response[T]) Fail(code int, message string, data ...T) {
	r.Body.Success = false // 失败状态
	r.Code = code
	r.Message = message
	if len(data) > 0 {
		r.Data = data[0]
	}

	body, err := r.Encode()
	if err != nil {
		uid := UniqId(16)
		Errorf("%s Response Encode() Error: %v", uid, err.Error())
		http.Error(r.writer, "Encode error: "+uid, http.StatusInternalServerError)
		return
	}
	r.Write(body)
}

// Text 响应text
func (r *Response[T]) Text(text string) {
	r.ContentType("text/plain")
	r.Write([]byte(text))
}

// Html 响应Html
func (r *Response[T]) Html(html string) {
	r.ContentType("text/html")
	r.Write([]byte(html))
}

// Xml 响应Xml
func (r *Response[T]) Xml(xml string) {
	r.ContentType("application/xml")
	r.Write([]byte(xml))
}

// Download 响应下载文件
//
//	filePath 文件路径
//	rename 重命名文件名
func (r *Response[T]) Download(filePath string, rename ...string) {
	// 打开图片文件
	file, err := os.Open(filePath)
	if err != nil {
		// 处理错误
		uid := UniqId(16)
		Errorf("%s Response File Open() path: %s, Error: %s", uid, filePath, err.Error())
		http.Error(r.writer, "Open file error: "+uid, http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// 响应的文件名
	fileName := ""
	if len(rename) == 0 {
		// 获取文件信息
		fileInfo, err := file.Stat()
		if err != nil {
			// 处理错误
			uid := UniqId(16)
			Errorf("%s Response File Stat() path: %s, Error: %s", uid, filePath, err.Error())
			http.Error(r.writer, "Stat file error: "+uid, http.StatusInternalServerError)
			return
		}
		fileName = fileInfo.Name()
	} else {
		fileName = rename[0]
	}

	// 设置Content-Type为文件的MIME类型
	if ctype := r.writer.Header().Get("Content-Type"); ctype == "" {
		ctype, err = GetFileType(file)
		if err != nil {
			// 处理错误
			uid := UniqId(16)
			Errorf("%s Response File GetFileType() path: %s, Error: %s", uid, filePath, err.Error())
			http.Error(r.writer, "file type error: "+uid, http.StatusInternalServerError)
			return
		}
		r.writer.Header().Set("Content-Type", ctype)
	}

	// herder 处理
	r.Herder(func(header http.Header) {
		// 设置Content-Disposition头，指定文件名
		header.Set("Content-Disposition", "attachment; filename="+fileName)
	})

	// 将图片数据写入响应
	_, err = io.Copy(r.writer, file)
	if err != nil {
		// 处理错误
		uid := UniqId(16)
		Errorf("%s Response File Copy() Error: %v", uid, err.Error())
		http.Error(r.writer, "io file error: "+uid, http.StatusInternalServerError)
		return
	}
}

// Show 响应显示文件内容：如图片
func (r *Response[T]) Show(filePath string) {
	// 打开图片文件
	file, err := os.Open(filePath)
	if err != nil {
		// 处理错误
		uid := UniqId(16)
		Errorf("%s Response File Open() path: %s, Error: %s", uid, filePath, err.Error())
		http.Error(r.writer, "Open file error: "+uid, http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// 设置Content-Type为文件的MIME类型
	if ctype := r.writer.Header().Get("Content-Type"); ctype == "" {
		ctype, err = GetFileType(file)
		if err != nil {
			// 处理错误
			uid := UniqId(16)
			Errorf("%s Response File GetFileType() path: %s, Error: %s", uid, filePath, err.Error())
			http.Error(r.writer, "file type error: "+uid, http.StatusInternalServerError)
			return
		}
		r.writer.Header().Set("Content-Type", ctype)
	}

	// 将图片数据写入响应
	_, err = io.Copy(r.writer, file)
	if err != nil {
		// 处理错误
		uid := UniqId(16)
		Errorf("%s Response File Copy() Error: %v", uid, err.Error())
		http.Error(r.writer, "io file error: "+uid, http.StatusInternalServerError)
		return
	}
}

// Write 写入响应数据
func (r *Response[T]) Write(body []byte) {
	r.writer.WriteHeader(r.statusCode)
	_, err := r.writer.Write(body)
	if err != nil {
		uid := UniqId(16)
		Errorf("%s Response Write() Error: %v", uid, err.Error())
		http.Error(r.writer, "write error: "+uid, http.StatusInternalServerError)
	}
}

// StatusCode 设置响应状态码，如：http.StatusOK
func (r *Response[T]) StatusCode(statusCode int) *Response[T] {
	r.statusCode = statusCode
	return r
}

// ContentType 设置响应头 Content-Type
func (r *Response[T]) ContentType(contentType string) *Response[T] {
	r.writer.Header().Set("Content-Type", contentType+"; charset=utf-8")
	return r
}

// Herder 设置响应头
func (r *Response[T]) Herder(f func(header http.Header)) *Response[T] {
	f(r.writer.Header())
	return r
}

// Encode 对数据编码
func (r *Response[T]) Encode() ([]byte, error) {
	return Marshal(r.Body)
}

// JsonResp 响应Json数据
func JsonResp[T any](w http.ResponseWriter, StatusCode ...int) *Response[T] {
	resp := &Response[T]{
		writer:     w,
		statusCode: http.StatusOK,
	}

	if len(StatusCode) > 0 {
		resp.statusCode = StatusCode[0]
	}

	return resp.ContentType("application/json")
}

// View 响应文本视图
func View(w http.ResponseWriter, StatusCode ...int) *Response[string] {
	resp := &Response[string]{
		writer:     w,
		statusCode: http.StatusOK,
	}

	if len(StatusCode) > 0 {
		resp.statusCode = StatusCode[0]
	}
	return resp
}

// Redirect 重定向, 状态码默认302
func Redirect(w http.ResponseWriter, url string, StatusCode ...int) {
	resp := &Response[string]{
		writer:     w,
		statusCode: http.StatusFound,
	}

	if len(StatusCode) > 0 {
		resp.statusCode = StatusCode[0]
	}

	resp.writer.Header().Set("Location", url)
	resp.writer.WriteHeader(resp.statusCode)
}
