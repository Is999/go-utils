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

// New 返回带追踪的error
func New(msg string) error {
	return newWrapError(msg, nil)
}

// Errorf 使用格式化传入错误消息，返回带追踪的error
func Errorf(format string, args ...any) error {
	return newWrapError(fmt.Sprintf(format, args...), nil)
}

// Wrap 对error接口进行包装
//  1. 如果error是一个nil则返回nil
//  2. 如果仅有err参数没有msg参数，进入 3、4步骤，否则包装成一个*wrapError类型返回
//  3. 如果error是*wrapError类型，返回原error
//  4. 其它类型的error则将error.Error信息作为msg, 包装成一个*wrapError类型返回
func Wrap(err error, msg ...string) error {
	if err == nil {
		return nil
	}

	if len(msg) > 0 {
		return newWrapError(msg[0], err)
	}

	var we *wrapError
	if As(err, &we) {
		return we
	}

	return newWrapError(err.Error(), err)
}

// Wrapf 对error接口进行包装，如果error是一个nil则返回nil
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}

	return newWrapError(fmt.Sprintf(format, args...), err)
}

// Is 检查err和target是否匹配， 同标准库errors.Is
//
//	使用到 Unwrap() error 接口
//	使用到 Is(error) bool 接口
func Is(err, target error) bool { return errors.Is(err, target) }

// As 检查err和target第一个匹配的err并赋值， 同标准库errors.As
//
//	使用到 Unwrap() error 接口
//	使用到 As(any) bool 接口
func As(err error, target any) bool { return errors.As(err, target) }

// Unwrap 对error Wrap的反向操作
//
//	使用到 Unwrap() error 接口
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// Trace 获取error日志追踪，返回 slog.LogValuer 接口
func Trace(err error) slog.LogValuer {
	if err == nil {
		return nil
	}
	var we *wrapError
	if errors.As(err, &we) {
		return we
	}

	return newWrapError(err.Error(), nil)
}

// TraceString 获取error日志追踪的字符串表示，兼容任意第三方日志库（如 zap、logrus 等）。
// 使用 fmt.Sprintf("%+v", err) 可获取相同信息，TraceString 提供更简洁的调用方式。
func TraceString(err error) string {
	if err == nil {
		return ""
	}
	var we *wrapError
	if errors.As(err, &we) {
		return we.String()
	}

	return newWrapError(err.Error(), nil).String()
}

// TraceJSON 获取error日志追踪的完整 JSON 字符串表示（含嵌套 wrap 信息），兼容任意第三方日志库。
func TraceJSON(err error) string {
	if err == nil {
		return ""
	}
	var we *wrapError
	if errors.As(err, &we) {
		return we.GoString()
	}

	return newWrapError(err.Error(), nil).GoString()
}

type stackTrace []uintptr

func (st stackTrace) callers(skip int) []*info {
	pcs := diff(st, callers(skip+1))
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

// LogValue 实现slog.LogValuer 接口
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
		b.WriteString(fmt.Sprintf(`"%s"`, v.String()))
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

// Error 实现Error接口
func (e *wrapError) Error() string {
	return e.msg
}

// Unwrap 实现Unwrap接口, 返回wrapError的子error
func (e *wrapError) Unwrap() error {
	return e.err
}

// Is 实现Is接口
//
//	e与target相等，两者类型相同并且msg相同并且第一个stackTrace相同返回true，否则返回false
func (e *wrapError) Is(target error) bool {
	if we, ok := target.(*wrapError); ok {
		return we.msg == e.msg && we.stackTrace[0] == e.stackTrace[0]
	}
	return false
}

// As 实现As接口
//
//	e与target两者类型相同返回true，否则返回false
func (e *wrapError) As(target any) bool {
	_, ok := target.(*wrapError)
	return ok
}

func (e *wrapError) traceMsg(depth int) string {
	if e.err == nil {
		return e.msg
	}
	if we, ok := (e.err).(*wrapError); ok {
		return fmt.Sprintf("%s; wrap-%d=%s", e.msg, depth, we.traceMsg(depth+1))
	}
	return fmt.Sprintf("%s; wrap-%d=%s", e.msg, depth, e.err.Error())
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
			b.WriteString(fmt.Sprintf(`,"err":%q`, e.err.Error()))
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

// LogValue 实现 slog.LogValuer 接口，对error的日志追踪
func (e *wrapError) LogValue() slog.Value {
	return slog.GroupValue(e.logGroup())
}

func (e *wrapError) logGroup() slog.Attr {
	attr := make([]any, 0, 4)
	attr = append(attr, slog.String("msg", e.msg))
	attr = append(attr, slog.Any("trace", e.stackTrace.LogValue()))
	if we, ok := (e.err).(*wrapError); ok {
		attr = append(attr, we.logGroup())
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

func (i *info) String() string {
	return fmt.Sprintf(`%s (%s:%d)`, i.name, i.file, i.line)
}

// diff 计算s1与s2的差集
func diff[T cmp.Ordered](s1, s2 []T) []T {
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
