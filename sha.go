package utils

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
)

// SHA1 返回字符串的 SHA-1 十六进制摘要。
//
// 安全说明：SHA-1 已不适合签名、证书、密码存储等安全场景；
// 该函数仅用于历史协议兼容或非安全哈希标识。
func SHA1(str string) string {
	sum := sha1.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}

// SHA256 返回字符串的 SHA-256 十六进制摘要。
// 适合普通摘要场景；密码存储仍应使用 bcrypt/argon2/scrypt 等专用算法。
func SHA256(str string) string {
	sum := sha256.Sum256([]byte(str))
	return hex.EncodeToString(sum[:])
}

// SHA512 返回字符串的 SHA-512 十六进制摘要。
// 适合普通摘要场景；密码存储仍应使用 bcrypt/argon2/scrypt 等专用算法。
func SHA512(str string) string {
	sum := sha512.Sum512([]byte(str))
	return hex.EncodeToString(sum[:])
}
