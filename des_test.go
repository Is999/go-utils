package utils_test

import (
	"encoding/base64"
	"testing"

	"github.com/Is999/go-utils"
)

func TestDES(t *testing.T) {
	type args struct {
		key     string
		iv      string
		mode    utils.CipherMode
		encode  utils.EncodeToString
		decode  utils.DecodeString
		padding utils.Pad
		unpad   utils.Unpad
		data    string
	}
	tests := []struct {
		name string
		args args
	}{
		// 8 字节 DES 密钥。
		{name: "001", args: args{key: "E9F1EFED", iv: "D073F7D4", mode: utils.CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "002", args: args{key: "E9F1EFED", iv: "D073F7D4", mode: utils.ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "003", args: args{key: "E9F1EFED", iv: "D073F7D4", mode: utils.CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "004", args: args{key: "E9F1EFED", iv: "D073F7D4", mode: utils.CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "005", args: args{key: "E9F1EFED", iv: "D073F7D4", mode: utils.OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "006", args: args{key: "E9F1EFED", mode: utils.CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "007", args: args{key: "E9F1EFED", mode: utils.ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "008", args: args{key: "E9F1EFED", mode: utils.CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "009", args: args{key: "E9F1EFED", mode: utils.CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "010", args: args{key: "E9F1EFED", mode: utils.OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		// 24 字节 3DES 密钥。
		{name: "011", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "E9F1EFED", mode: utils.CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "012", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "E9F1EFED", mode: utils.ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "013", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "E9F1EFED", mode: utils.CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "014", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "E9F1EFED", mode: utils.CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "015", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "E9F1EFED", mode: utils.OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 往返测试包含旧协议模式，逐项显式开启对应选项。
			opts := make([]utils.CipherOption, 0, 3)
			if len(tt.args.iv) > 0 {
				opts = append(opts, utils.WithIV(tt.args.iv))
			} else if tt.args.mode != utils.ECB {
				opts = append(opts, utils.WithAllowUnsafeKeyIV(true))
			}
			if tt.args.mode == utils.ECB {
				opts = append(opts, utils.WithAllowUnsafeECB(true))
			}
			if tt.args.mode == utils.CTR || tt.args.mode == utils.CFB || tt.args.mode == utils.OFB {
				opts = append(opts, utils.WithAllowUnsafeStreamMode(true))
			}
			a, err := utils.DES(tt.args.key, opts...)
			if err != nil {
				t.Fatalf("NewDES() error = %v", err)
			}

			encryptStr, err := a.Encrypt(tt.args.data, tt.args.mode, tt.args.encode, tt.args.padding)
			if err != nil {
				t.Fatalf("Encrypt() mode = %v error = %v", tt.args.mode, err)
			}

			got, err := a.Decrypt(encryptStr, tt.args.mode, tt.args.decode, tt.args.unpad)
			if err != nil {
				t.Fatalf("Decrypt() mode = %v error = %v", tt.args.mode, err)
			}

			if got != tt.args.data {
				t.Errorf("解密后数据不等于加密前数据 got = %v, want %v", got, tt.args.data)
			}
		})
	}
}
