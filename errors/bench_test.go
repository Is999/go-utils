package errors_test

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"testing"

	errutils "github.com/Is999/go-utils/errors"
)

var (
	benchErr  error
	benchBool bool
	benchText string
	benchCode int
)

func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchErr = errutils.New("test error")
	}
}

func BenchmarkErrorf(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchErr = errutils.Errorf("error: %d, %s", i, "test")
	}
}

func BenchmarkWrap(b *testing.B) {
	original := errutils.New("original")
	for i := 0; i < b.N; i++ {
		benchErr = errutils.Wrap(original, "wrapped")
	}
}

func BenchmarkWrapf(b *testing.B) {
	original := errutils.New("original")
	for i := 0; i < b.N; i++ {
		benchErr = errutils.Wrapf(original, "wrapped: %d", i)
	}
}

func BenchmarkWithMessage(b *testing.B) {
	original := errors.New("original")
	for i := 0; i < b.N; i++ {
		benchErr = errutils.WithMessage(original, "message")
	}
}

func BenchmarkWithMessagef(b *testing.B) {
	original := errors.New("original")
	for i := 0; i < b.N; i++ {
		benchErr = errutils.WithMessagef(original, "message: %d", i)
	}
}

func BenchmarkWithCode(b *testing.B) {
	original := errors.New("original")
	for i := 0; i < b.N; i++ {
		benchErr = errutils.WithCode(original, 409)
	}
}

func BenchmarkWrapMultiple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := errutils.New("original")
		err = errutils.Wrap(err, "level1")
		err = errutils.Wrap(err, "level2")
		err = errutils.Wrap(err, "level3")
		benchErr = err
	}
}

func BenchmarkIs(b *testing.B) {
	source := &typedError{msg: "source"}
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(source, "inner"), "middle"), "outer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = errutils.Is(err, source)
	}
}

func BenchmarkAs(b *testing.B) {
	source := &typedError{msg: "source"}
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(source, "inner"), "middle"), "outer")
	var target *typedError
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchBool = errutils.As(err, &target)
	}
}

func BenchmarkType(b *testing.B) {
	source := &typedError{msg: "source"}
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(source, "inner"), "middle"), "outer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, benchBool = errutils.AsType[*typedError](err)
	}
}

func BenchmarkSource(b *testing.B) {
	source := &typedError{msg: "source"}
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(source, "inner"), "middle"), "outer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchErr = errutils.Source(err)
	}
}

func BenchmarkCause(b *testing.B) {
	source := &typedError{msg: "source"}
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(source, "inner"), "middle"), "outer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchErr = errutils.Cause(err)
	}
}

func BenchmarkChain(b *testing.B) {
	source := &typedError{msg: "source"}
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(source, "inner"), "middle"), "outer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchCode = len(errutils.Chain(err))
	}
}

func BenchmarkSources(b *testing.B) {
	left := errutils.Wrap(io.EOF, "left")
	right := &typedError{msg: "right"}
	joined := errutils.Join(left, right)
	err := errutils.Wrap(joined, "top")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchCode = len(errutils.Sources(err))
	}
}

func BenchmarkErrorMethod(b *testing.B) {
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(&typedError{msg: "source"}, "inner"), "middle"), "outer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchText = err.Error()
	}
}

func BenchmarkFormatS(b *testing.B) {
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(&typedError{msg: "source"}, "inner"), "middle"), "outer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchText = fmt.Sprintf("%s", err)
	}
}

func BenchmarkFormatV(b *testing.B) {
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(&typedError{msg: "source"}, "inner"), "middle"), "outer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchText = fmt.Sprintf("%v", err)
	}
}

func BenchmarkFormatPlusV(b *testing.B) {
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(&typedError{msg: "source"}, "inner"), "middle"), "outer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchText = fmt.Sprintf("%+v", err)
	}
}

func BenchmarkFormatHashV(b *testing.B) {
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(&typedError{msg: "source"}, "inner"), "middle"), "outer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchText = fmt.Sprintf("%#v", err)
	}
}

func BenchmarkTraceString(b *testing.B) {
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(&typedError{msg: "source"}, "inner"), "middle"), "outer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchText = errutils.TraceString(err)
	}
}

func BenchmarkTraceJSON(b *testing.B) {
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(&typedError{msg: "source"}, "inner"), "middle"), "outer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchText = errutils.TraceJSON(err)
	}
}

// BenchmarkConcurrentRead 由测试框架分配恰好 b.N 次读取，避免固定分组遗漏余数。
func BenchmarkConcurrentRead(b *testing.B) {
	source := &typedError{msg: "source"}
	err := errutils.Wrap(errutils.Wrap(errutils.Wrap(source, "inner"), "middle"), "outer")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// 各 goroutine 独占结果，避免用汇总锁干扰并发读取成本。
		var localErr error
		var localBool bool
		var textSize int
		for pb.Next() {
			textSize += len(err.Error())
			textSize += len(fmt.Sprintf("%+v", err))
			textSize += len(errutils.TraceString(err))
			localBool = errutils.Is(err, source)
			localErr = errutils.Source(err)
		}
		runtime.KeepAlive(localErr)
		runtime.KeepAlive(localBool)
		runtime.KeepAlive(textSize)
	})
}

func BenchmarkCompareToStdlib(b *testing.B) {
	b.Run("New", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchErr = errors.New("test error")
		}
	})
	b.Run("Wrap", func(b *testing.B) {
		original := errors.New("original")
		for i := 0; i < b.N; i++ {
			benchErr = fmt.Errorf("wrapped: %w", original)
		}
	})
	b.Run("Is", func(b *testing.B) {
		source := &typedError{msg: "source"}
		err := fmt.Errorf("outer: %w", fmt.Errorf("middle: %w", fmt.Errorf("inner: %w", source)))
		for i := 0; i < b.N; i++ {
			benchBool = errors.Is(err, source)
		}
	})
	b.Run("As", func(b *testing.B) {
		source := &typedError{msg: "source"}
		err := fmt.Errorf("outer: %w", fmt.Errorf("middle: %w", fmt.Errorf("inner: %w", source)))
		var target *typedError
		for i := 0; i < b.N; i++ {
			benchBool = errors.As(err, &target)
		}
	})
}
