package errors

import (
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
)

// Trace 返回适配 slog 的结构化追踪值。
func Trace(err error) slog.LogValuer {
	if err == nil {
		return nil
	}
	return traceValue{err: err, withTrace: TraceEnabled()}
}

// TraceString 返回第三方日志库可直接打印的文本追踪。
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
func TraceJSON(err error) string {
	if err == nil {
		return ""
	}
	var b strings.Builder
	b.Grow(256)
	writeJSONTrace(&b, err, 0, TraceEnabled())
	return b.String()
}

// traceValue 是对外暴露给 slog 的结构化追踪包装器。
type traceValue struct {
	err       error
	depth     int
	withTrace bool
}

func (t traceValue) LogValue() slog.Value {
	return buildLogValue(t.err, t.depth, t.withTrace)
}

// nodeInfo 是统一渲染层使用的内部视图。
type nodeInfo struct {
	msg      string
	hasMsg   bool
	code     int
	hasCode  bool
	trace    stackTrace
	next     error
	children []error
	ctxKeys  []string
}

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
		info := nodeInfo{msg: e.err.Error(), hasMsg: true, next: e.err, ctxKeys: e.values}
		return info
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

func isJoinNode(info nodeInfo) bool {
	return len(info.children) > 0 && !info.hasMsg && !info.hasCode && len(info.trace) == 0 && info.next == nil
}

func flattenJoinCarrier(info nodeInfo) nodeInfo {
	current := info
	next := current.next
	for next != nil {
		nextInfo := inspectError(next)
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
		next = nextInfo.next
		current.next = next
	}
	return info
}

func formatError(s fmt.State, verb rune, err error) {
	switch verb {
	case 'v':
		if s.Flag('#') {
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

func writeTextTrace(b *strings.Builder, err error, depth int, withTrace bool) {
	if err == nil || depth >= maxChainDepth {
		return
	}
	info := inspectError(err)
	wrote := false
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

func buildLogValue(err error, depth int, withTrace bool) slog.Value {
	if err == nil {
		return slog.AnyValue(nil)
	}
	if depth >= maxChainDepth {
		return slog.StringValue(err.Error())
	}
	info := flattenJoinCarrier(inspectError(err))
	attrs := make([]slog.Attr, 0, 4)
	if info.hasCode {
		attrs = append(attrs, slog.Int("code", info.code))
	}
	if info.hasMsg {
		attrs = append(attrs, slog.String("msg", info.msg))
	}
	if len(info.ctxKeys) > 0 {
		ctxGroup := make([]any, 0, len(info.ctxKeys))
		for i := 0; i < len(info.ctxKeys); i += 2 {
			ctxGroup = append(ctxGroup, slog.String(info.ctxKeys[i], info.ctxKeys[i+1]))
		}
		attrs = append(attrs, slog.Group("ctx", ctxGroup...))
	}
	if withTrace && len(info.trace) > 0 {
		attrs = append(attrs, slog.Any("trace", info.trace))
	}
	switch {
	case len(info.children) > 0:
		attrs = append(attrs, slog.Any("errs", buildLogChildren(info.children, depth+1, withTrace)))
	case info.next != nil:
		attrs = append(attrs, slog.Any("err", traceValue{err: info.next, depth: depth + 1, withTrace: withTrace}))
	}
	return slog.GroupValue(attrs...)
}

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

func buildLogChildren(children []error, depth int, withTrace bool) []any {
	items := make([]any, 0, len(children))
	for _, child := range children {
		items = append(items, buildLogObject(child, depth, withTrace))
	}
	return items
}

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

func writeTraceFrames(b *strings.Builder, st stackTrace) {
	b.WriteByte('[')
	if len(st) == 0 {
		b.WriteByte(']')
		return
	}
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
	b.WriteByte(']')
}

func writeQuotedString(b *strings.Builder, s string) {
	var buf [512]byte
	quoted := strconv.AppendQuote(buf[:0], s)
	_, _ = b.Write(quoted)
}

//go:inline
func writeInt(b *strings.Builder, v int) {
	var buf [20]byte
	b.Write(strconv.AppendInt(buf[:0], int64(v), 10))
}

func writeFirstFrameLocation(b *strings.Builder, st stackTrace) {
	if len(st) == 0 {
		return
	}
	frame, _ := runtime.CallersFrames(st).Next()
	writeFrameLocation(b, frame)
}

func writeQuotedFrame(b *strings.Builder, frame runtime.Frame) {
	file := relativeProjectPath(frame.File)
	if needsJSONEscape(frame.Function) || needsJSONEscape(file) {
		writeQuotedString(b, buildFrameString(frame.Function, file, frame.Line))
		return
	}
	b.WriteByte('"')
	writeFrameText(b, frame.Function, file, frame.Line)
	b.WriteByte('"')
}

func buildFrameString(function, file string, line int) string {
	var b strings.Builder
	b.Grow(len(function) + len(file) + 16)
	b.WriteString(function)
	b.WriteString(" (")
	b.WriteString(file)
	b.WriteByte(':')
	writeInt(&b, line)
	b.WriteByte(')')
	return b.String()
}

func writeFrameLocation(b *strings.Builder, frame runtime.Frame) {
	file := relativeProjectPath(frame.File)
	b.WriteString(file)
	b.WriteByte(':')
	writeInt(b, frame.Line)
}

func writeFrameText(b *strings.Builder, function, file string, line int) {
	b.WriteString(function)
	b.WriteString(" (")
	b.WriteString(file)
	b.WriteByte(':')
	writeInt(b, line)
	b.WriteByte(')')
}

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
