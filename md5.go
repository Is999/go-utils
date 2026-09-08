package utils

import (
	"crypto/md5"
	"encoding/hex"
)

// MD5 返回 32 位小写十六进制摘要，仅用于旧协议或非安全哈希标识。
func MD5(str string) string {
	sum := md5.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}
