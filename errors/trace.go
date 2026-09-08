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

// Trace 返回延迟渲染的 slog 追踪值，err 为 nil 时返回 nil。
// 是否输出栈在本次调用时确定，之后切换开关不会改变已创建的追踪值。
func Trace(err error) slog.LogValuer {
	if err == nil {
		return nil
	}
	return traceValue{err: err, withTrace: TraceEnabled()}
}

// TraceString 输出消息、业务码及各层首个调用位置，不输出上下文字段；nil 返回空字符串。
func TraceString(err error) string {
	if err == nil {
		return ""
	}
	var b strings.Builder
	b.Grow(128)
	writeTextTrace(&b, err, 0, TraceEnabled())
	return b.String()
}

// TraceJSON 输出 code、msg、ctx 和 trace，单一原因使用 err，Join 子错误使用 errs 数组。
// err 为 nil 时返回空字符串；栈输出开关在本次调用时读取。
func TraceJSON(err error) string {
	if err == nil {
		return ""
	}
	var b strings.Builder
	b.Grow(256)
	writeJSONTrace(&b, err, 0, TraceEnabled())
	return b.String()
}

// traceValue 将节点展开推迟到 handler 消费属性时，日志被过滤时无需渲染。
type traceValue struct {
	err       error // 错误对象
	depth     int   // 当前递归深度
	withTrace bool  // 构造时记录的栈输出开关，整条延迟渲染链共用。
}

// LogValue 在消费属性时展开当前节点，单链原因继续交给 handler 延迟渲染。
func (t traceValue) LogValue() slog.Value {
	if t.err == nil {
		return slog.AnyValue(nil)
	}
	if t.depth >= maxChainDepth {
		return slog.StringValue(t.err.Error())
	}
	info := flattenJoinCarrier(inspectError(t.err))
	attrs := make([]slog.Attr, 0, 4)
	if info.hasCode {
		attrs = append(attrs, slog.Int("code", info.code))
	}
	if info.hasMsg {
		attrs = append(attrs, slog.String("msg", info.msg))
	}
	if len(info.ctxKeys) > 0 {
		// 直接构造属性组，避免把每个属性装箱为 any 后再由 slog.Group 拆出。
		ctxGroup := make([]slog.Attr, 0, len(info.ctxKeys)/2)
		for i := 0; i < len(info.ctxKeys); i += 2 {
			ctxGroup = append(ctxGroup, slog.String(info.ctxKeys[i], info.ctxKeys[i+1]))
		}
		attrs = append(attrs, slog.Attr{Key: "ctx", Value: slog.GroupValue(ctxGroup...)})
	}
	if t.withTrace && len(info.trace) > 0 {
		attrs = append(attrs, slog.Any("trace", info.trace))
	}
	switch {
	case len(info.children) > 0:
		// Join 数组在此展开，handler 不会递归解析数组里的 LogValuer。
		attrs = append(attrs, slog.Any("errs", buildLogChildren(info.children, t.depth+1, t.withTrace)))
	case info.next != nil:
		attrs = append(attrs, slog.Any("err", traceValue{err: info.next, depth: t.depth + 1, withTrace: t.withTrace}))
	}
	return slog.GroupValue(attrs...)
}

// nodeInfo 为 Text、JSON 和 slog 提供同一节点视图，不修改原错误或其子切片。
type nodeInfo struct {
	msg      string     // 错误消息
	hasMsg   bool       // 区分无消息字段与显式空消息。
	code     int        // 错误码
	hasCode  bool       // 区分无业务码与有效的 0 码。
	trace    stackTrace // 栈帧数组
	next     error      // 下一个错误（单链展开）
	children []error    // 子错误列表（Join 场景）
	ctxKeys  []string   // 上下文键值对 [key1, val1, key2, val2]
}

// inspectError 提取本层元数据；第三方多分支错误有非 nil 子节点时只展示子节点。
func inspectError(err error) nodeInfo {
	if err == nil {
		return nodeInfo{}
	}
	switch e := err.(type) {
	case *stackError:
		return nodeInfo{msg: e.msg, hasMsg: true, trace: e.trace, next: e.err}
	case *messageError:
		return nodeInfo{msg: e.msg, hasMsg: true, next: e.err}
	case *codeError:
		return nodeInfo{code: e.code, hasCode: true, next: e.err}
	case *contextError:
		return nodeInfo{msg: e.err.Error(), hasMsg: true, next: e.err, ctxKeys: e.values}
	default:
		if mu, ok := err.(multiUnwrapper); ok {
			children := compactErrors(mu.Unwrap())
			if len(children) > 0 {
				return nodeInfo{children: children}
			}
		}
		info := nodeInfo{msg: err.Error(), hasMsg: true}
		if u, ok := err.(unwrapper); ok {
			info.next = u.Unwrap()
		}
		return info
	}
}

// isJoinNode 仅识别没有消息、错误码、调用栈或单链原因的纯分支节点。
func isJoinNode(info nodeInfo) bool {
	return len(info.children) > 0 && !info.hasMsg && !info.hasCode && len(info.trace) == 0 && info.next == nil
}

// flattenJoinCarrier 将无冲突的包装元数据并到 Join 容器，冲突时保留原结构。
func flattenJoinCarrier(info nodeInfo) nodeInfo {
	current := info // 合并结果暂存于副本，确认到达 Join 前不改变原节点结构。
	for current.next != nil {
		nextInfo := inspectError(current.next)
		// 到达纯 Join 才提交，避免中途退出时丢失原有层级。
		if isJoinNode(nextInfo) {
			current.children = nextInfo.children
			current.next = nil
			return current
		}
		if len(nextInfo.children) > 0 {
			return info
		}
		merged := false
		if nextInfo.hasCode && !current.hasCode {
			current.code = nextInfo.code
			current.hasCode = true
			merged = true
		} else if nextInfo.hasCode {
			return info
		}
		if nextInfo.hasMsg && !current.hasMsg {
			current.msg = nextInfo.msg
			current.hasMsg = true
			// contextError 的上下文随消息一起上浮；外层已有消息时仍保留原嵌套。
			current.ctxKeys = nextInfo.ctxKeys
			merged = true
		} else if nextInfo.hasMsg {
			return info
		}
		if len(nextInfo.trace) > 0 && len(current.trace) == 0 {
			current.trace = nextInfo.trace
			merged = true
		} else if len(nextInfo.trace) > 0 {
			return info
		}
		if !merged {
			return info
		}
		current.next = nextInfo.next
	}
	return info
}

// formatError 忽略宽度和精度，仅按动词及 +/# 标志选择追踪格式。
func formatError(s fmt.State, verb rune, err error) {
	switch verb {
	case 'v':
		// %+v 和 %#v 使用 JSON，其余 %v 沿用文本格式。
		if s.Flag('+') || s.Flag('#') {
			_, _ = io.WriteString(s, TraceJSON(err))
			return
		}
		_, _ = io.WriteString(s, TraceString(err))
	case 's':
		_, _ = io.WriteString(s, TraceString(err))
	case 'q':
		_, _ = io.WriteString(s, strconv.Quote(TraceString(err)))
	}
}

// writeTextTrace 用 "; cause=" 分隔单链原因，Join 分支写入 joined=[...]，到达深度上限后停止。
func writeTextTrace(b *strings.Builder, err error, depth int, withTrace bool) {
	if err == nil || depth >= maxChainDepth {
		return
	}
	info := inspectError(err)
	wrote := false // 记录本层是否有字段；显式空消息也参与分隔符规则。
	if info.hasCode {
		b.WriteString("code=")
		writeInt(b, info.code)
		wrote = true
	}
	if info.hasMsg {
		if wrote {
			b.WriteString(", ")
		}
		b.WriteString(info.msg)
		wrote = true
	}
	if withTrace && len(info.trace) > 0 {
		if wrote {
			b.WriteString(" @ ")
		}
		writeFirstFrameLocation(b, info.trace)
		wrote = true
	}
	switch {
	case len(info.children) > 0:
		if wrote {
			b.WriteString("; ")
		}
		writeJoinedText(b, info.children, depth+1, withTrace)
	case info.next != nil:
		nextInfo := inspectError(info.next)
		if isJoinNode(nextInfo) {
			// Join 容器不额外输出 cause 层，其分支直接接在当前节点输出之后。
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

// writeJoinedText 按输入顺序写出 joined=[err1 | err2]，分支之间不增加递归深度。
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

const (
	// traceJSONHex 使用小写十六进制生成 JSON 的 Unicode 转义。
	traceJSONHex = "0123456789abcdef"
)

// writeJSONTrace 用对象表示节点；到达深度上限时只写消息字符串，仍保持 JSON 有效。
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
	wrote := false // 可选字段按实际输出补逗号，不能按零值判断字段是否存在。
	if info.hasCode {
		b.WriteString(`"code":`)
		writeInt(b, info.code)
		wrote = true
	}
	if info.hasMsg {
		if wrote {
			b.WriteByte(',')
		}
		b.WriteString(`"msg":`)
		writeQuotedString(b, info.msg)
		wrote = true
	}
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
	if withTrace && len(info.trace) > 0 {
		if wrote {
			b.WriteByte(',')
		}
		b.WriteString(`"trace":`)
		writeTraceFrames(b, info.trace)
		wrote = true
	}
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

// buildLogChildren 将 Join 分支表示为对象数组，避免把数组误编码为 slog 属性组。
func buildLogChildren(children []error, depth int, withTrace bool) []any {
	items := make([]any, 0, len(children))
	for _, child := range children {
		items = append(items, buildLogObject(child, depth, withTrace))
	}
	return items
}

// buildLogObject 生成数组元素所需的普通 map，内部值须自行渲染，不能依赖 handler 递归解析。
func buildLogObject(err error, depth int, withTrace bool) map[string]any {
	if err == nil {
		return map[string]any{"msg": ""}
	}
	if depth >= maxChainDepth {
		// 截断后仍保留剩余消息，Join 数组元素继续使用对象形状。
		return map[string]any{"msg": err.Error()}
	}
	info := flattenJoinCarrier(inspectError(err))
	obj := make(map[string]any, 4)
	if info.hasCode {
		obj["code"] = info.code
	}
	if info.hasMsg {
		obj["msg"] = info.msg
	}
	if len(info.ctxKeys) > 0 {
		ctx := make(map[string]string, len(info.ctxKeys)/2)
		for i := 0; i < len(info.ctxKeys); i += 2 {
			ctx[info.ctxKeys[i]] = info.ctxKeys[i+1]
		}
		obj["ctx"] = ctx
	}
	if withTrace && len(info.trace) > 0 {
		// handler 不会解析 map 内的 LogValuer；这里复用栈帧裁剪规则，避免输出原始 PC。
		attrs := info.trace.LogValue().Group()
		frames := make([]string, len(attrs))
		for i, attr := range attrs {
			frames[i] = attr.Value.String()
		}
		obj["trace"] = frames
	}
	switch {
	case len(info.children) > 0:
		obj["errs"] = buildLogChildren(info.children, depth+1, withTrace)
	case info.next != nil:
		obj["err"] = buildLogObject(info.next, depth+1, withTrace)
	}
	return obj
}

// writeTraceFrames 输出 "函数名 (文件:行号)" 数组，优先保留第一段连续项目帧。
func writeTraceFrames(b *strings.Builder, st stackTrace) {
	b.WriteByte('[')
	if len(st) == 0 {
		b.WriteByte(']')
		return
	}
	started := false // 是否已写入项目帧。
	if root := projectRoot(); root != "" {
		frames := runtime.CallersFrames(st)
		for range len(st) {
			frame, more := frames.Next()
			if isWithinRoot(frame.File, root) {
				if started {
					b.WriteByte(',')
				}
				writeQuotedFrame(b, frame)
				started = true
			} else if started {
				// 离开项目后不再纳入后续框架帧。
				break
			}
			if !more {
				break
			}
		}
	}
	if !started {
		// 找不到项目根或项目帧时保留原始栈，避免丢失失败位置。
		frames := runtime.CallersFrames(st)
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
	b.WriteByte(']')
}

// writeQuotedString 使用 JSON 字符串规则，避免消息或上下文中的控制字符破坏输出。
func writeQuotedString(b *strings.Builder, s string) {
	b.WriteByte('"')
	writeJSONEscapedContent(b, s)
	b.WriteByte('"')
}

// writeInt 使用 20 字节缓冲写入十进制整数，容量包含 int64 最小值的负号。
func writeInt(b *strings.Builder, v int) {
	var buf [20]byte
	b.Write(strconv.AppendInt(buf[:0], int64(v), 10))
}

// writeFirstFrameLocation 输出首个项目帧的 file:line，找不到时使用原始首帧。
func writeFirstFrameLocation(b *strings.Builder, st stackTrace) {
	if len(st) == 0 {
		return
	}
	if root := projectRoot(); root != "" {
		// 文本只输出一个位置，找到首个项目帧后无需继续解析。
		frames := runtime.CallersFrames(st)
		for range len(st) {
			frame, more := frames.Next()
			if isWithinRoot(frame.File, root) {
				writeFrameLocation(b, frame)
				return
			}
			if !more {
				break
			}
		}
	}
	// 找不到项目帧时仍保留原始首帧，避免丢失失败位置。
	frame, _ := runtime.CallersFrames(st).Next()
	writeFrameLocation(b, frame)
}

// writeQuotedFrame 复用消息字段的 JSON 转义规则，避免函数名或路径破坏输出。
func writeQuotedFrame(b *strings.Builder, frame runtime.Frame) {
	b.WriteByte('"')
	writeJSONEscapedContent(b, frame.Function)
	b.WriteString(" (")
	writeJSONEscapedContent(b, relativeProjectPath(frame.File))
	b.WriteByte(':')
	writeInt(b, frame.Line)
	b.WriteString(`)"`)
}

// writeJSONEscapedContent 写入不含外层引号的 JSON 字符串，保留 HTML 字符，替换非法 UTF-8。
func writeJSONEscapedContent(b *strings.Builder, s string) {
	start := 0 // s[start:i] 为无需转义的连续片段，遇到转义字符时一次写出。
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

		r, size := utf8.DecodeRuneInString(s[i:])
		// 合法的 U+FFFD 占多个字节；只有单字节 RuneError 表示非法编码。
		if r == utf8.RuneError && size == 1 {
			b.WriteString(s[start:i])
			b.WriteString(`\ufffd`)
			i++
			start = i
			continue
		}
		// 行分隔符沿用 encoding/json 的 Unicode 转义。
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

// writeFrameLocation 按 file:line 输出位置，仅模块内文件使用相对路径。
func writeFrameLocation(b *strings.Builder, frame runtime.Frame) {
	file := relativeProjectPath(frame.File)
	b.WriteString(file)
	b.WriteByte(':')
	writeInt(b, frame.Line)
}
