package utils

import (
	"encoding/json"
	"sync"
	"sync/atomic"
)

var (
	// setOptionsOnce 确保 Configure 全局配置只在程序生命周期内生效一次。
	setOptionsOnce sync.Once
	// configValue 使用原子指针保存配置快照，保证并发读取无锁且安全。
	configValue atomic.Pointer[options]
)

// Option 在首次 Configure 内按传入顺序应用，同类选项以后者为准。
type Option func(*options)

// _json 保存成对替换的编解码函数，运行期间可能被并发调用。
type _json struct {
	// encode 默认为 json.Marshal，自定义函数的错误直接交给调用方。
	encode Encode
	// decode 默认为 json.Unmarshal，目标对象由调用方提供。
	decode Decode
	// useStandard 决定 Response.Encode 是否为标准库错误补栈，自定义错误保持原样。
	useStandard bool
}

// options 保存发布后只读的配置快照。
type options struct {
	// json 在发布后不再修改，回调内部的并发安全由调用方保证。
	json _json
	// logger 为全局共享实例，默认跟随 slog.Default。
	logger Logger
}

// init 初始化默认配置，确保未显式 Configure 时也能安全使用。
func init() {
	configValue.Store(defaultOptions())
}

// defaultOptions 为包初始化和首次 Configure 分别创建默认快照，避免修改已发布实例。
func defaultOptions() *options {
	return &options{
		json: _json{
			encode:      json.Marshal,
			decode:      json.Unmarshal,
			useStandard: true,
		},
		logger: &slogLogger{},
	}
}

// WithJSON 成对替换 JSON 编解码函数；任一参数为 nil 时忽略此选项，回调须支持并发调用。
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

// WithLogger 设置支持并发调用的共享 Logger；nil 参数不改变当前选择。
func WithLogger(logger Logger) Option {
	return func(o *options) {
		if logger == nil {
			return
		}
		o.logger = logger
	}
}

// Configure 仅第一次调用生效，包括无参数调用；应在启动时设置，后续调用不会覆盖配置。
func Configure(opts ...Option) {
	setOptionsOnce.Do(func() {
		cfg := defaultOptions() // 本次尚未发布的快照，选项不会改写当前配置。
		for _, opt := range opts {
			if opt != nil {
				opt(cfg)
			}
		}
		// 选项全部应用后再发布，读取方不会看到中间状态。
		configValue.Store(cfg)
	})
}
