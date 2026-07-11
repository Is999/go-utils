package utils

import (
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Is999/go-utils/errors"
)

// ============================ Form 结构体 ============================

// Form HTTP 表单。
// 用于构建 multipart/form-data 类型的请求，支持文件和普通字段。
type Form struct {
	// Params 表单普通字段
	Params url.Values

	// Files 表单文件字段（字段名 -> 文件路径列表）
	Files url.Values

	// MaxSingleFileSize 单个文件大小上限，单位：字节；小于等于 0 表示不限制
	MaxSingleFileSize int64

	// MaxTotalFileSize 所有文件总大小上限，单位：字节；小于等于 0 表示不限制
	MaxTotalFileSize int64
}

// ============================ Form 构造方法 ============================

// NewForm 创建一个新的 Form 实例。
func NewForm() *Form {
	return &Form{
		Params: make(url.Values),
		Files:  make(url.Values),
	}
}

// SetMaxSingleFileSize 设置单个文件大小上限。
func (f *Form) SetMaxSingleFileSize(limit int64) *Form {
	f.MaxSingleFileSize = limit
	return f
}

// SetMaxTotalFileSize 设置所有文件总大小上限。
func (f *Form) SetMaxTotalFileSize(limit int64) *Form {
	f.MaxTotalFileSize = limit
	return f
}

// SetParam 设置单个表单字段。
func (f *Form) SetParam(key, value string) *Form {
	f.Params.Set(key, value)
	return f
}

// SetParams 批量设置表单字段。
func (f *Form) SetParams(params map[string]string) *Form {
	for key, value := range params {
		f.Params.Set(key, value)
	}
	return f
}

// AddParam 对表单字段添加多个值。
func (f *Form) AddParam(key string, values ...string) *Form {
	for _, value := range values {
		f.Params.Add(key, value)
	}
	return f
}

// AddParams 批量添加表单字段值。
func (f *Form) AddParams(params map[string][]string) *Form {
	for key, values := range params {
		for _, value := range values {
			f.Params.Add(key, value)
		}
	}
	return f
}

// DeleteParams 删除指定的表单字段。
func (f *Form) DeleteParams(keys ...string) {
	for _, key := range keys {
		f.Params.Del(key)
	}
}

// SetFile 设置单个文件字段。
func (f *Form) SetFile(fieldName, filePath string) *Form {
	f.Files.Set(fieldName, filePath)
	return f
}

// SetFiles 批量设置文件字段。
func (f *Form) SetFiles(files map[string]string) *Form {
	for name, path := range files {
		f.Files.Set(name, path)
	}
	return f
}

// AddFile 对文件字段添加多个文件路径。
func (f *Form) AddFile(fieldName string, filePath ...string) *Form {
	for _, path := range filePath {
		f.Files.Add(fieldName, path)
	}
	return f
}

// AddFiles 批量添加文件字段。
func (f *Form) AddFiles(files map[string][]string) *Form {
	for name, paths := range files {
		for _, path := range paths {
			f.Files.Add(name, path)
		}
	}
	return f
}

// DeleteFiles 删除指定的文件字段。
func (f *Form) DeleteFiles(fieldNames ...string) {
	for _, name := range fieldNames {
		f.Files.Del(name)
	}
}

// ============================ Form Reader ============================

// Reader 读取 Form 内容，转换为可上传的 body 和 content-type。
// 如果没有文件，返回 application/x-www-form-urlencoded 格式；
// 如果有文件，返回 multipart/form-data 格式。
func (f *Form) Reader() (body io.Reader, contentType string, err error) {
	// 无文件时返回 URL 编码格式
	if len(f.Files) == 0 {
		return strings.NewReader(f.Params.Encode()), "application/x-www-form-urlencoded", nil
	}

	// 先校验文件大小上限，避免请求发送过程中才发现超限。
	if err := f.validateFiles(); err != nil {
		return nil, "", errors.Tag(err)
	}

	// 使用 io.Pipe + multipart.Writer 流式拼装 body，避免大文件全量读入内存。
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	contentType = writer.FormDataContentType()
	go func() {
		if writeErr := f.writeMultipart(writer); writeErr != nil {
			_ = pipeWriter.CloseWithError(errors.Tag(writeErr))
			return
		}
		if closeErr := writer.Close(); closeErr != nil {
			_ = pipeWriter.CloseWithError(errors.Tag(closeErr))
			return
		}
		_ = pipeWriter.Close()
	}()
	return pipeReader, contentType, nil
}

// createFormFile 创建表单文件字段。
func (f *Form) createFormFile(writer *multipart.Writer, fieldName, filePath string) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return errors.Tag(err)
	}
	defer file.Close()

	// 创建表单文件头，仅暴露文件名，避免把本地目录结构写入 multipart。
	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return errors.Tag(err)
	}

	// 复制文件内容
	if _, err = io.Copy(part, file); err != nil {
		return errors.Tag(err)
	}

	return nil
}

// writeMultipart 将表单参数与文件按 multipart/form-data 规范流式写入 writer。
func (f *Form) writeMultipart(writer *multipart.Writer) error {
	// 处理普通表单字段
	for key, values := range f.Params {
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				return errors.Tag(err)
			}
		}
	}

	// 处理文件上传
	for fieldName, files := range f.Files {
		for _, filePath := range files {
			if err := f.createFormFile(writer, fieldName, filePath); err != nil {
				return errors.Tag(err)
			}
		}
	}
	return nil
}

// validateFiles 校验待上传文件是否满足普通文件约束与大小上限。
func (f *Form) validateFiles() error {
	var totalSize int64
	for fieldName, files := range f.Files {
		for _, filePath := range files {
			info, err := os.Stat(filePath)
			if err != nil {
				return errors.Tag(err)
			}
			if !info.Mode().IsRegular() {
				return errors.Errorf("multipart 文件必须是普通文件: field=%s path=%s", fieldName, filePath)
			}
			if f.MaxSingleFileSize > 0 && info.Size() > f.MaxSingleFileSize {
				return errors.Errorf("multipart 单文件大小超限: field=%s path=%s size=%d limit=%d", fieldName, filePath, info.Size(), f.MaxSingleFileSize)
			}
			totalSize += info.Size()
			if f.MaxTotalFileSize > 0 && totalSize > f.MaxTotalFileSize {
				return errors.Errorf("multipart 文件总大小超限: size=%d limit=%d", totalSize, f.MaxTotalFileSize)
			}
		}
	}
	return nil
}
