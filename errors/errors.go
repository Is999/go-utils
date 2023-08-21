package errors

import (
	"fmt"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
)

// New 错误信息：包装错误文件和行号
func New(text string) error {
	info := Caller(2)

	return &WrapError{
		Msg:   text,
		Trace: StackTrace{info},
	}
}

// Errorf 错误信息：包装错误文件和行号
func Errorf(format string, args ...any) error {
	info := Caller(2)

	return &WrapError{
		Msg:   fmt.Sprintf(format, args...),
		Trace: StackTrace{info},
	}
}

// Wrap 错误信息：包装错误文件和行号
func Wrap(err error) error {
	if err == nil {
		return nil
	}

	info := Caller(2)

	if _, ok := err.(*WrapError); ok {
		err.(*WrapError).Trace = append(err.(*WrapError).Trace, info)
		return err
	}

	return &WrapError{
		Msg:   err.Error(),
		Trace: StackTrace{info},
	}
}

// Trace 获取 error 追踪
func Trace(err error) StackTrace {
	if _, ok := err.(*WrapError); ok {
		return err.(*WrapError).Trace
	}

	return StackTrace{}
}

type StackTrace []*RuntimeInfo

func (t StackTrace) Caller() StackTrace {
	info := Caller(2)
	if t == nil {
		return StackTrace{info}
	}
	return append(t, info)
}

func (t StackTrace) LogValue() slog.Value {
	attrs := make([]slog.Attr, len(t))
	for k, v := range t {
		attrs[k] = slog.String(strconv.Itoa(k), v.String())
	}
	return slog.GroupValue(attrs...)
}

func (t StackTrace) String() string {
	b := strings.Builder{}
	b.WriteString(`{"trace":[`)
	l := len(t)
	for k, v := range t {
		b.WriteString(`"` + v.String() + `"`)
		if k < l-1 {
			b.WriteString(`,`)
		}
	}
	b.WriteString(`]}`)
	return b.String()
}

type WrapError struct {
	Msg   string
	Trace StackTrace
}

func (e WrapError) Error() string {
	return e.Msg
}

func (e WrapError) String() string {
	b := strings.Builder{}
	b.WriteString("{")
	b.WriteString(`"msg":"` + e.Msg + `",`)
	b.WriteString(strings.TrimRight(strings.TrimLeft(e.Trace.String(), `{`), `}`))
	b.WriteString("}")
	return b.String()
}

func (e WrapError) LogValue() slog.Value {
	return e.Trace.LogValue()
}

type RuntimeInfo struct {
	File string //文件名
	Line int    //行号
}

func (i RuntimeInfo) String() string {
	return fmt.Sprintf(`%s:%d`, i.File, i.Line)
}

// Caller 获取运行时行号、方法名、文件地址
func Caller(skip int) *RuntimeInfo {
	info := new(RuntimeInfo)
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		info.File = "???"
		info.Line = 0
		return info
	}

	info.File = file
	info.Line = line
	return info
}
