package errors

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// projectRoot 缓存首次渲染时从工作目录向上找到的模块根目录；找不到时返回空字符串。
var projectRoot = sync.OnceValue(func() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
})

// stackTrace 创建时只保存程序计数器，函数名和文件位置推迟到日志渲染时解析。
type stackTrace []uintptr

// LogValue 用数字键表示帧序号，值为 "func (file:line)"。
func (st stackTrace) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, len(st)) // 裁剪后的帧数不会超过原始栈深度。
	root := projectRoot()                  // 首次渲染时确定的模块根目录。
	if root == "" || len(st) == 0 {
		return slog.GroupValue(appendRawFrameAttrs(attrs, st)...)
	}

	// 只输出第一段连续项目帧，离开项目后不再纳入后续框架帧。
	frames := runtime.CallersFrames(st)
	started := false // 是否已进入项目帧区间。
	for range len(st) {
		frame, more := frames.Next()
		if isWithinRoot(frame.File, root) {
			started = true
			attrs = append(attrs, slog.String(strconv.Itoa(len(attrs)), frameString(frame)))
		} else if started {
			break
		}
		if !more {
			break
		}
	}
	if !started {
		// 找不到项目帧时回退原始栈，避免依赖库或非模块运行环境下丢失诊断信息。
		attrs = appendRawFrameAttrs(attrs[:0], st)
	}
	return slog.GroupValue(attrs...)
}

// stackError 记录首次补栈的位置，后续消息或业务码包装复用这份只读调用栈。
type stackError struct {
	msg   string     // 错误消息
	err   error      // 被包装的底层错误
	trace stackTrace // 调用栈追踪
}

// newStackError 从公开构造函数的调用方开始采集，新增转发层时需同步核对 skip。
func newStackError(msg string, err error) *stackError {
	return &stackError{msg: msg, err: err, trace: callers(2)}
}

// Error 连接本层与原因消息，不包含调用栈；nil 接收者返回空字符串。
func (e *stackError) Error() string {
	if e == nil {
		return ""
	}
	return composeErrorMessage(e.msg, e.err)
}

// Unwrap 返回被包装的底层错误，支持错误链展开。
func (e *stackError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Is 只匹配同一个错误对象；相同消息和调用位置不代表同一次错误。
func (e *stackError) Is(target error) bool {
	te, ok := target.(*stackError)
	return ok && e == te
}

// String 返回错误的文本追踪格式，等同于 TraceString。
func (e *stackError) String() string { return TraceString(e) }

// GoString 返回错误的 JSON 追踪格式，等同于 TraceJSON。
func (e *stackError) GoString() string { return TraceJSON(e) }

// Format 实现 fmt.Formatter 接口，支持格式化动词（%v/%s/%q 等）。
func (e *stackError) Format(s fmt.State, verb rune) {
	formatError(s, verb, e)
}

// MarshalJSON 实现 json.Marshaler 接口，返回 JSON 追踪格式。
func (e *stackError) MarshalJSON() ([]byte, error) {
	return []byte(TraceJSON(e)), nil
}

// MarshalText 实现 encoding.TextMarshaler 接口，返回文本追踪格式。
func (e *stackError) MarshalText() ([]byte, error) {
	return []byte(TraceString(e)), nil
}

// callers 跳过自身及额外 skip 层，只复制实际采集的程序计数器。
func callers(skip int) stackTrace {
	var pcs [maxStackDepth]uintptr                   // 临时容量按硬上限分配，返回结果只保留有效帧。
	n := runtime.Callers(skip+2, pcs[:StackDepth()]) // 跳过 runtime.Callers、callers 及指定的内部层级。
	if n == 0 {
		return nil
	}
	// 只复制实际采集到的 PC，避免返回局部数组切片导致保留 maxStackDepth 的完整底层数组。
	trace := make(stackTrace, n)
	copy(trace, pcs[:n])
	return trace
}

// frameString 输出 "function (file:line)"，模块内路径省略根目录。
func frameString(frame runtime.Frame) string {
	file := relativeProjectPath(frame.File)
	var b strings.Builder
	b.Grow(len(frame.Function) + len(file) + 16)
	b.WriteString(frame.Function)
	b.WriteString(" (")
	b.WriteString(file)
	b.WriteByte(':')
	writeInt(&b, frame.Line)
	b.WriteByte(')')
	return b.String()
}

// appendRawFrameAttrs 在没有项目帧可用时输出原始栈，避免丢失失败位置。
func appendRawFrameAttrs(attrs []slog.Attr, st stackTrace) []slog.Attr {
	frames := runtime.CallersFrames(st)
	for i := range len(st) {
		frame, more := frames.Next()
		attrs = append(attrs, slog.String(strconv.Itoa(i), frameString(frame)))
		if !more {
			break
		}
	}
	return attrs
}

// relativeProjectPath 对模块内文件使用相对路径，模块外保留原路径；分隔符统一为正斜杠。
func relativeProjectPath(file string) string {
	root := projectRoot()
	if root == "" || file == "" {
		return filepath.ToSlash(file)
	}
	// 采集到的文件通常直接位于模块根下，无需再由 filepath.Rel 清理路径。
	if len(file) > len(root) && strings.HasPrefix(file, root) && file[len(root)] == os.PathSeparator {
		return filepath.ToSlash(file[len(root)+1:])
	}
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return filepath.ToSlash(file)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(file)
	}
	return filepath.ToSlash(rel)
}

// isWithinRoot 按路径分隔符判断模块边界，避免把同名前缀目录当成子目录。
func isWithinRoot(file, root string) bool {
	if file == "" || root == "" {
		return false
	}
	if file == root {
		return true
	}
	// 常见情况下 frame.File 是 root 下的绝对路径，前缀快路径可以避开 filepath.Rel 的分配。
	if len(file) > len(root) && strings.HasPrefix(file, root) && file[len(root)] == os.PathSeparator {
		return true
	}
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
