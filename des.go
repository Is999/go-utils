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
		return nil, errors.Errorf("DES秘钥的长度只能是8字节，3DES秘钥的长度只能是24字节。当前预设置的秘钥[%s]长度: %d", key, len(key))
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
		return nil, errors.Errorf("3DES秘钥的长度只能是24字节。当前预设置的秘钥[%s]长度: %d", key, len(key))
	}

	return NewCipher(key, des.NewTripleDESCipher, opts...)
}
