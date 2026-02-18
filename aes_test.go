package utils_test

import (
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/Is999/go-utils"
)

func TestAES(t *testing.T) {
	type args struct {
		key       string
		iv        string
		mode      utils.McryptMode
		encode    utils.EncodeToString
		decode    utils.DecodeString
		padding   utils.Padding
		unPadding utils.UnPadding
		data      string
		enStr     string
	}
	tests := []struct {
		name string
		args args
	}{
		// key 16 bit
		{name: "001", args: args{key: "0D03E9F1EFEDA1B3", iv: "567FDEFD073F7D04", mode: utils.CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "+JxoH7FtkqIbUFSCiv8YYg=="}},
		{name: "002", args: args{key: "0D03E9F1EFEDA1B3", iv: "567FDEFD073F7D04", mode: utils.ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "4Ry2UwtN8ubicYh1crqtOQ=="}},
		{name: "003", args: args{key: "0D03E9F1EFEDA1B3", iv: "567FDEFD073F7D04", mode: utils.CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "2ui2buOQie8SsHEZEsYWTw=="}},
		{name: "004", args: args{key: "0D03E9F1EFEDA1B3", iv: "567FDEFD073F7D04", mode: utils.CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "2ui2buOQie8SsHEZEsYWTw=="}},
		{name: "005", args: args{key: "0D03E9F1EFEDA1B3", iv: "567FDEFD073F7D04", mode: utils.OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "2ui2buOQie8SsHEZEsYWTw=="}},
		{name: "006", args: args{key: "0D03E9F1EFEDA1B3", mode: utils.CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "dxnrw6ZNku0UjrCy9PXQpg=="}},
		{name: "007", args: args{key: "0D03E9F1EFEDA1B3", mode: utils.ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "4Ry2UwtN8ubicYh1crqtOQ=="}},
		{name: "008", args: args{key: "0D03E9F1EFEDA1B3", mode: utils.CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "zC45BSMW77rjY+HzFKWAgQ=="}},
		{name: "009", args: args{key: "0D03E9F1EFEDA1B3", mode: utils.CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "zC45BSMW77rjY+HzFKWAgQ=="}},
		{name: "010", args: args{key: "0D03E9F1EFEDA1B3", mode: utils.OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "zC45BSMW77rjY+HzFKWAgQ=="}},
		// key 24 bit
		{name: "011", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "8048399BA52A2E40", mode: utils.CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "EAFUmwnzgoFf5LqcMoEGXQ=="}},
		{name: "012", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "8048399BA52A2E40", mode: utils.ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "mTCgwi//jto7GT3NOl2Vxw=="}},
		{name: "013", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "8048399BA52A2E40", mode: utils.CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "XKApjF+Whk8wpw3igKBe1Q=="}},
		{name: "014", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "8048399BA52A2E40", mode: utils.CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "XKApjF+Whk8wpw3igKBe1Q=="}},
		{name: "015", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "8048399BA52A2E40", mode: utils.OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "XKApjF+Whk8wpw3igKBe1Q=="}},
		// key 32 bit
		{name: "021", args: args{key: "884100890d03e9f1efeda1b393ecba1b", iv: "8048399BA52A2E40", mode: utils.CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "HyflBLyZzsOOVTr2D6MUGA=="}},
		{name: "022", args: args{key: "884100890d03e9f1efeda1b393ecba1b", iv: "8048399BA52A2E40", mode: utils.ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "KIkQdT/sPWOtZ0xDZnzrLg=="}},
		{name: "023", args: args{key: "884100890d03e9f1efeda1b393ecba1b", iv: "8048399BA52A2E40", mode: utils.CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "umkKIG/KKMmT0pi3OpmHqw=="}},
		{name: "024", args: args{key: "884100890d03e9f1efeda1b393ecba1b", iv: "8048399BA52A2E40", mode: utils.CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "umkKIG/KKMmT0pi3OpmHqw=="}},
		{name: "025", args: args{key: "884100890d03e9f1efeda1b393ecba1b", iv: "8048399BA52A2E40", mode: utils.OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456", enStr: "umkKIG/KKMmT0pi3OpmHqw=="}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 实例化AES，并设置key
			opts := make([]utils.CipherOption, 0, 1)
			if len(tt.args.iv) > 0 {
				opts = append(opts, utils.WithIV(tt.args.iv))
			}
			a, err := utils.AES(tt.args.key, opts...)
			if err != nil {
				t.Errorf("NewAES() error = %v", err)
				return
			}

			// 加密数据
			encryptStr, err := a.Encrypt(tt.args.data, tt.args.mode, tt.args.encode, tt.args.padding)
			if err != nil {
				t.Errorf("Encrypt() mode = %v error = %v", tt.args.mode, err)
				return
			}

			t.Logf("Encrypt() mode = %v encryptStr = %v", tt.args.mode, encryptStr)

			// 对比加密串
			if tt.args.enStr != encryptStr {
				t.Errorf("加密串不相同 want = %v got = %v", tt.args.enStr, encryptStr)
				return
			}

			// 解密数据
			got, err := a.Decrypt(encryptStr, tt.args.mode, tt.args.decode, tt.args.unPadding)
			if err != nil {
				t.Errorf("Decrypt() mode = %v error = %v", tt.args.mode, err)
				return
			}

			if !reflect.DeepEqual(got, tt.args.data) {
				t.Errorf("解密后数据不等于加密前数据 got = %v, want %v", got, tt.args.data)
			}
		})
	}
}

func TestAESRandIV(t *testing.T) {
	type args struct {
		key       string
		iv        string
		mode      utils.McryptMode
		encode    utils.EncodeToString
		decode    utils.DecodeString
		padding   utils.Padding
		unPadding utils.UnPadding
		data      string
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "001", args: args{key: "1234567812345678", mode: utils.CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456"}},
		{name: "002", args: args{key: "1234567812345678", mode: utils.ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456"}},
		{name: "003", args: args{key: "1234567812345678", mode: utils.CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456"}},
		{name: "004", args: args{key: "1234567812345678", mode: utils.CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456"}},
		{name: "005", args: args{key: "1234567812345678", mode: utils.OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: utils.Pkcs7Padding, unPadding: utils.Pkcs7UnPadding, data: "123456"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 实例化AES，并设置key
			a, err := utils.AES(tt.args.key, utils.WithRandIV(true))
			if err != nil {
				t.Errorf("NewAES() error = %v", err)
				return
			}

			// 加密数据
			encryptStr, err := a.Encrypt(tt.args.data, tt.args.mode, tt.args.encode, tt.args.padding)
			if err != nil {
				t.Errorf("Encrypt() mode = %v error = %v", tt.args.mode, err)
				return
			}

			//t.Logf("Encrypt() mode = %v encryptStr = %v", tt.args.mode, encryptStr)

			// 解密数据
			got, err := a.Decrypt(encryptStr, tt.args.mode, tt.args.decode, tt.args.unPadding)
			if err != nil {
				t.Errorf("Decrypt() mode = %v error = %v", tt.args.mode, err)
				return
			}

			if !reflect.DeepEqual(got, tt.args.data) {
				t.Errorf("解密后数据不等于加密前数据 got = %v, want %v", got, tt.args.data)
			}
		})
	}
}
