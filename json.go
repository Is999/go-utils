package utils

// Marshal 对数据进行 JSON 编码
func Marshal(v any) ([]byte, error) {
	return currentConfig().json.encode(v)
}

// Unmarshal 对数据进行 JSON 解码
func Unmarshal(data []byte, v any) error {
	return currentConfig().json.decode(data, v)
}
