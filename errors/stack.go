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

// ============================ 全局变量 ============================

var (
	projectRootOnce sync.Once // 项目根路径一次性计算
	projectRootPath string    // 项目根路径缓存
)

// ============================ 栈追踪类型 ============================

// stackTrace 程序计数器数组类型，用于存储栈帧信息。
// 创建错误时只保存 uintptr 形式的程序计数器，在渲染时（LogValue/TraceString）才转为文件名和行号。
// 这样设计可以延迟路径裁剪和帧解析开销，只在真正需要展示时才进行耗时的 runtime.CallersFrames 操作。
type stackTrace []uintptr

// LogValue 实现 slog.LogValuer 接口，用于 slog 结构化输出。
// 将栈帧渲染为 slog.GroupValue，格式为 { "0": "func (file:line)", "1": "func (file:line)", ... }
func (st stackTrace) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, len(st)) // attrs 预分配为原始栈深度，避免项目帧全部命中时扩容。
	root := projectRoot()                  // root 是当前业务项目根目录，用于把 runtime/testing 等框架帧挡在输出外。
	if root == "" || len(st) == 0 {
		return slog.GroupValue(appendRawFrameAttrs(attrs, st)...)
	}

	// slog 渲染时边解析边裁剪项目帧，避免先生成 runtime.Frame 再二次遍历带来的额外开销。
	frames := runtime.CallersFrames(st) // frames 从原始 PC 懒解析得到，只有日志真正输出时才产生。
	started := false                    // started 标记已进入第一段连续项目帧，遇到后续非项目帧立即停止。
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

// ============================ stackError 带栈错误类型 ============================

// stackError 是真正带追踪栈的错误节点类型。
// 在创建时会调用 runtime.Callers 捕获当前调用栈，适用于需要记录错误发生位置的场景。
//
// 它保存错误消息、被包装的底层错误和调用栈信息。
type stackError struct {
	msg   string     // 错误消息
	err   error      // 被包装的底层错误
	trace stackTrace // 调用栈追踪
}

// newStackError 内部构造函数，创建带栈追踪的错误。
// 调用 runtime.Callers 采集原始 PC；项目帧裁剪推迟到渲染阶段，降低错误创建路径开销。
func newStackError(msg string, err error) *stackError {
	return &stackError{msg: msg, err: err, trace: callers(2)}
}

// Error 返回错误消息。
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

// Is 实现 errors.Is 接口，支持错误比较。
// 当目标错误也是 stackError 时，比较其是否完全相同。
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

// ============================ 栈帧采集函数 ============================

// callers 采集调用栈的程序计数器。
// 跳过 skip 指定的帧数，通常用于忽略错误库自身的栈帧。
// 这里只保存原始 PC，不在创建阶段解析 runtime.Frame 或裁剪项目路径，降低错误热路径开销。
func callers(skip int) stackTrace {
	var pcs [maxStackDepth]uintptr                   // pcs 是栈上临时缓冲，只承接本次 runtime.Callers 的原始 PC。
	n := runtime.Callers(skip+2, pcs[:StackDepth()]) // n 是实际采集到的帧数，受 StackDepth 全局配置约束。
	if n == 0 {
		return nil
	}
	// 只复制实际采集到的 PC，避免返回局部数组切片导致保留 maxStackDepth 的完整底层数组。
	trace := make(stackTrace, n)
	copy(trace, pcs[:n])
	return trace
}

// ============================ 栈帧渲染工具 ============================

// frameString 将单个栈帧渲染为字符串。
// 格式为：function (file:line)
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

// appendRawFrameAttrs 将原始 PC 栈渲染为 slog 属性。
// 仅在找不到项目根或没有项目帧时作为降级路径使用，保证异常运行环境下仍能看到完整诊断栈。
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

// relativeProjectPath 将绝对路径转换为相对于项目根目录的路径。
// 使用 filepath.Rel 计算相对路径，结果使用正斜杠分隔符。
func relativeProjectPath(file string) string {
	root := projectRoot() // root 来自 go.mod 向上查找结果，用于把源码绝对路径转成项目相对路径。
	if root == "" || file == "" {
		return filepath.ToSlash(file)
	}
	if rel, ok := relativeProjectPathFast(file, root); ok {
		return rel
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

// relativeProjectPathFast 使用字符串前缀快速计算项目相对路径。
// runtime.Frame.File 通常是项目根下的绝对路径，命中该分支可避开 filepath.Rel 的清理和分配成本。
func relativeProjectPathFast(file, root string) (string, bool) {
	if len(file) <= len(root) || !strings.HasPrefix(file, root) {
		return "", false
	}
	if file[len(root)] != os.PathSeparator {
		return "", false
	}
	return filepath.ToSlash(file[len(root)+1:]), true
}

// projectRoot 获取项目根目录路径。
// 通过向上查找包含 go.mod 文件的目录来确定项目根路径。
// 结果会被缓存，后续调用直接返回缓存值。
func projectRoot() string {
	projectRootOnce.Do(func() {
		wd, err := os.Getwd()
		if err != nil {
			return
		}
		dir := wd
		for {
			if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
				projectRootPath = dir
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				return
			}
			dir = parent
		}
	})
	return projectRootPath
}

// isWithinRoot 判断文件路径是否在指定根目录下。
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
