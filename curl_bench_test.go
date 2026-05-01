package utils_test

import "github.com/Is999/go-utils"

import (
	"net/url"
	"testing"
)

var (
	benchCurl *utils.Curl
	benchURL  string
	benchID   string
	benchBody int
)

func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchCurl = utils.NewCurl(utils.WithCurlLogger(curlTestLogger{}))
	}
}

func BenchmarkGenerateUniqID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchID = utils.GenerateUniqId(16)
	}
}

func BenchmarkBuildURL(b *testing.B) {
	params := url.Values{
		"page": []string{"2"},
		"q":    []string{"codex"},
	}
	for i := 0; i < b.N; i++ {
		benchURL, _ = utils.BuildUrl("https://example.com/search?lang=go", params)
	}
}

func BenchmarkFormReaderURLEncoded(b *testing.B) {
	form := utils.NewForm().
		SetParam("page", "2").
		SetParam("q", "codex")
	var buf [64]byte
	for i := 0; i < b.N; i++ {
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
