package errors_test

import (
	"fmt"
	"testing"

	"github.com/Is999/go-utils/errors"
)

// Tag 与 Wrap 分别测量保留消息和追加消息的成本；已有栈时 Tag 返回原错误，Wrap 仍创建包装。
func BenchmarkTagStdlibError(b *testing.B) {
	originalErr := fmt.Errorf("stdlib error")
	b.ResetTimer()
	for range b.N {
		benchErr = errors.Tag(originalErr)
	}
}

func BenchmarkWrapStdlibError(b *testing.B) {
	originalErr := fmt.Errorf("stdlib error")
	b.ResetTimer()
	for range b.N {
		benchErr = errors.Wrap(originalErr, "wrapped")
	}
}

func BenchmarkTagTrackedError(b *testing.B) {
	trackedErr := errors.New("already tracked")
	b.ResetTimer()
	for range b.N {
		benchErr = errors.Tag(trackedErr)
	}
}

func BenchmarkWrapTrackedError(b *testing.B) {
	trackedErr := errors.New("already tracked")
	b.ResetTimer()
	for range b.N {
		benchErr = errors.Wrap(trackedErr, "wrapped")
	}
}

// 连续 Tag 不增加已有栈的错误层级；对应 Wrap 基准保留逐层附加的消息。
func BenchmarkTagLayeredWrapping(b *testing.B) {
	originalErr := fmt.Errorf("original error")
	wrapped1 := errors.Tag(originalErr)
	wrapped2 := errors.Tag(wrapped1)
	wrapped3 := errors.Tag(wrapped2)

	b.ResetTimer()

	for range b.N {
		benchErr = errors.Tag(wrapped3)
	}
}

func BenchmarkWrapLayeredWrapping(b *testing.B) {
	originalErr := fmt.Errorf("original error")
	wrapped1 := errors.Wrap(originalErr, "layer1")
	wrapped2 := errors.Wrap(wrapped1, "layer2")
	wrapped3 := errors.Wrap(wrapped2, "layer3")

	b.ResetTimer()

	for range b.N {
		benchErr = errors.Wrap(wrapped3, "layer4")
	}
}

// HasStack 分开测量无栈叶子和已采集栈的错误。
func BenchmarkHasStackStdlib(b *testing.B) {
	originalErr := fmt.Errorf("stdlib error")
	b.ResetTimer()
	for range b.N {
		benchBool = errors.HasStack(originalErr)
	}
}

func BenchmarkHasStackTracked(b *testing.B) {
	trackedErr := errors.New("tracked error")
	b.ResetTimer()
	for range b.N {
		benchBool = errors.HasStack(trackedErr)
	}
}
