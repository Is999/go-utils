package utils_test

import (
	"encoding/base64"
	"testing"

	"github.com/Is999/go-utils"
)

func TestAES(t *testing.T) {
	type args struct {
		key     string
		iv      string
		mode    utils.CipherMode
		encode  utils.EncodeToString
		decode  utils.DecodeString
		padding utils.Pad
		unpad   utils.Unpad
		data    string
		enStr   string
	}
	tests := []struct {
		name string
		args args
	}{
		// 16 字节密钥（128 位）。
		{name: "001", args: args{key: "0D03E9F1EFEDA1B3", iv: "567FDEFD073F7D04", mode: utils.CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "+JxoH7FtkqIbUFSCiv8YYg=="}},
		{name: "002", args: args{key: "0D03E9F1EFEDA1B3", iv: "567FDEFD073F7D04", mode: utils.ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "4Ry2UwtN8ubicYh1crqtOQ=="}},
		{name: "003", args: args{key: "0D03E9F1EFEDA1B3", iv: "567FDEFD073F7D04", mode: utils.CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "2ui2buOQie8SsHEZEsYWTw=="}},
		{name: "004", args: args{key: "0D03E9F1EFEDA1B3", iv: "567FDEFD073F7D04", mode: utils.CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "2ui2buOQie8SsHEZEsYWTw=="}},
		{name: "005", args: args{key: "0D03E9F1EFEDA1B3", iv: "567FDEFD073F7D04", mode: utils.OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "2ui2buOQie8SsHEZEsYWTw=="}},
		{name: "006", args: args{key: "0D03E9F1EFEDA1B3", mode: utils.CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "dxnrw6ZNku0UjrCy9PXQpg=="}},
		{name: "007", args: args{key: "0D03E9F1EFEDA1B3", mode: utils.ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "4Ry2UwtN8ubicYh1crqtOQ=="}},
		{name: "008", args: args{key: "0D03E9F1EFEDA1B3", mode: utils.CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "zC45BSMW77rjY+HzFKWAgQ=="}},
		{name: "009", args: args{key: "0D03E9F1EFEDA1B3", mode: utils.CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "zC45BSMW77rjY+HzFKWAgQ=="}},
		{name: "010", args: args{key: "0D03E9F1EFEDA1B3", mode: utils.OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "zC45BSMW77rjY+HzFKWAgQ=="}},
		// 24 字节密钥（192 位）。
		{name: "011", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "8048399BA52A2E40", mode: utils.CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "EAFUmwnzgoFf5LqcMoEGXQ=="}},
		{name: "012", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "8048399BA52A2E40", mode: utils.ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "mTCgwi//jto7GT3NOl2Vxw=="}},
		{name: "013", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "8048399BA52A2E40", mode: utils.CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "XKApjF+Whk8wpw3igKBe1Q=="}},
		{name: "014", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "8048399BA52A2E40", mode: utils.CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "XKApjF+Whk8wpw3igKBe1Q=="}},
		{name: "015", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "8048399BA52A2E40", mode: utils.OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "XKApjF+Whk8wpw3igKBe1Q=="}},
		// 32 字节密钥（256 位）。
		{name: "021", args: args{key: "884100890d03e9f1efeda1b393ecba1b", iv: "8048399BA52A2E40", mode: utils.CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "HyflBLyZzsOOVTr2D6MUGA=="}},
		{name: "022", args: args{key: "884100890d03e9f1efeda1b393ecba1b", iv: "8048399BA52A2E40", mode: utils.ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "KIkQdT/sPWOtZ0xDZnzrLg=="}},
		{name: "023", args: args{key: "884100890d03e9f1efeda1b393ecba1b", iv: "8048399BA52A2E40", mode: utils.CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "umkKIG/KKMmT0pi3OpmHqw=="}},
		{name: "024", args: args{key: "884100890d03e9f1efeda1b393ecba1b", iv: "8048399BA52A2E40", mode: utils.CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "umkKIG/KKMmT0pi3OpmHqw=="}},
		{name: "025", args: args{key: "884100890d03e9f1efeda1b393ecba1b", iv: "8048399BA52A2E40", mode: utils.OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.PKCS7Pad, unpad: utils.PKCS7Unpad, data: "123456", enStr: "umkKIG/KKMmT0pi3OpmHqw=="}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 固定向量包含旧协议模式，逐项显式开启对应选项。
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
			a, err := utils.AES(tt.args.key, opts...)
			if err != nil {
				t.Fatalf("NewAES() error = %v", err)
			}

			encryptStr, err := a.Encrypt(tt.args.data, tt.args.mode, tt.args.encode, tt.args.padding)
			if err != nil {
				t.Fatalf("Encrypt() mode = %v error = %v", tt.args.mode, err)
			}

			// 固定向量约束密文格式，不能只检查加解密往返。
			if tt.args.enStr != encryptStr {
				t.Fatalf("加密串不相同 want = %v got = %v", tt.args.enStr, encryptStr)
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

func TestAESRandIV(t *testing.T) {
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
			// 随机 IV 随密文传递；ECB 沿用无 IV 的规则。
			opts := []utils.CipherOption{utils.WithRandIV(true)}
			if tt.args.mode == utils.ECB {
				opts = append(opts, utils.WithAllowUnsafeECB(true))
			}
			if tt.args.mode == utils.CTR || tt.args.mode == utils.CFB || tt.args.mode == utils.OFB {
				opts = append(opts, utils.WithAllowUnsafeStreamMode(true))
			}
			a, err := utils.AES(tt.args.key, opts...)
			if err != nil {
				t.Fatalf("NewAES() error = %v", err)
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
