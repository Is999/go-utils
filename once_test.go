package utils_test

import (
	"context"
	"errors"
	"strings"
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
	_ = o.Do(func() error {
		return nil
	}, 3)

	// 后续传入会失败的函数，也不能替换已缓存的成功结果。
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
	wantErr := errors.New("operation failed")
	for _, result := range []string{"success", "failure", "panic"} {
		t.Run(result, func(t *testing.T) {
			var o utils.Once
			started := make(chan struct{})
			release := make(chan struct{})
			done := make(chan error, 1)
			go func() {
				done <- o.Do(func() error {
					close(started)
					<-release
					if result == "panic" {
						panic("boom")
					}
					if result == "failure" {
						return wantErr
					}
					return nil
				}, 1)
			}()
			<-started

			// 执行与 Reset 均完成后，下一轮必须重新执行并缓存结果。
			resetDone := make(chan struct{})
			go func() {
				o.Reset()
				close(resetDone)
			}()
			close(release)
			err := <-done
			<-resetDone
			switch result {
			case "success":
				if err != nil {
					t.Fatal(err)
				}
			case "failure":
				if !errors.Is(err, wantErr) {
					t.Fatalf("Once.Do() error = %v, want original failure", err)
				}
			case "panic":
				if err == nil || !strings.Contains(err.Error(), "panic: boom") {
					t.Fatalf("Once.Do() error = %v, want recovered panic", err)
				}
			}

			calls := 0
			for range 2 {
				if err := o.Do(func() error { calls++; return nil }, 1); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 1 {
				t.Fatalf("calls after Reset() = %d, want 1", calls)
			}
		})
	}
}

func TestOnce_Do_Concurrent(t *testing.T) {
	var o utils.Once
	var wg sync.WaitGroup
	errCh := make(chan error, 10)
	var callCount atomic.Int32

	for range 10 {
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
	// 断言失败时也释放执行方，避免遗留等待 release 的 goroutine。
	t.Cleanup(func() {
		close(release)
		if err := <-done; err != nil {
			t.Errorf("executing Once.DoContext() error = %v, want nil", err)
		}
	})

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

func TestOnceFailureNamesOriginalFunction(t *testing.T) {
	// Do 内部会适配函数签名，错误仍须标记调用方函数，而不是内部适配闭包。
	wantErr := errors.New("operation failed")
	fn := func() error { return wantErr }
	var once utils.Once
	err := once.Do(fn, 1)
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), utils.GetFunctionName(fn)+" 尝试 1 次后依然失败") {
		t.Fatalf("Once.Do() error = %v, want original function and cause", err)
	}
	if cached := once.Do(fn, 1); cached != err {
		t.Fatalf("Once.Do() cached error = %v, want same error instance", cached)
	}

	once.Reset()
	panicFn := func() error { panic("boom") }
	err = once.Do(panicFn, 1)
	if err == nil || !strings.Contains(err.Error(), utils.GetFunctionName(panicFn)+" panic: boom") {
		t.Fatalf("Once.Do() panic error = %v, want original function name", err)
	}
}

func BenchmarkOnceCached(b *testing.B) {
	// 先完成初始化，单独测量后续重复调用的结果复用成本。
	var once utils.Once
	fn := func() error { return nil }
	if err := once.Do(fn, 1); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := once.Do(fn, 1); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOnceCachedParallel(b *testing.B) {
	// 多个 goroutine 共享同一已完成实例，记录缓存读取的锁竞争成本。
	var once utils.Once
	fn := func() error { return nil }
	if err := once.Do(fn, 1); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := once.Do(fn, 1); err != nil {
				b.Error(err)
				return
			}
		}
	})
}
