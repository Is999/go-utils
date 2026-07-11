package utils

import (
	"crypto/cipher"
	"crypto/rand"
	"io"

	"github.com/Is999/go-utils/errors"
)

// EncryptGCM 使用 GCM(AEAD) 模式加密。
//
// 安全说明：
//   - 该模式自带完整性校验，优先级高于 CBC/CTR/CFB/OFB 等非认证模式。
//   - 每次加密都会随机生成 nonce，并写入密文头部，解密时自动解析。
func (c *Cipher) EncryptGCM(data, additionalData []byte) ([]byte, error) {
	if err := c.check(); err != nil {
		return nil, errors.Tag(err)
	}

	aead, err := c.newGCM()
	if err != nil {
		return nil, errors.Tag(err)
	}

	nonceSize := aead.NonceSize()
	out := make([]byte, nonceSize, nonceSize+len(data)+aead.Overhead())
	if _, err = io.ReadFull(rand.Reader, out); err != nil {
		return nil, errors.Tag(err)
	}

	return aead.Seal(out, out[:nonceSize], data, additionalData), nil
}

// DecryptGCM 使用 GCM(AEAD) 模式解密。
func (c *Cipher) DecryptGCM(data, additionalData []byte) ([]byte, error) {
	if err := c.check(); err != nil {
		return nil, errors.Tag(err)
	}

	aead, err := c.newGCM()
	if err != nil {
		return nil, errors.Tag(err)
	}

	nonceSize := aead.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("GCM 密文太短，无法读取 nonce")
	}

	plaintext, err := aead.Open(nil, data[:nonceSize], data[nonceSize:], additionalData)
	if err != nil {
		return nil, errors.Tag(err)
	}
	return plaintext, nil
}

// EncryptGCMString 使用 GCM 模式加密字符串并编码输出。
func (c *Cipher) EncryptGCMString(data string, encode EncodeToString, additionalData []byte) (string, error) {
	if encode == nil {
		return "", errors.New("encode 不能为空")
	}
	encrypted, err := c.EncryptGCM([]byte(data), additionalData)
	if err != nil {
		return "", errors.Tag(err)
	}
	return encode(encrypted), nil
}

// DecryptGCMString 使用 GCM 模式解密编码后的密文字符串。
func (c *Cipher) DecryptGCMString(encrypt string, decode DecodeString, additionalData []byte) (string, error) {
	if decode == nil {
		return "", errors.New("decode 不能为空")
	}
	ciphertext, err := decode(encrypt)
	if err != nil {
		return "", errors.Tag(err)
	}
	decrypted, err := c.DecryptGCM(ciphertext, additionalData)
	if err != nil {
		return "", errors.Tag(err)
	}
	return string(decrypted), nil
}

// newGCM 创建 GCM AEAD 实例。
func (c *Cipher) newGCM() (cipher.AEAD, error) {
	if c.block.BlockSize() != 16 {
		return nil, errors.New("GCM 仅支持 16 字节分组算法，例如 AES")
	}
	aead, err := cipher.NewGCM(c.block)
	if err != nil {
		return nil, errors.Tag(err)
	}
	return aead, nil
}
