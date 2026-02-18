package utils_test

import (
	"crypto"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Is999/go-utils"
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
			if got, err := utils.GenerateKeyRSA(tt.args.path, tt.args.bits, tt.args.pkcs...); (err != nil) != tt.wantErr {
				t.Errorf("GenerateKeyRSA() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				// 考被文件
				for i, f := range got {
					err := utils.Copy(f, utils.Ternary(i == 0, pubFile, priFile))
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
		t.Errorf("ReadFile() WrapError = %v", err)
	}

	// 读取私钥文件内容
	pri, err := os.ReadFile(priFile)
	if err != nil {
		t.Errorf("ReadFile() WrapError = %v", err)
	}

	tests := []struct {
		name string
		args args
		//want   *_RSA
	}{
		{name: "001", args: args{publicKey: string(pub), privateKey: string(pri), hash: crypto.SHA256, encodeToString: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString}},
		{name: "002", args: args{publicKey: pubFile, privateKey: priFile, isFilePath: true, hash: crypto.MD5, encodeToString: hex.EncodeToString, decode: hex.DecodeString}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := make([]utils.RSAOption, 0, 1)
			if tt.args.isFilePath {
				opts = append(opts, utils.WithRSAFilePath(true))
			}
			r, err := utils.NewRSA(tt.args.publicKey, tt.args.privateKey, opts...)
			if err != nil {
				t.Errorf("NewRSA() WrapError = %v", err)
				return
			}

			// 源数据
			marshal, err := json.Marshal(map[string]interface{}{
				"Title": tt.name,
				"Content": strings.Repeat(`运行此代码时，当你在输入框中输入文本并点击㰆凭棥`, 131) + tt.name,
			})
			if err != nil {
				t.Errorf("json.Marshal() WrapError = %v", err)
				return
			}

			// t.Logf("json.Marshal() = %d %v\n", len(string(marshal)), string(marshal))

			// 公钥加密 PKCS1v15
			encodeString, err := r.Encrypt(string(marshal), tt.args.encodeToString)
			if err != nil {
				t.Errorf("Encrypt() WrapError = %v", err)
				return
			}
			//t.Logf("Encrypt() = %v\n", encodeString)

			// 私钥解密 PKCS1v15
			decryptString, err := r.Decrypt(encodeString, tt.args.decode)
			if err != nil {
				t.Errorf("Decrypt() WrapError = %v", err)
				return
			}
			//t.Logf("Decrypt() = %v\n", decryptString)

			// 公钥加密 OAEP
			encodeString, err = r.EncryptOAEP(string(marshal), tt.args.encodeToString, sha256.New())
			if err != nil {
				t.Errorf("Encrypt() WrapError = %v", err)
				return
			}
			//t.Logf("Encrypt() = %v\n", encodeString)

			// 私钥解密 OAEP
			decryptString, err = r.DecryptOAEP(encodeString, tt.args.decode, sha256.New())
			if err != nil {
				t.Errorf("Decrypt() WrapError = %v", err)
				return
			}
			//t.Logf("Decrypt() = %v\n", decryptString)

			if !reflect.DeepEqual(decryptString, string(marshal)) {
				t.Errorf("解密后数据不等于加密前数据 got = %v, want %v", decryptString, string(marshal))
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
		t.Errorf("ReadFile() WrapError = %v", err)
	}

	// 读取私钥文件内容
	pri, err := os.ReadFile(priFile)
	if err != nil {
		t.Errorf("ReadFile() WrapError = %v", err)
	}

	tests := []struct {
		name string
		args args
		//want   *_RSA
	}{
		{name: "001", args: args{publicKey: string(pub), privateKey: string(pri), hash: crypto.SHA256, encodeToString: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString}},
		{name: "002", args: args{publicKey: pubFile, privateKey: priFile, isFilePath: true, hash: crypto.MD5, encodeToString: hex.EncodeToString, decode: hex.DecodeString}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := make([]utils.RSAOption, 0, 1)
			if tt.args.isFilePath {
				opts = append(opts, utils.WithRSAFilePath(true))
			}
			privRsa, err := utils.NewPriRSA(tt.args.privateKey, opts...)
			if err != nil {
				t.Errorf("NewRSA() WrapError = %v", err)
				return
			}

			pubRsa, err := utils.NewPubRSA(tt.args.publicKey, opts...)
			if err != nil {
				t.Errorf("NewRSA() WrapError = %v", err)
				return
			}

			// 源数据
			marshal, err := json.Marshal(map[string]interface{}{
				"Title":   tt.name,
				"Content": strings.Repeat("测试内容8282@334&-", 1024) + tt.name,
			})
			if err != nil {
				t.Errorf("json.Marshal() WrapError = %v", err)
				return
			}

			// t.Logf("json.Marshal() = %d %v\n", len(string(marshal)), string(marshal))

			// 私钥签名 PKCS1v15
			sign, err := privRsa.Sign(string(marshal), tt.args.hash, tt.args.encodeToString)
			if err != nil {
				t.Errorf("Sign() WrapError = %v", err)
				return
			}
			//t.Logf("Sign() = %v\n", sign)

			// 公钥验签 PKCS1v15
			if err := pubRsa.Verify(string(marshal), sign, tt.args.hash, tt.args.decode); err != nil {
				t.Errorf("Verify() WrapError = %v", err)
				return
			} else {
				//t.Log("Verify() = 验证成功")
			}

			// 私钥签名 PSS
			sign, err = privRsa.SignPSS(string(marshal), tt.args.hash, tt.args.encodeToString, nil)
			if err != nil {
				t.Errorf("Sign() WrapError = %v", err)
				return
			}
			//t.Logf("Sign() = %v\n", sign)

			// 公钥验签 PSS
			if err := pubRsa.VerifyPSS(string(marshal), sign, tt.args.hash, tt.args.decode, nil); err != nil {
				t.Errorf("Verify() WrapError = %v", err)
				return
			} else {
				//t.Log("Verify() = 验证成功")
			}
		})
	}
}

func TestRSA_PEMHeaders(t *testing.T) {
	// 读取公钥文件内容
	pub, err := os.ReadFile(pubFile)
	if err != nil {
		t.Errorf("ReadFile() WrapError = %v", err)
	}

	//t.Logf("公钥 %s", string(pub))
	rPub := utils.RemovePEMHeaders(string(pub))
	//t.Logf("remove 公钥 %s", rPub)
	aPub, _ := utils.AddPEMHeaders(rPub, "public")
	//t.Logf("add 公钥 %s %v", aPub, strings.EqualFold(aPub, strings.TrimSpace(string(pub))))
	if !strings.EqualFold(aPub, strings.TrimSpace(string(pub))) {
		t.Errorf("转换后的公钥与原始公钥不相等")
	}

	// 读取私钥文件内容
	pri, err := os.ReadFile(priFile)
	if err != nil {
		t.Errorf("ReadFile() WrapError = %v", err)
	}
	//t.Logf("私钥 %s", string(pri))
	rPri := utils.RemovePEMHeaders(string(pri))
	//t.Logf("remove 私钥 %s", rPri)
	aPri, _ := utils.AddPEMHeaders(rPri, "private")
	//t.Logf("add 私钥 %s %v", aPri, strings.EqualFold(aPri, strings.TrimSpace(string(pri))))
	if !strings.EqualFold(aPri, strings.TrimSpace(string(pri))) {
		t.Errorf("转换后的私钥与原始私钥不相等")
	}
}
