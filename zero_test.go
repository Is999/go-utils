package utils

import (
	"reflect"
	"testing"
)

func TestZeroPadding(t *testing.T) {
	type args struct {
		data      []byte
		blockSize int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{name: "001", args: args{data: []byte{}, blockSize: 8}, want: []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{name: "002", args: args{data: []byte("A"), blockSize: 8}, want: []byte{65, 0, 0, 0, 0, 0, 0, 0}},
		{name: "003", args: args{data: []byte("钻"), blockSize: 8}, want: []byte{233, 146, 187, 0, 0, 0, 0, 0}},
		{name: "004", args: args{data: []byte("1234567"), blockSize: 8}, want: []byte{49, 50, 51, 52, 53, 54, 55, 0}},
		{name: "005", args: args{data: []byte("12345678"), blockSize: 8}, want: []byte{49, 50, 51, 52, 53, 54, 55, 56, 0, 0, 0, 0, 0, 0, 0, 0}},
		{name: "006", args: args{data: []byte("123456789"), blockSize: 8}, want: []byte{49, 50, 51, 52, 53, 54, 55, 56, 57, 0, 0, 0, 0, 0, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ZeroPadding(tt.args.data, tt.args.blockSize); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ZeroPadding() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZeroUnPadding(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{name: "001", args: args{data: []byte{0, 0, 0, 0, 0, 0, 0, 0}}, want: []byte{}, wantErr: false},
		{name: "002", args: args{data: []byte{65, 0, 0, 0, 0, 0, 0, 0}}, want: []byte("A"), wantErr: false},
		{name: "003", args: args{data: []byte{233, 146, 187, 0, 0, 0, 0, 0}}, want: []byte("钻"), wantErr: false},
		{name: "004", args: args{data: []byte{49, 50, 51, 52, 53, 54, 55, 0}}, want: []byte("1234567"), wantErr: false},
		{name: "005", args: args{data: []byte{49, 50, 51, 52, 53, 54, 55, 56, 0, 0, 0, 0, 0, 0, 0, 0}}, want: []byte("12345678"), wantErr: false},
		{name: "006", args: args{data: []byte{49, 50, 51, 52, 53, 54, 55, 56, 57, 0, 0, 0, 0, 0, 0, 0}}, want: []byte("123456789"), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ZeroUnPadding(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ZeroUnPadding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ZeroUnPadding() got = %v, want %v", got, tt.want)
			}
		})
	}
}
