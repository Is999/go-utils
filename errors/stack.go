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
// 只保存 uintptr 形式的程序计数器，在渲染时（LogValue/TraceString）才转为文件名和行号。
// 这样设计可以延迟解析开销，只在真正需要展示时才进行耗时的 runtime.CallersFrames 操作。
type stackTrace []uintptr

// LogValue 实现 slog.LogValuer 接口，用于 slog 结构化输出。
// 将栈帧渲染为 slog.GroupValue，格式为 { "0": "func (file:line)", "1": "func (file:line)", ... }
//
// 返回值：slog.Value，包含所有栈帧的属性组
func (st stackTrace) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, len(st))
	frames := runtime.CallersFrames(st)
	for i := range len(st) {
		frame, more := frames.Next()
		attrs = append(attrs, slog.String(strconv.Itoa(i), frameString(frame)))
		if !more {
			break
		}
	}
	return slog.GroupValue(attrs...)
}

// ============================ stackError 带栈错误类型 ============================

// stackError 是真正带追踪栈的错误节点类型。
// 在创建时会调用 runtime.Callers 捕获当前调用栈，适用于需要记录错误发生位置的场景。
//
// 字段说明：
//   - msg：错误消息
//   - err：被包装的底层错误
//   - trace：调用栈信息
type stackError struct {
	msg   string     // 错误消息
	err   error      // 被包装的底层错误
	trace stackTrace // 调用栈追踪
}

// newStackError 内部构造函数，创建带栈追踪的错误。
// 调用 runtime.Callers 采集调用栈，过滤掉项目外的 runtime 帧。
//
// 参数说明：
//   - msg：错误消息
//   - err：被包装的底层错误（可为 nil）
//
// 返回值：带栈追踪的错误对象
func newStackError(msg string, err error) *stackError {
	return &stackError{msg: msg, err: err, trace: callers(2)}
}

// Error 返回错误消息。
//
// 返回值：错误消息字符串
func (e *stackError) Error() string {
	if e == nil {
		return ""
	}
	return e.msg
}

// Unwrap 返回被包装的底层错误，支持错误链展开。
//
// 返回值：底层错误对象
func (e *stackError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Is 实现 errors.Is 接口，支持错误比较。
// 当目标错误也是 stackError 时，比较其是否完全相同。
//
// 参数说明：
//   - target：目标错误对象
//
// 返回值：true 表示两个错误相等
func (e *stackError) Is(target error) bool {
	te, ok := target.(*stackError)
	return ok && e == te
}

// String 返回错误的文本追踪格式，等同于 TraceString。
//
// 返回值：错误的多行文本表示
func (e *stackError) String() string { return TraceString(e) }

// GoString 返回错误的 JSON 追踪格式，等同于 TraceJSON。
//
// 返回值：错误的 JSON 字符串表示
func (e *stackError) GoString() string { return TraceJSON(e) }

// Format 实现 fmt.Formatter 接口，支持格式化动词（%v/%s/%q 等）。
//
// 参数说明：
//   - s：格式化状态
//   - verb：格式化动词
func (e *stackError) Format(s fmt.State, verb rune) {
	formatError(s, verb, e)
}

// MarshalJSON 实现 json.Marshaler 接口，返回 JSON 追踪格式。
//
// 返回值：JSON 字节数组
func (e *stackError) MarshalJSON() ([]byte, error) {
	return []byte(TraceJSON(e)), nil
}

// MarshalText 实现 encoding.TextMarshaler 接口，返回文本追踪格式。
//
// 返回值：错误文本表示的字节数组
func (e *stackError) MarshalText() ([]byte, error) {
	return []byte(TraceString(e)), nil
}

// ============================ 栈帧采集函数 ============================

// callers 采集调用栈的程序计数器。
// 跳过 skip 指定的帧数，通常用于忽略错误库自身的栈帧。
//
// 参数说明：
//   - skip：跳过的栈帧数
//
// 返回值：采集到的程序计数器数组
func callers(skip int) stackTrace {
	var pcs [maxStackDepth]uintptr
	n := runtime.Callers(skip+2, pcs[:StackDepth()])
	return trimProjectFrames(pcs[:n])
}

// ============================ 栈帧渲染工具 ============================

// frameString 将单个栈帧渲染为字符串。
// 格式为：function (file:line)
//
// 参数说明：
//   - frame：栈帧信息
//
// 返回值：格式化后的帧字符串
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

// ============================ 项目路径裁剪 ============================

// trimProjectFrames 裁剪栈帧，只保留项目内的栈帧。
// 从首个项目帧开始连续保留，碰到非项目帧即停止，
// 避免把 runtime/testing 等框架链路打进日志。
//
// 参数说明：
//   - st：原始栈追踪
//
// 返回值：裁剪后的栈追踪
func trimProjectFrames(st stackTrace) stackTrace {
	root := projectRoot()
	if root == "" || len(st) == 0 {
		return st
	}
	frames := runtime.CallersFrames(st)
	start := -1
	end := -1
	for i := range len(st) {
		frame, more := frames.Next()
		if isWithinRoot(frame.File, root) {
			if start < 0 {
				start = i
			}
			end = i + 1
		} else if start >= 0 {
			break
		}
		if !more {
			break
		}
	}
	if start < 0 || end <= start {
		return st
	}
	return st[start:end]
}

// relativeProjectPath 将绝对路径转换为相对于项目根目录的路径。
// 使用 filepath.Rel 计算相对路径，结果使用正斜杠分隔符。
//
// 参数说明：
//   - file：文件的绝对路径
//
// 返回值：相对于项目根目录的路径，使用正斜杠
func relativeProjectPath(file string) string {
	root := projectRoot()
	if root == "" || file == "" {
		return filepath.ToSlash(file)
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

// projectRoot 获取项目根目录路径。
// 通过向上查找包含 go.mod 文件的目录来确定项目根路径。
// 结果会被缓存，后续调用直接返回缓存值。
//
// 返回值：项目根目录的绝对路径
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
//
// 参数说明：
//   - file：文件路径
//   - root：根目录路径
//
// 返回值：true 表示文件在根目录下
func isWithinRoot(file, root string) bool {
	if file == "" || root == "" {
		return false
	}
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
