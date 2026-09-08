//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package utils_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/Is999/go-utils"
)

// TestArchiveRejectsFIFO 验证特殊文件在准备条目前被拒绝，避免等待管道的外部读写端。
func TestArchiveRejectsFIFO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pipe")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("tar", func(t *testing.T) {
		// 意外写头时立即报错，使旧实现也能同步结束，不会阻塞在 FIFO 的 Open。
		output := &archiveHeaderWriter{}
		writer := tar.NewWriter(output)
		defer writer.Close()
		err := utils.AddFileToTar(writer, path, "")
		if err == nil || !strings.Contains(err.Error(), "不支持的 tar 条目类型") {
			t.Fatalf("AddFileToTar() error = %v, want unsupported entry type", err)
		}
		if output.calls != 0 {
			t.Fatalf("archive writes = %d, want 0 before rejection", output.calls)
		}
	})

	t.Run("zip", func(t *testing.T) {
		var output bytes.Buffer
		writer := zip.NewWriter(&output)
		defer writer.Close()
		prepared := false
		// ZIP 会先创建压缩器再写头；在这里截断旧实现，同样无需打开 FIFO。
		writer.RegisterCompressor(zip.Deflate, func(io.Writer) (io.WriteCloser, error) {
			prepared = true
			return nil, io.ErrUnexpectedEOF
		})
		err := utils.AddFileToZip(writer, path, "")
		if err == nil || !strings.Contains(err.Error(), "不支持的 zip 条目类型") {
			t.Fatalf("AddFileToZip() error = %v, want unsupported entry type", err)
		}
		if prepared || output.Len() != 0 {
			t.Fatalf("compressor prepared = %t, archive bytes = %d; want no entry preparation", prepared, output.Len())
		}
	})
}

// archiveHeaderWriter 记录意外头部写入并立即失败，使 FIFO 回归不依赖超时或后台 goroutine。
type archiveHeaderWriter struct {
	calls int // 写入尝试次数，在单个子测试内同步访问。
}

// Write 不接收归档字节，用固定错误截断意外的后续读取。
func (w *archiveHeaderWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, io.ErrUnexpectedEOF
}
