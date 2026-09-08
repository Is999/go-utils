package utils

// Marshal 使用全局编码器生成 JSON，默认采用 encoding/json，错误保持编码器原样。
func Marshal(v any) ([]byte, error) {
	return configValue.Load().json.encode(v)
}

// Unmarshal 使用全局解码器写入 v；目标类型与错误语义由所配置的解码器决定。
func Unmarshal(data []byte, v any) error {
	return configValue.Load().json.decode(data, v)
}
