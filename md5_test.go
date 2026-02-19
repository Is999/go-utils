package utils_test

import (
	"testing"

	"github.com/Is999/go-utils"
)

func TestMd5(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "001", args: args{str: "ABC123&abc#"}, want: "4e2b0ebf5bdb830f2c6f2a6e057553c5"},
		{name: "002", args: args{str: "123456"}, want: "e10adc3949ba59abbe56e057f20f883e"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Md5(tt.args.str); got != tt.want {
				t.Errorf("Md5() = %v, want %v", got, tt.want)
			}
		})
	}
}
