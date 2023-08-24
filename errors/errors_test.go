package errors_test

import (
	"fmt"
	"github.com/Is999/go-utils/errors"
	"io"
	"testing"
)

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
			if err := errors.Unwrap(tt.args.err); err != tt.wantErr {
				t.Errorf("Unwrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
