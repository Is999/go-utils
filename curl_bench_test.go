package utils_test

import (
	"net/url"
	"testing"

	"github.com/Is999/go-utils"
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

func BenchmarkGenerateUniqueID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchID = utils.GenerateUniqueID(16)
	}
}

func BenchmarkBuildURL(b *testing.B) {
	params := url.Values{
		"page": []string{"2"},
		"q":    []string{"codex"},
	}
	for i := 0; i < b.N; i++ {
		benchURL, _ = utils.BuildURL("https://example.com/search?lang=go", params)
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
