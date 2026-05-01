package utils

import (
	"crypto/des"

	"github.com/Is999/go-utils/errors"
)

// DES 创建 DES/3DES 分组密码封装。
//
// 安全说明：DES 密钥空间过小，3DES 也属于历史兼容算法；新系统应优先使用 AES-GCM。
// 当 key 长度为 24 字节时自动使用 3DES，长度为 8 字节时使用 DES。
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

// DES3 创建 3DES 分组密码封装。
//
// 安全说明：3DES 仅用于旧协议兼容，新系统应优先使用 AES-GCM。
//
//	key 秘钥
func DES3(key string, opts ...CipherOption) (*Cipher, error) {
	if len(key) != 24 {
		return nil, errors.Errorf("3DES 密钥长度必须是 24 字节，当前长度: %d", len(key))
	}

	return NewCipher(key, des.NewTripleDESCipher, opts...)
}
