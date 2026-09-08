package utils_test

import (
	"crypto/aes"
	"encoding/base64"
	"testing"

	"github.com/Is999/go-utils"
)

// TestCipherZeroValue 保持零值实例先报密钥错误，不进入填充或 IV 处理。
func TestCipherZeroValue(t *testing.T) {
	var c utils.Cipher // 未经构造的公开零值。
	got, err := c.EncryptCBC(nil, utils.NoPad)
	if err == nil || err.Error() != "请先设置密钥" {
		t.Fatalf("EncryptCBC() error = %v, want 请先设置密钥", err)
	}
	if got != nil {
		t.Fatalf("EncryptCBC() = %v, want nil", got)
	}
}

func TestCipher(t *testing.T) {
	type args struct {
		key     string
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
		{name: "001", args: args{key: "1234567812345678", mode: utils.CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "002", args: args{key: "1234567812345678", mode: utils.ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "003", args: args{key: "1234567812345678", mode: utils.CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "004", args: args{key: "1234567812345678", mode: utils.CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
		{name: "005", args: args{key: "1234567812345678", mode: utils.OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 按测试模式选择 IV，旧协议模式需显式开启。
			opts := make([]utils.CipherOption, 0, 3)
			switch tt.args.mode {
			case utils.ECB:
				opts = append(opts, utils.WithAllowUnsafeECB(true))
			case utils.CBC:
				opts = append(opts, utils.WithRandIV(true))
			case utils.CTR, utils.CFB, utils.OFB:
				opts = append(opts, utils.WithRandIV(true), utils.WithAllowUnsafeStreamMode(true))
			}
			a, err := utils.NewCipher(tt.args.key, aes.NewCipher, opts...)
			if err != nil {
				t.Fatalf("NewCipher() error = %v", err)
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
