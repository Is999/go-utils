package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	setDefLevelOnce sync.Once

	metaPool sync.Pool

	// 日志级别，默认INFO
	defLevel = INFO

	// pid
	pid = os.Getpid()

	std = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)

	// 用户自定义输出日志
	logOutput func(meta *Meta, format string, v ...any)
)

func (s Level) String() string {
	switch s {
	case DEBUG:
		return "DBUG"
	case INFO:
		return "INFO"
	case WARNING:
		return "WARN"
	case ERROR:
		return "ERRO"
	case FATAL:
		return "FTAL"
	}
	return fmt.Sprintf("%T(%d)", s, s)
}

// SetLevel 设置日志级别，只能设置一次logLevel，请在程序入口设置
func SetLevel(level Level) {
	setDefLevelOnce.Do(func() {
		defLevel = level
	})
}

// Meta 日志数据
type Meta struct {
	// 调用日志的时间
	Time time.Time
	// 调用日志的文件
	File string
	// 调用日志的行号
	Line int
	// Depth
	Depth int
	// 日志等级
	Level Level
	// Thread ID
	Thread int
}

// metaPoolGet 获取meta pool
func metaPoolGet() (any, *Meta) {
	if metai := metaPool.Get(); metai != nil {
		return metai, metai.(*Meta)
	}
	meta := new(Meta)
	return meta, meta
}

// SetLogOutput 设置记录日志方法
func SetLogOutput(output func(meta *Meta, format string, v ...any)) {
	logOutput = output
}

// SetDefOutput 设置输出方式（使用标准库的 `log` 包来记录日志，设置该值才有意义）
func SetDefOutput(w io.Writer) {
	std.SetOutput(w)
}

// Output 输出日志
func Output(depth int, level Level, format string, v ...any) {
	// 判断日志级别
	if level < defLevel || level >= DISABLE {
		return
	}

	// 获取文件，行号
	_, file, line, ok := runtime.Caller(depth + 1)
	if !ok {
		file = "???"
		line = 0
	}

	metai, meta := metaPoolGet()

	*meta = Meta{
		Time:   time.Now(),
		File:   file,
		Line:   line,
		Depth:  depth + 1,
		Thread: pid,
		Level:  level,
	}

	// 输出日志
	if logOutput != nil {

		logOutput(meta, format, v...)
	} else {
		defOutput(meta, format, v...)
	}
	metaPool.Put(metai)
}

// defOutput 默认输出
func defOutput(meta *Meta, format string, v ...any) {
	std.Output(meta.Depth+2, fmt.Sprintf(fmt.Sprintf("[%s] id-%d %s:%d %s", meta.Level.String(), meta.Thread, meta.File, meta.Line, format), v...))
}

// Debug 记录debug级别日志
func Debug(v ...any) {
	Output(1, DEBUG, Format(v), v...)
}

// Debugf 记录debug级别日志
func Debugf(format string, v ...any) {
	Output(1, DEBUG, format, v...)
}

// Info 记录info级别日志
func Info(v ...any) {
	Output(1, INFO, Format(v), v...)
}

// Infof 记录info级别日志
func Infof(format string, v ...any) {
	Output(1, INFO, format, v...)
}

// Warn 记录warning级别日志
func Warn(v ...any) {
	Output(1, WARNING, Format(v), v...)
}

// Warnf 记录warning级别日志
func Warnf(format string, v ...any) {
	Output(1, WARNING, format, v...)
}

// Error 记录error级别日志
func Error(v ...any) {
	Output(1, ERROR, Format(v), v...)
}

// Errorf 记录error级别日志
func Errorf(format string, v ...any) {
	Output(1, ERROR, format, v...)
}

// Fatal 记录fatal级别日志, 并调用os.Exit()
func Fatal(v ...any) {
	Output(1, FATAL, Format(v), v...)
	os.Exit(1)
}

// Fatalf 记录fatal级别日志, 并调用os.Exit()
func Fatalf(format string, v ...any) {
	Output(1, FATAL, format, v...)
	os.Exit(1)
}

// Format 根据参数生成Format
func Format(args []any) string {
	if len(args) == 0 {
		return "\n"
	}

	b := make([]byte, 0, len(args)*3)
	for range args {
		b = append(b, "%v "...)
	}
	b[len(b)-1] = '\n' // Replace the last space with a newline.
	return string(b)
}
