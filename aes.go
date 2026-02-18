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
		return nil, errors.Errorf("AES秘钥的长度只能是16、24或32字节。当前预设置的秘钥[%s]长度: %d", key, len(key))
	case 16, 24, 32:
	}

	return NewCipher(key, aes.NewCipher, opts...)
}
