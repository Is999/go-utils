package utils_test

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Is999/go-utils"
)

func TestTar(t *testing.T) {
	srcDir := createArchiveFixture(t)
	tests := []struct {
		name    string
		files   []string
		tarFile string
		wantErr bool
	}{
		{name: "001", files: []string{srcDir}, tarFile: filepath.Join(t.TempDir(), "go-utils.tar"), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.Tar(tt.tarFile, tt.files); (err != nil) != tt.wantErr {
				t.Errorf("Tar() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
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
	tests := []struct {
		name      string
		files     []string
		tarGzFile string
		wantErr   bool
	}{
		{name: "001", files: []string{srcDir}, tarGzFile: filepath.Join(t.TempDir(), "go-utils.tar.gz"), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.TarGz(tt.tarGzFile, tt.files); (err != nil) != tt.wantErr {
				t.Errorf("TarGz() error = %v, wantErr %v", err, tt.wantErr)
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
		zipFile string
		destDir string
		wantErr bool
	}{
		{name: "001", zipFile: tarPath, destDir: filepath.Join(t.TempDir(), "tar")},
		{name: "002", zipFile: tarGzPath, destDir: filepath.Join(t.TempDir(), "targz")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.UnTar(tt.zipFile, tt.destDir); (err != nil) != tt.wantErr {
				t.Errorf("UnTar() error = %v, wantErr %v", err, tt.wantErr)
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

	if !utils.IsExist(filepath.Join(destDir, rootName, "README.md")) {
		t.Fatalf("missing extracted file: %s", filepath.Join(destDir, rootName, "README.md"))
	}
	if !utils.IsExist(filepath.Join(destDir, rootName, "conf", "app.yaml")) {
		t.Fatalf("missing extracted file: %s", filepath.Join(destDir, rootName, "conf", "app.yaml"))
	}
}
