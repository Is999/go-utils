package utils

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/cookiejar"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// TestCurlClosesBodyBeforeSendFailure 覆盖请求体移交后、Transport 接管前的资源释放。
func TestCurlClosesBodyBeforeSendFailure(t *testing.T) {
	wantErr := errors.New("callback stopped request")
	for name, setup := range map[string]func(*Curl){
		"request callback": func(c *Curl) {
			c.BeforeRequest(func(*http.Request) error { return wantErr })
		},
		"client callback": func(c *Curl) {
			c.BeforeClient(func(*http.Client) error { return wantErr })
		},
		"transport config": func(c *Curl) {
			c.SetProxyURL("%")
		},
	} {
		t.Run(name, func(t *testing.T) {
			body := &curlCloseTracker{Reader: strings.NewReader("request")}
			c := NewCurl()
			setup(c)
			if err := c.SendContext(context.Background(), http.MethodPost, "http://example.com", body); err == nil {
				t.Fatal("SendContext() succeeded, want preparation error")
			}
			if !body.closed {
				t.Fatal("request body was not closed after preparation failed")
			}
		})
	}
}

// curlCloseTracker 模拟不支持重放、需要由请求生命周期释放的流式请求体。
type curlCloseTracker struct {
	io.Reader      // 隐藏底层定位能力，保持与 io.PipeReader 相同的所有权边界。
	closed    bool // 记录关闭是否发生；单个测试请求串行访问。
}

// Close 只记录关闭信号，避免测试依赖真实文件描述符。
func (body *curlCloseTracker) Close() error {
	body.closed = true
	return nil
}

// TestCurlClosesBodyOnInvalidURL 确保构造 Request 失败也释放已移交的流式请求体。
func TestCurlClosesBodyOnInvalidURL(t *testing.T) {
	body := &curlCloseTracker{Reader: strings.NewReader("request")}
	if err := NewCurl().SendContext(context.Background(), http.MethodPost, "%", body); err == nil {
		t.Fatal("SendContext() succeeded with an invalid URL")
	}
	if !body.closed {
		t.Fatal("request body was not closed after request construction failed")
	}
}

// TestCurlNonRewindableBodyPreservesCause 确保无法重放的正文仍保留首次发送错误，而非只返回重试限制。
func TestCurlNonRewindableBodyPreservesCause(t *testing.T) {
	for _, maxRetry := range []uint8{1, 2} {
		t.Run(strconv.Itoa(int(maxRetry)), func(t *testing.T) {
			cause := errors.New("transport failure")
			body := &curlCloseTracker{Reader: strings.NewReader("request")}
			attempts := 0
			curl := NewCurl(WithCurlMaxRetry(maxRetry)).BeforeClient(func(client *http.Client) error {
				client.Transport = curlRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					attempts++
					// RoundTripper 即使返回错误也负责关闭请求体，不依赖实际网络资源。
					_ = req.Body.Close()
					return nil, cause
				})
				return nil
			})
			err := curl.Send(http.MethodPost, "http://example.com", body)
			if !errors.Is(err, cause) {
				t.Errorf("Send() error = %v, want original transport cause", err)
			}
			if attempts != 1 || !body.closed {
				t.Errorf("attempts = %d, body closed = %t; want one attempt and closed body", attempts, body.closed)
			}
		})
	}
}

// TestCurlRetryKeepsPreviousRequest 验证重试不会改写 Transport 仍可异步读取和关闭的前次请求。
func TestCurlRetryKeepsPreviousRequest(t *testing.T) {
	for _, withBody := range []bool{false, true} {
		t.Run("body="+strconv.FormatBool(withBody), func(t *testing.T) {
			jar, err := cookiejar.New(nil)
			if err != nil {
				t.Fatal(err)
			}
			original := &curlCloseTracker{Reader: strings.NewReader("payload")}
			replay := &curlCloseTracker{Reader: strings.NewReader("payload")}
			response := &curlCloseTracker{Reader: strings.NewReader("response")}
			var body io.Reader
			if withBody {
				body = original
			}
			secondAttempt, sendDone, firstClosed := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var events, payloads, cookies []string
			var firstCookie string
			attempts, replays := 0, 0
			curl := NewCurl(WithCurlMaxRetry(2)).BeforeRequest(func(req *http.Request) error {
				events = append(events, "request")
				jar.SetCookies(req.URL, []*http.Cookie{{Name: "session", Value: "value"}})
				if withBody {
					req.GetBody = func() (io.ReadCloser, error) {
						replays++
						return replay, nil
					}
				}
				return nil
			}).BeforeClient(func(client *http.Client) error {
				events = append(events, "client")
				// 禁用 Client 的超时浅拷贝，使测试直接覆盖 Curl 的请求隔离责任。
				client.Timeout, client.Jar = 0, jar
				client.Transport = curlRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					attempts++
					if attempts == 2 {
						close(secondAttempt)
						<-firstClosed
					}
					cookies = append(cookies, req.Header.Get("Cookie"))
					var data []byte
					if req.Body != nil {
						data, err = io.ReadAll(req.Body)
						if err != nil {
							t.Errorf("read attempt %d: %v", attempts, err)
						}
					}
					payloads = append(payloads, string(data))
					if attempts == 1 {
						go func() {
							// 第二次已进入 Transport 后才读取旧字段，避免靠睡眠猜测调度。
							select {
							case <-secondAttempt:
							case <-sendDone:
							}
							firstCookie = req.Header.Get("Cookie")
							if req.Body != nil {
								_ = req.Body.Close()
							}
							close(firstClosed)
						}()
						return nil, io.ErrUnexpectedEOF
					}
					if req.Body != nil {
						_ = req.Body.Close()
					}
					return &http.Response{StatusCode: http.StatusOK, Body: response}, nil
				})
				return nil
			}).AfterResponse(func(*http.Response) (bool, error) {
				events = append(events, "response")
				return false, nil
			}).AfterBody(func(data []byte) error {
				events = append(events, "body")
				if string(data) != "response" {
					t.Errorf("response body = %q", data)
				}
				return nil
			}).AfterDone(func(*http.Client, *http.Request, *http.Response) {
				events = append(events, "done")
				if response.closed {
					t.Error("response closed before AfterDone")
				}
			})
			err = curl.SendContext(context.Background(), http.MethodPost, "http://example.test", body)
			close(sendDone)
			if attempts > 0 {
				<-firstClosed
			}
			if err != nil {
				t.Fatal(err)
			}
			wantPayload, wantReplays := "", 0
			if withBody {
				wantPayload, wantReplays = "payload", 1
			}
			if attempts != 2 || replays != wantReplays || !slices.Equal(payloads, []string{wantPayload, wantPayload}) {
				t.Errorf("attempts=%d replays=%d payloads=%q", attempts, replays, payloads)
			}
			if firstCookie != "session=value" || !slices.Equal(cookies, []string{"session=value", "session=value"}) {
				t.Errorf("previous Cookie=%q, sent Cookies=%q; want one session=value per request", firstCookie, cookies)
			}
			if withBody && (!original.closed || !replay.closed) {
				t.Errorf("request bodies closed: original=%t replay=%t", original.closed, replay.closed)
			}
			if !response.closed || !slices.Equal(events, []string{"request", "client", "response", "body", "done"}) {
				t.Errorf("response closed=%t callbacks=%v", response.closed, events)
			}
		})
	}
}

// TestCurlCloneRestoresCursorAfterReadError 防止失败的快照读取影响调用方继续读取模板。
func TestCurlCloneRestoresCursorAfterReadError(t *testing.T) {
	reader := &curlReadError{Reader: strings.NewReader("request")}
	if _, err := reader.Seek(2, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if _, err := NewCurl().SetBody(reader).Clone(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Clone() error = %v, want io.ErrUnexpectedEOF", err)
	}
	if pos, err := reader.Seek(0, io.SeekCurrent); err != nil || pos != 2 {
		t.Fatalf("template position = %d, error = %v; want 2", pos, err)
	}
}

// curlReadError 模拟定位可用但中途读取失败的外部请求体。
type curlReadError struct {
	*strings.Reader // 底层游标允许检查失败后的恢复位置。
}

// Read 在交付字节后注入失败，覆盖已经移动游标的错误路径。
func (reader *curlReadError) Read(p []byte) (int, error) {
	n, _ := reader.Reader.Read(p)
	return n, io.ErrUnexpectedEOF
}

// TestCurlCloneCopiesCookieUnparsed 确保解析器保留的属性也不在克隆之间共享。
func TestCurlCloneCopiesCookieUnparsed(t *testing.T) {
	base := NewCurl().SetCookies(&http.Cookie{Name: "session", Unparsed: []string{"custom=base"}})
	cloned, err := base.Clone()
	if err != nil {
		t.Fatal(err)
	}
	cloned.GetCookie("session").Unparsed[0] = "custom=clone"
	if got := base.GetCookie("session").Unparsed[0]; got != "custom=base" {
		t.Fatalf("template cookie attribute = %q", got)
	}
}

// TestCurlNewRequestClonesMemoryBodyConcurrently 覆盖带请求体模板的并发派生和原始游标保留。
func TestCurlNewRequestClonesMemoryBodyConcurrently(t *testing.T) {
	const payload = "shared template request body"
	for name, reader := range map[string]io.ReadSeeker{
		"bytes":   bytes.NewReader([]byte(payload)),
		"strings": strings.NewReader(payload),
	} {
		t.Run(name, func(t *testing.T) {
			// 模板可能已被读取；派生请求仍从头发送，但不能改变模板游标。
			if _, err := reader.Seek(7, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			base := NewCurl().SetBody(reader)
			var wg sync.WaitGroup
			for range 32 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for range 8 {
						cloned, err := base.NewRequest()
						if err != nil {
							t.Errorf("NewRequest() error = %v", err)
							return
						}
						got, err := io.ReadAll(cloned.body)
						if err != nil || string(got) != payload {
							t.Errorf("cloned body = %q, error = %v", got, err)
							return
						}
					}
				}()
			}
			wg.Wait()
			if pos, err := reader.Seek(0, io.SeekCurrent); err != nil || pos != 7 {
				t.Fatalf("template position = %d, error = %v; want 7", pos, err)
			}
		})
	}
}

// BenchmarkCurlCloneMemoryBody 衡量模板派生时请求体快照的成本，大小单位为字节。
func BenchmarkCurlCloneMemoryBody(b *testing.B) {
	for _, size := range []int{1024, 64 * 1024} {
		payload := strings.Repeat("x", size)
		for name, reader := range map[string]io.Reader{
			"bytes":   bytes.NewReader([]byte(payload)),
			"strings": strings.NewReader(payload),
		} {
			b.Run(name+"/"+strconv.Itoa(size), func(b *testing.B) {
				base := NewCurl().SetBody(reader)
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					if _, err := base.Clone(); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// curlRoundTripFunc 用内存响应替代网络 I/O，保留 Client.Do 与 Curl 回调的真实调用链。
type curlRoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip 让测试响应参与标准库 Client 的请求生命周期。
func (roundTrip curlRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return roundTrip(req)
}

// TestCurlBasicAuthPreservesCredentials 覆盖空凭证字段及特殊字符，编码规则与标准库保持一致。
func TestCurlBasicAuthPreservesCredentials(t *testing.T) {
	for _, tc := range []struct {
		name          string // 子测试标识，区分默认配置与显式认证输入。
		username      string // 账号允许为空，库不增加格式校验。
		password      string // 密码允许为空，特殊字符交由标准库编码。
		wantBasicAuth bool   // 默认配置保持不发送 Authorization。
	}{
		{name: "default"},
		{name: "username and password", username: "alice", password: "secret", wantBasicAuth: true},
		{name: "empty password", username: "alice", wantBasicAuth: true},
		{name: "empty username", password: "token", wantBasicAuth: true},
		{name: "special characters", username: "al:ice", password: "p@:ss word/中文", wantBasicAuth: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := &http.Request{Header: make(http.Header)}
			if tc.wantBasicAuth {
				want.SetBasicAuth(tc.username, tc.password)
			}
			called := false
			c := NewCurl().SetBasicAuth(tc.username, tc.password).SetDefaultLogOutput(false)
			c.BeforeClient(func(client *http.Client) error {
				client.Transport = curlRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					called = true
					if got := req.Header.Get("Authorization"); got != want.Header.Get("Authorization") {
						t.Errorf("Authorization = %q, want %q", got, want.Header.Get("Authorization"))
					}
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: http.NoBody, Request: req}, nil
				})
				return nil
			})
			if err := c.GetContext(context.Background(), "http://example.com"); err != nil {
				t.Fatal(err)
			}
			if !called {
				t.Fatal("request did not reach the transport")
			}
		})
	}
}

// TestCurlCloneKeepsLazyConfigurationIndependent 覆盖未分配和已清空容器的公开配置与回调行为。
func TestCurlCloneKeepsLazyConfigurationIndependent(t *testing.T) {
	for _, initialized := range []bool{false, true} {
		t.Run(strconv.FormatBool(initialized), func(t *testing.T) {
			base := NewCurl()
			if initialized {
				base.SetHeader("X-Empty", "1").DeleteHeaders("X-Empty")
				base.SetParam("empty", "1").DeleteParams("empty")
				base.SetCookies()
			}
			cloned, err := base.Clone()
			if err != nil {
				t.Fatal(err)
			}
			WithCurlHeaders(map[string]string{"X-Clone": "1"})(cloned)
			WithCurlParams(map[string]string{"q": "value"})(cloned)
			WithCurlCookies(&http.Cookie{Name: "session", Value: "clone"})(cloned)
			cloned.Header().Set("X-Direct", "2")
			cloned.Params().Set("page", "3")
			if base.HasHeader("X-Clone") || base.HasHeader("X-Direct") || base.HasParam("q") || base.HasParam("page") || base.HasCookie("session") {
				t.Fatal("clone configuration changed the template")
			}

			// 发送前停止，既验证构造的请求，也核对 Clone 仍提供独立的 Client 配置对象。
			wantErr := errors.New("stop before sending")
			done := false
			cloned.BeforeRequest(func(req *http.Request) error {
				if req.Header.Get("X-Clone") != "1" || req.Header.Get("X-Direct") != "2" || req.Header.Get("Content-Type") != defaultCurlContentType {
					t.Errorf("request headers = %v", req.Header)
				}
				if req.URL.Query().Get("q") != "value" || req.URL.Query().Get("page") != "3" {
					t.Errorf("request query = %q", req.URL.RawQuery)
				}
				if cookie, err := req.Cookie("session"); err != nil || cookie.Value != "clone" {
					t.Errorf("request cookie = %v, error = %v", cookie, err)
				}
				return wantErr
			}).AfterDone(func(client *http.Client, req *http.Request, resp *http.Response) {
				done = true
				if client == nil || req != nil || resp != nil {
					t.Errorf("AfterDone client/request/response = %v/%v/%v", client, req, resp)
				}
			})
			if err := cloned.GetContext(context.Background(), "http://example.com"); !errors.Is(err, wantErr) || !done {
				t.Fatalf("GetContext() error = %v, AfterDone called = %v", err, done)
			}
		})
	}
}

// TestCurlDumpBodyLifecycle 从 SendContext 验证预览不消费正文，并在预览失败时释放两条读取流。
func TestCurlDumpBodyLifecycle(t *testing.T) {
	for _, test := range []struct {
		name        string // 场景名称用于定位请求体能力和失败位置。
		payload     string // 实际请求体；空字符串覆盖标准库 NoBody。
		stream      bool   // 流式请求体没有 GetBody，日志应跳过预览。
		getBodyFail bool   // 在建立预览流前返回错误。
		readFail    bool   // 预览流交付部分内容后返回错误。
	}{
		{name: "replayable", payload: "abcdef"},
		{name: "empty"},
		{name: "stream", payload: "abcdef", stream: true},
		{name: "get-body error", payload: "abcdef", getBodyFail: true},
		{name: "preview read error", payload: "abcdef", readFail: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var logs strings.Builder
			logger := &slogLogger{l: slog.New(slog.NewTextHandler(&logs, nil))}
			original := &curlCloseTracker{Reader: strings.NewReader(test.payload)}
			preview := &curlCloseTracker{Reader: &curlReadError{Reader: strings.NewReader(test.payload)}}
			var body io.Reader = strings.NewReader(test.payload)
			if test.stream || test.getBodyFail || test.readFail {
				body = original
			}
			sent := false
			curl := NewCurl(WithCurlLogger(logger), WithCurlDefLogOutput(true), WithCurlDumpBodyLimit(3)).
				SetDump(true).
				BeforeRequest(func(req *http.Request) error {
					if test.getBodyFail || test.readFail {
						req.GetBody = func() (io.ReadCloser, error) {
							if test.getBodyFail {
								return nil, io.ErrUnexpectedEOF
							}
							return preview, nil
						}
					}
					return nil
				}).
				BeforeClient(func(client *http.Client) error {
					client.Transport = curlRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						sent = true
						if req.Body != nil {
							data, err := io.ReadAll(req.Body)
							_ = req.Body.Close()
							if err != nil || string(data) != test.payload {
								t.Errorf("sent body = %q, error = %v", data, err)
							}
						}
						return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: http.NoBody}, nil
					})
					return nil
				})
			defer curl.CloseIdleConnections()
			err := curl.SendContext(context.Background(), http.MethodPost, "http://example.com", body)
			if test.getBodyFail || test.readFail {
				if !errors.Is(err, io.ErrUnexpectedEOF) || sent || !original.closed || test.readFail && !preview.closed {
					t.Fatalf("error=%v, sent=%v, original closed=%v, preview closed=%v", err, sent, original.closed, preview.closed)
				}
				return
			}
			if err != nil || !sent {
				t.Fatalf("SendContext() error = %v, sent = %v", err, sent)
			}
			if test.stream && !strings.Contains(logs.String(), "[skipped: non-rewindable]") {
				t.Fatalf("stream preview was not skipped: %s", logs.String())
			}
			if !test.stream && test.payload != "" && !strings.Contains(logs.String(), "[truncated]") {
				t.Fatalf("request preview was not truncated: %s", logs.String())
			}
		})
	}
}

// TestCurlPreviewLargeLimit 覆盖预览上限的整数边界，正文仍使用小数据完成真实发送链路。
func TestCurlPreviewLargeLimit(t *testing.T) {
	const requestBody = "request-preview-content"
	const responseBody = "response-preview-content"
	for _, limit := range []int64{math.MaxInt64 - 1, math.MaxInt64} {
		for _, dump := range []bool{false, true} {
			t.Run(strconv.FormatInt(limit, 10)+"/dump="+strconv.FormatBool(dump), func(t *testing.T) {
				var logs strings.Builder
				logger := &slogLogger{l: slog.New(slog.NewTextHandler(&logs, nil))}
				body := &curlCloseTracker{Reader: strings.NewReader(responseBody)}
				received := ""
				curl := NewCurl(WithCurlLogger(logger), WithCurlDefLogOutput(true),
					WithCurlLogBodyLimit(limit), WithCurlDumpBodyLimit(limit)).
					SetDump(dump).
					BeforeClient(func(client *http.Client) error {
						client.Transport = curlRoundTripFunc(func(req *http.Request) (*http.Response, error) {
							data, err := io.ReadAll(req.Body)
							_ = req.Body.Close()
							if err != nil || string(data) != requestBody {
								t.Errorf("request body = %q, error = %v", data, err)
							}
							return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: body}, nil
						})
						return nil
					}).AfterBody(func(data []byte) error {
					received = string(data)
					return nil
				})
				defer curl.CloseIdleConnections()
				if err := curl.SendContext(context.Background(), http.MethodPost, "http://example.com", strings.NewReader(requestBody)); err != nil {
					t.Fatal(err)
				}
				if received != responseBody || !body.closed {
					t.Fatalf("response body = %q, closed = %t", received, body.closed)
				}
				// 上限大于正文时，普通日志和 dump 都必须包含完整预览。
				if !strings.Contains(logs.String(), requestBody) || !strings.Contains(logs.String(), responseBody) || strings.Contains(logs.String(), "[truncated]") {
					t.Fatalf("unexpected preview: %s", logs.String())
				}
			})
		}
	}
}

// BenchmarkCurlSendResponse 覆盖请求准备、Client.Do、可选日志预览和响应读取。
func BenchmarkCurlSendResponse(b *testing.B) {
	for _, size := range []int{128, 4096, 65536} {
		payload := strings.Repeat("x", size)
		for _, logging := range []bool{false, true} {
			b.Run(strconv.Itoa(size)+"/logs="+strconv.FormatBool(logging), func(b *testing.B) {
				// 日志写入丢弃端，计入预览和编码成本，不计入文件或网络输出。
				logger := &slogLogger{l: slog.New(slog.NewTextHandler(io.Discard, nil))}
				curl := NewCurl(WithCurlLogger(logger), WithCurlDefLogOutput(logging)).
					BeforeClient(func(client *http.Client) error {
						client.Transport = curlRoundTripFunc(func(req *http.Request) (*http.Response, error) {
							return &http.Response{
								StatusCode:    http.StatusOK,
								Status:        "200 OK",
								Body:          io.NopCloser(strings.NewReader(payload)),
								ContentLength: int64(len(payload)),
								Request:       req,
							}, nil
						})
						return nil
					}).
					AfterBody(func(body []byte) error {
						if len(body) != size {
							b.Fatalf("response body size = %d, want %d", len(body), size)
						}
						return nil
					})
				defer curl.CloseIdleConnections()
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					if err := curl.GetContext(context.Background(), "http://example.com"); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// TestCurlPreservesStreamingTrailer 保留正文读取期间填入的尾字段，不能在发送前复制其映射。
func TestCurlPreservesStreamingTrailer(t *testing.T) {
	reader, writer := io.Pipe()
	var wg sync.WaitGroup
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
		wg.Wait()
	})
	curl := NewCurl(WithCurlMaxRetry(1)).BeforeRequest(func(req *http.Request) error {
		req.Trailer = http.Header{"X-Checksum": nil}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := io.WriteString(writer, "payload")
			if err == nil {
				// Trailer 在正文读取期间填入，EOF 后保持不变。
				req.Trailer.Set("X-Checksum", "complete")
			}
			_ = writer.CloseWithError(err)
		}()
		return nil
	}).BeforeClient(func(client *http.Client) error {
		client.Transport = curlRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			defer req.Body.Close()
			data, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			if string(data) != "payload" {
				t.Errorf("request body = %q, want payload", data)
			}
			if got := req.Trailer.Get("X-Checksum"); got != "complete" {
				t.Errorf("request trailer = %q, want complete", got)
			}
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		})
		return nil
	})
	if err := curl.SendContext(context.Background(), http.MethodPost, "http://example.test", reader); err != nil {
		t.Fatal(err)
	}
}
