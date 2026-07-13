// Package errors 提供带调用栈、错误码、上下文字段和结构化追踪的错误工具。
//
// 本包兼容标准库 errors.Is、errors.As、errors.Join 和 errors.Unwrap 语义，
// 并在包装标准库或第三方错误时补充可诊断的调用信息。
package errors
