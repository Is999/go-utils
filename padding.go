package utils

import "reflect"

var (
	// noPaddingFuncPtr 用函数身份识别内置 NoPad，保留自定义回调的执行。
	noPaddingFuncPtr = reflect.ValueOf(NoPad).Pointer()
	// noUnpadFuncPtr 用函数身份识别内置 NoUnpad，保留自定义回调的执行。
	noUnpadFuncPtr = reflect.ValueOf(NoUnpad).Pointer()
)

// NoPad 返回 data 的独立副本，忽略分组大小；空输入返回 nil。
func NoPad(data []byte, _ int) []byte {
	return append([]byte(nil), data...)
}

// NoUnpad 返回 data 的独立副本且错误始终为 nil；空输入返回 nil 切片。
func NoUnpad(data []byte) ([]byte, error) {
	return append([]byte(nil), data...), nil
}

// isNoPadFunc 判断当前填充函数是否为 NoPad。
func isNoPadFunc(padding Pad) bool {
	return padding != nil && reflect.ValueOf(padding).Pointer() == noPaddingFuncPtr
}

// isNoUnpadFunc 判断当前去填充函数是否为 NoUnpad。
func isNoUnpadFunc(unpad Unpad) bool {
	return unpad != nil && reflect.ValueOf(unpad).Pointer() == noUnpadFuncPtr
}
