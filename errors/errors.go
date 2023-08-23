package errors

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
)

func New(msg string) error {
	return newWrapError(msg, nil)
}

func Errorf(format string, args ...any) error {
	return newWrapError(fmt.Sprintf(format, args...), nil)
}

func Wrap(err error, msg ...string) error {
	if err == nil {
		return nil
	}

	if len(msg) > 0 {
		return newWrapError(msg[0], err)
	}

	var we *wrapError
	if errors.As(err, &we) {
		return we
	}

	return newWrapError(err.Error(), err)
}

// Trace 获取 err 追踪
func Trace(err error) slog.LogValuer {
	var we *wrapError
	if errors.As(err, &we) {
		return we
	}

	return newWrapError(err.Error(), nil)
}

type stackTrace []uintptr

func (st stackTrace) callers(skip int) []*info {
	pcs := ArrDiff(st, callers(skip+1))
	frames := runtime.CallersFrames(pcs)
	l := len(st)
	infos := make([]*info, 0, l)
	for i := 0; i < l; i++ {
		frame, more := frames.Next()
		infos = append(infos, &info{
			name: frame.Function,
			file: frame.File,
			line: frame.Line,
		})
		if !more {
			break
		}
	}
	return infos
}

func (st stackTrace) LogValue() slog.Value {
	infos := st.callers(1)
	attrs := make([]slog.Attr, len(infos))
	for i, v := range infos {
		attrs[i] = slog.String(strconv.Itoa(i), v.String())
	}
	return slog.GroupValue(attrs...)
}

func (st stackTrace) String() string {
	infos := st.callers(1)
	l := len(infos)

	b := strings.Builder{}
	b.WriteString(`[`)

	for i, v := range infos {
		b.WriteString(fmt.Sprintf("%q", v.String()))
		if i < l-1 {
			b.WriteString(`,`)
		}
	}

	b.WriteString(`]`)
	return b.String()
}

func newWrapError(msg string, err error) *wrapError {
	return &wrapError{
		msg:        msg,
		err:        err,
		stackTrace: callers(2),
	}
}

type wrapError struct {
	msg        string
	err        error
	stackTrace stackTrace
}

func (e *wrapError) Error() string {
	return e.msg
}

func (e *wrapError) Unwrap() error {
	return e.err
}

func (e *wrapError) Is(target error) bool {
	if we, ok := (target).(*wrapError); ok {
		return we == e || (we.msg == e.msg && we.stackTrace[0] == e.stackTrace[0])
	}
	return false
}

func (e *wrapError) As(target any) bool {
	if _, ok := (target).(*wrapError); ok {
		return true
	}
	return false
}

func (e *wrapError) traceMsg(depth int) string {
	if e.err == nil {
		return e.msg
	}
	if we, ok := (e.err).(*wrapError); ok {
		return fmt.Sprintf("%s; wrap-%d=%s", e.msg, depth, we.traceMsg(depth+1))
	}
	return e.msg
}

func (e *wrapError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = io.WriteString(s, e.String())
			return
		}
		if s.Flag('#') {
			_, _ = io.WriteString(s, e.GoString())
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, e.traceMsg(0))
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.traceMsg(0))
	}
}

func (e *wrapError) String() string {
	b := strings.Builder{}
	b.WriteString("{")
	b.WriteString(fmt.Sprintf(`"msg":%q`, e.msg))
	b.WriteString(`,"trace":` + e.stackTrace.String())
	b.WriteString("}")
	return b.String()
}

func (e *wrapError) GoString() string {
	b := strings.Builder{}
	b.WriteString("{")
	b.WriteString(fmt.Sprintf(`"msg":%q`, e.msg))
	b.WriteString(`,"trace":` + e.stackTrace.String())
	if e.err != nil {
		if we, ok := (e.err).(*wrapError); ok {
			b.WriteString(`,"err":` + we.GoString())
		} else {
			b.WriteString(`,"err":"` + e.err.Error() + `"`)
		}
	}
	b.WriteString("}")
	return b.String()
}

func (e *wrapError) MarshalJSON() ([]byte, error) {
	return []byte(e.GoString()), nil
}

func (e *wrapError) MarshalText() ([]byte, error) {
	return []byte(e.GoString()), nil
}

func (e *wrapError) LogValue() slog.Value {
	return slog.GroupValue(e.Group())
}

func (e *wrapError) Group() slog.Attr {
	attr := make([]any, 0, 4)
	attr = append(attr, slog.String("msg", e.msg))
	attr = append(attr, slog.Any("trace", e.stackTrace.LogValue()))
	if we, ok := (e.err).(*wrapError); ok {
		attr = append(attr, we.Group())
	}
	return slog.Group("wrap", attr...)
}

func callers(skip int) stackTrace {
	var pcs [100]uintptr
	n := runtime.Callers(skip+2, pcs[:])
	return pcs[0:n]
}

type info struct {
	name string
	file string //文件名
	line int    //行号
}

func (i info) String() string {
	return fmt.Sprintf(`%s (%s:%d)`, i.name, i.file, i.line)
}

// ArrDiff 计算s1与s2的差集
func ArrDiff[T cmp.Ordered](s1, s2 []T) []T {
	m := make(map[T]struct{}, len(s2))
	for i := 0; i < len(s2); i++ {
		m[s2[i]] = struct{}{}
	}
	var s = make([]T, 0, len(s1))
	for i := 0; i < len(s1); i++ {
		if _, ok := m[s1[i]]; !ok {
			s = append(s, s1[i])
		}
	}
	return s
}
