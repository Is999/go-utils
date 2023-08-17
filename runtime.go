package utils

import (
	"runtime"
)

type RuntimeInfo struct {
	Func string //方法名
	File string //文件名
	Line int    //行号
}

// GetRuntimeInfo 获取运行时行号、方法名、文件地址
func GetRuntimeInfo(skip int) *RuntimeInfo {
	info := new(RuntimeInfo)
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		info.File = "???"
		info.Line = 0
		return info
	}

	fPC := runtime.FuncForPC(pc)

	info.File = file
	info.Line = line
	info.Func = fPC.Name()
	return info
}
