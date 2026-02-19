package utils_test

import (
	"strings"
	"testing"

	"github.com/Is999/go-utils"
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

func TestGetFunctionName(t *testing.T) {
	// 普通函数
	t.Run("named_function", func(t *testing.T) {
		name := utils.GetFunctionName(utils.Md5)
		if !strings.Contains(name, "Md5") {
			t.Errorf("GetFunctionName() = %v, want contains 'Md5'", name)
		}
	})

	// 匿名函数
	t.Run("anonymous_function", func(t *testing.T) {
		fn := func() {}
		name := utils.GetFunctionName(fn)
		if name == "" || name == "Unknown Function" {
			t.Errorf("GetFunctionName() = %v, want a valid name", name)
		}
	})
}

func TestRuntimeInfo_InvalidSkip(t *testing.T) {
	// 超大skip值导致Caller失败
	info := utils.RuntimeInfo(999)
	if info.File != "Unknown File" {
		t.Errorf("RuntimeInfo(999).File = %v, want 'Unknown File'", info.File)
	}
	if info.Line != 0 {
		t.Errorf("RuntimeInfo(999).Line = %v, want 0", info.Line)
	}
}
