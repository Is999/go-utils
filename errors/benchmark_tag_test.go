package errors_test

import (
	"fmt"
	"testing"

	"github.com/Is999/go-utils/errors"
)

// 基准测试：Tag vs Wrap 性能对比
func BenchmarkTagStdlibError(b *testing.B) {
	originalErr := fmt.Errorf("stdlib error")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.Tag(originalErr)
	}
}

func BenchmarkWrapStdlibError(b *testing.B) {
	originalErr := fmt.Errorf("stdlib error")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.Wrap(originalErr, "wrapped")
	}
}

func BenchmarkTagTrackedError(b *testing.B) {
	trackedErr := errors.New("already tracked")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.Tag(trackedErr)
	}
}

func BenchmarkWrapTrackedError(b *testing.B) {
	trackedErr := errors.New("already tracked")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.Wrap(trackedErr, "wrapped")
	}
}

// 测试层层包装的性能
func BenchmarkTagLayeredWrapping(b *testing.B) {
	originalErr := fmt.Errorf("original error")
	wrapped1 := errors.Tag(originalErr)
	wrapped2 := errors.Tag(wrapped1)
	wrapped3 := errors.Tag(wrapped2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.Tag(wrapped3)
	}
}

func BenchmarkWrapLayeredWrapping(b *testing.B) {
	originalErr := fmt.Errorf("original error")
	wrapped1 := errors.Wrap(originalErr, "layer1")
	wrapped2 := errors.Wrap(wrapped1, "layer2")
	wrapped3 := errors.Wrap(wrapped2, "layer3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.Wrap(wrapped3, "layer4")
	}
}

// 测试HasStack的性能
func BenchmarkHasStackStdlib(b *testing.B) {
	originalErr := fmt.Errorf("stdlib error")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.HasStack(originalErr)
	}
}

func BenchmarkHasStackTracked(b *testing.B) {
	trackedErr := errors.New("tracked error")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.HasStack(trackedErr)
	}
}
