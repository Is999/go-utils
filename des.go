package utils

import (
	"crypto/des"
	"github.com/Is999/go-utils/errors"
)

// DES des加密解密
//
//	key 秘钥
//	isRandIV 随机生成IV: true 随机生成的IV会在加密后的密文开头
func DES(key string, isRandIV ...bool) (*Cipher, error) {
	switch len(key) {
	default:
		return nil, errors.Errorf("DES秘钥的长度只能是8字节，3DES秘钥的长度只能是24字节。当前预设置的秘钥[%s]长度: %d", key, len(key))
	case 24:
		return DES3(key, isRandIV...)
	case 8:
	}

	isRand := false
	if len(isRandIV) > 0 && isRandIV[0] {
		isRand = true
	}

	return NewCipher(key, des.NewCipher, isRand)
}

// DES3 des3加密解密
//
//	key 秘钥
//	isRandIV 随机生成IV: true 随机生成的IV会在加密后的密文开头
func DES3(key string, isRandIV ...bool) (*Cipher, error) {
	if len(key) != 24 {
		return nil, errors.Errorf("3DES秘钥的长度只能是24字节。当前预设置的秘钥[%s]长度: %d", key, len(key))
	}

	isRand := false
	if len(isRandIV) > 0 && isRandIV[0] {
		isRand = true
	}

	return NewCipher(key, des.NewTripleDESCipher, isRand)
}
