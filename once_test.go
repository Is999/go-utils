package utils_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Is999/go-utils"
)

func TestOnce_Do_Success(t *testing.T) {
	var o utils.Once
	callCount := 0
	err := o.Do(func() error {
		callCount++
		return nil
	}, 3)
	if err != nil {
		t.Errorf("Once.Do() error = %v, want nil", err)
	}
	if callCount != 1 {
		t.Errorf("Once.Do() callCount = %d, want 1", callCount)
	}
}

func TestOnce_Do_RetryThenSuccess(t *testing.T) {
	var o utils.Once
	callCount := 0
	err := o.Do(func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	}, 5)
	if err != nil {
		t.Errorf("Once.Do() error = %v, want nil", err)
	}
	if callCount != 3 {
		t.Errorf("Once.Do() callCount = %d, want 3", callCount)
	}
}

func TestOnce_Do_MaxRetriesExhausted(t *testing.T) {
	var o utils.Once
	callCount := 0
	err := o.Do(func() error {
		callCount++
		return errors.New("persistent error")
	}, 3)
	if err == nil {
		t.Error("Once.Do() error = nil, want error")
	}
	if callCount != 3 {
		t.Errorf("Once.Do() callCount = %d, want 3", callCount)
	}
}

func TestOnce_Do_AlreadyDone(t *testing.T) {
	var o utils.Once
	// 第一次成功执行
	_ = o.Do(func() error {
		return nil
	}, 3)

	// 第二次应直接返回nil
	callCount := 0
	err := o.Do(func() error {
		callCount++
		return errors.New("should not be called")
	}, 3)
	if err != nil {
		t.Errorf("Once.Do() second call error = %v, want nil", err)
	}
	if callCount != 0 {
		t.Errorf("Once.Do() second call callCount = %d, want 0", callCount)
	}
}

func TestOnce_Reset(t *testing.T) {
	var o utils.Once
	callCount := 0

	// 第一次执行
	_ = o.Do(func() error {
		callCount++
		return nil
	}, 3)
	if callCount != 1 {
		t.Errorf("After first Do, callCount = %d, want 1", callCount)
	}

	// 重置后可以再次执行
	o.Reset()
	_ = o.Do(func() error {
		callCount++
		return nil
	}, 3)
	if callCount != 2 {
		t.Errorf("After Reset and second Do, callCount = %d, want 2", callCount)
	}
}

func TestOnce_Do_Concurrent(t *testing.T) {
	var o utils.Once
	var wg sync.WaitGroup
	errCh := make(chan error, 10)
	var callCount atomic.Int32

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := o.Do(func() error {
				callCount.Add(1)
				return nil
			}, 3)
			errCh <- err
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Errorf("Once.Do() concurrent error = %v, want nil", err)
		}
	}
	if got := callCount.Load(); got != 1 {
		t.Errorf("Once.Do() concurrent callCount = %d, want 1", got)
	}
}

func TestOnce_Do_PanicShouldReturnError(t *testing.T) {
	var o utils.Once
	err := o.Do(func() error {
		panic("boom")
	}, 3)
	if err == nil {
		t.Fatal("Once.Do() panic expected error")
	}
}

func TestOnce_Do_MaxRetriesZeroShouldRunOnce(t *testing.T) {
	var o utils.Once
	callCount := 0
	err := o.Do(func() error {
		callCount++
		return errors.New("once error")
	}, 0)
	if err == nil {
		t.Fatal("Once.Do() expected error")
	}
	if callCount != 1 {
		t.Fatalf("Once.Do() callCount = %d, want 1", callCount)
	}
}

func TestOnce_DoContext_CancelWaitingCaller(t *testing.T) {
	var o utils.Once
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- o.DoContext(context.Background(), func(ctx context.Context) error {
			close(started)
			<-release
			return nil
		}, 1)
	}()

	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := o.DoContext(ctx, func(ctx context.Context) error {
		t.Fatal("waiting caller should not execute function")
		return nil
	}, 1)
	if err == nil {
		t.Fatal("Once.DoContext() error = nil, want error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Once.DoContext() error = %v, want context.DeadlineExceeded", err)
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("executing Once.DoContext() error = %v, want nil", err)
	}
}

func TestOnce_DoContext_CancelStopsRetryBackoff(t *testing.T) {
	var o utils.Once
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	err := o.DoContext(ctx, func(ctx context.Context) error {
		callCount++
		cancel()
		return errors.New("temporary error")
	}, 5)
	if err == nil {
		t.Fatal("Once.DoContext() error = nil, want error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Once.DoContext() error = %v, want context.Canceled", err)
	}
	if callCount != 1 {
		t.Fatalf("Once.DoContext() callCount = %d, want 1", callCount)
	}
}
