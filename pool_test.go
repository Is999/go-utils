package utils_test

import (
	"bytes"
	"testing"

	"github.com/Is999/go-utils"
)

func TestNewPool_Basic(t *testing.T) {
	pool := utils.NewPool(func() *bytes.Buffer {
		return new(bytes.Buffer)
	})

	buf := pool.Get()
	if buf == nil {
		t.Fatal("Pool.Get() returned nil")
	}

	buf.WriteString("hello")
	pool.Put(buf)

	buf2 := pool.Get()
	if buf2 == nil {
		t.Fatal("Pool.Get() returned nil after Put")
	}
}

func TestNewPool_WithReset(t *testing.T) {
	resetCalls := 0 // 直接验证归还回调，后续 Get 允许得到新对象。
	pool := utils.NewPool(func() *bytes.Buffer {
		return new(bytes.Buffer)
	}, utils.WithPoolReset(func(b *bytes.Buffer) {
		resetCalls++
		if b.String() != "test data" {
			t.Fatalf("reset buffer = %q, want test data", b.String())
		}
		b.Reset()
	}))

	buf := pool.Get()
	buf.WriteString("test data")
	pool.Put(buf)
	if resetCalls != 1 {
		t.Fatalf("reset calls = %d, want 1 before Put returns", resetCalls)
	}

	buf2 := pool.Get()
	if buf2.Len() != 0 {
		t.Errorf("Buffer should be reset after Put, got len = %d", buf2.Len())
	}
}

type testPoolItem struct {
	Name  string
	Value int
}

func TestNewPool_StructType(t *testing.T) {
	resetCalls := 0 // 记录同步回调，不依赖 sync.Pool 保留对象。
	pool := utils.NewPool(func() *testPoolItem {
		return &testPoolItem{}
	}, utils.WithPoolReset(func(item *testPoolItem) {
		resetCalls++
		if item.Name != "test" || item.Value != 42 {
			t.Fatalf("reset item = %+v, want original data", item)
		}
		item.Name = ""
		item.Value = 0
	}))

	item := pool.Get()
	item.Name = "test"
	item.Value = 42

	pool.Put(item)
	if resetCalls != 1 {
		t.Fatalf("reset calls = %d, want 1 before Put returns", resetCalls)
	}

	item2 := pool.Get()
	if item2.Name != "" || item2.Value != 0 {
		t.Errorf("Item should be reset: Name=%s, Value=%d", item2.Name, item2.Value)
	}
}

func TestNewPool_NilFactoryShouldFallback(t *testing.T) {
	pool := utils.NewPool[bytes.Buffer](nil)
	buf := pool.Get()
	if buf == nil {
		t.Fatal("Pool.Get() returned nil")
	}
}

func TestPoolPutNilShouldBeSafe(t *testing.T) {
	pool := utils.NewPool(func() *bytes.Buffer {
		return new(bytes.Buffer)
	})
	pool.Put(nil)
}
