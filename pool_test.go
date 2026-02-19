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

	// Get
	buf := pool.Get()
	if buf == nil {
		t.Fatal("Pool.Get() returned nil")
	}

	// Write and Put
	buf.WriteString("hello")
	pool.Put(buf)

	// Get again
	buf2 := pool.Get()
	if buf2 == nil {
		t.Fatal("Pool.Get() returned nil after Put")
	}
}

func TestNewPool_WithReset(t *testing.T) {
	pool := utils.NewPool(func() *bytes.Buffer {
		return new(bytes.Buffer)
	}, utils.WithPoolReset(func(b *bytes.Buffer) {
		b.Reset()
	}))

	// Get, write data, put back
	buf := pool.Get()
	buf.WriteString("test data")
	if buf.Len() == 0 {
		t.Error("Buffer should have data")
	}
	pool.Put(buf) // should be reset

	// Get again - buffer should have been reset
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
	pool := utils.NewPool(func() *testPoolItem {
		return &testPoolItem{}
	}, utils.WithPoolReset(func(item *testPoolItem) {
		item.Name = ""
		item.Value = 0
	}))

	item := pool.Get()
	item.Name = "test"
	item.Value = 42

	if item.Name != "test" || item.Value != 42 {
		t.Errorf("Unexpected values: Name=%s, Value=%d", item.Name, item.Value)
	}

	pool.Put(item)

	item2 := pool.Get()
	if item2.Name != "" || item2.Value != 0 {
		t.Errorf("Item should be reset: Name=%s, Value=%d", item2.Name, item2.Value)
	}
}
