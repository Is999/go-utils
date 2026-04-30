package utils

import (
	"crypto/des"

	"github.com/Is999/go-utils/errors"
)

// DES des加密解密
//
//	key 秘钥
func DES(key string, opts ...CipherOption) (*Cipher, error) {
	switch len(key) {
	default:
		return nil, errors.Errorf("DES 密钥长度必须是 8 字节，3DES 密钥长度必须是 24 字节，当前长度: %d", len(key))
	case 24:
		return DES3(key, opts...)
	case 8:
	}

	return NewCipher(key, des.NewCipher, opts...)
}

// DES3 des3加密解密
//
//	key 秘钥
func DES3(key string, opts ...CipherOption) (*Cipher, error) {
	if len(key) != 24 {
		return nil, errors.Errorf("3DES 密钥长度必须是 24 字节，当前长度: %d", len(key))
	}

	return NewCipher(key, des.NewTripleDESCipher, opts...)
}
