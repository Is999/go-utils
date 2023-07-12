package utils

import (
	"crypto"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

var (
	path    = "/tmp/"
	pubFile = path + "public.pem"
	priFile = path + "private.pem"
)

func TestGenerateKeyRSA(t *testing.T) {
	type args struct {
		path string
		bits int
		pkcs []bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{path: path, bits: 1024}, wantErr: false},
		{name: "002", args: args{path: path, bits: 1024, pkcs: []bool{false, false}}, wantErr: false},
		{name: "003", args: args{path: path, bits: 1024, pkcs: []bool{true, true}}, wantErr: false},
		{name: "004", args: args{path: path, bits: 1024, pkcs: []bool{false, true}}, wantErr: false},
		{name: "005", args: args{path: path, bits: 1024, pkcs: []bool{true, false}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := GenerateKeyRSA(tt.args.path, tt.args.bits, tt.args.pkcs...); (err != nil) != tt.wantErr {
				t.Errorf("GenerateKeyRSA() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				// 考被文件
				for i, f := range got {
					err := Copy(f, Ternary(i == 0, pubFile, priFile))
					if err != nil {
						t.Errorf("Copy() error = %v", err)
					}
				}
			}
		})
	}
}

func TestRSA(t *testing.T) {

	type args struct {
		publicKey      string
		privateKey     string
		isFilePath     bool
		hash           crypto.Hash
		encodeToString func([]byte) string
		decode         func(string) ([]byte, error)
	}

	// 读取公钥文件内容
	pub, err := os.ReadFile(pubFile)
	if err != nil {
		t.Errorf("ReadFile() err = %v", err)
	}

	// 读取私钥文件内容
	pri, err := os.ReadFile(priFile)
	if err != nil {
		t.Errorf("ReadFile() err = %v", err)
	}

	tests := []struct {
		name string
		args args
		//want   *RSA
	}{
		{name: "001", args: args{publicKey: string(pub), privateKey: string(pri), hash: crypto.SHA256, encodeToString: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString}},
		{name: "002", args: args{publicKey: pubFile, privateKey: priFile, isFilePath: true, hash: crypto.MD5, encodeToString: hex.EncodeToString, decode: hex.DecodeString}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRSA(tt.args.publicKey, tt.args.privateKey, tt.args.isFilePath)
			if err != nil {
				t.Errorf("NewRSA() err = %v", err)
				return
			}

			// r := &RSA{}

			// 判断是否设置公钥
			//if err := r.IsSetPublicKey(); err != nil {
			//	//t.Logf("IsSetPublicKey() = %v\n", err)
			//	// 设置公钥
			//	if err := r.SetPublicKey(tt.args.publicKey, tt.args.isFilePath); err != nil {
			//		t.Errorf("SetPublicKey() err = %v", err)
			//		return
			//	}
			//}

			// 判断是否设置私钥
			//if err := r.IsSetPrivateKey(); err != nil {
			//	//t.Logf("IsSetPrivateKey() = %v\n", err)
			//	// 设置私钥
			//	if err := r.SetPrivateKey(tt.args.privateKey, tt.args.isFilePath); err != nil {
			//		t.Errorf("SetPrivateKey() err = %v", err)
			//		return
			//	}
			//}

			// 源数据
			marshal, err := json.Marshal(map[string]interface{}{
				"Title":   tt.name,
				"Content": "测试内容8282@334&-" + tt.name,
			})
			if err != nil {
				t.Errorf("json.Marshal() err = %v", err)
				return
			}

			//t.Logf("json.Marshal() = %v\n", string(jsonEncode))

			// 公钥加密
			encodeString, err := r.Encrypt(string(marshal), tt.args.encodeToString)
			if err != nil {
				t.Errorf("Encrypt() err = %v", err)
				return
			}
			//t.Logf("Encrypt() = %v\n", encodeString)

			// 私钥解密
			decryptString, err := r.Decrypt(encodeString, tt.args.decode)
			if err != nil {
				t.Errorf("Decrypt() err = %v", err)
				return
			}
			//t.Logf("Decrypt() = %v\n", decryptString)

			if !reflect.DeepEqual(decryptString, string(marshal)) {
				t.Errorf("解密后数据不等于加密前数据 got = %v, want %v", decryptString, string(marshal))
			}

			// 私钥签名
			sign, err := r.Sign(string(marshal), tt.args.hash, tt.args.encodeToString)
			if err != nil {
				t.Errorf("Sign() err = %v", err)
				return
			}
			//t.Logf("Sign() = %v\n", sign)

			// 公钥验签
			if err := r.Verify(string(marshal), sign, tt.args.hash, tt.args.decode); err != nil {
				t.Errorf("Verify() err = %v", err)
				return
			} else {
				//t.Log("Verify() = 验证成功")
			}
		})
	}
}

func TestRSA_SignAndVerify(t *testing.T) {

	type args struct {
		publicKey      string
		privateKey     string
		isFilePath     bool
		hash           crypto.Hash
		encodeToString func([]byte) string
		decode         func(string) ([]byte, error)
	}

	// 读取公钥文件内容
	pub, err := os.ReadFile(pubFile)
	if err != nil {
		t.Errorf("ReadFile() err = %v", err)
	}

	// 读取私钥文件内容
	pri, err := os.ReadFile(priFile)
	if err != nil {
		t.Errorf("ReadFile() err = %v", err)
	}

	tests := []struct {
		name string
		args args
		//want   *RSA
	}{
		{name: "001", args: args{publicKey: string(pub), privateKey: string(pri), hash: crypto.SHA256, encodeToString: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString}},
		{name: "002", args: args{publicKey: pubFile, privateKey: priFile, isFilePath: true, hash: crypto.MD5, encodeToString: hex.EncodeToString, decode: hex.DecodeString}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			privRsa, err := NewPriRSA(tt.args.privateKey, tt.args.isFilePath)
			if err != nil {
				t.Errorf("NewRSA() err = %v", err)
				return
			}

			pubRsa, err := NewPubRSA(tt.args.publicKey, tt.args.isFilePath)
			if err != nil {
				t.Errorf("NewRSA() err = %v", err)
				return
			}

			// 源数据
			marshal, err := json.Marshal(map[string]interface{}{
				"Title":   tt.name,
				"Content": "测试内容8282@334&-" + tt.name,
			})
			if err != nil {
				t.Errorf("json.Marshal() err = %v", err)
				return
			}

			//t.Logf("json.Marshal() = %v\n", string(jsonEncode))

			// 公钥加密
			encodeString, err := pubRsa.Encrypt(string(marshal), tt.args.encodeToString)
			if err != nil {
				t.Errorf("Encrypt() err = %v", err)
				return
			}
			//t.Logf("Encrypt() = %v\n", encodeString)

			// 私钥解密
			decryptString, err := privRsa.Decrypt(encodeString, tt.args.decode)
			if err != nil {
				t.Errorf("Decrypt() err = %v", err)
				return
			}
			//t.Logf("Decrypt() = %v\n", decryptString)

			if !reflect.DeepEqual(decryptString, string(marshal)) {
				t.Errorf("解密后数据不等于加密前数据 got = %v, want %v", decryptString, string(marshal))
			}

			// 私钥签名
			sign, err := privRsa.Sign(string(marshal), tt.args.hash, tt.args.encodeToString)
			if err != nil {
				t.Errorf("Sign() err = %v", err)
				return
			}
			//t.Logf("Sign() = %v\n", sign)

			// 公钥验签
			if err := pubRsa.Verify(string(marshal), sign, tt.args.hash, tt.args.decode); err != nil {
				t.Errorf("Verify() err = %v", err)
				return
			} else {
				//t.Log("Verify() = 验证成功")
			}
		})
	}
}
