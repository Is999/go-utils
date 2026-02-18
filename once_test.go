package utils_test

import (
	"errors"
	"sync"
	"testing"

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

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := o.Do(func() error {
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
}
