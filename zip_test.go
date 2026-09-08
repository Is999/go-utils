package utils_test

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Is999/go-utils"
)

func TestZip(t *testing.T) {
	srcDir := createArchiveFixture(t)
	zipFile := filepath.Join(t.TempDir(), "go-utils.zip")
	if err := utils.Zip(zipFile, []string{srcDir}); err != nil {
		t.Fatal(err)
	}
}

func TestAddFileToZipHonorsBaseDirForDirectory(t *testing.T) {
	srcDir := createArchiveFixture(t)
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	if err := utils.AddFileToZip(writer, srcDir, "root"); err != nil {
		t.Fatalf("AddFileToZip() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool, len(reader.File))
	for _, file := range reader.File {
		names[file.Name] = true
	}
	want := filepath.ToSlash(filepath.Join("root", filepath.Base(srcDir), "README.md"))
	if !names[want] {
		t.Fatalf("zip entries missing %q, got %#v", want, names)
	}
}

func TestZipDirectoriesWorkWithStandardFS(t *testing.T) {
	// 标准库的 ZIP 文件系统按名称末尾的 / 识别目录，空目录也必须可枚举。
	source := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(filepath.Join(source, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "file.txt"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	if err := utils.AddFileToZip(writer, source, "root"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := fs.ReadDir(reader, "root/source")
	if err != nil {
		t.Fatalf("ReadDir(source) error = %v", err)
	}
	if len(entries) != 2 || entries[0].Name() != "empty" || !entries[0].IsDir() || entries[1].Name() != "file.txt" {
		t.Fatalf("ReadDir(source) = %v, want empty directory and file.txt", entries)
	}
	entries, err = fs.ReadDir(reader, "root/source/empty")
	if err != nil || len(entries) != 0 {
		t.Fatalf("ReadDir(empty) = (%v, %v), want empty directory", entries, err)
	}
	data, err := fs.ReadFile(reader, "root/source/file.txt")
	if err != nil || string(data) != "payload" {
		t.Fatalf("ReadFile() = (%q, %v), want payload", data, err)
	}
}

func TestArchivePathsPreserveSpaces(t *testing.T) {
	// 打包输入、输出目录和解包目录都保留原始名称，避免校验与实际 I/O 使用不同路径。
	root := t.TempDir()
	source := filepath.Join(root, " source ")
	if err := os.MkdirAll(filepath.Join(source, " empty "), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, " file "), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		suffix string
		pack   func(string, []string) error
		unpack func(string, string) error
	}{
		{suffix: ".zip", pack: utils.Zip, unpack: utils.UnZip},
		{suffix: ".tar", pack: utils.Tar, unpack: utils.UnTar},
		{suffix: ".tar.gz", pack: utils.TarGz, unpack: utils.UnTar},
	} {
		t.Run(tt.suffix, func(t *testing.T) {
			outputDir := filepath.Join(t.TempDir(), " archives ")
			if err := os.MkdirAll(outputDir, 0o755); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(outputDir, "bundle"+tt.suffix)
			if err := tt.pack(output, []string{source}); err != nil {
				t.Fatalf("pack() error = %v", err)
			}
			dest := filepath.Join(t.TempDir(), " extracted ")
			if err := tt.unpack(output, dest); err != nil {
				t.Fatalf("unpack() error = %v", err)
			}
			got, err := os.ReadFile(filepath.Join(dest, " source ", " file "))
			if err != nil || string(got) != "payload" {
				t.Fatalf("extracted file = (%q, %v), want payload", got, err)
			}
			if !utils.IsDir(filepath.Join(dest, " source ", " empty ")) {
				t.Fatal("empty directory was not preserved")
			}
			// 输出名的后缀判断仍允许末尾空白，但落盘时必须使用完整原始文件名。
			if err := tt.pack(output+" ", []string{source}); err != nil {
				t.Fatalf("pack(output with trailing space) error = %v", err)
			}
			if _, err := os.Stat(output + " "); err != nil {
				t.Fatal(err)
			}
			// 移除无空白的同名归档，避免误打开裁剪后的路径也能通过回归。
			if err := os.Remove(output); err != nil {
				t.Fatal(err)
			}
			// 同一公开库产出的归档应可直接解包，格式判定不得改写实际打开的文件名。
			spaceDest := t.TempDir()
			if err := tt.unpack(output+" ", spaceDest); err != nil {
				t.Fatalf("unpack(output with trailing space) error = %v", err)
			}
			got, err = os.ReadFile(filepath.Join(spaceDest, " source ", " file "))
			if err != nil || string(got) != "payload" {
				t.Fatalf("extracted trailing-space archive = (%q, %v), want payload", got, err)
			}
			// 同时存在裁剪后的目录时，仍须按原输入目录拒绝把输出归档写入自身。
			if err := os.MkdirAll(strings.TrimSpace(source), 0o755); err != nil {
				t.Fatal(err)
			}
			err = tt.pack(filepath.Join(source, "inside"+tt.suffix), []string{source})
			if err == nil || !strings.Contains(err.Error(), "输出文件不能位于待打包目录内") {
				t.Fatalf("pack(output inside source) error = %v", err)
			}
		})
	}
}

func TestUnZip(t *testing.T) {
	srcDir := createArchiveFixture(t)
	zipPath := filepath.Join(t.TempDir(), "go-utils.zip")
	if err := utils.Zip(zipPath, []string{srcDir}); err != nil {
		t.Fatalf("Zip() error = %v", err)
	}

	destDir := filepath.Join(t.TempDir(), "zip")
	if err := utils.UnZip(zipPath, destDir); err != nil {
		t.Fatal(err)
	}
	assertArchiveExtracted(t, destDir, filepath.Base(srcDir))
}

func TestUnZipRejectsPathTraversal(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "evil.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}

	writer := zip.NewWriter(file)
	entryWriter, err := writer.Create("../escape.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = entryWriter.Write([]byte("evil")); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(t.TempDir(), "unzip")
	err = utils.UnZip(zipPath, destDir)
	if err == nil {
		t.Fatal("UnZip() expected path traversal error")
	}
	if utils.IsExist(filepath.Join(filepath.Dir(destDir), "escape.txt")) {
		t.Fatal("UnZip() should not create files outside dest dir")
	}
}

func TestUnZipRestoresFileMode(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "mode.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}

	writer := zip.NewWriter(file)
	header := &zip.FileHeader{
		Name:   "bin/app.sh",
		Method: zip.Deflate,
	}
	header.SetMode(0o755)
	entryWriter, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = entryWriter.Write([]byte("#!/bin/sh\n")); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(t.TempDir(), "unzip")
	if err = utils.UnZip(zipPath, destDir); err != nil {
		t.Fatalf("UnZip() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(destDir, "bin", "app.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("unzip file mode = %v, want %v", info.Mode().Perm(), os.FileMode(0o755))
	}
}

func TestZipRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	targetFile := filepath.Join(root, "target.txt")
	if err := os.WriteFile(targetFile, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(root, "target-link.txt")
	if err := os.Symlink(targetFile, linkPath); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(t.TempDir(), "symlink.zip")
	if err := utils.Zip(zipPath, []string{linkPath}); err == nil {
		t.Fatal("Zip() expected symlink error")
	}
}

func TestZipRejectsOutputInsideSourceDir(t *testing.T) {
	srcDir := createArchiveFixture(t)
	zipPath := filepath.Join(srcDir, "self.zip")

	if err := utils.Zip(zipPath, []string{srcDir}); err == nil {
		t.Fatal("Zip() expected output-inside-source error")
	}
}

func TestUnZipRejectsSymlinkInDestinationPath(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "symlink-dest.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entryWriter, err := writer.Create("bin/app.sh")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = entryWriter.Write([]byte("#!/bin/sh\n")); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(t.TempDir(), "unzip")
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

	err = utils.UnZip(zipPath, destDir)
	if err == nil {
		t.Fatal("UnZip() expected symlink destination error")
	}
	if utils.IsExist(filepath.Join(outside, "app.sh")) {
		t.Fatal("UnZip() should not write through destination symlink")
	}
}

func TestUnZipRejectsSymlinkDestinationRoot(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "root-symlink.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entryWriter, err := writer.Create("app.sh")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = entryWriter.Write([]byte("#!/bin/sh\n")); err != nil {
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
	destLink := filepath.Join(t.TempDir(), "unzip-link")
	if err := os.Symlink(outside, destLink); err != nil {
		t.Fatal(err)
	}

	err = utils.UnZip(zipPath, destLink)
	if err == nil {
		t.Fatal("UnZip() expected symlink destination root error")
	}
	if utils.IsExist(filepath.Join(outside, "app.sh")) {
		t.Fatal("UnZip() should not write through destination root symlink")
	}
}

func TestUnZipRejectsSymlinkBeforeCreatingNestedDir(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "symlink-nested.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entryWriter, err := writer.Create("bin/new/app.sh")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = entryWriter.Write([]byte("#!/bin/sh\n")); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(t.TempDir(), "unzip")
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

	err = utils.UnZip(zipPath, destDir)
	if err == nil {
		t.Fatal("UnZip() expected symlink destination error")
	}
	if utils.IsExist(filepath.Join(outside, "new")) {
		t.Fatal("UnZip() should reject symlink before creating nested directories outside")
	}
}
