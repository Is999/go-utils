package utils

import (
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
			GetRuntimeInfo(tt.args.skip)
			//t.Logf("GetRuntimeInfo() = %+v", got)
		})
	}
}
