package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	apperrors "github.com/Is999/go-utils/errors"
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
			c.afterDone(c.cli, req, resp)
		}
	}()

	// 如果请求体支持 Seek，先回到起点，确保同一个 Curl 实例重复发送时请求体完整。
	if body, err = rewindRequestBody(body); err != nil {
		return apperrors.Wrap(err)
	}

	// 构建 Request
	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return apperrors.Wrap(err)
	}
	setRequestGetBody(req, body)

	// 设置请求头
	if c.header != nil && len(c.header) > 0 {
		if c.defLogOutput {
			c.Logger.Debug("set header")
		}
		req.Header = c.header.Clone()
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
		if err = c.beforeRequest(req); err != nil {
			return apperrors.Wrap(err)
		}
	}

	// 记录请求日志
	if c.defLogOutput && c.Logger.Enabled(context.Background(), LevelInfo) {
		if c.dump {
			dump, err := dumpRequestSafe(req, c.dumpBodyLimit)
			if err != nil {
				return apperrors.Wrap(err)
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
		return apperrors.Wrap(err)
	}

	// 执行 beforeClient 回调
	if c.beforeClient != nil {
		if c.defLogOutput {
			c.Logger.Debug("client()")
		}
		if err = c.beforeClient(c.cli); err != nil {
			return apperrors.Wrap(err)
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
			return apperrors.New("client.Do() retry body is not rewindable")
		}
		if i > 1 && req.GetBody != nil {
			req.Body, err = req.GetBody()
			if err != nil {
				return apperrors.Wrap(err)
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
			time.Sleep(retryDelay(i))
		}
	}

	if c.defLogOutput {
		c.Logger.Debug("client end", "time spent", time.Since(t1).String())
	}

	if err != nil {
		return apperrors.Errorf("client.Do() Retry %d times err: %v", maxRetry, err.Error())
	}

	var respBody []byte

	// 记录响应日志
	if c.defLogOutput && c.Logger.Enabled(context.Background(), LevelInfo) {
		if respBody, err = c.logResponse(resp); err != nil {
			return apperrors.Wrap(err)
		}
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK && !containsStatusCode(resp.StatusCode, c.statusCode) {
		return apperrors.Errorf("response error StatusCode: statusCode=%d, Status=%s", resp.StatusCode, resp.Status)
	}

	// 执行 afterResponse 回调
	if c.afterResponse != nil {
		if c.defLogOutput {
			c.Logger.Debug("response()")
		}
		isDone, err := c.afterResponse(resp)
		if err != nil {
			return apperrors.Wrap(err)
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
				return apperrors.Wrap(err)
			}
			respBody = buf.Bytes()
		}
		if err = c.afterBody(respBody); err != nil {
			return apperrors.Wrap(err)
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
			return nil, apperrors.Wrap(err)
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

// setRequestGetBody 为可 Seek 的请求体补充 GetBody，保证传输错误后可以安全重试。
//
// 参数说明：
//   - req：HTTP 请求对象。
//   - body：请求体。
func setRequestGetBody(req *http.Request, body io.Reader) {
	if req == nil || req.GetBody != nil || body == nil {
		return
	}
	readSeeker, ok := body.(io.ReadSeeker)
	if !ok {
		return
	}
	req.GetBody = func() (io.ReadCloser, error) {
		if _, err := readSeeker.Seek(0, io.SeekStart); err != nil {
			return nil, apperrors.Wrap(err)
		}
		return io.NopCloser(readSeeker), nil
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

	if req.Body != nil {
		b.WriteString("Request Body:\n")
		reqBody, restored, err := DrainBody(req.Body)
		if err != nil {
			c.Logger.Error("DrainBody error", "err", err.Error())
			return
		}
		req.Body = restored
		b.Write(reqBody)
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
	if c.dump {
		dump, err := dumpResponseSafe(resp, c.dumpBodyLimit)
		if err != nil {
			return nil, apperrors.Wrap(err)
		}
		c.Logger.Info("httputil.DumpResponse()", "response", dump)
	} else {
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Response Status: %s\n", resp.Status))
		b.WriteString("Response Body:\n")

		respBody, restored, err := DrainBody(resp.Body)
		if err != nil {
			return nil, apperrors.Wrap(err)
		}
		resp.Body = restored
		b.Write(respBody)
		c.Logger.Info("Response", "body", b.String())
		return respBody, nil
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
		return nil, b, apperrors.Wrap(err)
	}
	if err := b.Close(); err != nil {
		return nil, b, apperrors.Wrap(err)
	}
	bodyBytes := buf.Bytes()
	return bodyBytes, io.NopCloser(bytes.NewReader(bodyBytes)), nil
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
		return "", apperrors.Wrap(err)
	}

	if limit <= 0 || req.Body == nil || req.Body == http.NoBody {
		return string(dump), nil
	}

	if req.GetBody == nil {
		return string(dump) + "\nRequest Body: [skipped: non-rewindable]", nil
	}

	body, err := req.GetBody()
	if err != nil {
		return "", apperrors.Wrap(err)
	}
	defer body.Close()

	preview, truncated, err := readBodyPreview(body, limit)
	if err != nil {
		return "", apperrors.Wrap(err)
	}
	restored, err := req.GetBody()
	if err != nil {
		return "", apperrors.Wrap(err)
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
		return "", apperrors.Wrap(err)
	}

	if limit <= 0 || resp == nil || resp.Body == nil || resp.Body == http.NoBody {
		return string(dump), nil
	}

	preview, truncated, restored, err := readBodyPreviewAndRestore(resp.Body, limit)
	if err != nil {
		return "", apperrors.Wrap(err)
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
		return nil, false, apperrors.Wrap(err)
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
		return nil, false, body, apperrors.Wrap(err)
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
