package utils_test

import (
	"github.com/Is999/go-utils"
	"testing"
)

func TestGetEnv(t *testing.T) {
	type args struct {
		key        string
		defaultVal []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "001", args: args{key: "GOHOSTARCH", defaultVal: []string{"amd64"}}, want: "amd64"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.GetEnv(tt.args.key, tt.args.defaultVal...); got != tt.want {
				t.Errorf("GetEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
