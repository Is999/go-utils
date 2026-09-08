package utils_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Is999/go-utils"
)

func TestGetRuntimeInfo(t *testing.T) {
	info := utils.RuntimeInfo(0)
	if info.Func != utils.GetFunctionName(utils.RuntimeInfo) || filepath.Base(info.File) != "runtime.go" || info.Line <= 0 {
		t.Fatalf("RuntimeInfo(0) = %+v, want its own source frame", info)
	}

	// 相邻调用计算预期行号，避免硬编码文件绝对路径或源码行号。
	pc, file, line, ok := runtime.Caller(0)
	info = utils.RuntimeInfo(1)
	if !ok {
		t.Fatal("runtime.Caller() did not locate the test frame")
	}
	if info.Func != runtime.FuncForPC(pc).Name() || info.File != file || info.Line != line+1 {
		t.Fatalf("RuntimeInfo(1) = %+v, want caller at %s:%d", info, file, line+1)
	}
}

func TestGetFunctionName(t *testing.T) {
	t.Run("named_function", func(t *testing.T) {
		name := utils.GetFunctionName(utils.MD5)
		if !strings.Contains(name, "MD5") {
			t.Errorf("GetFunctionName() = %v, want contains 'MD5'", name)
		}
	})

	t.Run("anonymous_function", func(t *testing.T) {
		fn := func() {}
		name := utils.GetFunctionName(fn)
		if name == "" || name == "Unknown Function" {
			t.Errorf("GetFunctionName() = %v, want a valid name", name)
		}
	})

	t.Run("invalid_input", func(t *testing.T) {
		for _, input := range []any{nil, 123} {
			if name := utils.GetFunctionName(input); name != "Unknown Function" {
				t.Errorf("GetFunctionName(%v) = %v, want Unknown Function", input, name)
			}
		}
	})
}

func TestRuntimeInfo_InvalidSkip(t *testing.T) {
	info := utils.RuntimeInfo(999)
	if info.File != "Unknown File" {
		t.Errorf("RuntimeInfo(999).File = %v, want 'Unknown File'", info.File)
	}
	if info.Line != 0 {
		t.Errorf("RuntimeInfo(999).Line = %v, want 0", info.Line)
	}
}
