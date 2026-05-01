package utils

import (
	"crypto/md5"
	"encoding/hex"
)

// Md5 返回字符串的 MD5 十六进制摘要。
//
// 安全说明：MD5 已不适合密码存储、签名、完整性安全校验等安全场景；
// 该函数仅用于历史协议兼容或非安全哈希标识。
func Md5(str string) string {
	sum := md5.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}
