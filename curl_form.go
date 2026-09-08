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

// Form 保存普通字段和待上传文件路径；使用 NewForm 初始化可写映射。
// multipart 生成期间 Params/Files 须保持只读，读至 EOF 后可复用；提前关闭不等待后台退出。
type Form struct {
	// Params 按字段名保存多值；同一字段内保留添加顺序
	Params url.Values

	// Files 按字段名保存本地路径列表，文件由后台生成流程按路径打开。
	Files url.Values

	// MaxSingleFileSize 单文件预检上限，单位：字节；小于等于 0 不检查大小
	MaxSingleFileSize int64

	// MaxTotalFileSize 文件总大小预检上限，单位：字节；不含普通字段和 multipart 开销，非正数不限制
	MaxTotalFileSize int64
}

// NewForm 初始化字段与文件映射，默认不限制文件大小。
func NewForm() *Form {
	return &Form{
		Params: make(url.Values),
		Files:  make(url.Values),
	}
}

// SetMaxSingleFileSize 设置 Stat 预检的单文件字节上限，非正数不限制。
func (f *Form) SetMaxSingleFileSize(limit int64) *Form {
	f.MaxSingleFileSize = limit
	return f
}

// SetMaxTotalFileSize 设置 Stat 预检的文件总字节上限，非正数不限制。
func (f *Form) SetMaxTotalFileSize(limit int64) *Form {
	f.MaxTotalFileSize = limit
	return f
}

// SetParam 用单个值替换同名字段的全部旧值。
func (f *Form) SetParam(key, value string) *Form {
	f.Params.Set(key, value)
	return f
}

// SetParams 覆盖给定字段的值，不清空其他字段。
func (f *Form) SetParams(params map[string]string) *Form {
	for key, value := range params {
		f.Params.Set(key, value)
	}
	return f
}

// AddParam 按传入顺序追加同名字段的值，保留旧值和重复值。
func (f *Form) AddParam(key string, values ...string) *Form {
	for _, value := range values {
		f.Params.Add(key, value)
	}
	return f
}

// AddParams 逐项追加字段值，保留同名字段内的顺序，不保存传入切片。
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

// SetFile 替换同名字段的全部文件路径，文件在生成正文时打开。
func (f *Form) SetFile(fieldName, filePath string) *Form {
	f.Files.Set(fieldName, filePath)
	return f
}

// SetFiles 覆盖给定字段的文件路径，不清空其他字段。
func (f *Form) SetFiles(files map[string]string) *Form {
	for name, path := range files {
		f.Files.Set(name, path)
	}
	return f
}

// AddFile 按传入顺序追加文件路径，同名字段可包含多个文件。
func (f *Form) AddFile(fieldName string, filePath ...string) *Form {
	for _, path := range filePath {
		f.Files.Add(fieldName, path)
	}
	return f
}

// AddFiles 逐项追加文件路径，保留同名字段内的顺序，不保存传入切片。
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

// Reader 返回表单正文及 Content-Type；无文件时为 URL 编码字符串，有文件时为 multipart 流。
// multipart 正文是 io.ReadCloser；调用方须读完或关闭，交给 Curl 发送后由请求生命周期关闭。
// 文件大小只在返回前通过 Stat 预检，之后的文件变更不会被该上限限制。
func (f *Form) Reader() (body io.Reader, contentType string, err error) {
	if len(f.Files) == 0 {
		return strings.NewReader(f.Params.Encode()), "application/x-www-form-urlencoded", nil
	}

	// 在启动写入 goroutine 前检查文件类型和当前大小，失败时直接返回错误。
	if err := f.validateFiles(); err != nil {
		return nil, "", err
	}

	// 使用 io.Pipe + multipart.Writer 流式拼装 body，避免大文件全量读入内存。
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	contentType = writer.FormDataContentType()
	go func() {
		writeErr := f.writeMultipart(writer)
		if writeErr == nil {
			// 只有正文写入成功才补终止边界，避免覆盖更早发生的文件或字段错误。
			writeErr = writer.Close()
		}
		// nil 对应正常 EOF；写入失败则由读取端收到原始错误链。
		_ = pipeWriter.CloseWithError(errors.Tag(writeErr))
	}()
	return pipeReader, contentType, nil
}

// writeFormFile 按当前路径打开并流式写入单个文件，本次写入结束后关闭文件句柄。
func writeFormFile(writer *multipart.Writer, fieldName, filePath string) error {
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

	if _, err = io.Copy(part, file); err != nil {
		return errors.Tag(err)
	}

	return nil
}

// writeMultipart 先写普通字段再写文件；字段名的遍历顺序不固定，同名值按切片顺序写入。
func (f *Form) writeMultipart(writer *multipart.Writer) error {
	for key, values := range f.Params {
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				return errors.Tag(err)
			}
		}
	}

	for fieldName, files := range f.Files {
		for _, filePath := range files {
			if err := writeFormFile(writer, fieldName, filePath); err != nil {
				return errors.Tag(err)
			}
		}
	}
	return nil
}

// validateFiles 基于当前 Stat 结果检查普通文件和大小，不创建文件内容快照。
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
