package utils

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httputil"
	"slices"
	"strings"
	"time"

	"github.com/Is999/go-utils/errors"
)

// ============================ Send 请求发送 ============================

// Send 发起 HTTP 请求。
// 封装完整请求生命周期：构建 Request、配置 Client/Transport、执行重试、处理响应。
func (c *Curl) Send(method, url string, body io.Reader) (err error) {
	return c.SendContext(context.Background(), method, url, body)
}

// SendContext 发起带 context 的 HTTP 请求。
// 当 ctx 被取消时，会立即中断请求以及重试等待。
func (c *Curl) SendContext(ctx context.Context, method, url string, body io.Reader) (err error) {
	ctx = ensureContext(ctx)
	start := time.Now()

	// 设置请求 ID（未设置时自动生成）
	if c.requestID == "" {
		c.SetRequestID()
	}

	// 输出调试日志
	if c.defLogOutput {
		c.Logger.Debug("HTTP START", "time", start.Format(time.RFC3339Nano))
	}

	var (
		req  *http.Request
		resp *http.Response
	)
	defer func() {
		c.finishRequest(ctx, req, resp)
	}()

	req, err = c.prepareRequest(ctx, method, url, body)
	if err != nil {
		return err
	}
	if err := c.prepareClient(ctx); err != nil {
		return err
	}
	resp, err = c.sendWithRetry(ctx, req)
	if err != nil {
		return err
	}

	done, err := c.handleResponse(ctx, resp)
	if err != nil || done {
		return err
	}

	if c.defLogOutput {
		c.Logger.Debug("HTTP END", "total time spent", time.Since(start).String())
	}
	return nil
}

// finishRequest 执行完成回调并关闭响应体。
// afterDone 先于 Body.Close 执行，保留调用方在完成回调中读取响应对象的旧边界。
func (c *Curl) finishRequest(ctx context.Context, req *http.Request, resp *http.Response) {
	if c.afterDone != nil {
		if c.defLogOutput {
			c.Logger.Debug("done()")
		}
		c.afterDone(ctx, c.cli, req, resp)
	}
	if resp == nil || resp.Body == nil || resp.Body == http.NoBody {
		return
	}
	if c.defLogOutput {
		c.Logger.Debug("Close Response Body")
	}
	if err := resp.Body.Close(); err != nil {
		if c.defLogOutput {
			c.Logger.Error("Body.Close()", "err", err.Error())
		}
	}
}

// prepareRequest 构建请求并执行 Header、Cookie、认证、日志和发送前回调处理。
func (c *Curl) prepareRequest(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	req, err := buildHTTPRequest(ctx, method, rawURL, body)
	if err != nil {
		return nil, errors.Tag(err)
	}
	c.applyRequestOptions(req)
	if err = c.runBeforeRequest(ctx, req); err != nil {
		return nil, errors.Tag(err)
	}
	if err = c.logPreparedRequest(ctx, method, rawURL, req); err != nil {
		return nil, errors.Tag(err)
	}
	return req, nil
}

// buildHTTPRequest 构建可重放请求体的 HTTP Request。
func buildHTTPRequest(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	body, err := rewindRequestBody(body)
	if err != nil {
		return nil, errors.Tag(err)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, errors.Tag(err)
	}
	if err = setRequestGetBody(req, body); err != nil {
		return nil, errors.Tag(err)
	}
	return req, nil
}

// applyRequestOptions 写入 Header、Cookie 和 BasicAuth。
func (c *Curl) applyRequestOptions(req *http.Request) {
	header := c.ensureDefaultContentType()
	if len(header) > 0 {
		if c.defLogOutput {
			c.Logger.Debug("set header")
		}
		req.Header = header.Clone()
	}

	if len(c.cookies) > 0 {
		if c.defLogOutput {
			c.Logger.Debug("AddCookie()")
		}
		for _, cookie := range c.cookies {
			req.AddCookie(cookie)
		}
	}

	if c.username != "" && c.password != "" {
		if c.defLogOutput {
			c.Logger.Debug("SetBasicAuth()")
		}
		req.SetBasicAuth(c.username, c.password)
	}
}

// runBeforeRequest 执行发送前请求回调。
func (c *Curl) runBeforeRequest(ctx context.Context, req *http.Request) error {
	if c.beforeRequest != nil {
		if c.defLogOutput {
			c.Logger.Debug("request()")
		}
		if err := c.beforeRequest(ctx, req); err != nil {
			return errors.Tag(err)
		}
	}
	return nil
}

// logPreparedRequest 输出请求日志或 dump 内容。
func (c *Curl) logPreparedRequest(ctx context.Context, method, rawURL string, req *http.Request) error {
	if c.defLogOutput && c.Logger.Enabled(ctx, LevelInfo) {
		if c.dump {
			dump, err := dumpRequestSafe(req, c.dumpBodyLimit)
			if err != nil {
				return errors.Tag(err)
			}
			c.Logger.Info("httputil.DumpRequestOut()", "request", dump)
		} else {
			c.logRequest(method, rawURL, req)
		}
	}
	return nil
}

// prepareClient 初始化 HTTP Client、超时、Transport 和 client 回调。
func (c *Curl) prepareClient(ctx context.Context) error {
	if c.cli == nil {
		if c.defLogOutput {
			c.Logger.Debug("Init Client")
		}
		c.cli = &http.Client{}
	}
	c.cli.Timeout = c.timeout
	if c.cli.Timeout == 0 {
		c.cli.Timeout = defaultTimeout
	}
	if err := c.initTransport(); err != nil {
		return errors.Tag(err)
	}
	if c.beforeClient != nil {
		if c.defLogOutput {
			c.Logger.Debug("client()")
		}
		if err := c.beforeClient(ctx, c.cli); err != nil {
			return errors.Tag(err)
		}
	}
	return nil
}

// retryCount 返回本次请求最大尝试次数，包含首次请求。
func (c *Curl) retryCount() int {
	maxRetry := int(c.maxRetry)
	if maxRetry <= 0 {
		return 1
	}
	if maxRetry > defaultMaxRetries {
		return defaultMaxRetries
	}
	return maxRetry
}

// sendWithRetry 执行 HTTP 请求，并在请求体可重放时按配置重试。
func (c *Curl) sendWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	maxRetry := c.retryCount()
	start := time.Now()
	if c.defLogOutput {
		c.Logger.Debug("client start", "time", start.Format(time.RFC3339Nano))
	}

	var (
		resp *http.Response
		err  error
	)
	for i := 1; i <= maxRetry; i++ {
		if i > 1 && req.GetBody == nil && req.Body != nil && req.Body != http.NoBody {
			return nil, errors.New("client.Do() retry body is not rewindable")
		}
		if i > 1 && req.GetBody != nil {
			req.Body, err = req.GetBody()
			if err != nil {
				return nil, errors.Tag(err)
			}
		}
		resp, err = c.cli.Do(req)
		if err == nil {
			break
		}
		if i < maxRetry {
			if c.defLogOutput {
				c.Logger.Warn("client.Do()", "maxRetry", maxRetry, "currentRetry", i, "err", err.Error())
			}
			if err = waitRetry(ctx, i); err != nil {
				return nil, errors.Tag(err)
			}
		}
	}

	if c.defLogOutput {
		c.Logger.Debug("client end", "time spent", time.Since(start).String())
	}
	if err != nil {
		return nil, errors.Wrapf(err, "client.Do() Retry %d times err", maxRetry)
	}
	return resp, nil
}

// handleResponse 记录响应、校验状态码，并执行响应回调。
// 返回 done=true 表示 afterResponse 已接管后续处理，调用方应直接结束。
func (c *Curl) handleResponse(ctx context.Context, resp *http.Response) (done bool, err error) {
	var respBody []byte
	if c.defLogOutput && c.Logger.Enabled(ctx, LevelInfo) {
		if err = c.logResponse(resp); err != nil {
			return false, errors.Tag(err)
		}
	}
	if resp.StatusCode != http.StatusOK && !slices.Contains(c.statusCode, resp.StatusCode) {
		return false, errors.Errorf("response error StatusCode: statusCode=%d, Status=%s", resp.StatusCode, resp.Status)
	}
	if c.afterResponse != nil {
		if c.defLogOutput {
			c.Logger.Debug("response()")
		}
		done, err = c.afterResponse(ctx, resp)
		if err != nil {
			return false, errors.Tag(err)
		}
		if done {
			return true, nil
		}
	}
	if c.afterBody != nil {
		if c.defLogOutput {
			c.Logger.Debug("resolve()")
		}
		if respBody == nil {
			var buf bytes.Buffer
			if _, err = buf.ReadFrom(resp.Body); err != nil {
				return false, errors.Tag(err)
			}
			respBody = buf.Bytes()
		}
		if err = c.afterBody(ctx, respBody); err != nil {
			return false, errors.Tag(err)
		}
	}
	return false, nil
}

// rewindRequestBody 在请求发送前重置可回放请求体。
func rewindRequestBody(body io.Reader) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	if buffer, ok := body.(*bytes.Buffer); ok {
		return bytes.NewReader(append([]byte(nil), buffer.Bytes()...)), nil
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
func setRequestGetBody(req *http.Request, body io.Reader) error {
	if req.GetBody != nil || body == nil {
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
func (c *Curl) logRequest(method, url string, req *http.Request) {
	var b strings.Builder
	b.Grow(len(method) + len(url) + 3)
	b.WriteString(method)
	b.WriteString(": ")
	b.WriteString(url)
	b.WriteByte('\n')

	if req.Body != nil && req.Body != http.NoBody && c.logBodyLimit > 0 {
		b.WriteString("Request Body Preview:\n")
		reqBody, truncated, err := requestBodyPreview(req, c.logBodyLimit)
		if err != nil {
			c.Logger.Error("requestBodyPreview() error", "err", err.Error())
			return
		}
		appendBodyPreview(&b, reqBody, truncated)
	}
	c.Logger.Info("Request", "body", b.String())
}

// logResponse 记录响应日志，并在读取预览后恢复 Body。
func (c *Curl) logResponse(resp *http.Response) error {
	if c.dump {
		dump, err := dumpResponseSafe(resp, c.dumpBodyLimit)
		if err != nil {
			return errors.Tag(err)
		}
		c.Logger.Info("httputil.DumpResponse()", "response", dump)
		return nil
	}

	body, err := responseLogText(resp, c.logBodyLimit)
	if err != nil {
		return errors.Tag(err)
	}
	c.Logger.Info("Response", "body", body)
	return nil
}

// responseLogText 构造响应日志文本，并恢复被预览读取过的 Body。
func responseLogText(resp *http.Response, limit int64) (string, error) {
	var b strings.Builder
	b.WriteString("Response Status: ")
	b.WriteString(resp.Status)
	b.WriteByte('\n')
	if resp.Body == nil || resp.Body == http.NoBody || limit <= 0 {
		return b.String(), nil
	}

	preview, truncated, restored, err := readBodyPreviewAndRestore(resp.Body, limit)
	if err != nil {
		return "", errors.Tag(err)
	}
	resp.Body = restored

	b.WriteString("Response Body Preview:\n")
	appendBodyPreview(&b, preview, truncated)
	return b.String(), nil
}

// ============================ 响应处理方法 ============================

// BeforeRequest 请求发送前的回调。
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

// dumpRequestSafe 安全地获取请求详情预览。
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

	return formatDumpWithBody(string(dump), "Request Body", preview, truncated), nil
}

// dumpResponseSafe 安全地获取响应详情预览。
func dumpResponseSafe(resp *http.Response, limit int64) (string, error) {
	dump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		return "", errors.Tag(err)
	}

	if limit <= 0 || resp.Body == nil || resp.Body == http.NoBody {
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
func readBodyPreview(r io.Reader, limit int64) ([]byte, bool, error) {
	buf, truncated, err := readPreviewChunk(r, limit)
	if err != nil {
		return nil, false, errors.Tag(err)
	}
	return trimPreview(buf, truncated, limit), truncated, nil
}

// readBodyPreviewAndRestore 读取预览内容并恢复原始流。
func readBodyPreviewAndRestore(body io.ReadCloser, limit int64) ([]byte, bool, io.ReadCloser, error) {
	buf, truncated, err := readPreviewChunk(body, limit)
	if err != nil {
		return nil, false, body, errors.Tag(err)
	}
	restored := readCloser{
		Reader: io.MultiReader(bytes.NewReader(buf), body),
		Closer: body,
	}
	return trimPreview(buf, truncated, limit), truncated, restored, nil
}

// readPreviewChunk 读取 limit+1 字节，用额外 1 字节判断是否截断。
func readPreviewChunk(r io.Reader, limit int64) ([]byte, bool, error) {
	if limit <= 0 {
		return nil, false, nil
	}
	lr := &io.LimitedReader{R: r, N: limit + 1}
	buf, err := io.ReadAll(lr)
	if err != nil {
		return nil, false, errors.Tag(err)
	}
	return buf, int64(len(buf)) > limit, nil
}

// trimPreview 返回对外展示的预览片段，截断时去掉用于探测的额外字节。
func trimPreview(buf []byte, truncated bool, limit int64) []byte {
	if truncated {
		return buf[:limit]
	}
	return buf
}

// appendBodyPreview 写入日志 body 预览文本。
func appendBodyPreview(b *strings.Builder, preview []byte, truncated bool) {
	if len(preview) == 0 {
		return
	}
	b.Write(preview)
	if truncated {
		b.WriteString("\n...[truncated]")
	}
}

// readCloser 将恢复后的 Reader 和原始 Closer 组合成 io.ReadCloser。
type readCloser struct {
	io.Reader // 恢复后的读取流。
	io.Closer // 原始响应体关闭器。
}

// formatDumpWithBody 组装日志内容。
func formatDumpWithBody(header, label string, body []byte, truncated bool) string {
	if len(body) == 0 {
		return header
	}
	var b strings.Builder
	b.Grow(len(header) + len(label) + len(body) + 32)
	b.WriteString(header)
	b.WriteByte('\n')
	b.WriteString(label)
	b.WriteString(":\n")
	appendBodyPreview(&b, body, truncated)
	return b.String()
}
