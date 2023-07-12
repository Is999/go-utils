package utils

import (
	"crypto/md5"
	"encoding/hex"
)

// Md5 md5加密
func Md5(str string) string {
	sum := md5.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}
