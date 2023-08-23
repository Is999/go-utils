package utils_test

import (
	"github.com/Is999/go-utils"
	"testing"
)

func TestGetRuntimeInfo(t *testing.T) {
	type args struct {
		skip int
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "001", args: args{0}},
		{name: "002", args: args{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utils.RuntimeInfo(tt.args.skip)
			//t.Logf("RuntimeInfo() = %+v", got)
		})
	}
}
