package utils

import (
	"encoding/xml"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Is999/go-utils/errors"
)

// Response 默认值与响应头常量。
const (
	defaultResponseStatus      = http.StatusOK                     // 默认 HTTP 状态码，调用方未显式设置时按成功响应处理
	defaultJSONContentType     = "application/json; charset=utf-8" // JSON 响应默认 Content-Type，包含 UTF-8 字符集
	defaultTextContentType     = "text/plain; charset=utf-8"       // 纯文本响应的 UTF-8 媒体类型
	defaultHTMLContentType     = "text/html; charset=utf-8"        // HTML 响应的 UTF-8 媒体类型
	defaultXMLContentType      = "application/xml; charset=utf-8"  // XML 响应的 UTF-8 媒体类型
	defaultSuccessMessage      = "SUCCESS"                         // JSON 成功响应默认业务消息
	headerContentType          = "Content-Type"                    // Content-Type 响应头名，用于声明 body 媒体类型
	headerContentLength        = "Content-Length"                  // Content-Length 响应头名，用于文件手动输出时声明长度
	headerContentDisposition   = "Content-Disposition"             // Content-Disposition 响应头名，用于下载文件名
	headerContentTypeOptions   = "X-Content-Type-Options"          // X-Content-Type-Options 响应头名，用于禁止浏览器类型嗅探
	headerLocation             = "Location"                        // Location 响应头名，用于重定向目标地址
	responseFallbackAttachment = "download"                        // 下载文件名清洗为空时的兜底名称
)

// Response 为一次 HTTP 响应保存正文和提交状态，不能并发写入。
// 状态码与响应头应在正文写出前设置；它只跟踪自身写入，不感知外部对 ResponseWriter 的操作。
type Response struct {
	Body                            // JSON 响应体。
	statusCode  int                 // HTTP 状态码。
	statusSet   bool                // 是否显式设置过状态码。
	writer      http.ResponseWriter // HTTP 响应写入器。
	wroteHeader bool                // 当前构造器是否已提交最终响应头，包括 101 协议切换。
}

// ResponseOption 响应配置项。
type ResponseOption func(*Response)

// WithStatusCode 设置响应状态码。
//
// 仅接受 [100, 599]，其他值保留原配置；应在正文写出前设置。
func WithStatusCode(statusCode int) ResponseOption {
	return func(r *Response) {
		r.StatusCode(statusCode)
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
// 构造期间立即执行回调，不保存回调；nil 不作修改。
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
	Data    any    `json:"data"`    // 交给配置的 JSON 编码器，默认将 nil 编码为 null
}

// Success 成功响应 JSON 数据。
//
// 默认 HTTP 200；显式状态码优先。message 只使用首项，缺省或空串为 SUCCESS。
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
// 默认 HTTP 400；显式状态码优先。data 只使用首项，缺省编码为 null。
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

// Text 写出纯文本，并将 Content-Type 设为 UTF-8 文本类型。
func (r *Response) Text(data string) {
	if r == nil {
		return
	}
	r.setContentType(defaultTextContentType)
	r.writeString(data)
}

// HTML 原样写出 HTML，并将 Content-Type 设为 UTF-8 HTML 类型。
func (r *Response) HTML(data string) {
	if r == nil {
		return
	}
	r.setContentType(defaultHTMLContentType)
	r.writeString(data)
}

// XML 响应 XML 数据。
//
// 使用紧凑 XML 编码并添加 XML 声明；编码成功后才提交正文。
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
	r.Write(body)
}

// Download 响应下载文件。
//
// filePath 按原路径打开；rename 首项仅用于 Content-Disposition，不改变实际文件路径。
// 下载名会清理目录部分和控制字符；此入口使用内部 GET 请求。
func (r *Response) Download(filePath string, rename ...string) {
	r.DownloadRequest(nil, filePath, rename...)
}

// DownloadRequest 响应下载文件，并携带原始请求用于 Range/If-Modified-Since 等 HTTP 能力。
//
// req 为 nil 时使用内部 GET；尚未提交且未指定非 200 状态码时由 ServeContent 处理条件请求和 Range。
func (r *Response) DownloadRequest(req *http.Request, filePath string, rename ...string) {
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

// Show 使用内部 GET 请求输出文件；需要 HEAD、Range 或缓存校验时使用 ShowRequest。
func (r *Response) Show(filePath string) {
	r.ShowRequest(nil, filePath)
}

// ShowRequest 响应显示文件内容，并携带原始请求用于 Range/If-Modified-Since 等 HTTP 能力。
//
// req 为 nil 时使用内部 GET；尚未提交且未指定非 200 状态码时由 ServeContent 处理条件请求和 Range。
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
// 最终状态码只提交一次；1xx（除 101）配合空 body 可先发送临时响应。
// 写入失败只记日志，不能撤回已提交的内容。
func (r *Response) Write(body []byte) {
	if !r.writeHeader(len(body) > 0) {
		return
	}
	if len(body) == 0 {
		return
	}
	if _, err := r.writer.Write(body); err != nil {
		r.logError("Write writer.Write", err, "bytes", len(body))
	}
}

// StatusCode 设置待提交的状态码；无效值被忽略，已提交的状态码不会因此改变。
func (r *Response) StatusCode(statusCode int) *Response {
	if r == nil {
		return nil
	}
	if validHTTPStatus(statusCode) {
		r.statusCode = statusCode
		r.statusSet = true
	}
	return r
}

// ContentType 设置媒体类型；未带参数的文本类型自动补 UTF-8，已有参数原样保留。
func (r *Response) ContentType(contentType string) *Response {
	if r == nil || r.writer == nil {
		return r
	}
	if ct := normalizeContentType(contentType); ct != "" {
		r.setContentType(ct)
	}
	return r
}

// Header 立即调用 f 修改底层响应头；nil 不作修改，应在正文写出前调用。
func (r *Response) Header(f func(header http.Header)) *Response {
	if r == nil || r.writer == nil || f == nil {
		return r
	}
	f(r.writer.Header())
	return r
}

// Encode 使用当前 JSON 配置编码 Body，不写入 ResponseWriter，也不缓存编码结果。
// 默认编码与 Marshal 一致；保留标准库错误的堆栈包装，自定义编码器的错误原样返回。
func (r *Response) Encode() ([]byte, error) {
	cfg := configValue.Load() // 同一次编码使用一个配置快照。
	body, err := cfg.json.encode(r.Body)
	if cfg.json.useStandard {
		err = errors.Tag(err)
	}
	return body, err
}

// JSON 创建响应构造器，仅在未设置 Content-Type 时补 application/json; charset=utf-8。
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
	resp.writeHeader(false)
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

// validHTTPStatus 沿用本包接受的 [100, 599] 状态码范围。
func validHTTPStatus(statusCode int) bool {
	return statusCode >= 100 && statusCode <= 599
}

// setContentType 原样写入媒体类型，调用方已确定 charset，无需再次规范化。
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

// writeJSON 在编码成功后提交正文；编码失败交给 serverError，尚未提交响应时写出 500。
func (r *Response) writeJSON(desc string, args ...any) {
	body, err := r.Encode()
	if err != nil {
		r.serverError(desc, err, args...)
		return
	}
	r.Write(body)
}

// writeHeader 只提交一次最终状态；临时响应后写正文时按 net/http 规则补 200。
// hasBody 表示随后进入正文写入路径；返回 false 表示没有可用的 writer。
func (r *Response) writeHeader(hasBody bool) bool {
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
	// 101 已切换协议，其余 1xx 仍允许后续提交最终状态。
	if r.statusCode < http.StatusOK && r.statusCode != http.StatusSwitchingProtocols {
		if !hasBody {
			return true
		}
		r.statusCode = http.StatusOK
		r.writer.WriteHeader(r.statusCode)
	}
	r.wroteHeader = true
	return true
}

// writeString 使用 io.WriteString 输出正文；写入失败只记日志，不追加错误响应。
func (r *Response) writeString(body string) {
	if !r.writeHeader(body != "") {
		return
	}
	if body == "" {
		return
	}
	if _, err := io.WriteString(r.writer, body); err != nil {
		r.logError("Write io.WriteString", err, "bytes", len(body))
	}
}

// openFile 打开文件并拒绝目录响应；成功后由 DownloadRequest 或 ShowRequest 关闭句柄。
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

// serveFile 默认交给 ServeContent 处理 Range 和缓存条件；已有最终状态时直接输出文件。
func (r *Response) serveFile(req *http.Request, file *os.File, info os.FileInfo) {
	if r == nil || r.writer == nil || file == nil || info == nil {
		return
	}
	if req == nil {
		req = &http.Request{Method: http.MethodGet, Header: make(http.Header)}
	}

	// 已提交的状态不可再次交给 ServeContent 改写；显式非 200 状态也走手动复制路径。
	if r.wroteHeader || r.statusSet && r.statusCode != http.StatusOK {
		r.serveFileWithStatus(req, file, info)
		return
	}

	http.ServeContent(r.writer, req, info.Name(), info.ModTime(), file)
	r.wroteHeader = true
}

// serveFileWithStatus 按当前状态输出完整文件，不处理 Range 或缓存条件。
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
	r.writeHeader(req.Method != http.MethodHead)
	// HEAD 保留完整文件长度，只提交响应头。
	if req.Method == http.MethodHead {
		return
	}
	if _, err := io.Copy(r.writer, file); err != nil {
		// 正文可能已部分写出，不能再追加错误响应。
		r.logError("File io.Copy", err, "file", file.Name())
	}
}

// fileError 将不存在、权限不足和其他文件错误分别映射为 404、403、500，只在日志中记录路径和原因。
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

// writeHTTPError 仅在尚未提交时替换为错误响应，避免将错误文本追加到部分成功正文。
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

// normalizeContentType 仅为不含参数的文本媒体类型补 UTF-8；含分号的值保留调用方设置。
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

// safeAttachmentName 生成展示用下载名，移除控制字符并处理路径分隔符；空结果使用 download。
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
