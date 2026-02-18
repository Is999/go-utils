package utils

import (
	"encoding/json"
	"sync"
)

var (
	setOptionsOnce sync.Once
	config         = options{
		// 设置 json 编解码方法(三方开源库)，若未设置则默认使用 encoding/json(标准库)。
		json: _json{
			encode: json.Marshal,
			decode: json.Unmarshal,
		},
	}
)

// Option 用于配置全局设置入口的选项
type Option func(*options)

// 设置 json 编解码方法(三方开源库)，若未设置则默认使用 encoding/json(标准库)。
type _json struct {
	// 对数据进行 JSON 编码
	encode Encode
	// 对数据进行 JSON 解码
	decode Decode
}
type options struct {
	// 设置 json 编解码方法(三方开源库)，若未设置则默认使用 encoding/json(标准库)。
	json _json
}

// WithJSON 设置自定义 JSON 编码、解码方法
func WithJSON(encode Encode, decode Decode) Option {
	return func(o *options) {
		if encode == nil || decode == nil {
			return
		}
		o.json.encode = encode
		o.json.decode = decode
	}
}

// Configure 设置全局参数入口。只需在程序入口处设置一次。
func Configure(opts ...Option) {
	setOptionsOnce.Do(func() {
		cfg := options{
			// 设置 json 编解码方法(三方开源库)，若未设置则默认使用 encoding/json(标准库)。
			json: _json{
				encode: json.Marshal,
				decode: json.Unmarshal,
			},
		}
		for _, opt := range opts {
			if opt != nil {
				opt(&cfg)
			}
		}
		config = cfg
	})
}
