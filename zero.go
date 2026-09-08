package utils

import "github.com/Is999/go-utils/errors"

// ZeroPad 返回独立的零填充结果，已对齐的数据仍追加一整块。
// blockSize 不大于 0 时仅复制 data；去填充时无法保留明文原有的尾部零字节。
func ZeroPad(data []byte, blockSize int) []byte {
	if blockSize <= 0 {
		return append([]byte(nil), data...)
	}
	padding := blockSize - len(data)%blockSize
	out := make([]byte, len(data)+padding)
	copy(out, data)
	return out
}

// ZeroUnpad 去除全部尾部零字节，结果与 data 共享底层数组；空输入报错。
func ZeroUnpad(data []byte) ([]byte, error) {
	end := len(data) // 去填充结果的排他结束索引。
	if end == 0 {
		return nil, errors.New("ZeroUnpad() data 参数长度必须大于 0")
	}
	for end > 0 && data[end-1] == 0 {
		end--
	}
	return data[:end], nil
}
