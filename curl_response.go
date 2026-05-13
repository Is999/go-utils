package utils

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/Is999/go-utils/errors"
)

// ============================ Send 请求发送 ============================

// Send 发起 HTTP 请求。
// 封装完整请求生命周期：构建 Request、配置 Client/Transport、执行重试、处理响应。
//
// 参数说明：
//   - method：HTTP 方法（GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS）
//   - url：请求地址
//   - body：请求体
//
// 返回值：错误信息
func (c *Curl) Send(method, url string, body io.Reader) (err error) {
	return c.SendContext(context.Background(), method, url, body)
}

// SendContext 发起带 context 的 HTTP 请求。
// 当 ctx 被取消时，会立即中断请求以及重试等待。
func (c *Curl) SendContext(ctx context.Context, method, url string, body io.Reader) (err error) {
	ctx = ensureContext(ctx)

	// 记录请求开始时间
	t := time.Now()

	// 设置请求 ID（未设置时自动生成）
	if c.requestID == "" {
		c.SetRequestID()
	}

	// 输出调试日志
	if c.defLogOutput {
		c.Logger.Debug("HTTP START", "time", t.Format(time.RFC3339Nano))
	}

	// 定义请求和响应变量
	var (
		req  *http.Request
		resp *http.Response
	)

	// 请求完成后的资源清理
	defer func() {
		// 关闭 Response.Body
		defer func() {
			if resp == nil || resp.Body == nil || resp.Body == http.NoBody {
				return
			}

			if c.defLogOutput {
				c.Logger.Debug("Close Response Body")
			}

			if err := resp.Body.Close(); err != nil {
				c.Logger.Error("Body.Close()", "err", err.Error())
			}
		}()

		// 执行 afterDone 回调
		if c.afterDone != nil {
			if c.defLogOutput {
				c.Logger.Debug("done()")
			}
			c.afterDone(ctx, c.cli, req, resp)
		}
	}()

	// 如果请求体支持 Seek，先回到起点，确保同一个 Curl 实例重复发送时请求体完整。
	if body, err = rewindRequestBody(body); err != nil {
		return errors.Tag(err)
	}

	// 构建 Request
	req, err = http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return errors.Tag(err)
	}
	if err = setRequestGetBody(req, body); err != nil {
		return errors.Tag(err)
	}

	// 设置请求头。默认 Content-Type 在发送前补齐，避免 NewCurl 只作为模板使用时提前分配 Header。
	header := c.ensureDefaultContentType()
	if len(header) > 0 {
		if c.defLogOutput {
			c.Logger.Debug("set header")
		}
		req.Header = header.Clone()
	}

	// 设置 Cookie
	if c.cookies != nil && len(c.cookies) > 0 {
		if c.defLogOutput {
			c.Logger.Debug("AddCookie()")
		}
		for _, cookie := range c.cookies {
			req.AddCookie(cookie)
		}
	}

	// 设置 BasicAuth 认证
	if c.username != "" && c.password != "" {
		if c.defLogOutput {
			c.Logger.Debug("SetBasicAuth()")
		}
		req.SetBasicAuth(c.username, c.password)
	}

	// 执行 beforeRequest 回调
	if c.beforeRequest != nil {
		if c.defLogOutput {
			c.Logger.Debug("request()")
		}
		if err = c.beforeRequest(ctx, req); err != nil {
			return errors.Tag(err)
		}
	}

	// 记录请求日志
	if c.defLogOutput && c.Logger.Enabled(ctx, LevelInfo) {
		if c.dump {
			dump, err := dumpRequestSafe(req, c.dumpBodyLimit)
			if err != nil {
				return errors.Tag(err)
			}
			c.Logger.Info("httputil.DumpRequestOut()", "request", dump)
		} else {
			c.logRequest(method, url, req)
		}
	}

	// 初始化 Client（如未初始化）
	if c.cli == nil {
		if c.defLogOutput {
			c.Logger.Debug("Init Client")
		}
		c.cli = &http.Client{}
	}

	// 设置超时时间
	c.cli.Timeout = c.timeout
	if c.cli.Timeout == 0 {
		c.cli.Timeout = defaultTimeout
	}

	// 初始化 Transport
	if err = c.initTransport(); err != nil {
		return errors.Tag(err)
	}

	// 执行 beforeClient 回调
	if c.beforeClient != nil {
		if c.defLogOutput {
			c.Logger.Debug("client()")
		}
		if err = c.beforeClient(ctx, c.cli); err != nil {
			return errors.Tag(err)
		}
	}

	// 计算重试次数
	maxRetry := int(c.maxRetry)
	if maxRetry <= 0 {
		maxRetry = 1
	}
	if maxRetry > defaultMaxRetries {
		maxRetry = defaultMaxRetries
	}

	// 记录请求开始时间
	t1 := time.Now()
	if c.defLogOutput {
		c.Logger.Debug("client start", "time", t1.Format(time.RFC3339Nano))
	}

	// 执行请求（含重试逻辑）
	for i := 1; i <= maxRetry; i++ {
		if i > 1 && req.GetBody == nil && req.Body != nil && req.Body != http.NoBody {
			return errors.Tag(errors.New("client.Do() retry body is not rewindable"))
		}
		if i > 1 && req.GetBody != nil {
			req.Body, err = req.GetBody()
			if err != nil {
				return errors.Tag(err)
			}
		}
		resp, err = c.cli.Do(req)
		if err == nil {
			break
		}

		// 非最后一次重试，记录警告并等待
		if i < maxRetry {
			c.Logger.Warn("client.Do()", "maxRetry", maxRetry, "currentRetry", i, "err", err.Error())
			// 使用统一的指数退避和抖动策略，避免瞬时重试放大故障。
			if err = waitRetry(ctx, i); err != nil {
				return errors.Tag(err)
			}
		}
	}

	if c.defLogOutput {
		c.Logger.Debug("client end", "time spent", time.Since(t1).String())
	}

	if err != nil {
		return errors.Tag(errors.Errorf("client.Do() Retry %d times err: %v", maxRetry, err.Error()))
	}

	var respBody []byte

	// 记录响应日志
	if c.defLogOutput && c.Logger.Enabled(ctx, LevelInfo) {
		if respBody, err = c.logResponse(resp); err != nil {
			return errors.Tag(err)
		}
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK && !containsStatusCode(resp.StatusCode, c.statusCode) {
		return errors.Tag(errors.Errorf("response error StatusCode: statusCode=%d, Status=%s", resp.StatusCode, resp.Status))
	}

	// 执行 afterResponse 回调
	if c.afterResponse != nil {
		if c.defLogOutput {
			c.Logger.Debug("response()")
		}
		isDone, err := c.afterResponse(ctx, resp)
		if err != nil {
			return errors.Tag(err)
		}
		if isDone {
			return nil
		}
	}

	// 执行 afterBody 回调
	if c.afterBody != nil {
		if c.defLogOutput {
			c.Logger.Debug("resolve()")
		}
		if respBody == nil {
			var buf bytes.Buffer
			_, err = buf.ReadFrom(resp.Body)
			if err != nil {
				return errors.Tag(err)
			}
			respBody = buf.Bytes()
		}
		if err = c.afterBody(ctx, respBody); err != nil {
			return errors.Tag(err)
		}
	}

	if c.defLogOutput {
		c.Logger.Debug("HTTP END", "total time spent", time.Since(t).String())
	}

	return nil
}

// rewindRequestBody 在请求发送前重置可回放请求体。
//
// 参数说明：
//   - body：原始请求体。
//
// 返回值：重置后的请求体、错误信息。
func rewindRequestBody(body io.Reader) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	if seeker, ok := body.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, errors.Tag(err)
		}
	}
	if readSeeker, ok := body.(io.ReadSeeker); ok {
		if _, closes := body.(io.Closer); closes {
			return reusableReadSeeker{ReadSeeker: readSeeker}, nil
		}
	}
	return body, nil
}

// reusableReadSeeker 隐藏底层 Close 方法，让 net/http 在重试前不会关闭调用方持有的可回放请求体。
type reusableReadSeeker struct {
	io.ReadSeeker // 可重复定位的请求体。
}

// setRequestGetBody 为可安全重放的请求体补充 GetBody，保证日志预览和传输重试不会共享读取游标。
//
// 参数说明：
//   - req：HTTP 请求对象。
//   - body：请求体。
func setRequestGetBody(req *http.Request, body io.Reader) error {
	if req == nil || req.GetBody != nil || body == nil {
		return nil
	}

	switch v := body.(type) {
	case *bytes.Buffer:
		// bytes.Buffer 暴露的 Bytes() 可能被调用方继续修改，复制一份快照可保证后续重试内容稳定。
		setRequestGetBodyFromBytes(req, append([]byte(nil), v.Bytes()...))
		return nil
	case reusableReadSeeker:
		return setRequestGetBodyFromReadSeeker(req, v.ReadSeeker)
	case io.ReadSeeker:
		return setRequestGetBodyFromReadSeeker(req, v)
	default:
		// 流式 body 无法无损重放；保持 GetBody 为空，让重试逻辑在需要重试时明确返回不可重放错误。
		return nil
	}
}

// setRequestGetBodyFromReadSeeker 为内存型 ReadSeeker 创建独立快照。
//
// 降级策略：
//   - 只处理 bytes.Reader、strings.Reader 以及它们被 reusableReadSeeker 包装后的场景。
//   - 其它 ReadSeeker 可能是大文件或外部流，避免为了重试把未知体积内容读入内存。
func setRequestGetBodyFromReadSeeker(req *http.Request, readSeeker io.ReadSeeker) error {
	switch readSeeker.(type) {
	case *bytes.Reader, *strings.Reader:
	default:
		return nil
	}

	data, err := snapshotReadSeeker(readSeeker)
	if err != nil {
		return errors.Tag(err)
	}
	setRequestGetBodyFromBytes(req, data)
	return nil
}

// snapshotReadSeeker 从头读取 ReadSeeker 内容并恢复原始游标。
//
// 该函数只供已知内存型 reader 使用，业务意图是让 GetBody 每次返回独立 bytes.Reader，
// 避免日志预览、重试和实际发送共享同一个游标。
func snapshotReadSeeker(readSeeker io.ReadSeeker) ([]byte, error) {
	pos, err := readSeeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, errors.Tag(err)
	}
	if _, err = readSeeker.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Tag(err)
	}
	data, err := io.ReadAll(readSeeker)
	if err != nil {
		return nil, errors.Tag(err)
	}
	if _, err = readSeeker.Seek(pos, io.SeekStart); err != nil {
		return nil, errors.Tag(err)
	}
	return data, nil
}

// setRequestGetBodyFromBytes 基于不可变字节快照创建 GetBody。
func setRequestGetBodyFromBytes(req *http.Request, data []byte) {
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
}

// logRequest 记录请求日志（非 dump 模式）。
//
// 参数说明：
//   - method：HTTP 方法
//   - url：请求地址
//   - req：HTTP 请求
func (c *Curl) logRequest(method, url string, req *http.Request) {
	var b strings.Builder
	b.WriteString(method + ": " + url + "\n")

	if req != nil && req.Body != nil && req.Body != http.NoBody && c.logBodyLimit > 0 {
		b.WriteString("Request Body Preview:\n")
		reqBody, truncated, err := requestBodyPreview(req, c.logBodyLimit)
		if err != nil {
			c.Logger.Error("requestBodyPreview() error", "err", err.Error())
			return
		}
		b.WriteString(formatBodyPreview(reqBody, truncated))
	}
	c.Logger.Info("Request", "body", b.String())
}

// logResponse 记录响应日志。
//
// 参数说明：
//   - resp：HTTP 响应
//
// 返回值：错误信息
func (c *Curl) logResponse(resp *http.Response) ([]byte, error) {
	if resp == nil {
		c.Logger.Info("Response", "body", "<nil>")
		return nil, nil
	}

	isAllowedStatusCode := resp.StatusCode == http.StatusOK || containsStatusCode(resp.StatusCode, c.statusCode)
	if c.afterBody != nil && c.afterResponse == nil && isAllowedStatusCode && resp.Body != nil && resp.Body != http.NoBody {
		// 响应 body 还需要交给 afterBody 完整读取；日志阶段只截取预览并恢复流，避免大响应被 DrainBody 全量复制。
		if c.dump {
			dump, err := dumpResponseSafe(resp, c.dumpBodyLimit)
			if err != nil {
				return nil, errors.Tag(err)
			}
			c.Logger.Info("httputil.DumpResponse()", "response", dump)
			return nil, nil
		}

		var b strings.Builder
		b.WriteString("Response Status: ")
		b.WriteString(resp.Status)
		b.WriteByte('\n')
		if c.logBodyLimit > 0 {
			b.WriteString("Response Body Preview:\n")
			respBody, truncated, restored, err := readBodyPreviewAndRestore(resp.Body, c.logBodyLimit)
			if err != nil {
				return nil, errors.Tag(err)
			}
			resp.Body = restored
			b.WriteString(formatBodyPreview(respBody, truncated))
		}
		c.Logger.Info("Response", "body", b.String())
		return nil, nil
	}

	if c.dump {
		dump, err := dumpResponseSafe(resp, c.dumpBodyLimit)
		if err != nil {
			return nil, errors.Tag(err)
		}
		c.Logger.Info("httputil.DumpResponse()", "response", dump)
	} else {
		var b strings.Builder
		b.WriteString("Response Status: ")
		b.WriteString(resp.Status)
		b.WriteByte('\n')
		if resp != nil && resp.Body != nil && resp.Body != http.NoBody && c.logBodyLimit > 0 {
			b.WriteString("Response Body Preview:\n")
			var truncated bool
			var err error
			var respBody []byte
			respBody, truncated, resp.Body, err = readBodyPreviewAndRestore(resp.Body, c.logBodyLimit)
			if err != nil {
				return nil, errors.Tag(err)
			}
			b.WriteString(formatBodyPreview(respBody, truncated))
		}
		c.Logger.Info("Response", "body", b.String())
	}
	return nil, nil
}

// ============================ 响应处理方法 ============================

// BeforeRequest 请求发送前的回调。
//
// 参数说明：
//   - f：回调函数，返回 error 时中断请求
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) BeforeRequest(f func(request *http.Request) error) *Curl {
	if f == nil {
		c.beforeRequest = nil
		return c
	}
	c.beforeRequest = func(_ context.Context, request *http.Request) error {
		return f(request)
	}
	return c
}

// BeforeRequestContext 请求发送前的上下文回调。
func (c *Curl) BeforeRequestContext(f func(ctx context.Context, request *http.Request) error) *Curl {
	c.beforeRequest = f
	return c
}

// BeforeClient 请求发送前的 Client 回调。
//
// 参数说明：
//   - f：回调函数，返回 error 时中断请求
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) BeforeClient(f func(client *http.Client) error) *Curl {
	if f == nil {
		c.beforeClient = nil
		return c
	}
	c.beforeClient = func(_ context.Context, client *http.Client) error {
		return f(client)
	}
	return c
}

// BeforeClientContext 请求发送前的 Client 上下文回调。
func (c *Curl) BeforeClientContext(f func(ctx context.Context, client *http.Client) error) *Curl {
	c.beforeClient = f
	return c
}

// AfterResponse 请求发送后的回调。
//
// 参数说明：
//   - f：回调函数，isDone=true 时终止后续代码执行
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) AfterResponse(f func(response *http.Response) (isDone bool, err error)) *Curl {
	if f == nil {
		c.afterResponse = nil
		return c
	}
	c.afterResponse = func(_ context.Context, response *http.Response) (isDone bool, err error) {
		return f(response)
	}
	return c
}

// AfterResponseContext 请求发送后的上下文回调。
func (c *Curl) AfterResponseContext(f func(ctx context.Context, response *http.Response) (isDone bool, err error)) *Curl {
	c.afterResponse = f
	return c
}

// AfterBody 请求发送后对 Response.Body 的处理回调。
//
// 参数说明：
//   - f：回调函数，接收 body 字节数组
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) AfterBody(f func(body []byte) error) *Curl {
	if f == nil {
		c.afterBody = nil
		return c
	}
	c.afterBody = func(_ context.Context, body []byte) error {
		return f(body)
	}
	return c
}

// AfterBodyContext 请求发送后对 Response.Body 的上下文处理回调。
func (c *Curl) AfterBodyContext(f func(ctx context.Context, body []byte) error) *Curl {
	c.afterBody = f
	return c
}

// AfterDone 请求完成后的回调。
// 用于资源清理，如关闭连接等。
// 注意：client、request、response 有可能为 nil。
//
// 参数说明：
//   - f：回调函数
//
// 返回值：Curl 指针，支持链式调用
func (c *Curl) AfterDone(f func(client *http.Client, request *http.Request, response *http.Response)) *Curl {
	if f == nil {
		c.afterDone = nil
		return c
	}
	c.afterDone = func(_ context.Context, client *http.Client, request *http.Request, response *http.Response) {
		f(client, request, response)
	}
	return c
}

// AfterDoneContext 请求完成后的上下文回调。
func (c *Curl) AfterDoneContext(f func(ctx context.Context, client *http.Client, request *http.Request, response *http.Response)) *Curl {
	c.afterDone = f
	return c
}

// ============================ 内部工具函数 ============================

// DrainBody 读取 body 内容并恢复原始流。
//
// 参数说明：
//   - b：io.ReadCloser
//
// 返回值：body 内容、恢复的 ReadCloser、错误信息
func DrainBody(b io.ReadCloser) ([]byte, io.ReadCloser, error) {
	if b == nil || b == http.NoBody {
		return nil, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(b); err != nil {
		return nil, b, errors.Tag(err)
	}
	if err := b.Close(); err != nil {
		return nil, b, errors.Tag(err)
	}
	bodyBytes := buf.Bytes()
	return bodyBytes, io.NopCloser(bytes.NewReader(bodyBytes)), nil
}

// requestBodyPreview 获取请求体预览内容。
// 仅对支持 GetBody 的请求体读取预览，避免阻塞流式 body。
func requestBodyPreview(req *http.Request, limit int64) ([]byte, bool, error) {
	if req == nil || req.Body == nil || req.Body == http.NoBody || limit <= 0 {
		return nil, false, nil
	}
	if req.GetBody == nil {
		return []byte("[skipped: non-rewindable]"), false, nil
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, false, errors.Tag(err)
	}
	defer body.Close()
	return readBodyPreview(body, limit)
}

// containsStatusCode 检查状态码是否在列表中。
//
// 参数说明：
//   - code：状态码
//   - list：状态码列表
//
// 返回值：true 表示在列表中
func containsStatusCode(code int, list []int) bool {
	for _, v := range list {
		if v == code {
			return true
		}
	}
	return false
}

// dumpRequestSafe 安全地获取请求详情预览。
//
// 参数说明：
//   - req：HTTP 请求
//   - limit：预览长度上限
//
// 返回值：预览字符串、错误信息
func dumpRequestSafe(req *http.Request, limit int64) (string, error) {
	dump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		return "", errors.Tag(err)
	}

	if limit <= 0 || req.Body == nil || req.Body == http.NoBody {
		return string(dump), nil
	}

	if req.GetBody == nil {
		return string(dump) + "\nRequest Body: [skipped: non-rewindable]", nil
	}

	body, err := req.GetBody()
	if err != nil {
		return "", errors.Tag(err)
	}
	defer body.Close()

	preview, truncated, err := readBodyPreview(body, limit)
	if err != nil {
		return "", errors.Tag(err)
	}
	restored, err := req.GetBody()
	if err != nil {
		return "", errors.Tag(err)
	}
	req.Body = restored

	return formatDumpWithBody(string(dump), "Request Body", preview, truncated), nil
}

// dumpResponseSafe 安全地获取响应详情预览。
//
// 参数说明：
//   - resp：HTTP 响应
//   - limit：预览长度上限
//
// 返回值：预览字符串、错误信息
func dumpResponseSafe(resp *http.Response, limit int64) (string, error) {
	dump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		return "", errors.Tag(err)
	}

	if limit <= 0 || resp == nil || resp.Body == nil || resp.Body == http.NoBody {
		return string(dump), nil
	}

	preview, truncated, restored, err := readBodyPreviewAndRestore(resp.Body, limit)
	if err != nil {
		return "", errors.Tag(err)
	}
	resp.Body = restored

	return formatDumpWithBody(string(dump), "Response Body", preview, truncated), nil
}

// readBodyPreview 读取 body 预览内容。
//
// 参数说明：
//   - r：Reader
//   - limit：长度上限
//
// 返回值：预览内容、是否截断、错误信息
func readBodyPreview(r io.Reader, limit int64) ([]byte, bool, error) {
	lr := &io.LimitedReader{R: r, N: limit + 1}
	buf, err := io.ReadAll(lr)
	if err != nil {
		return nil, false, errors.Tag(err)
	}
	truncated := int64(len(buf)) > limit
	if truncated {
		return buf[:limit], true, nil
	}
	return buf, false, nil
}

// readBodyPreviewAndRestore 读取预览内容并恢复原始流。
//
// 参数说明：
//   - body：ReadCloser
//   - limit：长度上限
//
// 返回值：预览内容、是否截断、恢复的 ReadCloser、错误信息
func readBodyPreviewAndRestore(body io.ReadCloser, limit int64) ([]byte, bool, io.ReadCloser, error) {
	lr := &io.LimitedReader{R: body, N: limit + 1}
	buf, err := io.ReadAll(lr)
	if err != nil {
		return nil, false, body, errors.Tag(err)
	}
	truncated := int64(len(buf)) > limit
	preview := buf
	if truncated {
		preview = buf[:limit]
	}
	restored := readCloser{
		Reader: io.MultiReader(bytes.NewReader(buf), body),
		Closer: body,
	}
	return preview, truncated, restored, nil
}

// formatBodyPreview 将预览内容格式化为日志文本。
func formatBodyPreview(preview []byte, truncated bool) string {
	if len(preview) == 0 {
		return ""
	}
	if !truncated {
		return string(preview)
	}
	return string(preview) + "\n...[truncated]"
}

// readCloser 将恢复后的 Reader 和原始 Closer 组合成 io.ReadCloser。
type readCloser struct {
	io.Reader // 恢复后的读取流。
	io.Closer // 原始响应体关闭器。
}

// formatDumpWithBody 组装日志内容。
//
// 参数说明：
//   - header：头部内容
//   - label：标签
//   - body：body 内容
//   - truncated：是否截断
//
// 返回值：组装后的字符串
func formatDumpWithBody(header, label string, body []byte, truncated bool) string {
	if len(body) == 0 {
		return header
	}
	b := header + "\n" + label + ":\n" + string(body)
	if truncated {
		b += "\n...[truncated]"
	}
	return b
}
