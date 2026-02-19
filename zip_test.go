package utils_test

import (
	"testing"

	"github.com/Is999/go-utils"
)

func TestZip(t *testing.T) {
	type args struct {
		zipFiles    []string
		zipFileName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{zipFiles: []string{"./README.md", "./"}, zipFileName: "/tmp/go-utils.zip"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.Zip(tt.args.zipFileName, tt.args.zipFiles); (err != nil) != tt.wantErr {
				t.Errorf("Zip() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnZip(t *testing.T) {
	type args struct {
		zipFile string
		destDir string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{zipFile: "/tmp/go-utils.zip", destDir: "/tmp/zip/go-utils"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.UnZip(tt.args.zipFile, tt.args.destDir); (err != nil) != tt.wantErr {
				t.Errorf("UnZip() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
