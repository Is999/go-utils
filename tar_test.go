package utils_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Is999/go-utils"
)

func TestTar(t *testing.T) {
	srcDir := createArchiveFixture(t)
	tarFile := filepath.Join(t.TempDir(), "go-utils.tar")
	if err := utils.Tar(tarFile, []string{srcDir}); err != nil {
		t.Fatal(err)
	}
}

func TestAddFileToTarHonorsBaseDirForDirectory(t *testing.T) {
	srcDir := createArchiveFixture(t)
	var buf bytes.Buffer
	writer := tar.NewWriter(&buf)
	if err := utils.AddFileToTar(writer, srcDir, "root"); err != nil {
		t.Fatalf("AddFileToTar() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	reader := tar.NewReader(bytes.NewReader(buf.Bytes()))
	names := make(map[string]bool)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names[header.Name] = true
	}
	want := filepath.ToSlash(filepath.Join("root", filepath.Base(srcDir), "README.md"))
	if !names[want] {
		t.Fatalf("tar entries missing %q, got %#v", want, names)
	}
}

func TestTarGz(t *testing.T) {
	srcDir := createArchiveFixture(t)
	tarGzFile := filepath.Join(t.TempDir(), "go-utils.tar.gz")
	if err := utils.TarGz(tarGzFile, []string{srcDir}); err != nil {
		t.Fatal(err)
	}
}

func TestUnTarChecksGzipTrailer(t *testing.T) {
	// tar 的结束块可能先于 gzip 尾部校验被读取，仍须报告损坏或缺失的 gzip 尾部。
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	valid := buf.Bytes()
	corrupt := bytes.Clone(valid)
	corrupt[len(corrupt)-8] ^= 1
	for _, tt := range []struct {
		name string
		data []byte
		want error
	}{
		{name: "valid", data: valid},
		{name: "checksum", data: corrupt, want: gzip.ErrChecksum},
		{name: "truncated", data: valid[:len(valid)-4], want: io.ErrUnexpectedEOF},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "fixture.tar.gz")
			if err := os.WriteFile(path, tt.data, 0o600); err != nil {
				t.Fatal(err)
			}
			err := utils.UnTar(path, t.TempDir())
			if !errors.Is(err, tt.want) {
				t.Fatalf("UnTar() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestUnTarGzipPaddingAndMultistream(t *testing.T) {
	// gzip 成员边界可以落在 tar 数据内部；结束块之后的正常填充也应通过校验。
	var archive bytes.Buffer
	tw := tar.NewWriter(&archive)
	const content = "payload"
	if err := tw.WriteHeader(&tar.Header{Name: "file.txt", Mode: 0o644, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	data := archive.Bytes()
	for _, tt := range []struct {
		name   string
		chunks [][]byte
	}{
		{name: "padding", chunks: [][]byte{append(bytes.Clone(data), make([]byte, 1024)...)}},
		{name: "split members", chunks: [][]byte{data[:600], data[600:]}},
		{name: "trailing member", chunks: [][]byte{data, make([]byte, 1024)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var compressed bytes.Buffer
			for _, chunk := range tt.chunks {
				gz := gzip.NewWriter(&compressed)
				if _, err := gz.Write(chunk); err != nil {
					t.Fatal(err)
				}
				if err := gz.Close(); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(t.TempDir(), "fixture.tar.gz")
			if err := os.WriteFile(path, compressed.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			dest := t.TempDir()
			if err := utils.UnTar(path, dest); err != nil {
				t.Fatalf("UnTar() error = %v", err)
			}
			got, err := os.ReadFile(filepath.Join(dest, "file.txt"))
			if err != nil || string(got) != content {
				t.Fatalf("extracted file = (%q, %v), want %q", got, err, content)
			}
		})
	}
}

func TestUnTar(t *testing.T) {
	srcDir := createArchiveFixture(t)
	tarPath := filepath.Join(t.TempDir(), "go-utils.tar")
	if err := utils.Tar(tarPath, []string{srcDir}); err != nil {
		t.Fatalf("Tar() error = %v", err)
	}
	tarGzPath := filepath.Join(t.TempDir(), "go-utils.tar.gz")
	if err := utils.TarGz(tarGzPath, []string{srcDir}); err != nil {
		t.Fatalf("TarGz() error = %v", err)
	}

	tests := []struct {
		name    string
		tarFile string
		destDir string
	}{
		{name: "tar", tarFile: tarPath, destDir: filepath.Join(t.TempDir(), "tar")},
		{name: "tar.gz", tarFile: tarGzPath, destDir: filepath.Join(t.TempDir(), "targz")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.UnTar(tt.tarFile, tt.destDir); err != nil {
				t.Fatal(err)
			}
			assertArchiveExtracted(t, tt.destDir, filepath.Base(srcDir))
		})
	}
}

func TestUnTarRejectsPathTraversal(t *testing.T) {
	tarPath := filepath.Join(t.TempDir(), "evil.tar")
	file, err := os.Create(tarPath)
	if err != nil {
		t.Fatal(err)
	}

	writer := tar.NewWriter(file)
	if err = writer.WriteHeader(&tar.Header{
		Name: "../escape.txt",
		Mode: 0644,
		Size: int64(len("evil")),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = writer.Write([]byte("evil")); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(t.TempDir(), "untar")
	err = utils.UnTar(tarPath, destDir)
	if err == nil {
		t.Fatal("UnTar() expected path traversal error")
	}
	if utils.IsExist(filepath.Join(filepath.Dir(destDir), "escape.txt")) {
		t.Fatal("UnTar() should not create files outside dest dir")
	}
}

func TestUnTarRestoresFileMode(t *testing.T) {
	tarPath := filepath.Join(t.TempDir(), "mode.tar")
	file, err := os.Create(tarPath)
	if err != nil {
		t.Fatal(err)
	}

	writer := tar.NewWriter(file)
	if err = writer.WriteHeader(&tar.Header{
		Name: "bin/app.sh",
		Mode: 0755,
		Size: int64(len("#!/bin/sh\n")),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = writer.Write([]byte("#!/bin/sh\n")); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(t.TempDir(), "untar")
	if err = utils.UnTar(tarPath, destDir); err != nil {
		t.Fatalf("UnTar() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(destDir, "bin", "app.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Fatalf("untar file mode = %v, want %v", info.Mode().Perm(), os.FileMode(0755))
	}
}

func TestTarRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	targetFile := filepath.Join(root, "target.txt")
	if err := os.WriteFile(targetFile, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(root, "target-link.txt")
	if err := os.Symlink(targetFile, linkPath); err != nil {
		t.Fatal(err)
	}

	tarPath := filepath.Join(t.TempDir(), "symlink.tar")
	if err := utils.Tar(tarPath, []string{linkPath}); err == nil {
		t.Fatal("Tar() expected symlink error")
	}
}

func TestTarRejectsOutputInsideSourceDir(t *testing.T) {
	srcDir := createArchiveFixture(t)
	tarPath := filepath.Join(srcDir, "self.tar")

	if err := utils.Tar(tarPath, []string{srcDir}); err == nil {
		t.Fatal("Tar() expected output-inside-source error")
	}
}

func TestUnTarRejectsSymlinkInDestinationPath(t *testing.T) {
	tarPath := filepath.Join(t.TempDir(), "symlink-dest.tar")
	file, err := os.Create(tarPath)
	if err != nil {
		t.Fatal(err)
	}

	writer := tar.NewWriter(file)
	if err = writer.WriteHeader(&tar.Header{
		Name: "bin/app.sh",
		Mode: 0644,
		Size: int64(len("#!/bin/sh\n")),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = writer.Write([]byte("#!/bin/sh\n")); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(t.TempDir(), "untar")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(destDir, "bin")); err != nil {
		t.Fatal(err)
	}

	err = utils.UnTar(tarPath, destDir)
	if err == nil {
		t.Fatal("UnTar() expected symlink destination error")
	}
	if utils.IsExist(filepath.Join(outside, "app.sh")) {
		t.Fatal("UnTar() should not write through destination symlink")
	}
}

func TestUnTarRejectsSymlinkDestinationRoot(t *testing.T) {
	tarPath := filepath.Join(t.TempDir(), "root-symlink.tar")
	file, err := os.Create(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := tar.NewWriter(file)
	if err = writer.WriteHeader(&tar.Header{
		Name: "app.sh",
		Mode: 0644,
		Size: int64(len("#!/bin/sh\n")),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = writer.Write([]byte("#!/bin/sh\n")); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}

	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	destLink := filepath.Join(t.TempDir(), "untar-link")
	if err := os.Symlink(outside, destLink); err != nil {
		t.Fatal(err)
	}

	err = utils.UnTar(tarPath, destLink)
	if err == nil {
		t.Fatal("UnTar() expected symlink destination root error")
	}
	if utils.IsExist(filepath.Join(outside, "app.sh")) {
		t.Fatal("UnTar() should not write through destination root symlink")
	}
}

func createArchiveFixture(t *testing.T) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), "fixture")
	if err := os.MkdirAll(filepath.Join(root, "conf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hello archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "conf", "app.yaml"), []byte("name: go-utils\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func assertArchiveExtracted(t *testing.T, destDir, rootName string) {
	t.Helper()

	// 三种归档格式都要恢复正文，只有文件存在不能证明复制完成。
	for path, want := range map[string]string{
		"README.md":     "hello archive",
		"conf/app.yaml": "name: go-utils\n",
	} {
		got, err := os.ReadFile(filepath.Join(destDir, rootName, filepath.FromSlash(path)))
		if err != nil || string(got) != want {
			t.Fatalf("extracted %s = (%q, %v), want %q", path, got, err, want)
		}
	}
}
