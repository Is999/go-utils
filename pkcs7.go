package utils

import "github.com/Is999/go-utils/errors"

// PKCS7Pad 返回独立的填充结果，已对齐的数据仍追加一整块。
// blockSize 有效范围为 1～255；超出范围时仅复制 data。
func PKCS7Pad(data []byte, blockSize int) []byte {
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

// PKCS7Unpad 校验并去除尾部填充值，成功结果与 data 共享底层数组；空输入报错。
func PKCS7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("PKCS7Unpad() data 参数长度必须大于 0")
	}

	padding := int(data[length-1])
	if padding == 0 || padding > length {
		return nil, errors.New("PKCS7Unpad() padding 长度异常")
	}
	// PKCS#7 要求每个填充字节都等于填充长度，不能只检查末字节。
	for i := length - padding; i < length; i++ {
		if int(data[i]) != padding {
			return nil, errors.New("PKCS7Unpad() padding 内容异常")
		}
	}
	return data[:length-padding], nil
}
