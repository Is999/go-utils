package utils

import (
	"encoding/base64"
	"reflect"
	"testing"
)

func TestDES(t *testing.T) {
	type args struct {
		key       string
		iv        string
		mode      McryptMode
		encode    Encode
		decode    Decode
		padding   Padding
		unPadding UnPadding
		data      string
	}
	tests := []struct {
		name string
		args args
	}{
		// key 8 bit
		{name: "001", args: args{key: "E9F1EFED", iv: "D073F7D4", mode: CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "002", args: args{key: "E9F1EFED", iv: "D073F7D4", mode: ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "003", args: args{key: "E9F1EFED", iv: "D073F7D4", mode: CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "004", args: args{key: "E9F1EFED", iv: "D073F7D4", mode: CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "005", args: args{key: "E9F1EFED", iv: "D073F7D4", mode: OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "006", args: args{key: "E9F1EFED", mode: CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "007", args: args{key: "E9F1EFED", mode: ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "008", args: args{key: "E9F1EFED", mode: CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "009", args: args{key: "E9F1EFED", mode: CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "010", args: args{key: "E9F1EFED", mode: OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		// key 24 bit
		{name: "011", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "E9F1EFED", mode: CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "012", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "E9F1EFED", mode: ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "013", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "E9F1EFED", mode: CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "014", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "E9F1EFED", mode: CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "015", args: args{key: "9F9CE8D28048399BA52A2E40", iv: "E9F1EFED", mode: OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 实例化DES，并设置key
			a, err := DES(tt.args.key)
			if err != nil {
				t.Errorf("NewDES() error = %v", err)
				return
			}

			// 设置iv
			if len(tt.args.iv) > 0 {
				if err := a.SetIv(tt.args.iv); err != nil {
					t.Errorf("SetIv() error = %v", err)
					return
				}
			}

			// 加密数据
			encryptStr, err := a.Encrypt(tt.args.data, tt.args.mode, tt.args.encode, tt.args.padding)
			if err != nil {
				t.Errorf("Encrypt() mode = %v error = %v", tt.args.mode, err)
				return
			}

			// t.Logf("Encrypt() mode = %v encryptStr = %v", tt.args.mode, encryptStr)

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
