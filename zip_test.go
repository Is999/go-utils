package utils_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Is999/go-utils"
)

func TestZip(t *testing.T) {
	srcDir := createArchiveFixture(t)
	tests := []struct {
		name    string
		files   []string
		zipFile string
		wantErr bool
	}{
		{name: "001", files: []string{srcDir}, zipFile: filepath.Join(t.TempDir(), "go-utils.zip"), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.Zip(tt.zipFile, tt.files); (err != nil) != tt.wantErr {
				t.Errorf("Zip() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
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

func TestUnZip(t *testing.T) {
	srcDir := createArchiveFixture(t)
	zipPath := filepath.Join(t.TempDir(), "go-utils.zip")
	if err := utils.Zip(zipPath, []string{srcDir}); err != nil {
		t.Fatalf("Zip() error = %v", err)
	}

	tests := []struct {
		name    string
		zipFile string
		destDir string
		wantErr bool
	}{
		{name: "001", zipFile: zipPath, destDir: filepath.Join(t.TempDir(), "zip")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.UnZip(tt.zipFile, tt.destDir); (err != nil) != tt.wantErr {
				t.Errorf("UnZip() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertArchiveExtracted(t, tt.destDir, filepath.Base(srcDir))
		})
	}
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
