package utils

import (
	"crypto/des"

	"github.com/Is999/go-utils/errors"
)

// DES 按原始密钥字节创建加密器：8 字节使用 DES，24 字节使用 3DES。
// DES/3DES 仅适合旧协议兼容，新协议建议使用 AES-GCM。
func DES(key string, opts ...CipherOption) (*Cipher, error) {
	switch len(key) {
	case 8:
		return NewCipher(key, des.NewCipher, opts...)
	case 24:
		return NewCipher(key, des.NewTripleDESCipher, opts...)
	default:
		return nil, errors.Errorf("DES 密钥长度必须是 8 字节，3DES 密钥长度必须是 24 字节，当前长度: %d", len(key))
	}
}

// TripleDES 创建用于旧协议兼容的 3DES 加密器，key 须为 24 个原始字节。
func TripleDES(key string, opts ...CipherOption) (*Cipher, error) {
	if len(key) != 24 {
		return nil, errors.Errorf("3DES 密钥长度必须是 24 字节，当前长度: %d", len(key))
	}

	return NewCipher(key, des.NewTripleDESCipher, opts...)
}
