package errors_test

import (
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/Is999/go-utils/errors"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		msg  string
	}{
		{name: "001", msg: "test error"},
		{name: "002", msg: "另一个错误"},
		{name: "003", msg: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.msg)
			if err == nil {
				t.Error("New() returned nil")
			}
			if err.Error() != tt.msg {
				t.Errorf("New() error message = %v, want %v", err.Error(), tt.msg)
			}
		})
	}
}

func TestErrorf(t *testing.T) {
	tests := []struct {
		name   string
		format string
		args   []any
		want   string
	}{
		{name: "001", format: "error: %s", args: []any{"test"}, want: "error: test"},
		{name: "002", format: "code: %d, msg: %s", args: []any{100, "error"}, want: "code: 100, msg: error"},
		{name: "003", format: "no args", args: nil, want: "no args"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.Errorf(tt.format, tt.args...)
			if err == nil {
				t.Error("Errorf() returned nil")
			}
			if err.Error() != tt.want {
				t.Errorf("Errorf() error message = %v, want %v", err.Error(), tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	wrapErr := errors.New("wrap error")

	tests := []struct {
		name    string
		err     error
		msg     []string
		wantNil bool
	}{
		{name: "001", err: nil, msg: nil, wantNil: true},
		{name: "002", err: originalErr, msg: nil, wantNil: false},
		{name: "003", err: originalErr, msg: []string{"wrapped"}, wantNil: false},
		{name: "004", err: wrapErr, msg: nil, wantNil: false},
		{name: "005", err: wrapErr, msg: []string{"double wrapped"}, wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.Wrap(tt.err, tt.msg...)
			if tt.wantNil && err != nil {
				t.Errorf("Wrap() = %v, want nil", err)
			}
			if !tt.wantNil && err == nil {
				t.Error("Wrap() returned nil, want non-nil")
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	originalErr := fmt.Errorf("original error")

	tests := []struct {
		name    string
		err     error
		format  string
		args    []any
		wantNil bool
	}{
		{name: "001", err: nil, format: "test", args: nil, wantNil: true},
		{name: "002", err: originalErr, format: "wrapped: %s", args: []any{"info"}, wantNil: false},
		{name: "003", err: originalErr, format: "code: %d", args: []any{500}, wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.Wrapf(tt.err, tt.format, tt.args...)
			if tt.wantNil && err != nil {
				t.Errorf("Wrapf() = %v, want nil", err)
			}
			if !tt.wantNil && err == nil {
				t.Error("Wrapf() returned nil, want non-nil")
			}
		})
	}
}

func TestAs(t *testing.T) {
	err := errors.New("test error")
	wrapErr := errors.Wrap(err, "wrapped")

	// 测试 As 函数
	t.Run("001", func(t *testing.T) {
		var target interface{ Error() string }
		if !errors.As(err, &target) {
			t.Error("As() should return true for wrapError")
		}
	})

	t.Run("002", func(t *testing.T) {
		var target interface{ Error() string }
		if !errors.As(wrapErr, &target) {
			t.Error("As() should return true for wrapped wrapError")
		}
	})
}

func TestTrace(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "001", err: errors.New("test error")},
		{name: "002", err: errors.Wrap(fmt.Errorf("original"), "wrapped")},
		{name: "003", err: fmt.Errorf("standard error")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trace := errors.Trace(tt.err)
			if trace == nil {
				t.Error("Trace() returned nil")
			}
			// 验证实现了 slog.LogValuer 接口
			var _ slog.LogValuer = trace
		})
	}
}

func TestIs(t *testing.T) {
	err := fmt.Errorf("原始测试错误")
	err2 := errors.New("原始测试错误")
	type args struct {
		err    error
		target error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "001", args: args{
			err:    errors.Wrapf(io.EOF, "包装测试错误"),
			target: io.EOF,
		}, want: true},
		{name: "002", args: args{
			err:    errors.Wrapf(fmt.Errorf("原始测试错误"), "包装测试错误"),
			target: io.EOF,
		}, want: false},
		{name: "003", args: args{
			err:    errors.Wrapf(fmt.Errorf("原始测试错误"), "包装测试错误"),
			target: fmt.Errorf("原始测试错误"),
		}, want: false},
		{name: "004", args: args{
			err:    errors.Wrapf(err, "包装测试错误"),
			target: err,
		}, want: true},
		{name: "005", args: args{
			err:    errors.Wrapf(errors.New("原始测试错误"), "包装测试错误"),
			target: err,
		}, want: false},
		{name: "006", args: args{
			err:    errors.Wrapf(errors.New("原始测试错误"), "包装测试错误"),
			target: errors.New("原始测试错误"),
		}, want: false},
		{name: "007", args: args{
			err:    errors.Wrapf(err2, "包装测试错误"),
			target: err2,
		}, want: true},
		{name: "008", args: args{
			err:    err2,
			target: err2,
		}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.args.err, tt.args.target); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnwrap(t *testing.T) {
	err := fmt.Errorf("原始测试错误")
	type args struct {
		err error
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{name: "001", args: args{err: errors.Wrapf(io.EOF, "包装测试错误")}, wantErr: io.EOF},
		{name: "002", args: args{err: errors.Wrapf(err, "包装测试错误")}, wantErr: err},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := errors.Unwrap(tt.args.err); !errors.Is(err, tt.wantErr) {
				t.Errorf("Unwrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestErrorFormat(t *testing.T) {
	err := errors.New("test error")
	wrapErr := errors.Wrap(fmt.Errorf("original"), "wrapped")

	// 测试 %s 格式化
	s := fmt.Sprintf("%s", err)
	if s == "" {
		t.Error("Error format with s returned empty string")
	}

	// 测试 %v 格式化
	v := fmt.Sprintf("%v", err)
	if v == "" {
		t.Error("Error format with v returned empty string")
	}

	// 测试 %+v 格式化
	pv := fmt.Sprintf("%+v", err)
	if pv == "" {
		t.Error("Error format with +v returned empty string")
	}

	// 测试 %#v 格式化
	hv := fmt.Sprintf("%#v", wrapErr)
	if hv == "" {
		t.Error("Error format with #v returned empty string")
	}

	// 测试 %q 格式化
	q := fmt.Sprintf("%q", err)
	if q == "" {
		t.Error("Error format with q returned empty string")
	}
}
