package utils

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"net/http/httputil"
	"slices"
	"strings"
	"time"

	"github.com/Is999/go-utils/errors"
)

// Send 使用后台上下文发送指定 method、URL 和 body；生命周期与 SendContext 相同。
func (c *Curl) Send(method, url string, body io.Reader) error {
	return c.SendContext(context.Background(), method, url, body)
}

// SendContext 发起带 context 的 HTTP 请求。
// nil ctx 使用后台上下文；取消会传递给请求、回调和重试等待，不会强行中断用户回调。
// 普通流式 io.ReadCloser 交由请求生命周期关闭，发送前失败也会释放；
// 支持定位的 io.ReadSeekCloser 保留给调用方复用和关闭，但自动重试仍要求 Request.GetBody。
// 回调在当前 goroutine 串行执行，重试期间不重复执行 BeforeRequest 和 BeforeClient。
func (c *Curl) SendContext(ctx context.Context, method, url string, body io.Reader) (err error) {
	ctx = ensureContext(ctx)
	start := time.Now()

	if c.requestID == "" {
		c.SetRequestID()
	}

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
		// 尚未交给 Transport 的流式请求体由这里释放，避免 multipart 写端一直等待。
		if req.Body != nil {
			_ = req.Body.Close()
		}
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

// finishRequest 在成功或失败后结束响应生命周期。
func (c *Curl) finishRequest(ctx context.Context, req *http.Request, resp *http.Response) {
	// 回调先于关闭执行，仍可读取前面未消费的正文。
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
	// 不排空未读正文，HTTP/1.x 连接可能无法复用。
	if err := resp.Body.Close(); err != nil {
		// 关闭错误仅记录，不覆盖请求结果。
		if c.defLogOutput {
			c.Logger.Error("Body.Close()", "err", err.Error())
		}
	}
}

// prepareRequest 按请求配置、BeforeRequest、日志预览的顺序准备请求。
func (c *Curl) prepareRequest(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	req, err := buildHTTPRequest(ctx, method, rawURL, body)
	if err != nil {
		return nil, errors.Tag(err)
	}
	defer func() {
		// 回调或日志准备失败时，net/http 尚未接管 Body 的关闭责任。
		if err != nil && req.Body != nil {
			_ = req.Body.Close()
		}
	}()
	c.applyRequestOptions(req)
	if c.beforeRequest != nil {
		if c.defLogOutput {
			c.Logger.Debug("request()")
		}
		if err = c.beforeRequest(ctx, req); err != nil {
			return nil, errors.Tag(err)
		}
	}
	if err = c.logPreparedRequest(ctx, method, rawURL, req); err != nil {
		return nil, errors.Tag(err)
	}
	return req, nil
}

// buildHTTPRequest 重置可定位正文后创建 Request，普通流保留当前读取位置。
func buildHTTPRequest(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	body, err := rewindRequestBody(body)
	if err != nil {
		return nil, errors.Tag(err)
	}
	// 标准库为 bytes.Reader 和 strings.Reader 设置 GetBody；其他 Reader 不会自动获得重试能力。
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		if closer, ok := body.(io.Closer); ok {
			_ = closer.Close()
		}
		return nil, errors.Tag(err)
	}
	return req, nil
}

// applyRequestOptions 将请求头复制到本次 Request，再叠加 Cookie 和 BasicAuth。
func (c *Curl) applyRequestOptions(req *http.Request) {
	header := c.ensureDefaultContentType()
	if c.defLogOutput {
		c.Logger.Debug("set header")
	}
	req.Header = header.Clone()

	if len(c.cookies) > 0 {
		if c.defLogOutput {
			c.Logger.Debug("AddCookie()")
		}
		for _, cookie := range c.cookies {
			req.AddCookie(cookie)
		}
	}

	// 两者均空时保持默认请求头；只填写账号或密码也属于有效的认证配置。
	if c.username != "" || c.password != "" {
		if c.defLogOutput {
			c.Logger.Debug("SetBasicAuth()")
		}
		req.SetBasicAuth(c.username, c.password)
	}
}

// logPreparedRequest 仅在 Info 日志启用时预览请求；dump 失败会中止发送。
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

// prepareClient 为本次发送应用客户端配置，保留可复用的 Client 实例。
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
		return err
	}
	// 回调最后执行，允许调用方覆盖已应用的超时和 Transport。
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

// sendWithRetry 仅重试 Client.Do 返回的错误，不重试 HTTP 状态码或响应回调失败。
// 有正文时从 GetBody 重建读取流；是否允许重复发送由调用方按业务语义决定。
func (c *Curl) sendWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	maxRetry := max(1, min(int(c.maxRetry), defaultMaxRetries)) // 尝试次数包含首次请求，范围为 1 到 5。
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
			return nil, errors.Wrap(err, "client.Do() retry body is not rewindable")
		}
		// 每次复制请求和请求头，隔离异步 Transport 访问及 CookieJar 写入；Trailer 仍与正文共享。
		attempt := *req
		attempt.Header = req.Header.Clone()
		if i > 1 && req.GetBody != nil {
			attempt.Body, err = req.GetBody()
			if err != nil {
				return nil, errors.Tag(err)
			}
		}
		resp, err = c.cli.Do(&attempt)
		if err == nil {
			break
		}
		if i < maxRetry {
			if c.defLogOutput {
				c.Logger.Warn("client.Do()", "maxRetry", maxRetry, "currentRetry", i, "err", err.Error())
			}
			// 等待成功不能覆盖发送错误，下一轮可能因正文无法重放而直接返回。
			if waitErr := waitRetry(ctx, i); waitErr != nil {
				return nil, waitErr
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
// done=true 只跳过 afterBody；完成回调和响应体关闭仍由 SendContext 执行。
func (c *Curl) handleResponse(ctx context.Context, resp *http.Response) (done bool, err error) {
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
		// afterResponse 可以先消费部分内容，afterBody 接收当前流中剩余的数据。
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, errors.Tag(err)
		}
		if err = c.afterBody(ctx, respBody); err != nil {
			return false, errors.Tag(err)
		}
	}
	return false, nil
}

// rewindRequestBody 从 bytes.Buffer 未读部分建立快照，其他可定位正文回到起点。
func rewindRequestBody(body io.Reader) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	if buffer, ok := body.(*bytes.Buffer); ok {
		return bytes.NewReader(bytes.Clone(buffer.Bytes())), nil
	}
	if seeker, ok := body.(io.ReadSeeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, errors.Tag(err)
		}
		// 可定位正文留给调用方重复使用，发送时隐藏其 Close。
		if _, closes := body.(io.Closer); closes {
			return reusableReadSeeker{ReadSeeker: seeker}, nil
		}
	}
	return body, nil
}

// reusableReadSeeker 隐藏 Close，避免 net/http 关闭调用方需要重复发送的文件等可定位正文。
// 它不提供 GetBody，单次 Send 内自动重试需另外配置该能力。
type reusableReadSeeker struct {
	io.ReadSeeker // 可重复定位的请求体。
}

// logRequest 记录普通请求日志；预览失败只记错误，不阻止请求发送。
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

// logResponse 记录响应日志；预览成功后恢复 Body，失败则交由请求结束路径关闭。
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

// responseLogText 在预览成功后拼回已读字节，后续回调仍能读取完整正文。
func responseLogText(resp *http.Response, limit int64) (string, error) {
	var b strings.Builder
	b.WriteString("Response Status: ")
	b.WriteString(resp.Status)
	b.WriteByte('\n')
	// 流式响应可用 SetLogBodyLimit(0) 跳过预读，避免等待正文填满预览。
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

// BeforeRequest 在请求配置应用后、日志预览前执行；返回错误中止发送，nil 清除回调。
// 替换 Request.Body 时，原正文的关闭与新正文的 GetBody 由回调负责。
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

// BeforeRequestContext 与 BeforeRequest 时机相同，并接收本次 SendContext 的上下文。
func (c *Curl) BeforeRequestContext(f func(ctx context.Context, request *http.Request) error) *Curl {
	c.beforeRequest = f
	return c
}

// BeforeClient 在超时和 Transport 初始化后执行，可覆盖 Client 配置；nil 清除回调。
// 重试不重复执行此回调；返回错误时不进入网络请求。
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

// BeforeClientContext 与 BeforeClient 时机相同，并接收本次请求上下文。
func (c *Curl) BeforeClientContext(f func(ctx context.Context, client *http.Client) error) *Curl {
	c.beforeClient = f
	return c
}

// AfterResponse 在日志预览和状态校验通过后执行；nil 清除回调。
// 返回 true 或错误会跳过 AfterBody，响应体仍在 AfterDone 之后关闭。
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

// AfterResponseContext 与 AfterResponse 的顺序和关闭归属相同，并接收请求上下文。
func (c *Curl) AfterResponseContext(f func(ctx context.Context, response *http.Response) (isDone bool, err error)) *Curl {
	c.afterResponse = f
	return c
}

// AfterBody 接收 AfterResponse 处理后剩余的完整正文；nil 清除回调。
// 正文会一次读入内存，日志预览限额不限制这里的读取大小。
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

// AfterBodyContext 与 AfterBody 的读取规则相同，并接收请求上下文。
func (c *Curl) AfterBodyContext(f func(ctx context.Context, body []byte) error) *Curl {
	c.afterBody = f
	return c
}

// AfterDone 在成功或失败后执行，先于响应体关闭；nil 清除回调。
// 准备或发送失败时部分参数可能为 nil；响应正文也可能已被前面的回调消费。
// request 是准备阶段的原始请求；最终响应对应的请求可从非 nil 的 response.Request 获取。
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

// AfterDoneContext 与 AfterDone 的执行和关闭顺序相同，并接收请求上下文。
func (c *Curl) AfterDoneContext(f func(ctx context.Context, client *http.Client, request *http.Request, response *http.Response)) *Curl {
	c.afterDone = f
	return c
}

// DrainBody 全量读取并关闭原 body，成功时返回正文和共享该字节切片的内存读取流。
// 读取或关闭失败时返回原 body，其位置可能已改变，后续清理由调用方负责。
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

// requestBodyPreview 从 GetBody 创建预览流，不消费发送中的正文。
func requestBodyPreview(req *http.Request, limit int64) ([]byte, bool, error) {
	// 无法重放的流直接跳过，避免预览阻塞上传或提前消费正文。
	if req.GetBody == nil {
		return []byte("[skipped: non-rewindable]"), false, nil
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, false, errors.Tag(err)
	}
	defer body.Close()
	buf, truncated, err := readPreviewChunk(body, limit)
	if err != nil {
		return nil, false, errors.Tag(err)
	}
	return trimPreview(buf, truncated, limit), truncated, nil
}

// dumpRequestSafe 保留请求头并限长展示正文，不消费原请求体。
func dumpRequestSafe(req *http.Request, limit int64) (string, error) {
	// 标准库会按正数 ContentLength 缓冲临时正文，内存开销不受展示上限控制。
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

	preview, truncated, err := requestBodyPreview(req, limit)
	if err != nil {
		return "", errors.Tag(err)
	}

	return formatDumpWithBody(string(dump), "Request Body", preview, truncated), nil
}

// dumpResponseSafe 保留响应头并限长展示正文，预览成功后恢复 Body。
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

// readBodyPreviewAndRestore 将读出的字节拼回原流，Close 仍转发给原响应体。
func readBodyPreviewAndRestore(body io.ReadCloser, limit int64) ([]byte, bool, io.ReadCloser, error) {
	buf, truncated, err := readPreviewChunk(body, limit)
	if err != nil {
		// 读取失败不恢复正文，已消费部分内容的原流交给上层关闭。
		return nil, false, body, errors.Tag(err)
	}
	// 探测截断的额外字节也必须拼回，不能只恢复展示片段。
	restored := readCloser{
		Reader: io.MultiReader(bytes.NewReader(buf), body),
		Closer: body,
	}
	return trimPreview(buf, truncated, limit), truncated, restored, nil
}

// readPreviewChunk 最多读取 limit+1 字节，用额外 1 字节判断是否截断。
func readPreviewChunk(r io.Reader, limit int64) ([]byte, bool, error) {
	if limit <= 0 {
		return nil, false, nil
	}
	// 最大上限不再增加探测字节，避免整数溢出后被当作空流。
	lr := &io.LimitedReader{R: r, N: min(limit, math.MaxInt64-1) + 1}
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
