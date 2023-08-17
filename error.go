package utils

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// Wrap 错误信息：包装错误文件和行号
func Wrap(err error) error {
	if err == nil {
		return nil
	}

	info := GetRuntimeInfo(2)

	if _, ok := err.(*WrapError); ok {
		err.(*WrapError).Trace.Trace = append(err.(*WrapError).Trace.Trace, info)
		return err
	}

	return &WrapError{
		Msg: err.Error(),
		Trace: &StackTrace{
			Trace: []*RuntimeInfo{info},
		},
	}
}

// Error 错误信息：包装错误文件和行号
func Error(format string, args ...any) error {
	info := GetRuntimeInfo(2)

	return &WrapError{
		Msg: Ternary(len(args) == 0, format, fmt.Sprintf(format, args...)),
		Trace: &StackTrace{
			Trace: []*RuntimeInfo{info},
		},
	}
}

// Trace error 转换 Trace
func Trace(err error) *StackTrace {
	if _, ok := err.(*WrapError); ok {
		return err.(*WrapError).Trace
	}

	return &StackTrace{}
}

type StackTrace struct {
	Trace []*RuntimeInfo
}

func (t *StackTrace) LogValue() slog.Value {
	attrs := make([]slog.Attr, len(t.Trace))
	for k, v := range t.Trace {
		attrs[k] = slog.String(strconv.Itoa(k), fmt.Sprintf("%s:%d", v.File, v.Line))
	}
	return slog.GroupValue(attrs...)
}

type WrapError struct {
	Msg   string
	Trace *StackTrace
}

func (e WrapError) Error() string {
	return e.Msg
}

func (e WrapError) String() string {
	b := strings.Builder{}
	b.WriteString("{")
	b.WriteString(`"msg":"` + e.Msg + `",`)
	b.WriteString(`"trace":[`)
	l := len(e.Trace.Trace)
	for i, s := range e.Trace.Trace {
		b.WriteString(fmt.Sprintf(`"%s:%d"`, s.File, s.Line))
		if i < l-1 {
			b.WriteString(",")
		}
	}
	b.WriteString("]}")
	return b.String()
}

func (e WrapError) LogValue() slog.Value {
	return e.Trace.LogValue()
}
