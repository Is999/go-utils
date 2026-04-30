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

var (
	projectRootOnce sync.Once
	projectRootPath string
)

// stackTrace 只保存程序计数器，渲染时再转为文件名和行号。
type stackTrace []uintptr

// LogValue 用于 slog 输出结构化 trace。
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

// stackError 是真正带追踪栈的错误节点。
type stackError struct {
	msg   string
	err   error
	trace stackTrace
}

func newStackError(msg string, err error) *stackError {
	return &stackError{msg: msg, err: err, trace: callers(2)}
}

func (e *stackError) Error() string {
	if e == nil {
		return ""
	}
	return e.msg
}

func (e *stackError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *stackError) Is(target error) bool {
	te, ok := target.(*stackError)
	return ok && e == te
}

func (e *stackError) String() string   { return TraceString(e) }
func (e *stackError) GoString() string { return TraceJSON(e) }

func (e *stackError) Format(s fmt.State, verb rune) {
	formatError(s, verb, e)
}

func (e *stackError) MarshalJSON() ([]byte, error) {
	return []byte(TraceJSON(e)), nil
}

func (e *stackError) MarshalText() ([]byte, error) {
	return []byte(TraceString(e)), nil
}

func callers(skip int) stackTrace {
	var pcs [maxStackDepth]uintptr
	n := runtime.Callers(skip+2, pcs[:StackDepth()])
	return trimProjectFrames(pcs[:n])
}

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

// trimProjectFrames 只保留项目内、且从首个项目帧开始连续的一段栈，避免把 runtime/testing 链路打进日志。
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
