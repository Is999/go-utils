package utils

import (
	"crypto/aes"

	"github.com/Is999/go-utils/errors"
)

// AES AES加密解密
//
//	key 秘钥
func AES(key string, opts ...CipherOption) (*Cipher, error) {
	switch len(key) {
	default:
		return nil, errors.Errorf("AES 密钥长度必须是 16、24 或 32 字节，当前长度: %d", len(key))
	case 16, 24, 32:
	}

	return NewCipher(key, aes.NewCipher, opts...)
}
