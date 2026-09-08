package utils_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/iotest"

	"github.com/Is999/go-utils"
)

// TestFindFiles 使用独立目录验证匹配和递归，避免访问仓库之外的文件。
func TestFindFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"alpha.txt", "app.yaml", "beta.log", "nested/app.yaml", "nested/deep/beta.log"} {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name    string
		depth   bool
		match   []string
		want    []string // 相对测试目录的路径，顺序与目录遍历结果一致。
		wantErr bool
	}{
		{name: "all", want: []string{"alpha.txt", "app.yaml", "beta.log"}},
		{name: "wildcard", match: []string{"*"}, want: []string{"alpha.txt", "app.yaml", "beta.log"}},
		{name: "wildcard ignores rules", match: []string{"*", ".txt"}, want: []string{"alpha.txt", "app.yaml", "beta.log"}},
		{name: "prefix", match: []string{"p", "a"}, want: []string{"alpha.txt", "app.yaml"}},
		{name: "suffix", match: []string{"s", ".log"}, want: []string{"beta.log"}},
		{name: "exact", match: []string{"e", "app.yaml", "beta.log"}, want: []string{"app.yaml", "beta.log"}},
		{name: "single name", match: []string{"app.yaml"}, want: []string{"app.yaml"}},
		{name: "no match", match: []string{"e", "absent"}},
		{name: "regexp", match: []string{"r", `^a.*\.txt$`, `^beta\.log$`}, want: []string{"alpha.txt", "beta.log"}},
		{name: "invalid regexp", match: []string{"r", "["}, wantErr: true},
		{name: "recursive", depth: true, want: []string{"alpha.txt", "app.yaml", "beta.log", "nested/app.yaml", "nested/deep/beta.log"}},
		{name: "recursive prefix", depth: true, match: []string{"p", "app", "beta"}, want: []string{"app.yaml", "beta.log", "nested/app.yaml", "nested/deep/beta.log"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.FindFiles(dir, tt.depth, tt.match...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FindFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
			paths := make([]string, 0, len(got))
			for _, file := range got {
				if !filepath.IsAbs(file.Path) || file.IsDir() || file.Name() != filepath.Base(file.Path) {
					t.Fatalf("FindFiles() invalid file metadata: %+v", file)
				}
				rel, err := filepath.Rel(dir, file.Path)
				if err != nil {
					t.Fatal(err)
				}
				paths = append(paths, filepath.ToSlash(rel))
			}
			if !slices.Equal(paths, tt.want) {
				t.Fatalf("FindFiles() paths = %#v, want %#v", paths, tt.want)
			}
		})
	}
}

func TestIsDir(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "001", args: args{"./errors"}, want: true},
		{name: "002", args: args{"./slices.go"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.IsDir(tt.args.path); got != tt.want {
				t.Errorf("IsDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsFile(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "001", args: args{"./errors"}, want: false},
		{name: "002", args: args{"./slices.go"}, want: true},
		{name: "003", args: args{"./array.go"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.IsFile(tt.args.path); got != tt.want {
				t.Errorf("IsFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsExist(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "001", args: args{"./errors"}, want: true},
		{name: "002", args: args{"./slices.go"}, want: true},
		{name: "003", args: args{"./not_exist.go"}, want: false},
		{name: "004", args: args{"./file.go"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.IsExist(tt.args.path); got != tt.want {
				t.Errorf("IsExist() = %v, want %v", got, tt.want)
			}
			if size, err := utils.Size(tt.args.path); (err != nil) == tt.want {
				t.Errorf("Size() path = %v, size = %v, WrapError %v", tt.args.path, utils.FormatFileSize(size, 4), err)
			}
		})
	}
}

func TestCopy(t *testing.T) {
	want, err := os.ReadFile("json.go")
	if err != nil {
		t.Fatal(err)
	}
	// 源码只读，复制目标由当前用例独占并清理。
	dest := filepath.Join(t.TempDir(), "json.txt")
	if err := utils.Copy("json.go", dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("Copy() content = %q, error = %v; want source content", got, err)
	}
}

func TestCopyRejectsSameSourceAndDestination(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "same.txt")
	original := []byte("copy same file should keep content")
	if err := os.WriteFile(fileName, original, 0644); err != nil {
		t.Fatal(err)
	}

	err := utils.Copy(fileName, fileName)
	if err == nil {
		t.Fatal("Copy() expected error when source equals destination")
	}

	got, readErr := os.ReadFile(fileName)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(got) != string(original) {
		t.Fatalf("file content changed after rejected Copy(): got %q want %q", got, original)
	}
}

func TestCopyRejectsSameUnderlyingFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	dst := filepath.Join(dir, "target.txt")
	original := []byte("copy hardlink should keep content")
	if err := os.WriteFile(src, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(src, dst); err != nil {
		t.Skipf("os.Link() unsupported: %v", err)
	}

	err := utils.Copy(src, dst)
	if err == nil {
		t.Fatal("Copy() expected error when destination is same underlying file")
	}

	got, readErr := os.ReadFile(src)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(got) != string(original) {
		t.Fatalf("source content changed after rejected Copy(): got %q want %q", got, original)
	}
}

func TestCopyRejectsSymlinkDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	outside := filepath.Join(dir, "outside.txt")
	link := filepath.Join(dir, "target-link.txt")
	original := []byte("copy should reject symlink destination")
	if err := os.WriteFile(src, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("outside"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	err := utils.Copy(src, link)
	if err == nil {
		t.Fatal("Copy() expected error when destination is symlink")
	}

	got, readErr := os.ReadFile(outside)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(got) != "outside" {
		t.Fatalf("symlink target content changed after rejected Copy(): got %q want %q", got, "outside")
	}
}

func TestScan(t *testing.T) {
	want, err := os.ReadFile("file.go")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open("file.go")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var content []byte
	lines := 0
	err = utils.Scan(file, func(number int, line []byte, readErr error) error {
		lines++
		if number != lines || readErr != nil {
			t.Fatalf("Scan() callback = (%d, %v), want (%d, nil)", number, readErr, lines)
		}
		// 源码 fixture 使用 LF，补回分隔符后应与原文件逐字节一致。
		content = append(content, line...)
		content = append(content, '\n')
		return nil
	}, int(utils.MB))
	if err != nil || !bytes.Equal(content, want) {
		t.Fatalf("Scan() read %d bytes, error = %v; want %d matching bytes", len(content), err, len(want))
	}
}

// go test -bench=Scan$ -run ^$  -count 5 -benchmem
func BenchmarkScan(b *testing.B) {
	// 每次操作包含打开、完整扫描和关闭，不额外复制正文。
	for b.Loop() {
		file, err := os.Open("file.go")
		if err != nil {
			b.Fatal(err)
		}
		readErr := utils.Scan(file, func(int, []byte, error) error { return nil }, int(utils.MB))
		closeErr := file.Close()
		if readErr != nil {
			b.Fatal(readErr)
		}
		if closeErr != nil {
			b.Fatal(closeErr)
		}
	}
}

func TestLine(t *testing.T) {
	want, err := os.ReadFile("file.go")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open("file.go")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var content []byte
	lines := 0
	err = utils.Line(file, func(number int, line []byte, done bool) error {
		if number != lines+1 {
			t.Fatalf("Line() number = %d, want %d", number, lines+1)
		}
		content = append(content, line...)
		if done {
			lines++
			content = append(content, '\n')
		}
		return nil
	})
	if err != nil || !bytes.Equal(content, want) {
		t.Fatalf("Line() read %d bytes, error = %v; want %d matching bytes", len(content), err, len(want))
	}
}

func TestLineKeepsPhysicalLineNumberAcrossLongLineChunks(t *testing.T) {
	input := strings.Repeat("a", 70*1024) + "\nsecond"
	expectedLine := 1
	completedLines := 0
	callbacks := 0

	err := utils.Line(strings.NewReader(input), func(line int, _ []byte, done bool) error {
		callbacks++
		if line != expectedLine {
			t.Fatalf("Line() callback line = %d, want %d", line, expectedLine)
		}
		if done {
			expectedLine++
			completedLines++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Line() error = %v", err)
	}
	if callbacks < 3 {
		t.Fatalf("Line() callbacks = %d, want at least 3", callbacks)
	}
	if completedLines != 2 {
		t.Fatalf("Line() completed lines = %d, want 2", completedLines)
	}
}

func TestLineDataWithError(t *testing.T) {
	// Reader 可以同时返回有效尾块和读取错误；回调处理尾块后仍须返回原读取错误。
	readErr := errors.New("read failed")
	for _, handlerErr := range []error{nil, utils.DONE, errors.New("handler failed")} {
		t.Run(fmt.Sprint(handlerErr), func(t *testing.T) {
			read := false
			reader := fileReadFunc(func(p []byte) (int, error) {
				if read {
					return 0, io.EOF
				}
				read = true
				return copy(p, "payload"), readErr
			})
			var content string
			err := utils.Line(reader, func(number int, data []byte, done bool) error {
				if number != 1 || !done {
					t.Fatalf("Line() callback = (%d, %t), want (1, true)", number, done)
				}
				content = string(data)
				return handlerErr
			})
			wantErr := readErr
			if handlerErr != nil {
				wantErr = handlerErr
			}
			if handlerErr == utils.DONE {
				wantErr = nil
			}
			if content != "payload" || !errors.Is(err, wantErr) {
				t.Fatalf("Line() = (%q, %v), want (payload, %v)", content, err, wantErr)
			}
		})
	}
}

// TestLineCompletesFinalLine 保证末行完成通知不依赖 Reader 是否同时返回数据与 EOF。
func TestLineCompletesFinalLine(t *testing.T) {
	block := strings.Repeat("x", bufio.MaxScanTokenSize)
	for _, tt := range []struct {
		name  string
		input string
		want  []string // 回调拼接并完成的物理行，不含 LF/CRLF。
	}{
		{name: "empty"},
		{name: "below boundary", input: block[1:], want: []string{block[1:]}},
		{name: "at boundary", input: block, want: []string{block}},
		{name: "above boundary", input: block + "x", want: []string{block + "x"}},
		{name: "two blocks", input: block + block, want: []string{block + block}},
		{name: "after complete line", input: "first\n" + block, want: []string{"first", block}},
		{name: "with LF", input: block + "\n", want: []string{block}},
		{name: "split CRLF", input: block[1:] + "\r\n", want: []string{block[1:]}},
		{name: "trailing CR", input: block + "\r", want: []string{block + "\r"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, eofWithData := range []bool{false, true} {
				var input io.Reader = strings.NewReader(tt.input)
				if eofWithData {
					input = iotest.DataErrReader(input)
				}
				var lines []string // 仅在收到 done 后提交完整行。
				var line strings.Builder
				err := utils.Line(input, func(number int, data []byte, done bool) error {
					if number != len(lines)+1 {
						t.Fatalf("line number = %d, want %d", number, len(lines)+1)
					}
					line.Write(data)
					if done {
						lines = append(lines, line.String())
						line.Reset()
					}
					return nil
				})
				if err != nil || !slices.Equal(lines, tt.want) || line.Len() != 0 {
					t.Fatalf("Line() completed %d lines, pending %d bytes, error = %v; want %d lines (EOF with data: %t)", len(lines), line.Len(), err, len(tt.want), eofWithData)
				}
			}
		})
	}
}

// TestLineFinalNotificationError 在空末块通知中仍遵循 DONE、回调错误、读取错误的优先级。
func TestLineFinalNotificationError(t *testing.T) {
	block := strings.Repeat("x", bufio.MaxScanTokenSize)
	readErr := errors.New("read failed")
	handlerErr := errors.New("handler failed")
	for _, tt := range []struct {
		name        string
		readErr     error
		callbackErr error
		wantErr     error
	}{
		{name: "EOF", readErr: io.EOF},
		{name: "EOF and DONE", readErr: io.EOF, callbackErr: utils.DONE},
		{name: "EOF and callback error", readErr: io.EOF, callbackErr: handlerErr, wantErr: handlerErr},
		{name: "read error", readErr: readErr, wantErr: readErr},
		{name: "read error and DONE", readErr: readErr, callbackErr: utils.DONE},
		{name: "read and callback errors", readErr: readErr, callbackErr: handlerErr, wantErr: handlerErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// 先交付完整缓冲区，再单独报告 EOF 或读取失败。
			reader := io.MultiReader(strings.NewReader(block), fileReadFunc(func([]byte) (int, error) {
				return 0, tt.readErr
			}))
			completed := false // 结束通知必须发生在同一行的空末块。
			err := utils.Line(reader, func(number int, data []byte, done bool) error {
				if number != 1 {
					t.Fatalf("line number = %d, want 1", number)
				}
				if !done {
					return nil
				}
				if completed || len(data) != 0 {
					t.Fatalf("unexpected final notification: completed=%t, bytes=%d", completed, len(data))
				}
				completed = true
				return tt.callbackErr
			})
			if !completed || !errors.Is(err, tt.wantErr) {
				t.Fatalf("Line() completed=%t, error=%v; want completed=true, error=%v", completed, err, tt.wantErr)
			}
		})
	}
}

func TestLineFraming(t *testing.T) {
	// 记录每次回调的实际片段，覆盖 CRLF 恰好跨越内部缓冲区的情况。
	type chunk struct {
		line int
		data string
		done bool
	}
	block := strings.Repeat("x", bufio.MaxScanTokenSize)
	for _, tt := range []struct {
		name  string
		input string
		want  []chunk
	}{
		{name: "empty"},
		{name: "line endings", input: "\n\r\nx\r\r\nlast\r", want: []chunk{{1, "", true}, {2, "", true}, {3, "x\r", true}, {4, "last\r", true}}},
		{name: "long line", input: block + "tail\n", want: []chunk{{1, block, false}, {1, "tail", true}}},
		{name: "split CRLF", input: block[1:] + "\r\nnext", want: []chunk{{1, block[1:], false}, {1, "", true}, {2, "next", true}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, eofWithData := range []bool{false, true} {
				var input io.Reader = strings.NewReader(tt.input)
				var reference io.Reader = strings.NewReader(tt.input)
				if eofWithData {
					input = iotest.DataErrReader(input)
					reference = iotest.DataErrReader(reference)
				}
				var got []chunk
				err := utils.Line(input, func(number int, data []byte, done bool) error {
					got = append(got, chunk{number, string(data), done})
					return nil
				})
				if err != nil || !slices.Equal(got, tt.want) {
					t.Fatalf("Line() returned %d chunks, error = %v; want %d chunks with matching numbers, data and done flags", len(got), err, len(tt.want))
				}
				// 成功读取时逐片段对照标准库，保留长行和 CRLF 的分块协议。
				reader := bufio.NewReaderSize(reference, bufio.MaxScanTokenSize)
				var standard []chunk
				number := 1
				for {
					line, prefix, err := reader.ReadLine()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
					standard = append(standard, chunk{number, string(line), !prefix})
					if !prefix {
						number++
					}
				}
				if !slices.Equal(got, standard) {
					t.Fatalf("Line() framing differs from bufio.ReadLine (EOF with data: %t)", eofWithData)
				}
			}
		})
	}
}

// go test -bench=Line$ -run ^$  -count 5 -benchmem
func BenchmarkLine(b *testing.B) {
	// 每次操作包含打开、完整读取和关闭，不在循环间复用文件游标。
	for b.Loop() {
		file, err := os.Open("file.go")
		if err != nil {
			b.Fatal(err)
		}
		readErr := utils.Line(file, func(int, []byte, bool) error { return nil })
		closeErr := file.Close()
		if readErr != nil {
			b.Fatal(readErr)
		}
		if closeErr != nil {
			b.Fatal(closeErr)
		}
	}
}

func TestRead(t *testing.T) {
	want, err := os.ReadFile("file.go")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open("file.go")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var content []byte
	err = utils.Read(file, func(size int, block []byte) error {
		if size != len(block) {
			t.Fatalf("Read() block size = %d, want %d", size, len(block))
		}
		content = append(content, block...)
		return nil
	})
	if err != nil || !bytes.Equal(content, want) {
		t.Fatalf("Read() read %d bytes, error = %v; want %d matching bytes", len(content), err, len(want))
	}
}

// fileReadFunc 用于逐次控制 Reader 的数据和错误，覆盖文件读取以外的合法返回组合。
type fileReadFunc func([]byte) (int, error)

func (f fileReadFunc) Read(p []byte) (int, error) { return f(p) }

func TestReadTransientEmptyRead(t *testing.T) {
	// 首次空读不代表 EOF；下次同时返回最后一块数据和 EOF。
	reads := 0
	reader := fileReadFunc(func(p []byte) (int, error) {
		reads++
		if reads == 1 {
			return 0, nil
		}
		return copy(p, "payload"), io.EOF
	})
	var content strings.Builder
	err := utils.Read(reader, func(size int, data []byte) error {
		if size != len(data) {
			t.Fatalf("Read() block size = %d, want %d", size, len(data))
		}
		content.Write(data)
		return nil
	})
	if err != nil || content.String() != "payload" || reads != 2 {
		t.Fatalf("Read() = (%q, %v), reads = %d; want (payload, nil), 2", content.String(), err, reads)
	}
}

func TestReadDataWithError(t *testing.T) {
	// Reader 可以在同一次调用中返回有效数据和错误；数据先交给回调，再传播错误。
	readErr := errors.New("read failed")
	for _, handlerErr := range []error{nil, utils.DONE, errors.New("handler failed")} {
		t.Run(fmt.Sprint(handlerErr), func(t *testing.T) {
			reader := fileReadFunc(func(p []byte) (int, error) {
				return copy(p, "payload"), readErr
			})
			var content string
			err := utils.Read(reader, func(_ int, data []byte) error {
				content = string(data)
				return handlerErr
			})
			wantErr := readErr
			if handlerErr != nil {
				wantErr = handlerErr
			}
			if handlerErr == utils.DONE {
				wantErr = nil
			}
			if content != "payload" || !errors.Is(err, wantErr) {
				t.Fatalf("Read() = (%q, %v), want (payload, %v)", content, err, wantErr)
			}
		})
	}
}

// go test -bench=Read$ -run ^$  -count 5 -benchmem
func BenchmarkRead(b *testing.B) {
	// 每次操作包含打开、读到 EOF 和关闭，回调不额外复制正文。
	for b.Loop() {
		file, err := os.Open("file.go")
		if err != nil {
			b.Fatal(err)
		}
		readErr := utils.Read(file, func(int, []byte) error { return nil })
		closeErr := file.Close()
		if readErr != nil {
			b.Fatal(readErr)
		}
		if closeErr != nil {
			b.Fatal(closeErr)
		}
	}
}

func TestWrite(t *testing.T) {
	type args struct {
		fileName string
		perm     os.FileMode
		isAppend bool
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{fileName: "test.log", perm: 0744, isAppend: false}},
		{name: "002", args: args{fileName: "test2.log", perm: 0744, isAppend: true}},
		{name: "003", args: args{fileName: "test/test/test.log", perm: 0711, isAppend: false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 各用例独占目录，追加与多级目录测试不继承其他运行留下的数据。
			fileName := filepath.Join(t.TempDir(), tt.args.fileName)
			w, err := utils.NewWrite(fileName, utils.WithWriteAppend(tt.args.isAppend), utils.WithWritePerm(tt.args.perm))
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWrite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer func() {
				if err := w.Close(); err != nil {
					t.Errorf("Close() WrapError %v", err)
				}
			}()

			var group sync.WaitGroup
			for i := 1; i <= 30; i++ {
				group.Go(func() {
					if i%3 == 0 {
						for j := range 10 {
							_, err := w.Write(fmt.Appendf(nil, "Write %d-%d Name %v; 太液仙舟迥，西园引上才。未晓征车度，鸡鸣关早开。\n", i, j, tt.name))
							if err != nil {
								t.Errorf("Write() error = %v", err)
								return
							}
						}
					} else if i%3 == 1 {
						for j := range 10 {
							_, err := w.WriteString(fmt.Sprintf("WriteString %d-%d Name %v; 隔户杨柳弱袅袅，恰似十五女儿腰。谁谓朝来不作意，狂风挽断最长条。\n", i, j, tt.name))
							if err != nil {
								t.Errorf("WriteString() error = %v", err)
								return
							}
						}

					} else {
						_, err := w.WriteBuf(func(write *bufio.Writer) (int, error) {
							for j := range 10000 {
								_, err := write.WriteString(fmt.Sprintf("WriteBuf %d-%d Name %v; 红酥肯放琼苞碎。探著南枝开遍未。不知酝藉几多香，但见包藏无限意。道人憔悴春窗底。闷损阑干愁不倚。要来小酌便来休，未必明朝风不起。\n", i, j, tt.name))
								if err != nil {
									return 0, err
								}
							}
							return 0, nil
						})
						if err != nil {
							t.Errorf("WriteBuf() error = %v", err)
							return
						}
					}
				})
			}
			group.Wait()

			file, err := os.Open(fileName)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			scanner := bufio.NewScanner(file)
			lines := 0
			for scanner.Scan() {
				lines++
			}
			if err := scanner.Err(); err != nil {
				t.Fatal(err)
			}
			// 每种写入方式各有 10 个 worker，全部返回后不应丢失任何一行。
			if want := 10 * (10 + 10 + 10000); lines != want {
				t.Fatalf("written lines = %d, want %d", lines, want)
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name string
		size int64
		want string
	}{
		{name: "bytes", size: 500, want: "500B"},
		{name: "zero", size: 0, want: "0B"},
		{name: "1KB", size: 1024, want: "1.0000K"},
		{name: "1MB", size: 1024 * 1024, want: "1.0000M"},
		{name: "1GB", size: 1024 * 1024 * 1024, want: "1.0000G"},
		{name: "1.5KB", size: 1536, want: "1.5000K"},
		{name: "mixed", size: 2560, want: "2.5000K"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.FormatFileSize(tt.size, 4)
			if got != tt.want {
				t.Errorf("FormatFileSize(%d) = %v, want %v", tt.size, got, tt.want)
			}
		})
	}
}

func TestFileType(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{name: "go_file", filePath: "./file.go", wantErr: false},
		{name: "mod_file", filePath: "./go.mod", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.filePath)
			if err != nil {
				t.Fatalf("os.Open() error = %v", err)
			}
			defer f.Close()

			ctype, err := utils.FileType(f)
			if (err != nil) != tt.wantErr {
				t.Errorf("FileType() error = %v, wantErr %v", err, tt.wantErr)
			}
			if ctype == "" {
				t.Error("FileType() returned empty string")
			}
		})
	}
}

// TestWriteBufReturnsFlushError 区分缓冲写入成功与最终 Flush 失败。
func TestWriteBufReturnsFlushError(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "flush.log")
	if err := os.WriteFile(fileName, nil, 0600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	w := &utils.WriteFile{File: file}
	defer w.Close()

	// 短内容只进入缓冲区，回调成功后 Flush 才会触达只读文件。
	const content = "flush error test"
	called := false
	n, err := w.WriteBuf(func(write *bufio.Writer) (int, error) {
		called = true
		n, writeErr := write.WriteString(content)
		if writeErr != nil {
			t.Fatalf("buffered WriteString() error = %v", writeErr)
		}
		return n, nil
	})
	if !called || n != len(content) {
		t.Fatalf("WriteBuf() called = %v, n = %d, want true, %d", called, n, len(content))
	}
	if pathErr, ok := errors.AsType[*os.PathError](err); !ok || pathErr.Op != "write" || pathErr.Path != fileName {
		t.Fatalf("WriteBuf() error = %v, want write PathError for %q", err, fileName)
	}
}

// TestWriteFileRejectsClosedWrites 验证三种写入入口均拒绝关闭句柄，不执行用户回调。
func TestWriteFileRejectsClosedWrites(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "closed.log")
	w, err := utils.NewWrite(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	called := false
	for name, write := range map[string]func() (int, error){
		"Write":       func() (int, error) { return w.Write([]byte("closed")) },
		"WriteString": func() (int, error) { return w.WriteString("closed") },
		"WriteBuf": func() (int, error) {
			return w.WriteBuf(func(*bufio.Writer) (int, error) {
				called = true
				return 0, nil
			})
		},
	} {
		t.Run(name, func(t *testing.T) {
			n, err := write()
			if called || n != 0 || err == nil || err.Error() != "文件已关闭" {
				t.Fatalf("%s() called = %v, n = %d, error = %v, want false, 0, 文件已关闭", name, called, n, err)
			}
		})
	}
	if n, err := w.WriteBuf(nil); n != 0 || err == nil || err.Error() != "handler 不能为空" {
		t.Fatalf("WriteBuf(nil) = (%d, %v), want handler error before closed-file error", n, err)
	}
}

// TestWriteBufConcurrentClose 验证写回调与关闭并发完成后，正文仍完整保留。
func TestWriteBufConcurrentClose(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "concurrent-close.log")
	w, err := utils.NewWrite(fileName)
	if err != nil {
		t.Fatal(err)
	}

	const content = "in-flight write"
	started := make(chan struct{})
	allowReturn := make(chan struct{})
	writeDone := make(chan error, 1)
	closeDone := make(chan error, 1)

	go func() {
		_, err := w.WriteBuf(func(write *bufio.Writer) (int, error) {
			n, err := write.WriteString(content)
			close(started)
			<-allowReturn
			return n, err
		})
		writeDone <- err
	}()

	<-started

	go func() {
		closeDone <- w.Close()
	}()

	close(allowReturn)

	writeErr, closeErr := <-writeDone, <-closeDone
	if writeErr != nil || closeErr != nil {
		t.Fatalf("WriteBuf() error = %v, Close() error = %v", writeErr, closeErr)
	}
	got, err := os.ReadFile(fileName)
	if err != nil || string(got) != content {
		t.Fatalf("file after Close() = (%q, %v), want %q", got, err, content)
	}
}

func TestWriteFileAtomicReplacesWholeFile(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "atomic.txt")
	if err := os.WriteFile(fileName, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := utils.WriteFileAtomic(fileName, []byte("new-content"), 0644); err != nil {
		t.Fatalf("WriteFileAtomic() error = %v", err)
	}

	got, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-content" {
		t.Fatalf("file content = %q, want %q", got, "new-content")
	}
}

func TestWriteStringAtomicReplacesWholeFile(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "atomic-string.txt")
	if err := os.WriteFile(fileName, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := utils.WriteStringAtomic(fileName, "new-string", 0644); err != nil {
		t.Fatalf("WriteStringAtomic() error = %v", err)
	}

	got, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-string" {
		t.Fatalf("file content = %q, want %q", got, "new-string")
	}
}

func TestWriteFileAtomicRejectsSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	realFile := filepath.Join(dir, "real.txt")
	linkFile := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(realFile, []byte("real"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realFile, linkFile); err != nil {
		t.Fatal(err)
	}

	err := utils.WriteFileAtomic(linkFile, []byte("blocked"), 0644)
	if err == nil {
		t.Fatal("WriteFileAtomic() expected symlink target error")
	}

	got, readErr := os.ReadFile(realFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "real" {
		t.Fatalf("real file content changed: got %q want %q", got, "real")
	}
}

func TestWriteFileAtomicRejectsSymlinkDirectory(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(dir, "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}

	err := utils.WriteFileAtomic(filepath.Join(linkDir, "blocked.txt"), []byte("blocked"), 0644)
	if err == nil {
		t.Fatal("WriteFileAtomic() expected symlink directory error")
	}
	if utils.IsExist(filepath.Join(realDir, "blocked.txt")) {
		t.Fatal("WriteFileAtomic() should not write through symlink directory")
	}
}

func TestNewWriteRejectsSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	realFile := filepath.Join(dir, "real.log")
	linkFile := filepath.Join(dir, "link.log")
	if err := os.WriteFile(realFile, []byte("real"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realFile, linkFile); err != nil {
		t.Fatal(err)
	}

	w, err := utils.NewWrite(linkFile)
	if err == nil {
		_ = w.Close()
		t.Fatal("NewWrite() expected symlink target error")
	}
}

func TestNewWriteRejectsSymlinkDirectory(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(dir, "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}

	w, err := utils.NewWrite(filepath.Join(linkDir, "blocked.log"))
	if err == nil {
		_ = w.Close()
		t.Fatal("NewWrite() expected symlink directory error")
	}
	if utils.IsExist(filepath.Join(realDir, "blocked.log")) {
		t.Fatal("NewWrite() should not write through symlink directory")
	}
}

func TestWritePathsPreserveSpaces(t *testing.T) {
	// 空格属于文件系统路径本身，空值校验不能改写包含首尾空格的目录或文件名。
	dir := filepath.Join(t.TempDir(), " directory ")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	atomicPath := filepath.Join(dir, " atomic ")
	if err := utils.WriteFileAtomic(atomicPath, []byte("payload"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic() error = %v", err)
	}
	writePath := filepath.Join(dir, " writer ")
	writer, err := utils.NewWrite(writePath, utils.WithWritePerm(0o600))
	if err != nil {
		t.Fatalf("NewWrite() error = %v", err)
	}
	if _, err := writer.WriteString("payload"); err != nil {
		_ = writer.Close()
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{atomicPath, writePath} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != "payload" {
			t.Fatalf("ReadFile(%q) = (%q, %v), want payload", path, got, err)
		}
	}
}

// TestFileTypePreservesOffset 验证类型取自文件头，短文件和完整缓冲区读取都不影响调用方游标。
func TestFileTypePreservesOffset(t *testing.T) {
	var content bytes.Buffer
	if err := png.Encode(&content, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		fileName string // 未知扩展名触发读取；已知扩展名无需检查正文。
		content  []byte // PNG 尾部填充用于覆盖一次读满 512 字节的情况。
		want     string // 保持标准库 MIME 类型及 charset 格式。
	}{
		{name: "short PNG", fileName: "sample", content: content.Bytes(), want: "image/png"},
		{name: "long PNG", fileName: "sample.unknown", content: append(bytes.Clone(content.Bytes()), make([]byte, 512)...), want: "image/png"},
		{name: "known extension", fileName: "sample.txt", content: content.Bytes(), want: "text/plain; charset=utf-8"},
		{name: "empty", fileName: "sample", want: "text/plain; charset=utf-8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fileName := filepath.Join(t.TempDir(), tc.fileName)
			if err := os.WriteFile(fileName, tc.content, 0644); err != nil {
				t.Fatal(err)
			}
			f, err := os.Open(fileName)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			for _, offset := range []int64{0, int64(len(tc.content) / 2), int64(len(tc.content))} {
				if _, err := f.Seek(offset, io.SeekStart); err != nil {
					t.Fatal(err)
				}
				if got, err := utils.FileType(f); err != nil || got != tc.want {
					t.Errorf("FileType() at offset %d = (%q, %v), want (%q, nil)", offset, got, err, tc.want)
				}
				if got, err := f.Seek(0, io.SeekCurrent); err != nil || got != offset {
					t.Fatalf("file offset = %d, error = %v; want %d", got, err, offset)
				}
			}
		})
	}
}

// TestFileTypeClosedFile 保留已知扩展名无需读文件的行为，实际读取失败时返回原始错误链。
func TestFileTypeClosedFile(t *testing.T) {
	for _, fileName := range []string{"sample", "sample.png"} {
		t.Run(fileName, func(t *testing.T) {
			f, err := os.Create(filepath.Join(t.TempDir(), fileName))
			if err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			got, err := utils.FileType(f)
			if fileName == "sample.png" {
				if err != nil || got != "image/png" {
					t.Fatalf("FileType() = (%q, %v), want (image/png, nil)", got, err)
				}
			} else if !errors.Is(err, os.ErrClosed) || got != "" {
				t.Fatalf("FileType() = (%q, %v), want empty type and os.ErrClosed", got, err)
			}
		})
	}
}

// go test -bench=Write$ -run ^$  -count 5 -benchmem
func BenchmarkWrite(t *testing.B) {
	type args struct {
		fileName string
		perm     os.FileMode
		isAppend bool
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{fileName: "test3.log", perm: 0744, isAppend: false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.B) {
			fileName := filepath.Join(t.TempDir(), tt.args.fileName)
			w, err := utils.NewWrite(fileName, utils.WithWriteAppend(tt.args.isAppend), utils.WithWritePerm(tt.args.perm))
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWrite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer func() {
				if err := w.Close(); err != nil {
					t.Errorf("Close() WrapError %v", err)
				}
			}()

			// 每次迭代写入 10 行，WriteBuf 返回前完成刷新。
			for t.Loop() {
				_, err := w.WriteBuf(func(write *bufio.Writer) (int, error) {
					for range 10 {
						_, err := write.WriteString("红酥肯放琼苞碎。探著南枝开遍未。不知酝藉几多香，但见包藏无限意。道人憔悴春窗底。闷损阑干愁不倚。要来小酌便来休，未必明朝风不起。\n")
						if err != nil {
							return 0, err
						}
					}
					return 0, nil
				})
				if err != nil {
					t.Errorf("WriteBuf() error = %v", err)
					return
				}
			}
		})
	}
}
