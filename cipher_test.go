package utils

import (
	"crypto/aes"
	"encoding/base64"
	"reflect"
	"testing"
)

func TestCipher(t *testing.T) {
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
		{name: "001", args: args{key: "1234567812345678", mode: CBC, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "002", args: args{key: "1234567812345678", mode: ECB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "003", args: args{key: "1234567812345678", mode: CTR, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "004", args: args{key: "1234567812345678", mode: CFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
		{name: "005", args: args{key: "1234567812345678", mode: OFB, encode: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString, padding: Pkcs7Padding, unPadding: Pkcs7UnPadding, data: "123456"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 实例化Cipher，并设置key
			a, err := NewCipher(tt.args.key, aes.NewCipher, false)
			if err != nil {
				t.Errorf("NewCipher() error = %v", err)
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
