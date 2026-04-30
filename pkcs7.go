package utils

import "github.com/Is999/go-utils/errors"

// Pkcs7Padding 按 PKCS#7 规则填充数据。
//
// 返回值总是新切片，避免 append 复用调用方底层数组导致原始数据被意外修改。
func Pkcs7Padding(data []byte, blockSize int) []byte {
	if blockSize <= 0 || blockSize > 255 {
		return append([]byte(nil), data...)
	}
	padding := blockSize - len(data)%blockSize
	out := make([]byte, len(data)+padding)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(padding)
	}
	return out
}

// Pkcs7UnPadding 去除 PKCS#7 填充。
//
// 会严格校验填充长度和每个填充字节，避免损坏密文被误当作合法明文。
func Pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("Pkcs7UnPadding() data 参数长度必须大于 0")
	}

	padding := int(data[length-1])
	if padding == 0 || padding > length {
		return nil, errors.New("Pkcs7UnPadding() padding 长度异常")
	}
	for i := length - padding; i < length; i++ {
		if int(data[i]) != padding {
			return nil, errors.New("Pkcs7UnPadding() padding 内容异常")
		}
	}
	return data[:length-padding], nil
}
