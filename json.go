package utils

// Marshal 对数据进行 JSON 编码
func Marshal(v any) ([]byte, error) {
	return config.json.encode(v)
}

// Unmarshal 对数据进行 JSON 解码
func Unmarshal(data []byte, v any) error {
	return config.json.decode(data, v)
}
