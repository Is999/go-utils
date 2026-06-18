package utils

import "reflect"

// 填充函数指针用于快速识别无需填充的策略。
var (
	// noPaddingFuncPtr 记录 NoPad 函数指针，用于快速识别“流模式不填充”策略。
	noPaddingFuncPtr = reflect.ValueOf(NoPad).Pointer()
	// noUnpadFuncPtr 记录 NoUnpad 函数指针，用于快速识别“流模式不去填充”策略。
	noUnpadFuncPtr = reflect.ValueOf(NoUnpad).Pointer()
)

// NoPad 表示不做任何填充，适合 CTR/CFB/OFB/GCM 等流式或 AEAD 模式。
// 返回值会复制原始数据，避免调用方后续修改原切片影响加密流程。
func NoPad(data []byte, _ int) []byte {
	return append([]byte(nil), data...)
}

// NoUnpad 表示不做任何去填充，适合 CTR/CFB/OFB/GCM 等流式或 AEAD 模式。
// 返回值会复制原始数据，避免调用方后续修改结果影响后续处理。
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
