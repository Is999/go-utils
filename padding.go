package utils

import "reflect"

var (
	// noPaddingFuncPtr 记录 NoPadding 函数指针，用于快速识别“流模式不填充”策略。
	noPaddingFuncPtr = reflect.ValueOf(NoPadding).Pointer()
	// noUnPaddingFuncPtr 记录 NoUnPadding 函数指针，用于快速识别“流模式不去填充”策略。
	noUnPaddingFuncPtr = reflect.ValueOf(NoUnPadding).Pointer()
)

// NoPadding 表示不做任何填充，适合 CTR/CFB/OFB/GCM 等流式或 AEAD 模式。
// 返回值会复制原始数据，避免调用方后续修改原切片影响加密流程。
func NoPadding(data []byte, _ int) []byte {
	return append([]byte(nil), data...)
}

// NoUnPadding 表示不做任何去填充，适合 CTR/CFB/OFB/GCM 等流式或 AEAD 模式。
// 返回值会复制原始数据，避免调用方后续修改结果影响后续处理。
func NoUnPadding(data []byte) ([]byte, error) {
	return append([]byte(nil), data...), nil
}

// isNoPaddingFunc 判断当前填充函数是否为 NoPadding。
func isNoPaddingFunc(padding Padding) bool {
	return padding != nil && reflect.ValueOf(padding).Pointer() == noPaddingFuncPtr
}

// isNoUnPaddingFunc 判断当前去填充函数是否为 NoUnPadding。
func isNoUnPaddingFunc(unPadding UnPadding) bool {
	return unPadding != nil && reflect.ValueOf(unPadding).Pointer() == noUnPaddingFuncPtr
}
