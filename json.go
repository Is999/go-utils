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

// JSONOption 用于配置自定义 JSON 编解码方法
type JSONOption func(*jsonOptions)

type jsonOptions struct {
	encode func(v any) ([]byte, error)
	decode func(data []byte, v any) error
}

// WithJSONEncoder 设置自定义 JSON 编码方法
func WithJSONEncoder(encode func(v any) ([]byte, error)) JSONOption {
	return func(o *jsonOptions) {
		o.encode = encode
	}
}

// WithJSONDecoder 设置自定义 JSON 解码方法
func WithJSONDecoder(decode func(data []byte, v any) error) JSONOption {
	return func(o *jsonOptions) {
		o.decode = decode
	}
}

// SetJsonMethod 设置json编码解码方法(三方开源库)， 若未设置则默认使用encoding/json(标准库)。只需在程序入口处设置一次
func SetJsonMethod(opts ...JSONOption) {
	setJsonOnce.Do(func() {
		var cfg jsonOptions
		for _, opt := range opts {
			if opt != nil {
				opt(&cfg)
			}
		}
		jsonEncode = cfg.encode
		jsonDecode = cfg.decode
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
