// Package errors 提供带调用栈、错误码、上下文字段和结构化追踪的错误工具。
//
// 本包兼容标准库 errors.Is、errors.As、errors.Join、errors.Unwrap 和
// errors.AsType 语义，并在包装标准库或第三方错误时补充可诊断的调用信息。
// 包装对象创建后只读；自定义错误的 Error、Unwrap、As 方法由调用方保证并发读取安全。
// 追踪输出沿用 Join 的多分支格式，只展开子错误；多分支包装自身的消息不单独显示。
// 需要为整组错误附加说明时，可在外层调用 Wrap 或 WithMessage。
package errors
