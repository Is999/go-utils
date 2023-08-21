package utils

import (
	"bytes"
	"github.com/Is999/go-utils/errors"
)

// ZeroPadding 填充
func ZeroPadding(data []byte, blockSize int) []byte {
	//判断缺少几位长度。最少1，最多 blockSize
	padding := blockSize - len(data)%blockSize
	//补足位数。把切片[]byte{byte(0)}复制padding个
	padText := bytes.Repeat([]byte{0}, padding)
	return append(data, padText...)
}

// ZeroUnPadding 填充的反向操作
func ZeroUnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("ZeroUnPadding() data 参数长度必须大于0！")
	}

	// 去除填充
	return bytes.TrimRightFunc(data, func(r rune) bool {
		return r == rune(0)
	}), nil
}
