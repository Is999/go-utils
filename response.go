package utils

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Is999/go-utils/errors"
)

// Response 默认值与响应头常量。
const (
	defaultResponseStatus      = http.StatusOK                     // 默认 HTTP 状态码，调用方未显式设置时按成功响应处理
	defaultJSONContentType     = "application/json; charset=utf-8" // JSON 响应默认 Content-Type，包含 UTF-8 字符集
	defaultTextContentType     = "text/plain; charset=utf-8"       // 纯文本响应默认 Content-Type，避免每次 Text 调用重复规范化
	defaultHTMLContentType     = "text/html; charset=utf-8"        // HTML 响应默认 Content-Type，避免每次 HTML 调用重复规范化
	defaultXMLContentType      = "application/xml; charset=utf-8"  // XML 响应默认 Content-Type，避免每次 XML 调用重复规范化
	defaultSuccessMessage      = "SUCCESS"                         // JSON 成功响应默认业务消息
	headerContentType          = "Content-Type"                    // Content-Type 响应头名，用于声明 body 媒体类型
	headerContentLength        = "Content-Length"                  // Content-Length 响应头名，用于文件手动输出时声明长度
	headerContentDisposition   = "Content-Disposition"             // Content-Disposition 响应头名，用于下载文件名
	headerContentTypeOptions   = "X-Content-Type-Options"          // X-Content-Type-Options 响应头名，用于禁止浏览器类型嗅探
	headerLocation             = "Location"                        // Location 响应头名，用于重定向目标地址
	responseFallbackAttachment = "download"                        // 下载文件名清洗为空时的兜底名称
	responseJSONHex            = "0123456789abcdef"                // JSON 字符串控制字符转义使用的小写十六进制表，与 encoding/json 输出保持一致
)

// Response HTTP 响应构造器。
//
// 设计目标：
//   - 易用：保留 JSON(w).Success(...)、View(w).Text(...) 等链式调用。
//   - 高性能：文本写入使用 io.WriteString，文件响应使用 http.ServeContent。
//   - 稳定：统一管理状态码和响应头，避免重复 WriteHeader。
//   - 安全：文件响应禁止目录输出，下载文件名会清洗 CR/LF 等危险字符。
type Response struct {
	Body                            // JSON 响应体。
	statusCode  int                 // HTTP 状态码。
	statusSet   bool                // 是否显式设置过状态码。
	writer      http.ResponseWriter // HTTP 响应写入器。
	wroteHeader bool                // 是否已经写出响应头。
}

// ResponseOption 响应配置项。
type ResponseOption func(*Response)

// WithStatusCode 设置响应状态码。
//
// 仅接受标准 HTTP 状态码范围 [100, 599]，非法值会被忽略，避免 net/http panic。
func WithStatusCode(statusCode int) ResponseOption {
	return func(r *Response) {
		if r != nil {
			r.setStatusCode(statusCode)
		}
	}
}

// WithContentType 设置响应头 Content-Type。
//
// 对 text/*、application/json、application/xml 等文本类型自动追加 charset=utf-8；
// 对 image/*、application/octet-stream 等二进制类型保持原值，避免错误 charset。
func WithContentType(contentType string) ResponseOption {
	return func(r *Response) {
		r.ContentType(contentType)
	}
}

// WithHeader 自定义响应头。
//
// 回调为 nil 时不做任何操作，便于条件化配置。
func WithHeader(f func(header http.Header)) ResponseOption {
	return func(r *Response) {
		r.Header(f)
	}
}

// Body JSON 响应体。
type Body struct {
	Success bool   `json:"success"` // 响应状态：true 成功，false 失败
	Code    int    `json:"code"`    // 业务识别码
	Message string `json:"message"` // 响应信息
	Data    any    `json:"data"`    // 响应数据
}

// Success 成功响应 JSON 数据。
//
// code 为业务识别码，data 为响应数据，message 可选；未指定 message 时默认为 SUCCESS。
func (r *Response) Success(code int, data any, message ...string) {
	if r == nil {
		return
	}
	msg := defaultSuccessMessage
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	r.Body = Body{
		Success: true,
		Code:    code,
		Message: msg,
		Data:    data,
	}
	if !r.statusSet {
		r.statusCode = http.StatusOK
	}
	r.writeJSON("Success Encode", "data", data)
}

// Fail 失败响应 JSON 数据。
//
// code 为业务识别码，message 为响应信息，data 可选。
func (r *Response) Fail(code int, message string, data ...any) {
	if r == nil {
		return
	}
	r.Body = Body{
		Success: false,
		Code:    code,
		Message: message,
	}
	if len(data) > 0 {
		r.Data = data[0]
	}
	if !r.statusSet {
		r.statusCode = http.StatusBadRequest
	}
	r.writeJSON("Fail Encode", "data", data)
}

// Text 响应纯文本。
func (r *Response) Text(data string) {
	if r == nil {
		return
	}
	r.setContentType(defaultTextContentType)
	r.writeString(data)
}

// HTML 响应 HTML 文本。
func (r *Response) HTML(data string) {
	if r == nil {
		return
	}
	r.setContentType(defaultHTMLContentType)
	r.writeString(data)
}

// XML 响应 XML 数据。
//
// 使用 xml.Marshal 而不是 MarshalIndent，减少生产接口中的额外 CPU 和内存开销。
func (r *Response) XML(data any) {
	if r == nil {
		return
	}
	xmlData, err := xml.Marshal(data)
	if err != nil {
		r.serverError("XML xml.Marshal", err, "data", data)
		return
	}
	r.setContentType(defaultXMLContentType)

	body := make([]byte, 0, len(xml.Header)+len(xmlData))
	body = append(body, xml.Header...)
	body = append(body, xmlData...)
	r.writeBytes(body)
}

// Download 响应下载文件。
//
// filePath 为本地文件路径，rename 可指定下载文件名。文件名会被清洗后写入
// Content-Disposition，防止 CR/LF 注入和路径穿透式文件名。
func (r *Response) Download(filePath string, rename ...string) {
	r.download(nil, filePath, rename...)
}

// DownloadRequest 响应下载文件，并携带原始请求用于 Range/If-Modified-Since 等 HTTP 能力。
//
// 新代码建议在 Handler 中优先使用该方法；Download 会使用一个内部 GET 请求兜底。
func (r *Response) DownloadRequest(req *http.Request, filePath string, rename ...string) {
	r.download(req, filePath, rename...)
}

// download 统一处理下载文件逻辑。
func (r *Response) download(req *http.Request, filePath string, rename ...string) {
	file, info, ok := r.openFile(filePath, "Download")
	if !ok {
		return
	}
	defer file.Close()

	fileName := info.Name()
	if len(rename) > 0 && strings.TrimSpace(rename[0]) != "" {
		fileName = rename[0]
	}
	fileName = safeAttachmentName(fileName)

	r.Header(func(header http.Header) {
		header.Set(headerContentDisposition, mime.FormatMediaType("attachment", map[string]string{
			"filename": fileName,
		}))
		header.Set(headerContentTypeOptions, "nosniff")
	})
	r.serveFile(req, file, info)
}

// Show 响应显示文件内容，如图片、PDF、文本等。
func (r *Response) Show(filePath string) {
	r.ShowRequest(nil, filePath)
}

// ShowRequest 响应显示文件内容，并携带原始请求用于 Range/If-Modified-Since 等 HTTP 能力。
//
// 新代码建议在 Handler 中优先使用该方法；Show 会使用一个内部 GET 请求兜底。
func (r *Response) ShowRequest(req *http.Request, filePath string) {
	file, info, ok := r.openFile(filePath, "Show")
	if !ok {
		return
	}
	defer file.Close()

	r.Header(func(header http.Header) {
		header.Set(headerContentTypeOptions, "nosniff")
	})
	r.serveFile(req, file, info)
}

// Write 写入原始字节响应。
//
// 该方法只会写一次状态码；后续多次调用只继续写 body，避免重复 WriteHeader。
func (r *Response) Write(body []byte) {
	if r == nil {
		return
	}
	r.writeBytes(body)
}

// StatusCode 设置响应状态码，如 http.StatusOK。
func (r *Response) StatusCode(statusCode int) *Response {
	if r == nil {
		return nil
	}
	r.setStatusCode(statusCode)
	return r
}

// ContentType 设置响应头 Content-Type。
func (r *Response) ContentType(contentType string) *Response {
	if r == nil || r.writer == nil {
		return r
	}
	if ct := normalizeContentType(contentType); ct != "" {
		r.setContentType(ct)
	}
	return r
}

// Header 设置响应头。
func (r *Response) Header(f func(header http.Header)) *Response {
	if r == nil || r.writer == nil || f == nil {
		return r
	}
	f(r.writer.Header())
	return r
}

// Encode 对 JSON 响应体编码。
// 使用默认标准库 JSON 时走手写响应包壳快路径；配置第三方 JSON 后回退到自定义编码器，避免改变调用方语义。
func (r *Response) Encode() ([]byte, error) {
	if currentConfig().json.useStandard {
		return encodeResponseBody(r.Body)
	}
	return Marshal(r.Body)
}

// JSON 创建 JSON 响应构造器。
func JSON(w http.ResponseWriter, opts ...ResponseOption) *Response {
	resp := newResponse(w, http.StatusOK, opts...)
	if resp.writer != nil && resp.writer.Header().Get(headerContentType) == "" {
		resp.setContentType(defaultJSONContentType)
	}
	return resp
}

// View 创建文本/文件响应构造器。
func View(w http.ResponseWriter, opts ...ResponseOption) *Response {
	return newResponse(w, http.StatusOK, opts...)
}

// Redirect 重定向。
//
// url 为重定向地址。状态码仅接受 3xx，非 3xx 会回退为 302。
func Redirect(w http.ResponseWriter, url string, opts ...ResponseOption) {
	resp := newResponse(w, http.StatusFound, opts...)
	if resp.writer == nil {
		return
	}
	if resp.statusCode < http.StatusMultipleChoices || resp.statusCode >= http.StatusBadRequest {
		resp.statusCode = http.StatusFound
	}
	resp.writer.Header().Set(headerLocation, cleanHeaderValue(url))
	resp.writeHeader()
}

// newResponse 创建响应构造器并应用选项。
func newResponse(w http.ResponseWriter, statusCode int, opts ...ResponseOption) *Response {
	resp := &Response{
		writer:     w,
		statusCode: statusCode,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(resp)
		}
	}
	if !validHTTPStatus(resp.statusCode) {
		resp.statusCode = defaultResponseStatus
		resp.statusSet = false
	}
	return resp
}

// encodeResponseBody 编码统一 JSON 响应包壳。
// success/code/message 四个固定字段不走反射；data 仍交给 encoding/json，保持任意业务数据的标准库语义。
func encodeResponseBody(body Body) ([]byte, error) {
	// dataJSON 是业务数据的 JSON 片段，数据来源为 Body.Data；nil 会按标准库语义输出 null。
	dataJSON, err := json.Marshal(body.Data)
	if err != nil {
		return nil, errors.Tag(err)
	}

	// out 预估固定字段与业务数据容量，减少 append 扩容；message 可能包含转义字符，容量只做保守估计。
	out := make([]byte, 0, len(dataJSON)+len(body.Message)+64)
	out = append(out, `{"success":`...)
	out = strconv.AppendBool(out, body.Success)
	out = append(out, `,"code":`...)
	out = strconv.AppendInt(out, int64(body.Code), 10)
	out = append(out, `,"message":`...)
	out = appendJSONQuotedString(out, body.Message)
	out = append(out, `,"data":`...)
	out = append(out, dataJSON...)
	out = append(out, '}')
	return out, nil
}

// appendJSONQuotedString 按 encoding/json 的默认 escapeHTML 规则追加 JSON 字符串。
// 业务边界：Response 快路径只用于标准库配置，因此必须转义 <、>、&、U+2028、U+2029 与非法 UTF-8，确保输出与 json.Marshal 语义一致。
func appendJSONQuotedString(dst []byte, s string) []byte {
	dst = append(dst, '"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if b >= 0x20 && b != '\\' && b != '"' && b != '<' && b != '>' && b != '&' {
				i++
				continue
			}
			dst = append(dst, s[start:i]...)
			switch b {
			case '\\', '"':
				dst = append(dst, '\\', b)
			case '\b':
				dst = append(dst, '\\', 'b')
			case '\f':
				dst = append(dst, '\\', 'f')
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				// 控制字符和 HTML 敏感字符统一写成 \u00xx，保持标准库 json.Marshal 的默认转义策略。
				dst = append(dst, '\\', 'u', '0', '0', responseJSONHex[b>>4], responseJSONHex[b&0x0f])
			}
			i++
			start = i
			continue
		}

		// r 是当前 UTF-8 字符；非法编码按标准库语义降级为 \ufffd，避免写出非 JSON 文本。
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			dst = append(dst, s[start:i]...)
			dst = append(dst, `\ufffd`...)
			i++
			start = i
			continue
		}
		if r == '\u2028' || r == '\u2029' {
			dst = append(dst, s[start:i]...)
			dst = append(dst, '\\', 'u', '2', '0', '2', responseJSONHex[r&0x0f])
			i += size
			start = i
			continue
		}
		i += size
	}
	dst = append(dst, s[start:]...)
	dst = append(dst, '"')
	return dst
}

// setStatusCode 校验并记录 HTTP 状态码。
func (r *Response) setStatusCode(statusCode int) {
	if validHTTPStatus(statusCode) {
		r.statusCode = statusCode
		r.statusSet = true
	}
}

// validHTTPStatus 判断状态码是否在 net/http 支持范围内。
func validHTTPStatus(statusCode int) bool {
	return statusCode >= 100 && statusCode <= 599
}

// setContentType 写入已规范化的 Content-Type。
// 调用方负责传入业务需要的 charset；该方法不再重复 normalize，服务于 Text/HTML/XML/JSON 热路径。
func (r *Response) setContentType(contentType string) {
	if r == nil || r.writer == nil || contentType == "" {
		return
	}
	header := r.writer.Header()
	if header == nil {
		return
	}
	values := header[headerContentType] // 复用已有值的底层数组，减少重复设置时的分配。
	if len(values) == 0 {
		header[headerContentType] = []string{contentType}
		return
	}
	header[headerContentType] = append(values[:0], contentType)
}

// writeJSON 将当前 Body 编码为 JSON 并写出。
func (r *Response) writeJSON(desc string, args ...any) {
	body, err := r.Encode()
	if err != nil {
		r.serverError(desc, err, args...)
		return
	}
	r.writeBytes(body)
}

// writeHeader 写出响应头，确保状态码只写一次。
func (r *Response) writeHeader() bool {
	if r == nil || r.writer == nil {
		return false
	}
	if r.wroteHeader {
		return true
	}
	if !validHTTPStatus(r.statusCode) {
		r.statusCode = defaultResponseStatus
	}
	r.writer.WriteHeader(r.statusCode)
	r.wroteHeader = true
	return true
}

// writeBytes 写出字节响应体。
func (r *Response) writeBytes(body []byte) {
	if !r.writeHeader() {
		return
	}
	if len(body) == 0 {
		return
	}
	if _, err := r.writer.Write(body); err != nil {
		r.logError("Write writer.Write", err, "bytes", len(body))
	}
}

// writeString 写出字符串响应体，避免额外 []byte 分配。
func (r *Response) writeString(body string) {
	if !r.writeHeader() {
		return
	}
	if body == "" {
		return
	}
	if _, err := io.WriteString(r.writer, body); err != nil {
		r.logError("Write io.WriteString", err, "bytes", len(body))
	}
}

// openFile 打开文件并拒绝目录响应。
func (r *Response) openFile(filePath, desc string) (*os.File, os.FileInfo, bool) {
	if r == nil {
		return nil, nil, false
	}
	file, err := os.Open(filePath)
	if err != nil {
		r.fileError(desc+" os.Open", filePath, err)
		return nil, nil, false
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		r.fileError(desc+" file.Stat", filePath, err)
		return nil, nil, false
	}
	if info.IsDir() {
		_ = file.Close()
		r.fileError(desc+" directory", filePath, os.ErrNotExist)
		return nil, nil, false
	}
	return file, info, true
}

// serveFile 根据状态码选择 ServeContent 或手动复制路径。
func (r *Response) serveFile(req *http.Request, file *os.File, info os.FileInfo) {
	if r == nil || r.writer == nil || file == nil || info == nil {
		return
	}
	if req == nil {
		req = &http.Request{Method: http.MethodGet, Header: make(http.Header)}
	}

	// 若调用方显式设置了非 200 状态码，则尊重该状态码并走手动复制路径。
	// 常规文件响应使用 ServeContent，可自动处理 Content-Type、Content-Length、Last-Modified。
	if r.statusSet && r.statusCode != http.StatusOK {
		r.serveFileWithStatus(req, file, info)
		return
	}

	http.ServeContent(r.writer, req, info.Name(), info.ModTime(), file)
	r.wroteHeader = true
}

// serveFileWithStatus 使用显式状态码输出文件内容。
func (r *Response) serveFileWithStatus(req *http.Request, file *os.File, info os.FileInfo) {
	if r.writer.Header().Get(headerContentType) == "" {
		ctype, err := FileType(file)
		if err != nil {
			r.serverError("File FileType", err, "file", file.Name())
			return
		}
		r.writer.Header().Set(headerContentType, ctype)
	}
	r.writer.Header().Set(headerContentLength, strconv.FormatInt(info.Size(), 10))
	r.writeHeader()
	if req.Method == http.MethodHead {
		return
	}
	if _, err := io.Copy(r.writer, file); err != nil {
		r.logError("File io.Copy", err, "file", file.Name())
	}
}

// fileError 将文件访问错误转换为安全 HTTP 错误响应。
func (r *Response) fileError(desc, filePath string, err error) {
	statusCode := http.StatusInternalServerError
	switch {
	case os.IsNotExist(err):
		statusCode = http.StatusNotFound
	case os.IsPermission(err):
		statusCode = http.StatusForbidden
	}
	id := r.logError(desc, err, "file", filePath, "status", statusCode)
	r.writeHTTPError(statusCode, http.StatusText(statusCode)+", code-"+id)
}

// serverError 输出内部错误响应并记录日志。
func (r *Response) serverError(desc string, err error, args ...any) {
	id := r.logError(desc, err, args...)
	r.writeHTTPError(http.StatusInternalServerError, "Response error, code-"+id)
}

// writeHTTPError 写出 HTTP 错误响应，已写头时不再重复写入。
func (r *Response) writeHTTPError(statusCode int, message string) {
	if r == nil || r.writer == nil || r.wroteHeader {
		return
	}
	r.statusCode = statusCode
	r.statusSet = true
	http.Error(r.writer, message, statusCode)
	r.wroteHeader = true
}

// logError 记录错误并返回可暴露给调用方的追踪 ID。
func (r *Response) logError(desc string, err error, args ...any) string {
	id := UniqueID(16)
	if err == nil {
		return id
	}

	logArgs := make([]any, 0, len(args)+6)
	logArgs = append(logArgs, "code", id, "desc", desc, "err", err.Error())
	logArgs = append(logArgs, args...)
	if logger := Log(); logger != nil {
		logger.Error(err.Error(), logArgs...)
	}
	return id
}

// normalizeContentType 规范化 Content-Type，并为文本类型补充 UTF-8。
func normalizeContentType(contentType string) string {
	ct := strings.TrimSpace(contentType)
	if ct == "" {
		return ""
	}
	if strings.Contains(ct, ";") {
		return ct
	}

	lower := strings.ToLower(ct)
	switch {
	case strings.HasPrefix(lower, "text/"),
		lower == "application/json",
		lower == "application/xml",
		lower == "text/xml",
		lower == "application/javascript",
		lower == "application/x-www-form-urlencoded",
		strings.HasSuffix(lower, "+json"),
		strings.HasSuffix(lower, "+xml"):
		return ct + "; charset=utf-8"
	default:
		return ct
	}
}

// safeAttachmentName 清洗下载文件名，避免响应头注入和路径语义。
func safeAttachmentName(name string) string {
	name = filepath.Base(cleanHeaderValue(strings.TrimSpace(name)))
	name = strings.Map(func(r rune) rune {
		switch {
		case r < 32 || r == 127:
			return -1
		case r == '/' || r == '\\':
			return '_'
		default:
			return r
		}
	}, name)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return responseFallbackAttachment
	}
	return name
}

// cleanHeaderValue 移除响应头值中的换行符，避免 CRLF 注入。
func cleanHeaderValue(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\r", ""), "\n", "")
}
