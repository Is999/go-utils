package utils_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Is999/go-utils"
)

func TestURLPathMergesQuery(t *testing.T) {
	got, err := utils.URLPath("https://example.test/path?a=1", url.Values{
		"b": []string{"2"},
	})
	if err != nil {
		t.Fatalf("URLPath() error = %v", err)
	}
	if got != "https://example.test/path?a=1&b=2" {
		t.Fatalf("URLPath() = %q", got)
	}

	if _, err = utils.URLPath("://bad-url", url.Values{"a": []string{"1"}}); err == nil {
		t.Fatal("URLPath() should reject invalid URL")
	}
}

func TestCurlConfigAndMethods(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Header.Get("X-Test") != "yes" {
			t.Errorf("missing X-Test header")
		}
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	c := utils.NewCurl(
		utils.WithCurlTimeout(2*time.Second),
		utils.WithCurlContentType("text/plain"),
		utils.WithCurlHeader("X-Test", "yes"),
		utils.WithCurlHeaders(map[string]string{"X-More": "1"}),
		utils.WithCurlParams(map[string]string{"page": "1"}),
		utils.WithCurlBody(strings.NewReader("body")),
		utils.WithCurlBodyBytes([]byte("bytes")),
		utils.WithCurlCookies(&http.Cookie{Name: "sid", Value: "1"}),
		utils.WithCurlBasicAuth("u", "p"),
		utils.WithCurlProxyURL(""),
		utils.WithCurlInsecureSkipVerify(true),
		utils.WithCurlRootCAs(""),
		utils.WithCurlCertKey("", ""),
		utils.WithCurlStatusCode(http.StatusAccepted),
		utils.WithCurlMaxRetry(1),
		utils.WithCurlDump(false),
		utils.WithCurlDumpBodyLimit(16),
		utils.WithCurlLogBodyLimit(16),
		utils.WithCurlRequestID("rid"),
	)

	c.AddHeader("X-List", "a", "b").
		AddHeaders(map[string][]string{"X-Add": {"c"}}).
		AddParam("tag", "a", "b").
		AddParams(map[string][]string{"kind": {"x"}}).
		SetUserAgent("go-utils-test").
		SetProxyURL("").
		InsecureSkipVerify(true).
		SetRootCAs("").
		SetCertKey("", "").
		SetDefLogOutput(false).
		SetLogBodyLimit(8)

	if !c.HasHeader("X-Test") || len(c.GetHeaderValues("X-List")) != 2 {
		t.Fatal("header helpers did not record values")
	}
	if !c.HasParam("page") || len(c.GetParamValues("tag")) != 2 {
		t.Fatal("param helpers did not record values")
	}
	if !c.HasCookie("sid") || c.GetCookie("sid").Value != "1" {
		t.Fatal("cookie helpers did not record value")
	}
	if c.GetRequestID() != "rid" {
		t.Fatal("request id should use configured value")
	}
	if got := c.GetStatusCode(); len(got) != 1 || got[0] != http.StatusAccepted {
		t.Fatalf("GetStatusCode() = %#v", got)
	}

	c.DeleteHeaders("X-More").DeleteParams("kind").DeleteCookies("sid")
	c.ResetHeader(http.Header{"X-Test": []string{"yes"}})
	c.ResetParams(url.Values{"reset": []string{"1"}})
	c.SetCookies(&http.Cookie{Name: "sid", Value: "2"}).ClearCookies()
	c.CloseIdleConnections()

	for _, call := range []struct {
		name string
		run  func() error
	}{
		{"send", func() error { return c.Send(http.MethodGet, server.URL, nil) }},
		{"put", func() error { return c.Put(server.URL) }},
		{"patch", func() error { return c.Patch(server.URL) }},
		{"head", func() error { return c.Head(server.URL) }},
		{"delete", func() error { return c.Delete(server.URL) }},
		{"options", func() error { return c.Options(server.URL) }},
	} {
		if err := call.run(); err != nil {
			t.Fatalf("%s error = %v", call.name, err)
		}
	}
	if len(methods) != 6 {
		t.Fatalf("methods = %#v", methods)
	}
}

func TestCurlCallbacksAndDrainBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ReadAll() error = %v", err)
		}
		if string(body) != "payload" {
			t.Errorf("request body = %q, want payload", body)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	var (
		beforeRequestCalled bool
		afterResponseCalled bool
		afterDoneCalled     bool
	)
	curl := utils.NewCurl().
		BeforeRequest(func(req *http.Request) error {
			beforeRequestCalled = true
			req.Header.Set("X-Before", "yes")
			return nil
		}).
		AfterResponse(func(resp *http.Response) (bool, error) {
			afterResponseCalled = true
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
			}
			return false, nil
		}).
		AfterDone(func(client *http.Client, req *http.Request, resp *http.Response) {
			afterDoneCalled = true
			if client == nil || req == nil || resp == nil {
				t.Fatalf("AfterDone() args contain nil: client=%v req=%v resp=%v", client, req, resp)
			}
		})

	if err := curl.Send(http.MethodPost, server.URL, strings.NewReader("payload")); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !beforeRequestCalled || !afterResponseCalled || !afterDoneCalled {
		t.Fatalf("callback flags = before:%v response:%v done:%v", beforeRequestCalled, afterResponseCalled, afterDoneCalled)
	}

	body, restored, err := utils.DrainBody(io.NopCloser(strings.NewReader("drain")))
	if err != nil {
		t.Fatalf("DrainBody() error = %v", err)
	}
	if string(body) != "drain" {
		t.Fatalf("DrainBody() body = %q, want drain", body)
	}
	restoredBody, err := io.ReadAll(restored)
	if err != nil {
		t.Fatalf("ReadAll(restored) error = %v", err)
	}
	if !bytes.Equal(restoredBody, body) {
		t.Fatalf("restored body = %q, want %q", restoredBody, body)
	}
	if body, restored, err = utils.DrainBody(nil); err != nil || len(body) != 0 || restored != http.NoBody {
		t.Fatalf("DrainBody(nil) = body:%q restored:%v err:%v", body, restored, err)
	}
}

func TestFormDeleteAndReader(t *testing.T) {
	form := utils.NewForm().
		SetParam("a", "1").
		AddParam("b", "2", "3").
		SetParams(map[string]string{"c": "4"}).
		AddParams(map[string][]string{"d": {"5"}}).
		SetFile("file", "missing.txt").
		AddFile("more", "a.txt", "b.txt").
		SetFiles(map[string]string{"single": "c.txt"}).
		AddFiles(map[string][]string{"many": {"d.txt"}})

	form.DeleteParams("b", "missing")
	form.DeleteFiles("file", "missing")
	if _, ok := form.Params["b"]; ok {
		t.Fatal("DeleteParams() should remove key")
	}
	if _, ok := form.Files["file"]; ok {
		t.Fatal("DeleteFiles() should remove key")
	}

	plain := utils.NewForm().SetParam("a", "1")
	body, contentType, err := plain.Reader()
	if err != nil {
		t.Fatalf("Reader() error = %v", err)
	}
	if contentType != "application/x-www-form-urlencoded" {
		t.Fatalf("contentType = %q", contentType)
	}
	data, _ := io.ReadAll(body)
	if string(data) != "a=1" {
		t.Fatalf("body = %q", data)
	}

	if _, err = json.Marshal(form.Params); err != nil {
		t.Fatalf("params should remain JSON serializable: %v", err)
	}
}
