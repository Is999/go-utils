package errors

import (
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ============================ 链路追踪 API ============================

// Trace 返回适配 slog 的结构化追踪值。
// 返回的 slog.LogValuer 实现会在日志打印时延迟渲染错误信息。
// 日志输出包含 code、msg、ctx 等字段，可通过 SetTraceEnabled 控制是否输出 trace 数组。
//
// 参数说明：
//   - err：错误对象，nil 时返回 nil
//
// 返回值：slog.LogValuer 接口，调用 LogValue() 时渲染完整错误信息
func Trace(err error) slog.LogValuer {
	if err == nil {
		return nil
	}
	return traceValue{err: err, withTrace: TraceEnabled()}
}

// TraceString 返回第三方日志库可直接打印的文本追踪。
// 格式为：code=xxx, msg=xxx @ file:line; cause=xxx @ file:line
// 使用预分配策略减少内存分配，追踪深度受 maxChainDepth 限制。
//
// 参数说明：
//   - err：错误对象，nil 时返回空字符串
//
// 返回值：多行文本追踪字符串，每层错误占一行
func TraceString(err error) string {
	if err == nil {
		return ""
	}
	var b strings.Builder
	b.Grow(128)
	writeTextTrace(&b, err, 0, TraceEnabled())
	return b.String()
}

// TraceJSON 返回结构化 JSON 追踪字符串，便于第三方日志或落盘。
// JSON 结构包含 code（错误码）、msg（消息）、ctx（上下文）、trace（栈帧数组）等字段。
// Join 多错误场景下使用 errs 数组扁平化输出，避免嵌套噪声。
//
// 参数说明：
//   - err：错误对象，nil 时返回空字符串
//
// 返回值：JSON 格式的错误追踪字符串
func TraceJSON(err error) string {
	if err == nil {
		return ""
	}
	var b strings.Builder
	b.Grow(256)
	writeJSONTrace(&b, err, 0, TraceEnabled())
	return b.String()
}

// ============================ slog 渲染结构 ============================

// traceValue 是对外暴露给 slog 的结构化追踪包装器。
// 实现 slog.LogValuer 接口，支持延迟渲染和日志级别过滤。
//
// 字段说明：
//   - err：错误对象
//   - depth：当前渲染深度，用于限制链式输出的递归层数
//   - withTrace：是否渲染 trace 栈帧数组
type traceValue struct {
	err       error // 错误对象
	depth     int   // 当前递归深度
	withTrace bool  // 是否渲染 trace 数组
}

// LogValue 实现 slog.LogValuer 接口，返回 slog 结构化值。
// 该方法仅在日志系统决定输出该属性时才会被调用，实现延迟渲染。
//
// 返回值：slog.Value，结构化包含 code、msg、ctx、trace、err 等字段
func (t traceValue) LogValue() slog.Value {
	return buildLogValue(t.err, t.depth, t.withTrace)
}

// ============================ 统一渲染视图 ============================

// nodeInfo 是统一渲染层使用的内部视图结构。
// 遍历错误链时将各类错误统一转换为该结构，确保 Text/JSON/slog 三种输出格式一致。
//
// 字段说明：
//   - msg：错误消息内容
//   - hasMsg：是否存在自定义消息（区分空消息和未设置）
//   - code：错误码
//   - hasCode：是否设置了错误码
//   - trace：栈帧数组，仅 stackError 类型会设置
//   - next：下一个错误，用于单链展开
//   - children：子错误列表，用于 Join 多错误场景
//   - ctxKeys：扁平化的上下文键值对数组
type nodeInfo struct {
	msg      string     // 错误消息
	hasMsg   bool       // 是否设置了消息
	code     int        // 错误码
	hasCode  bool       // 是否设置了错误码
	trace    stackTrace // 栈帧数组
	next     error      // 下一个错误（单链展开）
	children []error    // 子错误列表（Join 场景）
	ctxKeys  []string   // 上下文键值对 [key1, val1, key2, val2]
}

// inspectError 将任意错误对象转换为统一的 nodeInfo 视图。
// 使用类型断言识别 stackError、messageError、codeError、contextError 等内置类型，
// 对标准库错误和其他第三方错误使用接口检查（multiUnwrapper/unwrapper）。
//
// 参数说明：
//   - err：待检查的错误对象
//
// 返回值：统一格式的 nodeInfo 结构
func inspectError(err error) nodeInfo {
	if err == nil {
		return nodeInfo{}
	}
	switch e := err.(type) {
	case *stackError:
		// 带栈追踪的错误，提取消息和栈帧
		return nodeInfo{msg: e.msg, hasMsg: true, trace: e.trace, next: e.err}
	case *messageError:
		// 仅带消息的轻量级错误
		return nodeInfo{msg: e.msg, hasMsg: true, next: e.err}
	case *codeError:
		// 带错误码的错误
		return nodeInfo{code: e.code, hasCode: true, next: e.err}
	case *contextError:
		// 带上下文键值的错误
		return nodeInfo{msg: e.err.Error(), hasMsg: true, next: e.err, ctxKeys: e.values}
	default:
		// 处理标准库 errors.Join 等多链错误
		if mu, ok := err.(multiUnwrapper); ok {
			children := compactErrors(mu.Unwrap())
			if len(children) > 0 {
				return nodeInfo{children: children}
			}
		}
		// 处理标准错误和第三方错误
		info := nodeInfo{msg: err.Error(), hasMsg: true}
		if u, ok := err.(unwrapper); ok {
			info.next = u.Unwrap()
		}
		return info
	}
}

// isJoinNode 判断 nodeInfo 是否为 Join 节点。
// Join 节点特征：无自定义消息、无错误码、无栈追踪、无 next 链，仅有 children。
//
// 参数说明：
//   - info：待检查的 nodeInfo
//
// 返回值：true 表示是 Join 节点
func isJoinNode(info nodeInfo) bool {
	return len(info.children) > 0 && !info.hasMsg && !info.hasCode && len(info.trace) == 0 && info.next == nil
}

// flattenJoinCarrier 扁平化 Join 节点的载体信息。
// 当 Wrap/WithCode 等包装器包裹 Join 时，将 Join 的 children 上浮，
// 避免 JSON 输出出现冗余嵌套层级，实现更紧凑的 errs 数组结构。
//
// 合并规则：
//   - 优先保留外层消息（msg）、错误码（code）
//   - 优先保留外层栈追踪（trace），内层不再重复采集
//   - 如果内层也有消息/错误码/栈追踪，则停止上浮，保持原有结构
//
// 参数说明：
//   - info：待处理的 nodeInfo
//
// 返回值：扁平化后的 nodeInfo
func flattenJoinCarrier(info nodeInfo) nodeInfo {
	current := info
	next := current.next
	for next != nil {
		nextInfo := inspectError(next)
		// 遇到 Join 节点，将其 children 上浮到当前位置
		if isJoinNode(nextInfo) {
			current.children = nextInfo.children
			current.next = nil
			return current
		}
		// 内层已有 children，停止上浮
		if len(nextInfo.children) > 0 {
			return info
		}
		merged := false
		// 合并错误码：外层无码时继承内层码
		if nextInfo.hasCode && !current.hasCode {
			current.code = nextInfo.code
			current.hasCode = true
			merged = true
		} else if nextInfo.hasCode {
			// 内层外层都有码，停止合并
			return info
		}
		// 合并消息：外层无消息时继承内层消息
		if nextInfo.hasMsg && !current.hasMsg {
			current.msg = nextInfo.msg
			current.hasMsg = true
			merged = true
		} else if nextInfo.hasMsg {
			return info
		}
		// 合并栈追踪：外层无栈时继承内层栈
		if len(nextInfo.trace) > 0 && len(current.trace) == 0 {
			current.trace = nextInfo.trace
			merged = true
		} else if len(nextInfo.trace) > 0 {
			return info
		}
		// 无任何合并发生，停止
		if !merged {
			return info
		}
		next = nextInfo.next
		current.next = next
	}
	return info
}

// ============================ fmt 格式化支持 ============================

// formatError 实现 fmt.Formatter 接口，支持 %v/%s/%q 等格式化动词。
// 根据格式化标志和动词选择不同的输出格式：
//   - %v / %s：文本追踪格式（TraceString）
//   - %+v / %#v：JSON 追踪格式（TraceJSON）
//   - %q：带引号的文本追踪
//
// 参数说明：
//   - s：格式化状态，可获取标志位和输出目标
//   - verb：格式化动词（v/s/q 等）
//   - err：要格式化的错误对象
func formatError(s fmt.State, verb rune, err error) {
	switch verb {
	case 'v':
		if s.Flag('#') {
			// %+v 或 %#v，输出 JSON 格式
			_, _ = io.WriteString(s, TraceJSON(err))
			return
		}
		// %v，输出文本格式
		_, _ = io.WriteString(s, TraceString(err))
	case 's':
		// %s，输出文本格式
		_, _ = io.WriteString(s, TraceString(err))
	case 'q':
		// %q，输出带引号的文本格式
		_, _ = io.WriteString(s, strconv.Quote(TraceString(err)))
	}
}

// ============================ 文本追踪渲染 ============================

// writeTextTrace 渲染文本格式的错误追踪。
// 格式为：code=xxx, msg=xxx @ file:line; cause=xxx @ file:line
// 通过递归遍历错误链，每层错误占一个分段，使用 "; cause=" 分隔。
//
// 参数说明：
//   - b：字符串构建器，用于追加输出
//   - err：要渲染的错误对象
//   - depth：当前递归深度，防止链过深
//   - withTrace：是否渲染栈帧位置信息
func writeTextTrace(b *strings.Builder, err error, depth int, withTrace bool) {
	if err == nil || depth >= maxChainDepth {
		return
	}
	info := inspectError(err)
	wrote := false
	// 渲染错误码
	if info.hasCode {
		b.WriteString("code=")
		writeInt(b, info.code)
		wrote = true
	}
	// 渲染错误消息
	if info.hasMsg {
		if wrote {
			b.WriteString(", ")
		}
		b.WriteString(info.msg)
		wrote = true
	}
	// 渲染栈帧位置（首个帧）
	if withTrace && len(info.trace) > 0 {
		if wrote {
			b.WriteString(" @ ")
		}
		writeFirstFrameLocation(b, info.trace)
		wrote = true
	}
	// 处理 Join 多错误场景
	switch {
	case len(info.children) > 0:
		if wrote {
			b.WriteString("; ")
		}
		writeJoinedText(b, info.children, depth+1, withTrace)
	case info.next != nil:
		nextInfo := inspectError(info.next)
		if isJoinNode(nextInfo) {
			if wrote {
				b.WriteString("; ")
			}
			writeJoinedText(b, nextInfo.children, depth+1, withTrace)
			return
		}
		if wrote {
			b.WriteString("; cause=")
		}
		writeTextTrace(b, info.next, depth+1, withTrace)
	}
}

// writeJoinedText 渲染 Join 多错误的文本格式。
// 格式为：joined=[err1 @ loc | err2 @ loc | err3]
//
// 参数说明：
//   - b：字符串构建器
//   - children：子错误列表
//   - depth：递归深度
//   - withTrace：是否渲染栈位置
func writeJoinedText(b *strings.Builder, children []error, depth int, withTrace bool) {
	b.WriteString("joined=[")
	for i, child := range children {
		if i > 0 {
			b.WriteString(" | ")
		}
		writeTextTrace(b, child, depth, withTrace)
	}
	b.WriteByte(']')
}

// ============================ JSON 追踪渲染 ============================

const (
	// traceJSONHex 是 TraceJSON 字符串转义使用的十六进制表。
	// 业务意图：错误消息和上下文值可能来自外部输入，必须按 JSON 规则转义控制字符，避免日志落盘后无法解析。
	traceJSONHex = "0123456789abcdef"
)

// writeJSONTrace 渲染 JSON 格式的错误追踪。
// 结构为：{"code":xxx,"msg":"xxx","ctx":{"k":"v"},"trace":["loc1","loc2"],"err":{...}}
// Join 场景下使用 "errs" 数组扁平化输出，避免嵌套。
//
// 参数说明：
//   - b：字符串构建器
//   - err：要渲染的错误对象
//   - depth：递归深度
//   - withTrace：是否渲染 trace 数组
func writeJSONTrace(b *strings.Builder, err error, depth int, withTrace bool) {
	if err == nil {
		b.WriteString("null")
		return
	}
	if depth >= maxChainDepth {
		writeQuotedString(b, err.Error())
		return
	}
	info := flattenJoinCarrier(inspectError(err))
	b.WriteByte('{')
	wrote := false
	// 渲染错误码
	if info.hasCode {
		b.WriteString(`"code":`)
		writeInt(b, info.code)
		wrote = true
	}
	// 渲染错误消息
	if info.hasMsg {
		if wrote {
			b.WriteByte(',')
		}
		b.WriteString(`"msg":`)
		writeQuotedString(b, info.msg)
		wrote = true
	}
	// 渲染上下文键值对
	if len(info.ctxKeys) > 0 {
		if wrote {
			b.WriteByte(',')
		}
		b.WriteString(`"ctx":{`)
		for i := 0; i < len(info.ctxKeys); i += 2 {
			if i > 0 {
				b.WriteByte(',')
			}
			writeQuotedString(b, info.ctxKeys[i])
			b.WriteByte(':')
			writeQuotedString(b, info.ctxKeys[i+1])
		}
		b.WriteByte('}')
		wrote = true
	}
	// 渲染栈帧数组
	if withTrace && len(info.trace) > 0 {
		if wrote {
			b.WriteByte(',')
		}
		b.WriteString(`"trace":`)
		writeTraceFrames(b, info.trace)
		wrote = true
	}
	// 渲染子错误或下一个错误
	switch {
	case len(info.children) > 0:
		if wrote {
			b.WriteByte(',')
		}
		writeJSONChildren(b, info.children, depth+1, withTrace)
	case info.next != nil:
		if wrote {
			b.WriteByte(',')
		}
		b.WriteString(`"err":`)
		writeJSONTrace(b, info.next, depth+1, withTrace)
	}
	b.WriteByte('}')
}

// writeJSONChildren 渲染 Join 多错误的 JSON 数组。
//
// 参数说明：
//   - b：字符串构建器
//   - children：子错误列表
//   - depth：递归深度
//   - withTrace：是否渲染栈追踪
func writeJSONChildren(b *strings.Builder, children []error, depth int, withTrace bool) {
	b.WriteString(`"errs":[`)
	for i, child := range children {
		if i > 0 {
			b.WriteByte(',')
		}
		writeJSONTrace(b, child, depth, withTrace)
	}
	b.WriteByte(']')
}

// ============================ slog 结构化渲染 ============================

// buildLogValue 构建 slog 结构化追踪值。
// 返回 slog.GroupValue，包含 code、msg、ctx、trace、errs 等属性。
//
// 参数说明：
//   - err：错误对象
//   - depth：递归深度
//   - withTrace：是否渲染 trace
//
// 返回值：slog.Value，结构化错误信息
func buildLogValue(err error, depth int, withTrace bool) slog.Value {
	if err == nil {
		return slog.AnyValue(nil)
	}
	if depth >= maxChainDepth {
		return slog.StringValue(err.Error())
	}
	info := flattenJoinCarrier(inspectError(err))
	attrs := make([]slog.Attr, 0, 4)
	// 添加错误码
	if info.hasCode {
		attrs = append(attrs, slog.Int("code", info.code))
	}
	// 添加消息
	if info.hasMsg {
		attrs = append(attrs, slog.String("msg", info.msg))
	}
	// 添加上下文键值对
	if len(info.ctxKeys) > 0 {
		ctxGroup := make([]any, 0, len(info.ctxKeys))
		for i := 0; i < len(info.ctxKeys); i += 2 {
			ctxGroup = append(ctxGroup, slog.String(info.ctxKeys[i], info.ctxKeys[i+1]))
		}
		attrs = append(attrs, slog.Group("ctx", ctxGroup...))
	}
	// 添加栈追踪
	if withTrace && len(info.trace) > 0 {
		attrs = append(attrs, slog.Any("trace", info.trace))
	}
	// 添加子错误或下一错误
	switch {
	case len(info.children) > 0:
		attrs = append(attrs, slog.Any("errs", buildLogChildren(info.children, depth+1, withTrace)))
	case info.next != nil:
		attrs = append(attrs, slog.Any("err", traceValue{err: info.next, depth: depth + 1, withTrace: withTrace}))
	}
	return slog.GroupValue(attrs...)
}

// buildLogChildren 构建 slog 子错误数组。
//
// 参数说明：
//   - children：子错误列表
//   - depth：递归深度
//   - withTrace：是否渲染栈追踪
//
// 返回值：slog 可序列化的 []any 数组
func buildLogChildren(children []error, depth int, withTrace bool) []any {
	items := make([]any, 0, len(children))
	for _, child := range children {
		items = append(items, buildLogObject(child, depth, withTrace))
	}
	return items
}

// buildLogObject 构建单个错误的 slog 对象表示。
//
// 参数说明：
//   - err：错误对象
//   - depth：递归深度
//   - withTrace：是否渲染栈追踪
//
// 返回值：map[string]any 结构
func buildLogObject(err error, depth int, withTrace bool) map[string]any {
	if err == nil || depth >= maxChainDepth {
		return map[string]any{"msg": ""}
	}
	info := flattenJoinCarrier(inspectError(err))
	obj := make(map[string]any, 4)
	if info.hasCode {
		obj["code"] = info.code
	}
	if info.hasMsg {
		obj["msg"] = info.msg
	}
	if withTrace && len(info.trace) > 0 {
		obj["trace"] = info.trace
	}
	switch {
	case len(info.children) > 0:
		obj["errs"] = buildLogChildren(info.children, depth+1, withTrace)
	case info.next != nil:
		obj["err"] = buildLogObject(info.next, depth+1, withTrace)
	}
	return obj
}

// ============================ 栈帧渲染工具 ============================

// writeTraceFrames 渲染栈帧数组为 JSON 格式。
// 格式为：["file1:10","file2:20","file3:30"]
//
// 参数说明：
//   - b：字符串构建器
//   - st：栈帧数组
func writeTraceFrames(b *strings.Builder, st stackTrace) {
	b.WriteByte('[')
	if len(st) == 0 {
		b.WriteByte(']')
		return
	}
	if !writeProjectTraceFrames(b, st) {
		writeRawTraceFrames(b, st)
	}
	b.WriteByte(']')
}

// writeProjectTraceFrames 将项目内连续栈帧渲染为 JSON 数组元素。
// 数据来源是 runtime.Callers 捕获的原始 PC；渲染时才解析 runtime.Frame 并裁剪项目边界，避免创建错误时承担路径处理成本。
//
// 参数说明：
//   - b：字符串构建器
//   - st：原始 PC 栈追踪
//
// 返回值：true 表示已找到并写入项目帧；false 表示需要调用方降级输出原始栈
func writeProjectTraceFrames(b *strings.Builder, st stackTrace) bool {
	root := projectRoot() // root 是项目边界，命中后只输出第一段连续业务栈帧。
	if root == "" || len(st) == 0 {
		return false
	}

	frames := runtime.CallersFrames(st) // frames 从原始 PC 懒解析，避免错误创建时承担 JSON 输出成本。
	started := false                    // started 表示已进入项目帧区间，后续遇到非项目帧即完成裁剪。
	wrote := false                      // wrote 标记 JSON 数组是否已有元素，用于安全写入逗号分隔符。
	for range len(st) {
		frame, more := frames.Next()
		if isWithinRoot(frame.File, root) {
			started = true
			if wrote {
				b.WriteByte(',')
			}
			writeQuotedFrame(b, frame)
			wrote = true
		} else if started {
			break
		}
		if !more {
			break
		}
	}
	return started
}

// writeRawTraceFrames 将原始 PC 栈帧渲染为 JSON 数组元素。
// 这是找不到项目根或项目帧时的降级策略，宁可保留完整诊断信息，也不静默输出空 trace。
//
// 参数说明：
//   - b：字符串构建器
//   - st：原始 PC 栈追踪
func writeRawTraceFrames(b *strings.Builder, st stackTrace) {
	frames := runtime.CallersFrames(st) // frames 是降级输出使用的原始调用栈，可能包含框架帧但不丢诊断信息。
	for i := range len(st) {
		frame, more := frames.Next()
		if i > 0 {
			b.WriteByte(',')
		}
		writeQuotedFrame(b, frame)
		if !more {
			break
		}
	}
}

// writeQuotedString 将字符串写入 Builder 并进行 JSON 转义。
// 业务意图：TraceJSON 可能承载外部错误消息和 context 值，必须输出严格 JSON 字符串而不是 Go 字符串字面量。
//
// 参数说明：
//   - b：字符串构建器
//   - s：要写入的字符串
func writeQuotedString(b *strings.Builder, s string) {
	b.WriteByte('"')
	writeJSONEscapedContent(b, s)
	b.WriteByte('"')
}

// writeInt 将整数写入 Builder，使用栈缓冲区避免分配。
//
//go:inline
func writeInt(b *strings.Builder, v int) {
	var buf [20]byte
	b.Write(strconv.AppendInt(buf[:0], int64(v), 10))
}

// writeFirstFrameLocation 渲染栈帧的第一个位置。
// 格式为：file:line
//
// 参数说明：
//   - b：字符串构建器
//   - st：栈帧数组
func writeFirstFrameLocation(b *strings.Builder, st stackTrace) {
	if len(st) == 0 {
		return
	}
	if writeFirstProjectFrameLocation(b, st) {
		return
	}
	writeFirstRawFrameLocation(b, st)
}

// writeFirstProjectFrameLocation 写入首个项目内栈帧的位置。
// TraceString 只展示首个业务失败位置，因此找到第一帧项目路径后即可停止，避免解析完整调用栈。
//
// 参数说明：
//   - b：字符串构建器
//   - st：原始 PC 栈追踪
//
// 返回值：true 表示已写入项目内位置；false 表示调用方需要降级使用原始首帧
func writeFirstProjectFrameLocation(b *strings.Builder, st stackTrace) bool {
	root := projectRoot() // root 用来跳过错误库外层框架帧，优先定位第一帧业务代码。
	if root == "" || len(st) == 0 {
		return false
	}
	frames := runtime.CallersFrames(st) // frames 按需解析到第一帧项目路径后即停止，保护文本日志性能。
	for range len(st) {
		frame, more := frames.Next()
		if isWithinRoot(frame.File, root) {
			writeFrameLocation(b, frame)
			return true
		}
		if !more {
			break
		}
	}
	return false
}

// writeFirstRawFrameLocation 写入原始栈的首帧位置。
// 这是项目根不可用或没有项目帧时的降级策略，确保 TraceString 仍然给出可定位的失败位置。
//
// 参数说明：
//   - b：字符串构建器
//   - st：原始 PC 栈追踪
func writeFirstRawFrameLocation(b *strings.Builder, st stackTrace) {
	frame, _ := runtime.CallersFrames(st).Next()
	writeFrameLocation(b, frame)
}

// writeQuotedFrame 将单个栈帧写入 Builder，带 JSON 引号。
// 如果函数名或文件路径包含特殊字符，先进行 JSON 转义。
//
// 参数说明：
//   - b：字符串构建器
//   - frame：栈帧信息
func writeQuotedFrame(b *strings.Builder, frame runtime.Frame) {
	file := relativeProjectPath(frame.File)
	if needsJSONEscape(frame.Function) || needsJSONEscape(file) {
		b.WriteByte('"')
		writeJSONEscapedContent(b, frame.Function)
		b.WriteString(" (")
		writeJSONEscapedContent(b, file)
		b.WriteByte(':')
		writeInt(b, frame.Line)
		b.WriteString(`)"`)
		return
	}
	b.WriteByte('"')
	writeFrameText(b, frame.Function, file, frame.Line)
	b.WriteByte('"')
}

// writeJSONEscapedContent 写入 JSON 字符串内部内容，不包含外层引号。
// 业务意图：栈帧函数名或路径偶发包含引号、反斜杠、控制字符时直接转义片段，避免先拼接完整帧字符串再二次转义。
//
// 参数说明：
//   - b：字符串构建器
//   - s：待写入的字符串片段，数据来源为 runtime.Frame.Function 或裁剪后的文件路径
func writeJSONEscapedContent(b *strings.Builder, s string) {
	start := 0 // start 是尚未写入的安全片段起点，用于批量写出普通字符减少 Write 调用。
	for i := 0; i < len(s); {
		if c := s[i]; c < utf8.RuneSelf {
			if c >= 0x20 && c != '\\' && c != '"' {
				i++
				continue
			}
			b.WriteString(s[start:i])
			switch c {
			case '\\', '"':
				b.WriteByte('\\')
				b.WriteByte(c)
			case '\b':
				b.WriteString(`\b`)
			case '\f':
				b.WriteString(`\f`)
			case '\n':
				b.WriteString(`\n`)
			case '\r':
				b.WriteString(`\r`)
			case '\t':
				b.WriteString(`\t`)
			default:
				// 其他控制字符必须写成 \u00xx；\xNN 是 Go 字符串语法，不是合法 JSON。
				b.WriteString(`\u00`)
				b.WriteByte(traceJSONHex[c>>4])
				b.WriteByte(traceJSONHex[c&0x0f])
			}
			i++
			start = i
			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:]) // r 是当前 UTF-8 字符；非法字节需要按 JSON 兼容方式降级。
		if r == utf8.RuneError && size == 1 {
			b.WriteString(s[start:i])
			b.WriteString(`\ufffd`)
			i++
			start = i
			continue
		}
		if r == '\u2028' || r == '\u2029' {
			b.WriteString(s[start:i])
			b.WriteString(`\u202`)
			b.WriteByte(traceJSONHex[r&0x0f])
			i += size
			start = i
			continue
		}
		i += size
	}
	b.WriteString(s[start:])
}

// writeFrameLocation 将帧位置写入 Builder。
// 格式为：file:line
//
// 参数说明：
//   - b：字符串构建器
//   - frame：栈帧信息
func writeFrameLocation(b *strings.Builder, frame runtime.Frame) {
	file := relativeProjectPath(frame.File)
	b.WriteString(file)
	b.WriteByte(':')
	writeInt(b, frame.Line)
}

// writeFrameText 将帧的完整信息写入 Builder。
// 格式为：function (file:line)
//
// 参数说明：
//   - b：字符串构建器
//   - function：函数名
//   - file：文件路径
//   - line：行号
func writeFrameText(b *strings.Builder, function, file string, line int) {
	b.WriteString(function)
	b.WriteString(" (")
	b.WriteString(file)
	b.WriteByte(':')
	writeInt(b, line)
	b.WriteByte(')')
}

// needsJSONEscape 检查字符串是否包含需要 JSON 转义的字符。
// 检查范围：反斜杠、引号、控制字符（< 0x20）。
//
// 参数说明：
//   - s：待检查的字符串
//
// 返回值：true 表示需要转义
func needsJSONEscape(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\', '"':
			return true
		default:
			if s[i] < 0x20 {
				return true
			}
		}
	}
	return false
}
