package utils

import (
	"encoding/json"
	"sync"
	"sync/atomic"
)

// 全局配置状态，使用 once 与原子指针保证并发安全。
var (
	// setOptionsOnce 确保 Configure 全局配置只在程序生命周期内生效一次。
	setOptionsOnce sync.Once
	// configValue 使用原子指针保存配置快照，保证并发读取无锁且安全。
	configValue atomic.Pointer[options]
)

// Option 用于配置全局设置入口的选项
type Option func(*options)

// 设置 json 编解码方法(三方开源库)，若未设置则默认使用 encoding/json(标准库)。
type _json struct {
	// 对数据进行 JSON 编码
	encode Encode
	// 对数据进行 JSON 解码
	decode Decode
	// useStandard 表示当前是否仍使用标准库 JSON 编解码。
	// 业务意图：响应体可在标准库语义下走手写包壳快路径；用户注入第三方 JSON 时必须回退到自定义实现。
	useStandard bool
}

// options 保存全局配置快照。
// 结构体内部字段只在 Configure 阶段写入，运行期通过原子指针只读访问。
type options struct {
	// 设置 json 编解码方法(三方开源库)，若未设置则默认使用 encoding/json(标准库)。
	json _json
	// 设置日志(三方日志库)，若未设置则默认使用 log/slog(标准库)。
	logger Logger
}

// init 初始化默认配置，确保未显式 Configure 时也能安全使用。
func init() {
	configValue.Store(defaultOptions())
}

// defaultOptions 返回默认配置快照。
func defaultOptions() *options {
	return &options{
		// 设置 json 编解码方法(三方开源库)，若未设置则默认使用 encoding/json(标准库)。
		json: _json{
			encode:      json.Marshal,
			decode:      json.Unmarshal,
			useStandard: true,
		},
		// 设置日志(三方日志库)，若未设置则默认使用 log/slog(标准库)。
		logger: &slogLogger{},
	}
}

// currentConfig 返回当前生效的全局配置快照。
func currentConfig() *options {
	return configValue.Load()
}

// WithJSON 设置自定义 JSON 编码、解码方法
func WithJSON(encode Encode, decode Decode) Option {
	return func(o *options) {
		if encode == nil || decode == nil {
			return
		}
		o.json.encode = encode
		o.json.decode = decode
		o.json.useStandard = false
	}
}

// WithLogger 设置自定义 Logger，若未设置则默认使用 log/slog 标准库。
func WithLogger(logger Logger) Option {
	return func(o *options) {
		if logger == nil {
			return
		}
		o.logger = logger
	}
}

// Configure 设置全局参数入口。只需在程序入口处设置一次。
func Configure(opts ...Option) {
	setOptionsOnce.Do(func() {
		cfg := *defaultOptions()
		for _, opt := range opts {
			if opt != nil {
				opt(&cfg)
			}
		}
		configValue.Store(&cfg)
	})
}
