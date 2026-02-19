package utils_test

import (
	"testing"

	"github.com/Is999/go-utils"
)

func TestTar(t *testing.T) {
	type args struct {
		zipFiles    []string
		zipFileName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{zipFiles: []string{"./README.md", "./"}, zipFileName: "/tmp/go-utils.tar"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.Tar(tt.args.zipFileName, tt.args.zipFiles); (err != nil) != tt.wantErr {
				t.Errorf("Tar() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTarGz(t *testing.T) {
	type args struct {
		zipFiles    []string
		zipFileName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{zipFiles: []string{"./README.md", "./"}, zipFileName: "/tmp/go-utils.tar.gz"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.TarGz(tt.args.zipFileName, tt.args.zipFiles); (err != nil) != tt.wantErr {
				t.Errorf("TarGz() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnTar(t *testing.T) {
	type args struct {
		zipFile string
		destDir string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{zipFile: "/tmp/go-utils.tar", destDir: "/tmp/tar/go-utils"}, wantErr: false},
		{name: "002", args: args{zipFile: "/tmp/go-utils.tar.gz", destDir: "/tmp/targz/go-utils"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.UnTar(tt.args.zipFile, tt.args.destDir); (err != nil) != tt.wantErr {
				t.Errorf("UnTar() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
