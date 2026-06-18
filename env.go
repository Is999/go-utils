package utils

import (
	"os"
)

// - 获取环境变量 os.Getenv(key)
// - 设置环境变量 os.Setenv(key, value)
// - 删除环境变量 os.Unsetenv(key)

// GetEnv 获取环境变量值。
// 如果环境变量未设置或值为空，且提供了默认值参数，则返回默认值。
func GetEnv(key string, defaultVal ...string) string {
	val := os.Getenv(key)
	if val == "" && len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return val
}
