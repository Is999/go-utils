package utils

import (
	"reflect"
	"runtime"
)

type Frame struct {
	Func string //方法名
	File string //文件名
	Line int    //行号
}

// RuntimeInfo 获取运行时行号、方法名、文件地址
func RuntimeInfo(skip int) *Frame {
	info := new(Frame)
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		info.File = "Unknown File"
		info.Line = 0
		return info
	}

	fPC := runtime.FuncForPC(pc)

	info.File = file
	info.Line = line
	info.Func = fPC.Name()
	return info
}

// GetFunctionName 获取函数名（普通函数、结构体方法或匿名函数）
func GetFunctionName(i interface{}) string {
	pc := reflect.ValueOf(i).Pointer()
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "Unknown Function"
	}
	return fn.Name()
}
