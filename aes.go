package utils

import (
	"crypto/aes"
	"github.com/Is999/go-utils/errors"
)

// AES AES加密解密
//
//	key 秘钥
//	isRandIV 随机生成IV: true 随机生成的IV会在加密后的密文开头
func AES(key string, isRandIV ...bool) (*Cipher, error) {
	switch len(key) {
	default:
		return nil, errors.Errorf("AES秘钥的长度只能是16、24或32字节。当前预设置的秘钥[%s]长度: %d", key, len(key))
	case 16, 24, 32:
	}

	isRand := false
	if len(isRandIV) > 0 && isRandIV[0] {
		isRand = true
	}

	return NewCipher(key, aes.NewCipher, isRand)
}
