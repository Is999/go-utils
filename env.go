package utils

import "os"

// GetEnv 获取环境变量值。
// 如果环境变量未设置或值为空，且提供了默认值参数，则返回默认值。
func GetEnv(key string, defaultVal ...string) string {
	val := os.Getenv(key)
	if val == "" && len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return val
}
