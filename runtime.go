package utils

import (
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
