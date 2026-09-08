package utils

import (
	"crypto/cipher"
	"crypto/rand"
	"io"

	"github.com/Is999/go-utils/errors"
)

// EncryptGCM 返回由 nonce、密文和认证标签组成的独立切片，每次调用生成新的随机 nonce。
// additionalData 可为 nil，仅参与认证，不写入密文；GCM 不使用 Cipher 的 IV 配置。
func (c *Cipher) EncryptGCM(data, additionalData []byte) ([]byte, error) {
	if err := c.check(); err != nil {
		return nil, err
	}

	aead, err := c.newGCM()
	if err != nil {
		return nil, errors.Tag(err)
	}

	nonceSize := aead.NonceSize()
	// 预留 nonce 前缀及标签容量，Seal 直接追加密文和标签。
	out := make([]byte, nonceSize, nonceSize+len(data)+aead.Overhead())
	if _, err = io.ReadFull(rand.Reader, out); err != nil {
		return nil, errors.Tag(err)
	}

	return aead.Seal(out, out[:nonceSize], data, additionalData), nil
}

// DecryptGCM 读取密文头部 nonce，并用相同 additionalData 验证和解密；认证失败不返回明文。
func (c *Cipher) DecryptGCM(data, additionalData []byte) ([]byte, error) {
	if err := c.check(); err != nil {
		return nil, err
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
		// 认证失败不向调用方交付任何解密内容。
		return nil, errors.Tag(err)
	}
	return plaintext, nil
}

// EncryptGCMString 按 EncryptGCM 的格式加密字符串，再交由非 nil 的 encode 编码。
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

// DecryptGCMString 先由非 nil 的 decode 解码，再按 DecryptGCM 的格式认证和解密。
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

// newGCM 为已初始化的 Cipher 创建 AEAD，GCM 要求分组大小为 16 字节。
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
