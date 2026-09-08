package utils

import (
	"reflect"
	"runtime"
)

// Frame 保存调用栈中的函数、文件和行号信息。
type Frame struct {
	Func string // 包路径限定的函数名，保留匿名函数和方法的运行时后缀。
	File string // 编译器记录的源文件路径，受构建时 trimpath 设置影响。
	Line int    // 源文件行号，从 1 开始；未获取到时为 0。
}

// RuntimeInfo 获取指定栈帧；skip 为 0 指向本函数，为 1 指向调用方，失败时返回 Unknown 标记。
func RuntimeInfo(skip int) *Frame {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return &Frame{File: "Unknown File"}
	}

	fPC := runtime.FuncForPC(pc)
	if fPC == nil {
		return &Frame{Func: "Unknown Function"}
	}
	return &Frame{Func: fPC.Name(), File: file, Line: line}
}

// GetFunctionName 返回运行时函数名；nil、非函数或无法定位时返回 "Unknown Function"。
func GetFunctionName(i any) string {
	if i == nil {
		return "Unknown Function"
	}
	value := reflect.ValueOf(i)
	if value.Kind() != reflect.Func || value.IsNil() {
		return "Unknown Function"
	}
	pc := value.Pointer()
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "Unknown Function"
	}
	return fn.Name()
}
