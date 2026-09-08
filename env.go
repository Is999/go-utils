package utils

import "os"

// GetEnv 获取环境变量值。
// 未设置与已设为空字符串按同样方式处理；只使用 defaultVal 第一项。
func GetEnv(key string, defaultVal ...string) string {
	val := os.Getenv(key)
	if val == "" && len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return val
}
