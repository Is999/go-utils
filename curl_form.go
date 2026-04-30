package utils

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"strings"

	apperrors "github.com/Is999/go-utils/errors"
)

// ============================ Form 结构体 ============================

// Form HTTP 表单。
// 用于构建 multipart/form-data 类型的请求，支持文件和普通字段。
type Form struct {
	// Params 表单普通字段
	Params url.Values

	// Files 表单文件字段（字段名 -> 文件路径列表）
	Files url.Values
}

// ============================ Form 构造方法 ============================

// NewForm 创建一个新的 Form 实例。
//
// 返回值：*Form 表单指针
func NewForm() *Form {
	return &Form{
		Params: make(url.Values),
		Files:  make(url.Values),
	}
}

// SetParam 设置单个表单字段。
//
// 参数说明：
//   - key：字段名
//   - value：字段值
//
// 返回值：Form 指针，支持链式调用
func (f *Form) SetParam(key, value string) *Form {
	f.Params.Set(key, value)
	return f
}

// SetParams 批量设置表单字段。
//
// 参数说明：
//   - params：字段字典
//
// 返回值：Form 指针，支持链式调用
func (f *Form) SetParams(params map[string]string) *Form {
	for key, value := range params {
		f.SetParam(key, value)
	}
	return f
}

// AddParam 对表单字段添加多个值。
//
// 参数说明：
//   - key：字段名
//   - values：多个值
//
// 返回值：Form 指针，支持链式调用
func (f *Form) AddParam(key string, values ...string) *Form {
	for _, value := range values {
		f.Params.Add(key, value)
	}
	return f
}

// AddParams 批量添加表单字段值。
//
// 参数说明：
//   - params：字段字典，值为切片
//
// 返回值：Form 指针，支持链式调用
func (f *Form) AddParams(params map[string][]string) *Form {
	for key, values := range params {
		if values != nil && len(values) > 0 {
			f.AddParam(key, values...)
		}
	}
	return f
}

// DelParams 删除指定的表单字段。
//
// 参数说明：
//   - keys：可变数量的字段名
func (f *Form) DelParams(keys ...string) {
	for _, key := range keys {
		f.Params.Del(key)
	}
}

// SetFile 设置单个文件字段。
//
// 参数说明：
//   - fieldName：表单字段名
//   - filePath：文件路径
//
// 返回值：Form 指针，支持链式调用
func (f *Form) SetFile(fieldName, filePath string) *Form {
	f.Files.Set(fieldName, filePath)
	return f
}

// SetFiles 批量设置文件字段。
//
// 参数说明：
//   - files：文件字典（字段名 -> 文件路径）
//
// 返回值：Form 指针，支持链式调用
func (f *Form) SetFiles(files map[string]string) *Form {
	for name, path := range files {
		f.SetFile(name, path)
	}
	return f
}

// AddFile 对文件字段添加多个文件路径。
//
// 参数说明：
//   - fieldName：表单字段名
//   - filePath：可变数量的文件路径
//
// 返回值：Form 指针，支持链式调用
func (f *Form) AddFile(fieldName string, filePath ...string) *Form {
	for _, path := range filePath {
		f.Files.Add(fieldName, path)
	}
	return f
}

// AddFiles 批量添加文件字段。
//
// 参数说明：
//   - files：文件字典，值为切片
//
// 返回值：Form 指针，支持链式调用
func (f *Form) AddFiles(files map[string][]string) *Form {
	for name, paths := range files {
		if paths != nil && len(paths) > 0 {
			f.AddFile(name, paths...)
		}
	}
	return f
}

// DelFiles 删除指定的文件字段。
//
// 参数说明：
//   - fieldNames：可变数量的字段名
func (f *Form) DelFiles(fieldNames ...string) {
	for _, name := range fieldNames {
		f.Files.Del(name)
	}
}

// ============================ Form Reader ============================

// Reader 读取 Form 内容，转换为可上传的 body 和 content-type。
// 如果没有文件，返回 application/x-www-form-urlencoded 格式；
// 如果有文件，返回 multipart/form-data 格式。
//
// 返回值：
//   - body：io.Reader 请求体
//   - contentType：Content-Type
//   - err：错误信息
func (f *Form) Reader() (body io.Reader, contentType string, err error) {
	// 无文件时返回 URL 编码格式
	if f.Files == nil || len(f.Files) == 0 {
		return strings.NewReader(f.Params.Encode()), "application/x-www-form-urlencoded", nil
	}

	// 创建 multipart writer
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)

	// 处理普通表单字段
	if f.Params != nil {
		for key, values := range f.Params {
			for _, value := range values {
				if err := writer.WriteField(key, value); err != nil {
					return nil, "", apperrors.Wrap(err)
				}
			}
		}
	}

	// 处理文件上传
	for fieldName, files := range f.Files {
		for _, filePath := range files {
			if err := f.createFormFile(writer, fieldName, filePath); err != nil {
				return nil, "", apperrors.Wrap(err)
			}
		}
	}

	// 关闭 writer
	if err := writer.Close(); err != nil {
		return nil, "", apperrors.Wrap(err)
	}

	return buf, writer.FormDataContentType(), nil
}

// createFormFile 创建表单文件字段。
//
// 参数说明：
//   - writer：multipart.Writer
//   - fieldName：字段名
//   - filePath：文件路径
//
// 返回值：错误信息
func (f *Form) createFormFile(writer *multipart.Writer, fieldName, filePath string) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return apperrors.Wrap(err)
	}
	defer file.Close()

	// 创建表单文件头
	part, err := writer.CreateFormFile(fieldName, filePath)
	if err != nil {
		return apperrors.Wrap(err)
	}

	// 复制文件内容
	if _, err = io.Copy(part, file); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}
