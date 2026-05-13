package utils_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Is999/go-utils"
)

func TestCipherConcurrentRandIV(t *testing.T) {
	c, err := utils.AES("1234567812345678", utils.WithRandIV(true))
	if err != nil {
		t.Fatal(err)
	}

	const data = "concurrent payload"
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 64; j++ {
				encrypted, err := c.Encrypt(data, utils.CBC, base64.StdEncoding.EncodeToString, utils.Pkcs7Padding)
				if err != nil {
					t.Errorf("Encrypt() error = %v", err)
					return
				}
				decrypted, err := c.Decrypt(encrypted, utils.CBC, base64.StdEncoding.DecodeString, utils.Pkcs7UnPadding)
				if err != nil {
					t.Errorf("Decrypt() error = %v", err)
					return
				}
				if decrypted != data {
					t.Errorf("decrypted = %q, want %q", decrypted, data)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestCipherEncryptToDecryptToCTRNoPadding(t *testing.T) {
	c, err := utils.AES("1234567812345678", utils.WithRandIV(true))
	if err != nil {
		t.Fatal(err)
	}

	// data 是流模式原始业务数据，用于验证 NoPadding 快路径不会额外复制或裁剪内容。
	data := []byte("payload for dst reuse")
	// encryptedDst 是调用方复用的密文缓冲区，前缀用于验证 EncryptTo 会追加而不是覆盖历史内容。
	encryptedDst := []byte("prefix:")
	encrypted, err := c.EncryptTo(encryptedDst, data, utils.CTR, utils.NoPadding)
	if err != nil {
		t.Fatalf("EncryptTo() error = %v", err)
	}
	if !bytes.HasPrefix(encrypted, encryptedDst) {
		t.Fatalf("EncryptTo() should keep dst prefix, got %q", encrypted)
	}

	// decryptedDst 是调用方复用的明文缓冲区，解密输入跳过前缀后应恢复原始 data。
	decryptedDst := []byte("plain:")
	decrypted, err := c.DecryptTo(decryptedDst, encrypted[len(encryptedDst):], utils.CTR, utils.NoUnPadding)
	if err != nil {
		t.Fatalf("DecryptTo() error = %v", err)
	}
	if !bytes.Equal(decrypted[len(decryptedDst):], data) {
		t.Fatalf("DecryptTo() = %q, want suffix %q", decrypted, data)
	}
}

func TestPkcs7UnPaddingRejectsInvalidPadding(t *testing.T) {
	tests := [][]byte{
		{},
		{1, 2, 3, 0},
		{1, 2, 3, 5},
		{1, 2, 3, 2, 3},
	}
	for _, tt := range tests {
		if _, err := utils.Pkcs7UnPadding(tt); err == nil {
			t.Fatalf("utils.Pkcs7UnPadding(%v) expected error", tt)
		}
	}
}

func TestPaddingDoesNotMutateInput(t *testing.T) {
	src := make([]byte, 2, 16)
	copy(src, "ab")

	padded := utils.Pkcs7Padding(src, 8)
	padded[0] = 'x'
	if string(src) != "ab" {
		t.Fatalf("utils.Pkcs7Padding mutated input: %q", src)
	}

	zeroPadded := utils.ZeroPadding(src, 8)
	zeroPadded[0] = 'y'
	if string(src) != "ab" {
		t.Fatalf("utils.ZeroPadding mutated input: %q", src)
	}

	noPadding := utils.NoPadding(src, 8)
	noPadding[0] = 'z'
	if string(src) != "ab" {
		t.Fatalf("utils.NoPadding mutated input: %q", src)
	}
}

func TestCipherRejectsNilCallbacks(t *testing.T) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Encrypt("data", utils.CBC, nil, utils.Pkcs7Padding); err == nil {
		t.Fatal("Encrypt() with nil encode expected error")
	}
	if _, err := c.Encrypt("data", utils.CBC, base64.StdEncoding.EncodeToString, nil); err == nil {
		t.Fatal("Encrypt() with nil padding expected error")
	}
	if _, err := c.Decrypt("data", utils.CBC, nil, utils.Pkcs7UnPadding); err == nil {
		t.Fatal("Decrypt() with nil decode expected error")
	}
	if _, err := c.DecryptBytes([]byte("data"), utils.CBC, nil); err == nil {
		t.Fatal("DecryptBytes() with nil unPadding expected error")
	}
}

func TestCipherRejectsUnsafeStreamModesByDefault(t *testing.T) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.EncryptCFB([]byte("data"), utils.Pkcs7Padding); err == nil {
		t.Fatal("EncryptCFB() expected error when unsafe stream mode is disabled")
	}
	if _, err := c.EncryptOFB([]byte("data"), utils.Pkcs7Padding); err == nil {
		t.Fatal("EncryptOFB() expected error when unsafe stream mode is disabled")
	}
}

func TestCipherRejectsECBByDefault(t *testing.T) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.EncryptECB([]byte("data"), utils.Pkcs7Padding); err == nil {
		t.Fatal("EncryptECB() expected error when utils.ECB is disabled")
	}
}

func TestCipherRejectsImplicitKeyIVByDefault(t *testing.T) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.EncryptCBC([]byte("data"), utils.Pkcs7Padding); err == nil {
		t.Fatal("EncryptCBC() expected error when IV is not configured")
	}
	if _, err := c.EncryptCTR([]byte("data"), utils.Pkcs7Padding); err == nil {
		t.Fatal("EncryptCTR() expected error when IV is not configured")
	}
}

func TestCipherAllowsUnsafeStreamModesWhenEnabled(t *testing.T) {
	c, err := utils.AES("1234567812345678", utils.WithAllowUnsafeStreamMode(true), utils.WithAllowUnsafeKeyIV(true))
	if err != nil {
		t.Fatal(err)
	}

	encrypted, err := c.EncryptCFB([]byte("legacy-data"), utils.Pkcs7Padding)
	if err != nil {
		t.Fatalf("EncryptCFB() error = %v", err)
	}
	decrypted, err := c.DecryptCFB(encrypted, utils.Pkcs7UnPadding)
	if err != nil {
		t.Fatalf("DecryptCFB() error = %v", err)
	}
	if string(decrypted) != "legacy-data" {
		t.Fatalf("DecryptCFB() = %q, want %q", decrypted, "legacy-data")
	}
}

func TestCipherAllowsUnsafeECBWhenEnabled(t *testing.T) {
	c, err := utils.AES("1234567812345678", utils.WithAllowUnsafeECB(true))
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := c.EncryptECB([]byte("legacy-data"), utils.Pkcs7Padding)
	if err != nil {
		t.Fatalf("EncryptECB() error = %v", err)
	}
	decrypted, err := c.DecryptECB(encrypted, utils.Pkcs7UnPadding)
	if err != nil {
		t.Fatalf("DecryptECB() error = %v", err)
	}
	if string(decrypted) != "legacy-data" {
		t.Fatalf("DecryptECB() = %q, want %q", decrypted, "legacy-data")
	}
}

func TestCipherAllowsUnsafeKeyIVWhenEnabled(t *testing.T) {
	c, err := utils.AES("1234567812345678", utils.WithAllowUnsafeKeyIV(true))
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := c.EncryptCBC([]byte("legacy-data"), utils.Pkcs7Padding)
	if err != nil {
		t.Fatalf("EncryptCBC() error = %v", err)
	}
	decrypted, err := c.DecryptCBC(encrypted, utils.Pkcs7UnPadding)
	if err != nil {
		t.Fatalf("DecryptCBC() error = %v", err)
	}
	if string(decrypted) != "legacy-data" {
		t.Fatalf("DecryptCBC() = %q, want %q", decrypted, "legacy-data")
	}
}

func TestCipherCTRNoPadding(t *testing.T) {
	c, err := utils.AES("1234567812345678", utils.WithRandIV(true))
	if err != nil {
		t.Fatal(err)
	}

	raw := []byte("short")
	encrypted, err := c.EncryptCTR(raw, utils.NoPadding)
	if err != nil {
		t.Fatalf("EncryptCTR() error = %v", err)
	}
	decrypted, err := c.DecryptCTR(encrypted, utils.NoUnPadding)
	if err != nil {
		t.Fatalf("DecryptCTR() error = %v", err)
	}
	if string(decrypted) != string(raw) {
		t.Fatalf("DecryptCTR() = %q, want %q", decrypted, raw)
	}
}

func TestCipherGCM(t *testing.T) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		t.Fatal(err)
	}

	aad := []byte("request-id=r-1")
	encrypted, err := c.EncryptGCM([]byte("secure payload"), aad)
	if err != nil {
		t.Fatalf("EncryptGCM() error = %v", err)
	}
	decrypted, err := c.DecryptGCM(encrypted, aad)
	if err != nil {
		t.Fatalf("DecryptGCM() error = %v", err)
	}
	if string(decrypted) != "secure payload" {
		t.Fatalf("DecryptGCM() = %q, want %q", decrypted, "secure payload")
	}
}

func TestCipherGCMRejectsTamperedAAD(t *testing.T) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		t.Fatal(err)
	}

	encrypted, err := c.EncryptGCM([]byte("secure payload"), []byte("aad-1"))
	if err != nil {
		t.Fatalf("EncryptGCM() error = %v", err)
	}
	if _, err = c.DecryptGCM(encrypted, []byte("aad-2")); err == nil {
		t.Fatal("DecryptGCM() expected auth error")
	}
}

func TestCipherGCMRejectsDES(t *testing.T) {
	c, err := utils.DES("12345678")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.EncryptGCM([]byte("secure payload"), nil); err == nil {
		t.Fatal("EncryptGCM() expected unsupported block size error")
	}
}

func TestRSAWithoutPEMHeaders(t *testing.T) {
	dir := t.TempDir()
	files, err := utils.GenerateKeyRSA(dir, 2048)
	if err != nil {
		t.Fatal(err)
	}

	pub, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	pri, err := os.ReadFile(files[1])
	if err != nil {
		t.Fatal(err)
	}

	r, err := utils.NewRSA(utils.RemovePEMHeaders(string(pub)), utils.RemovePEMHeaders(string(pri)))
	if err != nil {
		t.Fatal(err)
	}

	encrypted, err := r.EncryptOAEP("hello", base64.StdEncoding.EncodeToString, sha256.New())
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := r.DecryptOAEP(encrypted, base64.StdEncoding.DecodeString, sha256.New())
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "hello" {
		t.Fatalf("decrypted = %q, want hello", decrypted)
	}

	if mode := fileMode(t, files[1]); mode.Perm() != 0o600 {
		t.Fatalf("private key perm = %v, want 0600", mode.Perm())
	}
}

func BenchmarkCipherAESCBCEncrypt(b *testing.B) {
	c, err := utils.AES("1234567812345678", utils.WithIV("1234567812345678"))
	if err != nil {
		b.Fatal(err)
	}
	data := []byte("benchmark payload")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchCipherBytes, err = c.EncryptBytes(data, utils.CBC, utils.Pkcs7Padding)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCipherAESCBCDecrypt(b *testing.B) {
	c, err := utils.AES("1234567812345678", utils.WithIV("1234567812345678"))
	if err != nil {
		b.Fatal(err)
	}
	encrypted, err := c.EncryptBytes([]byte("benchmark payload"), utils.CBC, utils.Pkcs7Padding)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchCipherBytes, err = c.DecryptBytes(encrypted, utils.CBC, utils.Pkcs7UnPadding)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCipherAESCTRNoPaddingEncrypt(b *testing.B) {
	c, err := utils.AES("1234567812345678", utils.WithRandIV(true))
	if err != nil {
		b.Fatal(err)
	}
	data := []byte("benchmark payload")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchCipherBytes, err = c.EncryptCTR(data, utils.NoPadding)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCipherAESCTRNoPaddingEncryptTo(b *testing.B) {
	c, err := utils.AES("1234567812345678", utils.WithRandIV(true))
	if err != nil {
		b.Fatal(err)
	}
	// data 是流模式加密输入，NoPadding 下 EncryptTo 可直接写入复用缓冲区。
	data := []byte("benchmark payload")
	// dst 是循环复用的密文缓冲区，用于衡量减少输出切片分配后的收益。
	dst := make([]byte, 0, len(data)+16)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dst = dst[:0]
		benchCipherBytes, err = c.EncryptTo(dst, data, utils.CTR, utils.NoPadding)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCipherAESCTRNoPaddingDecrypt(b *testing.B) {
	c, err := utils.AES("1234567812345678", utils.WithRandIV(true))
	if err != nil {
		b.Fatal(err)
	}
	encrypted, err := c.EncryptCTR([]byte("benchmark payload"), utils.NoPadding)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchCipherBytes, err = c.DecryptCTR(encrypted, utils.NoUnPadding)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCipherAESCTRNoPaddingDecryptTo(b *testing.B) {
	c, err := utils.AES("1234567812345678", utils.WithRandIV(true))
	if err != nil {
		b.Fatal(err)
	}
	encrypted, err := c.EncryptCTR([]byte("benchmark payload"), utils.NoPadding)
	if err != nil {
		b.Fatal(err)
	}
	// dst 是循环复用的明文缓冲区，用于衡量减少输出切片分配后的收益。
	dst := make([]byte, 0, len(encrypted))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dst = dst[:0]
		benchCipherBytes, err = c.DecryptTo(dst, encrypted, utils.CTR, utils.NoUnPadding)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCipherAESGCMEncrypt(b *testing.B) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		b.Fatal(err)
	}
	data := []byte("benchmark payload")
	aad := []byte("bench")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchCipherBytes, err = c.EncryptGCM(data, aad)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCipherAESGCMDecrypt(b *testing.B) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		b.Fatal(err)
	}
	aad := []byte("bench")
	encrypted, err := c.EncryptGCM([]byte("benchmark payload"), aad)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchCipherBytes, err = c.DecryptGCM(encrypted, aad)
		if err != nil {
			b.Fatal(err)
		}
	}
}

var benchCipherBytes []byte

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode()
}
