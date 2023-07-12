package utils

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
)

// Sha1 sha1加密
func Sha1(str string) string {
	sum := sha1.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}

// Sha256 sha256加密
func Sha256(str string) string {
	sum := sha256.Sum256([]byte(str))
	return hex.EncodeToString(sum[:])
}

// Sha512 sha512加密
func Sha512(str string) string {
	sum := sha512.Sum512([]byte(str))
	return hex.EncodeToString(sum[:])
}
