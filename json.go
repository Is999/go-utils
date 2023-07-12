package utils

import (
	"encoding/json"
	"sync"
)

var (
	setJsonOnce sync.Once
	// 对数据进行 JSON 编码
	jsonEncode func(v any) ([]byte, error)
	// 对数据进行 JSON 解码
	jsonDecode func(data []byte, v any) error
)

// SetJsonMethod 设置json编码解码方法(三方开源库)， 若未设置则默认使用encoding/json(标准库)。只需在程序入口处设置一次
func SetJsonMethod(encode func(v any) ([]byte, error), decode func(data []byte, v any) error) {
	setJsonOnce.Do(func() {
		jsonEncode = encode
		jsonDecode = decode
	})
}

// Marshal 对数据进行 JSON 编码
func Marshal(v any) ([]byte, error) {
	if jsonEncode == nil {
		return json.Marshal(v)
	}
	return jsonEncode(v)
}

// Unmarshal 对数据进行 JSON 解码
func Unmarshal(data []byte, v any) error {
	if jsonDecode == nil {
		return json.Unmarshal(data, v)
	}
	return jsonDecode(data, v)
}
