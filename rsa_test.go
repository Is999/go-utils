package utils_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Is999/go-utils"
)

// TestGenerateKeyRSA 覆盖默认格式和四种公私钥组合，生成文件须能由公开入口重新导入。
func TestGenerateKeyRSA(t *testing.T) {
	tests := []struct {
		name string
		pkcs []bool // 依次选择公钥 PKIX/PKCS#1、私钥 PKCS#1/PKCS#8。
	}{
		{name: "default"},
		{name: "PKCS1-PKCS8", pkcs: []bool{false, false}},
		{name: "PKIX-PKCS1", pkcs: []bool{true, true}},
		{name: "PKCS1-PKCS1", pkcs: []bool{false, true}},
		{name: "PKIX-PKCS8", pkcs: []bool{true, false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.GenerateKeyRSA(t.TempDir(), 2048, tt.pkcs...)
			if err != nil {
				t.Fatalf("GenerateKeyRSA() error = %v", err)
			}
			if len(got) != 2 {
				t.Fatalf("GenerateKeyRSA() returned %d files, want 2", len(got))
			}
			if _, err := utils.NewRSA(got[0], got[1], utils.WithRSAFilePath(true)); err != nil {
				t.Fatalf("NewRSA() cannot read generated keys: %v", err)
			}
		})
	}
}

// TestRSA 覆盖文本和文件两种密钥来源。
func TestRSA(t *testing.T) {
	type args struct {
		publicKey      string
		privateKey     string
		isFilePath     bool
		encodeToString func([]byte) string
		decode         func(string) ([]byte, error)
	}

	pubFile, priFile := rsaKeyFiles(t)
	pub := mustReadRSAFile(t, pubFile)
	pri := mustReadRSAFile(t, priFile)

	tests := []struct {
		name string
		args args
	}{
		{name: "001", args: args{publicKey: string(pub), privateKey: string(pri), encodeToString: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString}},
		{name: "002", args: args{publicKey: pubFile, privateKey: priFile, isFilePath: true, encodeToString: hex.EncodeToString, decode: hex.DecodeString}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := utils.NewRSA(tt.args.publicKey, tt.args.privateKey, utils.WithRSAFilePath(tt.args.isFilePath))
			if err != nil {
				t.Fatalf("NewRSA() WrapError = %v", err)
			}

			// 文本超过单个 RSA 分块，验证跨分块的拼接结果。
			marshal, err := json.Marshal(map[string]any{
				"Title":   tt.name,
				"Content": strings.Repeat("运行此代码时，当你在输入框中输入文本并点击提交按钮", 131) + tt.name,
			})
			if err != nil {
				t.Fatalf("json.Marshal() WrapError = %v", err)
			}

			encodeString, err := r.Encrypt(string(marshal), tt.args.encodeToString)
			if err != nil {
				t.Fatalf("Encrypt() WrapError = %v", err)
			}

			decryptString, err := r.Decrypt(encodeString, tt.args.decode)
			if err != nil {
				t.Fatalf("Decrypt() WrapError = %v", err)
			}

			if decryptString != string(marshal) {
				t.Errorf("PKCS1v15 解密后数据不等于加密前数据 got = %v, want %v", decryptString, string(marshal))
			}

			encodeString, err = r.EncryptOAEP(string(marshal), tt.args.encodeToString, sha256.New())
			if err != nil {
				t.Fatalf("Encrypt() WrapError = %v", err)
			}

			decryptString, err = r.DecryptOAEP(encodeString, tt.args.decode, sha256.New())
			if err != nil {
				t.Fatalf("Decrypt() WrapError = %v", err)
			}

			if decryptString != string(marshal) {
				t.Errorf("解密后数据不等于加密前数据 got = %v, want %v", decryptString, string(marshal))
			}

			encodeString, err = r.EncryptOAEPHash(string(marshal), tt.args.encodeToString, crypto.SHA256)
			if err != nil {
				t.Fatalf("EncryptOAEPHash() WrapError = %v", err)
			}

			decryptString, err = r.DecryptOAEPHash(encodeString, tt.args.decode, crypto.SHA256)
			if err != nil {
				t.Fatalf("DecryptOAEPHash() WrapError = %v", err)
			}

			if decryptString != string(marshal) {
				t.Errorf("OAEPHash 解密后数据不等于加密前数据 got = %v, want %v", decryptString, string(marshal))
			}
		})
	}
}

// TestRSA_SignAndVerify 覆盖文本和文件密钥的两种签名方式。
func TestRSA_SignAndVerify(t *testing.T) {
	type args struct {
		publicKey      string
		privateKey     string
		isFilePath     bool
		hash           crypto.Hash
		encodeToString func([]byte) string
		decode         func(string) ([]byte, error)
	}

	pubFile, priFile := rsaKeyFiles(t)
	pub := mustReadRSAFile(t, pubFile)
	pri := mustReadRSAFile(t, priFile)

	tests := []struct {
		name string
		args args
	}{
		{name: "001", args: args{publicKey: string(pub), privateKey: string(pri), hash: crypto.SHA256, encodeToString: base64.StdEncoding.EncodeToString, decode: base64.StdEncoding.DecodeString}},
		{name: "002", args: args{publicKey: pubFile, privateKey: priFile, isFilePath: true, hash: crypto.SHA512, encodeToString: hex.EncodeToString, decode: hex.DecodeString}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			privRsa, err := utils.NewPriRSA(tt.args.privateKey, utils.WithRSAFilePath(tt.args.isFilePath))
			if err != nil {
				t.Fatalf("NewRSA() WrapError = %v", err)
			}

			pubRsa, err := utils.NewPubRSA(tt.args.publicKey, utils.WithRSAFilePath(tt.args.isFilePath))
			if err != nil {
				t.Fatalf("NewRSA() WrapError = %v", err)
			}

			// 传入完整长消息，摘要计算由签名入口完成。
			marshal, err := json.Marshal(map[string]any{
				"Title":   tt.name,
				"Content": strings.Repeat("测试内容8282@334&-", 1024) + tt.name,
			})
			if err != nil {
				t.Fatalf("json.Marshal() WrapError = %v", err)
			}

			sign, err := privRsa.Sign(string(marshal), tt.args.hash, tt.args.encodeToString)
			if err != nil {
				t.Fatalf("Sign() WrapError = %v", err)
			}

			if err := pubRsa.Verify(string(marshal), sign, tt.args.hash, tt.args.decode); err != nil {
				t.Fatalf("Verify() WrapError = %v", err)
			}

			sign, err = privRsa.SignPSS(string(marshal), tt.args.hash, tt.args.encodeToString, nil)
			if err != nil {
				t.Fatalf("Sign() WrapError = %v", err)
			}

			if err := pubRsa.VerifyPSS(string(marshal), sign, tt.args.hash, tt.args.decode, nil); err != nil {
				t.Fatalf("Verify() WrapError = %v", err)
			}
		})
	}
}

func TestRSA_PEMHeaders(t *testing.T) {
	// 这里只验证文本分行和标记恢复，无需生成可用密钥或依赖文件。
	for _, tt := range []struct {
		keyType string
		pemType string
	}{
		{keyType: "public", pemType: "PUBLIC KEY"},
		{keyType: "private", pemType: "RSA PRIVATE KEY"},
	} {
		t.Run(tt.keyType, func(t *testing.T) {
			original := "-----BEGIN " + tt.pemType + "-----\n" + strings.Repeat("Ab0+", 16) + "\nAQID\n-----END " + tt.pemType + "-----"
			got, err := utils.AddPEMHeaders(utils.RemovePEMHeaders(original), tt.keyType)
			if err != nil {
				t.Fatal(err)
			}
			if got != original {
				t.Fatalf("PEM round trip = %q, want %q", got, original)
			}
		})
	}
}

// TestRSAKeyDERPreservesBytes 验证二进制密钥尾部的空白字节不会被当作文本裁剪。
func TestRSAKeyDERPreservesBytes(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := privateKey.PublicKey
	// 固定指数的 DER 末字节为 0x09，回归不依赖随机密钥恰好以空白字节结尾。
	publicKey.E = 65545
	pkcs1 := x509.MarshalPKCS1PublicKey(&publicKey)
	pkix, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := x509.ParsePKCS1PublicKey(pkcs1); err != nil {
		t.Fatal(err)
	}
	if _, err := x509.ParsePKIXPublicKey(pkix); err != nil {
		t.Fatal(err)
	}

	for _, format := range []struct {
		name    string
		pemType string
		der     []byte
	}{
		{name: "PKCS1", pemType: "RSA PUBLIC KEY", der: pkcs1},
		{name: "PKIX", pemType: "PUBLIC KEY", der: pkix},
	} {
		t.Run(format.name, func(t *testing.T) {
			if format.der[len(format.der)-1] != '\t' {
				t.Fatal("DER fixture must end with a tab byte")
			}
			for _, input := range []struct {
				name string
				data string
			}{
				{name: "DER", data: string(format.der)},
				{name: "PEM", data: " \t\n" + string(pem.EncodeToMemory(&pem.Block{Type: format.pemType, Bytes: format.der})) + " \r\n"},
				{name: "base64", data: " \t\n" + base64.StdEncoding.EncodeToString(format.der) + " \r\n"},
			} {
				t.Run(input.name, func(t *testing.T) {
					t.Run("string", func(t *testing.T) {
						if _, err := utils.NewPubRSA(input.data); err != nil {
							t.Fatal(err)
						}
					})
					t.Run("file", func(t *testing.T) {
						path := filepath.Join(t.TempDir(), "public.key")
						if err := os.WriteFile(path, []byte(input.data), 0o600); err != nil {
							t.Fatal(err)
						}
						if _, err := utils.NewPubRSA(path, utils.WithRSAFilePath(true)); err != nil {
							t.Fatal(err)
						}
					})
				})
			}
		})
	}
}

func TestRemovePEMHeadersPreservesBodyAndCaseRules(t *testing.T) {
	// 头尾标记沿用 Unicode 大写匹配规则；正文中的大小写和非标记横线均原样保留。
	for _, tt := range []struct {
		input string
		want  string
	}{
		{input: "-----BEGIN PUBLIC KEY-----\nAbCd+/==\n-----END PUBLIC KEY-----", want: "AbCd+/=="},
		{input: "  -----begin private key-----\r\n AbCd \r\n efGh \r\n -----end private key-----  ", want: "AbCdefGh"},
		{input: "-----begın key-----\nAbCd\n-----end key-----", want: "AbCd"},
		{input: "-----非标记-----\n正文ßı\n----BEGIN KEY-----", want: "-----非标记-----正文ßı----BEGIN KEY-----"},
		{input: "\n \t ", want: ""},
	} {
		if got := utils.RemovePEMHeaders(tt.input); got != tt.want {
			t.Fatalf("RemovePEMHeaders(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func BenchmarkRemovePEMHeaders(b *testing.B) {
	// 多行正文规模接近 PEM 私钥，计量解析过程而非密钥生成。
	body := strings.Repeat("AbCdEfGhIjKlMnOpQrStUvWxYz0123456789+/aBcDeFgHiJkLmNoPqRsTuVwXyZ\n", 26)
	pem := "-----BEGIN PRIVATE KEY-----\n" + body + "-----END PRIVATE KEY-----"
	b.ReportAllocs()
	b.SetBytes(int64(len(pem)))
	for b.Loop() {
		benchRSAString = utils.RemovePEMHeaders(pem)
	}
}

func TestGenerateKeyRSARejectsWeakBits(t *testing.T) {
	if _, err := utils.GenerateKeyRSA(t.TempDir(), 1024); err == nil {
		t.Fatal("GenerateKeyRSA() expected weak bits error")
	}
}

func TestGenerateKeyRSARejectsSymlinkDirectory(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	linkDir := filepath.Join(root, "link")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}

	if _, err := utils.GenerateKeyRSA(linkDir, 2048); err == nil {
		t.Fatal("GenerateKeyRSA() expected symlink directory error")
	}

	entries, err := os.ReadDir(realDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("real key dir should stay empty, got %d entries", len(entries))
	}
}

func TestRSARejectsWeakHash(t *testing.T) {
	_, priFile := rsaKeyFiles(t)
	r, err := utils.NewPriRSA(string(mustReadRSAFile(t, priFile)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Sign("hello", crypto.MD5, base64.StdEncoding.EncodeToString); err == nil {
		t.Fatal("Sign() expected weak hash error")
	}
	if _, err = r.DecryptOAEPHash("", base64.StdEncoding.DecodeString, crypto.SHA1); err == nil {
		t.Fatal("DecryptOAEPHash() expected weak hash error")
	}
}

// rsaKeyFiles 的文件由当前测试或基准独占，同一调用内复用密钥并由 TempDir 清理。
func rsaKeyFiles(t testing.TB) (string, string) {
	t.Helper()
	files, err := utils.GenerateKeyRSA(t.TempDir(), 2048)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("GenerateKeyRSA() returned %d files, want 2", len(files))
	}
	return files[0], files[1]
}

// mustReadRSAFile 读取失败即停止当前用例，避免后续误报密钥格式错误。
func mustReadRSAFile(t testing.TB, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// benchmarkRSA 在 b.Loop 前生成并解析密钥，不计入加解密耗时。
func benchmarkRSA(b *testing.B) *utils.RSA {
	b.Helper()

	pubFile, priFile := rsaKeyFiles(b)
	r, err := utils.NewRSA(string(mustReadRSAFile(b, pubFile)), string(mustReadRSAFile(b, priFile)))
	if err != nil {
		b.Fatal(err)
	}
	return r
}

func BenchmarkRSAEncryptOAEP(b *testing.B) {
	r := benchmarkRSA(b)
	data := strings.Repeat("rsa-benchmark-payload-", 4)
	var err error

	b.ReportAllocs()
	for b.Loop() {
		benchRSAString, err = r.EncryptOAEP(data, base64.StdEncoding.EncodeToString, sha256.New())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRSADecryptOAEP(b *testing.B) {
	r := benchmarkRSA(b)
	encrypted, err := r.EncryptOAEP(strings.Repeat("rsa-benchmark-payload-", 4), base64.StdEncoding.EncodeToString, sha256.New())
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		benchRSAString, err = r.DecryptOAEP(encrypted, base64.StdEncoding.DecodeString, sha256.New())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRSASignPSS(b *testing.B) {
	r := benchmarkRSA(b)
	data := strings.Repeat("rsa-sign-payload-", 8)
	var err error

	b.ReportAllocs()
	for b.Loop() {
		benchRSAString, err = r.SignPSS(data, crypto.SHA256, base64.StdEncoding.EncodeToString, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRSAVerifyPSS(b *testing.B) {
	r := benchmarkRSA(b)
	data := strings.Repeat("rsa-sign-payload-", 8)
	sign, err := r.SignPSS(data, crypto.SHA256, base64.StdEncoding.EncodeToString, nil)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := r.VerifyPSS(data, sign, crypto.SHA256, base64.StdEncoding.DecodeString, nil); err != nil {
			b.Fatal(err)
		}
	}
}

// benchRSAString 保留最近一次基准输出。
var benchRSAString string
