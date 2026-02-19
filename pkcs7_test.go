package utils_test

import (
	"reflect"
	"testing"

	"github.com/Is999/go-utils"
)

func TestPkcs7Padding(t *testing.T) {
	type args struct {
		data      []byte
		blockSize int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{name: "001", args: args{data: []byte{}, blockSize: 8}, want: []byte{8, 8, 8, 8, 8, 8, 8, 8}},
		{name: "002", args: args{data: []byte("A"), blockSize: 8}, want: []byte{65, 7, 7, 7, 7, 7, 7, 7}},
		{name: "003", args: args{data: []byte("钻"), blockSize: 8}, want: []byte{233, 146, 187, 5, 5, 5, 5, 5}},
		{name: "004", args: args{data: []byte("1234567"), blockSize: 8}, want: []byte{49, 50, 51, 52, 53, 54, 55, 1}},
		{name: "005", args: args{data: []byte("12345678"), blockSize: 8}, want: []byte{49, 50, 51, 52, 53, 54, 55, 56, 8, 8, 8, 8, 8, 8, 8, 8}},
		{name: "006", args: args{data: []byte("123456789"), blockSize: 8}, want: []byte{49, 50, 51, 52, 53, 54, 55, 56, 57, 7, 7, 7, 7, 7, 7, 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Pkcs7Padding(tt.args.data, tt.args.blockSize); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Pkcs7Padding() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPkcs7UnPadding(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{name: "001", args: args{data: []byte{8, 8, 8, 8, 8, 8, 8, 8}}, want: []byte{}, wantErr: false},
		{name: "002", args: args{data: []byte{65, 7, 7, 7, 7, 7, 7, 7}}, want: []byte("A"), wantErr: false},
		{name: "003", args: args{data: []byte{233, 146, 187, 5, 5, 5, 5, 5}}, want: []byte("钻"), wantErr: false},
		{name: "004", args: args{data: []byte{49, 50, 51, 52, 53, 54, 55, 1}}, want: []byte("1234567"), wantErr: false},
		{name: "005", args: args{data: []byte{49, 50, 51, 52, 53, 54, 55, 56, 8, 8, 8, 8, 8, 8, 8, 8}}, want: []byte("12345678"), wantErr: false},
		{name: "006", args: args{data: []byte{49, 50, 51, 52, 53, 54, 55, 56, 57, 7, 7, 7, 7, 7, 7, 7}}, want: []byte("123456789"), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.Pkcs7UnPadding(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Pkcs7UnPadding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Pkcs7UnPadding() got = %v, want %v", got, tt.want)
			}
		})
	}
}
