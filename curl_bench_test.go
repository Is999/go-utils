package utils_test

import (
	"testing"

	"github.com/Is999/go-utils"
)

var (
	benchCurl *utils.Curl
	benchBody int
)

// BenchmarkNew 计量请求实例与默认配置的构造成本。
func BenchmarkNew(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		benchCurl = utils.NewCurl(utils.WithCurlLogger(curlTestLogger{}))
	}
}

// BenchmarkCurlRequestID 使用空日志器测量 ID 生成与请求头更新，不包含 Curl 构造。
func BenchmarkCurlRequestID(b *testing.B) {
	curl := utils.NewCurl(utils.WithCurlLogger(curlTestLogger{}))
	b.ResetTimer()
	for range b.N {
		curl.SetRequestID()
	}
}

// BenchmarkFormReaderURLEncoded 计量固定字段的编码与正文读取，不包含表单配置。
func BenchmarkFormReaderURLEncoded(b *testing.B) {
	form := utils.NewForm().
		SetParam("page", "2").
		SetParam("q", "alice")
	var buf [64]byte
	b.ResetTimer()
	for range b.N {
		body, contentType, err := form.Reader()
		if err != nil {
			b.Fatal(err)
		}
		if contentType == "" {
			b.Fatal("empty content type")
		}
		benchBody, _ = body.Read(buf[:])
	}
}
