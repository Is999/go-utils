package utils

import "github.com/Is999/go-utils/errors"

// ZeroPadding 使用 0 字节填充到分组大小的整数倍。
//
// 注意：ZeroPadding 无法区分明文末尾真实的 0 字节和填充字节，协议允许时优先使用 PKCS#7。
func ZeroPadding(data []byte, blockSize int) []byte {
	if blockSize <= 0 {
		return append([]byte(nil), data...)
	}
	padding := blockSize - len(data)%blockSize
	out := make([]byte, len(data)+padding)
	copy(out, data)
	return out
}

// ZeroUnPadding 去除尾部 0 字节填充。
func ZeroUnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("ZeroUnPadding() data 参数长度必须大于 0")
	}
	end := length
	for end > 0 && data[end-1] == 0 {
		end--
	}
	return data[:end], nil
}
