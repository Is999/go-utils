package utils

import (
	"encoding/xml"
	"io"
	"log/slog"
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
	Code    int    `json:"code"`    // 响应识别码
	Message string `json:"message"` // 响应信息
	Data    T      `json:"data"`    // 响应数据
}

// Success 成功响应返回Json数据
//
//	code 响应识别码
//	data 响应数据
//	message 响应信息
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
		// 记录日志
		slog.Error(err.Error(), "trace", slog.GroupValue(
			slog.String("code", uid),
			slog.String("desc", "Success r.Encode()"),
			slog.Any("data", data),
		))
		// 响应
		http.Error(r.writer, "Json encoding error, code-"+uid, http.StatusInternalServerError)
		return
	}
	r.Write(body)
}

// Fail 失败响应返回Json数据
//
//	code 响应识别码
//	message 响应信息
//	data 响应数据
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
		// 记录日志
		slog.Error(err.Error(), "trace", slog.GroupValue(
			slog.String("code", uid),
			slog.String("desc", "Fail r.Encode()"),
			slog.Any("data", data),
		))
		// 响应
		http.Error(r.writer, "Json encoding error, code-"+uid, http.StatusInternalServerError)
		return
	}
	r.Write(body)
}

// Text 响应text
func (r *Response[T]) Text(data string) {
	r.ContentType("text/plain")
	r.Write([]byte(data))
}

// Html 响应Html
func (r *Response[T]) Html(data string) {
	r.ContentType("text/html")
	r.Write([]byte(data))
}

// Xml 响应Xml
func (r *Response[T]) Xml(data any) {
	xmlData, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		uid := UniqId(16)
		// 记录日志
		slog.Error(err.Error(), "trace", slog.GroupValue(
			slog.String("code", uid),
			slog.String("desc", "Xml xml.MarshalIndent()"),
			slog.Any("data", data),
		))
		// 响应
		http.Error(r.writer, "Xml encoding error, code-"+uid, http.StatusInternalServerError)
		return
	}

	r.ContentType("application/xml")
	r.Write([]byte(xml.Header))
	r.Write(xmlData)
}

// Download 响应下载文件
//
//	filePath 文件路径
//	rename 重命名文件名
func (r *Response[T]) Download(filePath string, rename ...string) {
	// 打开图片文件
	file, err := os.Open(filePath)
	if err != nil {
		uid := UniqId(16)
		// 记录日志
		slog.Error(err.Error(), "trace", slog.GroupValue(
			slog.String("code", uid),
			slog.String("desc", "Download os.Open()"),
			slog.String("filePath", filePath),
		))
		// 响应
		http.Error(r.writer, "Open file error, code-"+uid, http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// 响应的文件名
	fileName := ""
	if len(rename) == 0 {
		// 获取文件信息
		fileInfo, err := file.Stat()
		if err != nil {
			uid := UniqId(16)
			// 记录日志
			slog.Error(err.Error(), "trace", slog.GroupValue(
				slog.String("code", uid),
				slog.String("desc", "Download file.Stat()"),
				slog.String("filePath", filePath),
			))
			// 响应
			http.Error(r.writer, "Stat file error, code-"+uid, http.StatusInternalServerError)
			return
		}
		fileName = fileInfo.Name()
	} else {
		fileName = rename[0]
	}

	// 设置Content-Type为文件的MIME类型
	if ctype := r.writer.Header().Get("Content-Type"); ctype == "" {
		ctype, err = FileType(file)
		if err != nil {
			uid := UniqId(16)
			// 记录日志
			slog.Error(err.Error(), "trace", slog.GroupValue(
				slog.String("code", uid),
				slog.String("desc", "Download FileType()"),
				slog.String("filePath", filePath),
			))
			// 响应
			http.Error(r.writer, "File type error, code-"+uid, http.StatusInternalServerError)
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
		uid := UniqId(16)
		// 记录日志
		slog.Error(err.Error(), "trace", slog.GroupValue(
			slog.String("code", uid),
			slog.String("desc", "Download io.Copy"),
			slog.String("filePath", filePath),
		))
		// 响应
		http.Error(r.writer, "io error, code-"+uid, http.StatusInternalServerError)
		return
	}
}

// Show 响应显示文件内容：如图片
func (r *Response[T]) Show(filePath string) {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		uid := UniqId(16)
		// 记录日志
		slog.Error(err.Error(), "trace", slog.GroupValue(
			slog.String("code", uid),
			slog.String("desc", "Show os.Open()"),
			slog.String("filePath", filePath),
		))
		// 响应
		http.Error(r.writer, "Open file error, code-"+uid, http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// 设置Content-Type为文件的MIME类型
	if ctype := r.writer.Header().Get("Content-Type"); ctype == "" {
		ctype, err = FileType(file)
		if err != nil {
			uid := UniqId(16)
			// 记录日志
			slog.Error(err.Error(), "trace", slog.GroupValue(
				slog.String("code", uid),
				slog.String("desc", "Show FileType()"),
				slog.String("filePath", filePath),
			))
			// 响应
			http.Error(r.writer, "File type error, code-"+uid, http.StatusInternalServerError)
			return
		}
		r.writer.Header().Set("Content-Type", ctype)
	}

	// 将图片数据写入响应
	_, err = io.Copy(r.writer, file)
	if err != nil {
		uid := UniqId(16)
		// 记录日志
		slog.Error(err.Error(), "trace", slog.GroupValue(
			slog.String("code", uid),
			slog.String("desc", "Show io.Copy"),
			slog.String("filePath", filePath),
		))
		// 响应
		http.Error(r.writer, "io error, code-"+uid, http.StatusInternalServerError)
		return
	}
}

// Write 写入响应数据
func (r *Response[T]) Write(body []byte) {
	r.writer.WriteHeader(r.statusCode)
	_, err := r.writer.Write(body)
	if err != nil {
		uid := UniqId(16)
		slog.Error("Response Write()", "UID", uid, "err", err.Error())
		// 记录日志
		slog.Error(err.Error(), "trace", slog.GroupValue(
			slog.String("code", uid),
			slog.String("desc", "Write r.writer.Write"),
			slog.String("body", string(body)),
		))
		// 响应
		http.Error(r.writer, "Write error, code-"+uid, http.StatusInternalServerError)
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
//
//	statusCode 响应状态码：默认响应 200
func JsonResp[T any](w http.ResponseWriter, statusCode ...int) *Response[T] {
	resp := &Response[T]{
		writer:     w,
		statusCode: http.StatusOK,
	}

	if len(statusCode) > 0 {
		resp.statusCode = statusCode[0]
	}

	return resp.ContentType("application/json")
}

// View 响应文本视图
//
//	statusCode 响应状态码：默认响应 200
func View(w http.ResponseWriter, statusCode ...int) *Response[string] {
	resp := &Response[string]{
		writer:     w,
		statusCode: http.StatusOK,
	}

	if len(statusCode) > 0 {
		resp.statusCode = statusCode[0]
	}
	return resp
}

// Redirect 重定向
//
//	url 重定向地址
//	statusCode 响应状态码：默认响应 302
func Redirect(w http.ResponseWriter, url string, statusCode ...int) {
	resp := &Response[string]{
		writer:     w,
		statusCode: http.StatusFound,
	}

	if len(statusCode) > 0 {
		resp.statusCode = statusCode[0]
	}

	resp.writer.Header().Set("Location", url)
	resp.writer.WriteHeader(resp.statusCode)
}
