package utils

import (
	"bytes"

	"github.com/Is999/go-utils/errors"
)

// Pkcs7Padding 填充
func Pkcs7Padding(data []byte, blockSize int) []byte {
	//判断缺少几位长度。最少1，最多 blockSize
	padding := blockSize - len(data)%blockSize
	//补足位数。把切片[]byte{byte(padding)}复制padding个
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// Pkcs7UnPadding 填充的反向操作
func Pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("Pkcs7UnPadding() data 参数长度必须大于0！")
	}
	//获取填充的个数
	unPadding := int(data[length-1])
	end := length - unPadding
	if end < 0 {
		return nil, errors.New("Pkcs7UnPadding() data 参数长度异常！")
	}
	return data[:end], nil
}
