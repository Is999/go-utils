package utils

import (
	"fmt"
	"log/slog"
	"runtime"
	"strconv"
)

// Wrap 错误信息：包装错误文件和行号
func Wrap(err error) error {
	if err == nil {
		return nil
	}

	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}

	if _, ok := err.(*WrapError); ok {
		err.(*WrapError).Trace = append(err.(*WrapError).Trace, fmt.Sprintf("%s:%d", file, line))
		return err
	}

	return &WrapError{
		Msg:   err.Error(),
		Trace: []string{fmt.Sprintf("%s:%d", file, line)},
	}
}

func LogError(err error) {
	if err == nil {
		return
	}

	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}

	if _, ok := err.(*WrapError); ok {
		err.(*WrapError).Trace = append(err.(*WrapError).Trace, fmt.Sprintf("%s:%d", file, line))
		slog.Error(err.(*WrapError).Msg, "trace", err)
	} else {
		slog.Error(err.Error(), "trace", fmt.Sprintf("%s:%d", file, line))
	}
}

// Error 错误信息：包装错误文件和行号
func Error(format string, args ...any) error {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}

	return &WrapError{
		Msg:   Ternary(len(args) == 0, format, fmt.Sprintf(format, args...)),
		Trace: []string{fmt.Sprintf("%s:%d", file, line)},
	}
}

type WrapError struct {
	Msg   string
	Trace []string
}

func (e WrapError) Error() string {
	return e.String()
}

func (e WrapError) String() string {
	return fmt.Sprintf("%s %+v", e.Msg, e.Trace)
}

func (e WrapError) LogValue() slog.Value {
	attrs := make([]slog.Attr, len(e.Trace))
	for k, v := range e.Trace {
		attrs[k] = slog.String(strconv.Itoa(k), v)
	}
	return slog.GroupValue(attrs...)
}
