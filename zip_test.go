package utils_test

import (
	"archive/zip"
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
