package utils

import (
	"crypto/aes"

	"github.com/Is999/go-utils/errors"
)

// AES 创建 AES 加密器，key 按原始字节使用，长度须为 16、24 或 32 字节。
func AES(key string, opts ...CipherOption) (*Cipher, error) {
	switch len(key) {
	case 16, 24, 32:
		return NewCipher(key, aes.NewCipher, opts...)
	default:
		return nil, errors.Errorf("AES 密钥长度必须是 16、24 或 32 字节，当前长度: %d", len(key))
	}
}
