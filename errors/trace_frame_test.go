package errors

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestQuotedFrameJSON 验证栈帧与普通错误消息遵循相同的 JSON 字符串规则。
func TestQuotedFrameJSON(t *testing.T) {
	for _, file := range []string{
		"pkg/file.go",
		"pkg/中文.go",
		"pkg/quote\"line\n.go",
		"pkg/line\u2028paragraph\u2029.go",
		"pkg/invalid\xff.go",
		"pkg/a&b.go",
	} {
		t.Run(strconv.Quote(file), func(t *testing.T) {
			frame := runtime.Frame{Function: "example.处理", File: file, Line: 17}
			var got strings.Builder
			writeQuotedFrame(&got, frame)

			// TraceJSON 保留 HTML 字符，其他转义规则与标准 JSON 编码器一致。
			var want bytes.Buffer
			encoder := json.NewEncoder(&want)
			encoder.SetEscapeHTML(false)
			if err := encoder.Encode(frame.Function + " (" + file + ":17)"); err != nil {
				t.Fatal(err)
			}
			if got.String() != strings.TrimSuffix(want.String(), "\n") {
				t.Fatalf("quoted frame = %q, want %q", got.String(), want.String())
			}
		})
	}
}

// BenchmarkQuotedFrame 分别计量普通路径、中文路径和需要转义的路径。
func BenchmarkQuotedFrame(b *testing.B) {
	for _, file := range []string{"pkg/file.go", "pkg/中文.go", "pkg/quote\"line.go"} {
		b.Run(file, func(b *testing.B) {
			// runtime.Frame.File 通常是模块内的绝对路径，包含实际使用的相对路径裁剪。
			frame := runtime.Frame{Function: "example.Handler", File: filepath.Join(projectRoot(), file), Line: 17}
			b.ReportAllocs()
			for b.Loop() {
				var out strings.Builder
				out.Grow(256)
				writeQuotedFrame(&out, frame)
				if out.Len() == 0 {
					b.Fatal("empty frame")
				}
			}
		})
	}
}
