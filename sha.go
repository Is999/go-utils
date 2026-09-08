package utils

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
)

// SHA1 返回 40 位小写十六进制摘要，仅用于旧协议或非安全哈希标识。
func SHA1(str string) string {
	sum := sha1.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}

// SHA256 返回 64 位小写十六进制摘要，不进行加盐或密钥派生。
func SHA256(str string) string {
	sum := sha256.Sum256([]byte(str))
	return hex.EncodeToString(sum[:])
}

// SHA512 返回 128 位小写十六进制摘要，不进行加盐或密钥派生。
func SHA512(str string) string {
	sum := sha512.Sum512([]byte(str))
	return hex.EncodeToString(sum[:])
}
