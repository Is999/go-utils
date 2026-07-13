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
				encrypted, err := c.Encrypt(data, utils.CBC, base64.StdEncoding.EncodeToString, utils.PKCS7Pad)
				if err != nil {
					t.Errorf("Encrypt() error = %v", err)
					return
				}
				decrypted, err := c.Decrypt(encrypted, utils.CBC, base64.StdEncoding.DecodeString, utils.PKCS7Unpad)
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

func TestCipherEncryptToDecryptToCTRNoPad(t *testing.T) {
	c, err := utils.AES("1234567812345678", utils.WithRandIV(true), utils.WithAllowUnsafeStreamMode(true))
	if err != nil {
		t.Fatal(err)
	}

	// data 是流模式原始业务数据，用于验证 NoPad 快路径不会额外复制或裁剪内容。
	data := []byte("payload for dst reuse")
	// encryptedDst 是调用方复用的密文缓冲区，前缀用于验证 EncryptTo 会追加而不是覆盖历史内容。
	encryptedDst := []byte("prefix:")
	encrypted, err := c.EncryptTo(encryptedDst, data, utils.CTR, utils.NoPad)
	if err != nil {
		t.Fatalf("EncryptTo() error = %v", err)
	}
	if !bytes.HasPrefix(encrypted, encryptedDst) {
		t.Fatalf("EncryptTo() should keep dst prefix, got %q", encrypted)
	}

	// decryptedDst 是调用方复用的明文缓冲区，解密输入跳过前缀后应恢复原始 data。
	decryptedDst := []byte("plain:")
	decrypted, err := c.DecryptTo(decryptedDst, encrypted[len(encryptedDst):], utils.CTR, utils.NoUnpad)
	if err != nil {
		t.Fatalf("DecryptTo() error = %v", err)
	}
	if !bytes.Equal(decrypted[len(decryptedDst):], data) {
		t.Fatalf("DecryptTo() = %q, want suffix %q", decrypted, data)
	}
}

func TestCipherGCMStringRoundTrip(t *testing.T) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		t.Fatal(err)
	}

	const plaintext = "gcm payload"
	aad := []byte("request-id:1")
	encrypted, err := c.EncryptGCMString(plaintext, base64.StdEncoding.EncodeToString, aad)
	if err != nil {
		t.Fatalf("EncryptGCMString() error = %v", err)
	}
	decrypted, err := c.DecryptGCMString(encrypted, base64.StdEncoding.DecodeString, aad)
	if err != nil {
		t.Fatalf("DecryptGCMString() error = %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("DecryptGCMString() = %q, want %q", decrypted, plaintext)
	}
	if _, err := c.EncryptGCMString(plaintext, nil, aad); err == nil {
		t.Fatal("EncryptGCMString() with nil encoder expected error")
	}
	if _, err := c.DecryptGCMString(encrypted, nil, aad); err == nil {
		t.Fatal("DecryptGCMString() with nil decoder expected error")
	}
}

func TestPKCS7UnpadRejectsInvalidPadding(t *testing.T) {
	tests := [][]byte{
		{},
		{1, 2, 3, 0},
		{1, 2, 3, 5},
		{1, 2, 3, 2, 3},
	}
	for _, tt := range tests {
		if _, err := utils.PKCS7Unpad(tt); err == nil {
			t.Fatalf("utils.PKCS7Unpad(%v) expected error", tt)
		}
	}
}

func TestPaddingDoesNotMutateInput(t *testing.T) {
	src := make([]byte, 2, 16)
	copy(src, "ab")

	padded := utils.PKCS7Pad(src, 8)
	padded[0] = 'x'
	if string(src) != "ab" {
		t.Fatalf("utils.PKCS7Pad mutated input: %q", src)
	}

	zeroPadded := utils.ZeroPad(src, 8)
	zeroPadded[0] = 'y'
	if string(src) != "ab" {
		t.Fatalf("utils.ZeroPad mutated input: %q", src)
	}

	noPadding := utils.NoPad(src, 8)
	noPadding[0] = 'z'
	if string(src) != "ab" {
		t.Fatalf("utils.NoPad mutated input: %q", src)
	}

	noUnpadding, err := utils.NoUnpad(src)
	if err != nil {
		t.Fatalf("utils.NoUnpad() error = %v", err)
	}
	noUnpadding[0] = 'w'
	if string(src) != "ab" {
		t.Fatalf("utils.NoUnpad mutated input: %q", src)
	}
}

func TestCipherRejectsNilCallbacks(t *testing.T) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Encrypt("data", utils.CBC, nil, utils.PKCS7Pad); err == nil {
		t.Fatal("Encrypt() with nil encode expected error")
	}
	if _, err := c.Encrypt("data", utils.CBC, base64.StdEncoding.EncodeToString, nil); err == nil {
		t.Fatal("Encrypt() with nil padding expected error")
	}
	if _, err := c.Decrypt("data", utils.CBC, nil, utils.PKCS7Unpad); err == nil {
		t.Fatal("Decrypt() with nil decode expected error")
	}
	if _, err := c.DecryptBytes([]byte("data"), utils.CBC, nil); err == nil {
		t.Fatal("DecryptBytes() with nil unpad expected error")
	}
}

func TestCipherRejectsUnsafeStreamModesByDefault(t *testing.T) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.EncryptCTR([]byte("data"), utils.NoPad); err == nil {
		t.Fatal("EncryptCTR() expected error when unsafe stream mode is disabled")
	}
	if _, err := c.EncryptCFB([]byte("data"), utils.PKCS7Pad); err == nil {
		t.Fatal("EncryptCFB() expected error when unsafe stream mode is disabled")
	}
	if _, err := c.EncryptOFB([]byte("data"), utils.PKCS7Pad); err == nil {
		t.Fatal("EncryptOFB() expected error when unsafe stream mode is disabled")
	}
}

func TestCipherRejectsECBByDefault(t *testing.T) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.EncryptECB([]byte("data"), utils.PKCS7Pad); err == nil {
		t.Fatal("EncryptECB() expected error when utils.ECB is disabled")
	}
}

func TestCipherBlockModesNoPadRejectPartialBlockWithoutPanic(t *testing.T) {
	c, err := utils.AES(
		"1234567812345678",
		utils.WithIV("abcdefgh12345678"),
		utils.WithAllowUnsafeECB(true),
	)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		run  func() error
	}{
		{
			name: "EncryptECB",
			run: func() error {
				_, err := c.EncryptECB([]byte("short"), utils.NoPad)
				return err
			},
		},
		{
			name: "EncryptCBC",
			run: func() error {
				_, err := c.EncryptCBC([]byte("short"), utils.NoPad)
				return err
			},
		},
		{
			name: "EncryptBytes ECB",
			run: func() error {
				_, err := c.EncryptBytes([]byte("short"), utils.ECB, utils.NoPad)
				return err
			},
		},
		{
			name: "EncryptBytes CBC",
			run: func() error {
				_, err := c.EncryptBytes([]byte("short"), utils.CBC, utils.NoPad)
				return err
			},
		},
		{
			name: "EncryptTo ECB",
			run: func() error {
				_, err := c.EncryptTo(nil, []byte("short"), utils.ECB, utils.NoPad)
				return err
			},
		},
		{
			name: "EncryptTo CBC",
			run: func() error {
				_, err := c.EncryptTo(nil, []byte("short"), utils.CBC, utils.NoPad)
				return err
			},
		},
		{
			name: "Encrypt string ECB",
			run: func() error {
				_, err := c.Encrypt("short", utils.ECB, base64.StdEncoding.EncodeToString, utils.NoPad)
				return err
			},
		},
		{
			name: "Encrypt string CBC",
			run: func() error {
				_, err := c.Encrypt("short", utils.CBC, base64.StdEncoding.EncodeToString, utils.NoPad)
				return err
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("%s panicked: %v", tt.name, recovered)
				}
			}()
			if err := tt.run(); err == nil {
				t.Fatalf("%s expected partial-block error", tt.name)
			}
		})
	}
}

func TestCipherNoPadEmptyRoundTrip(t *testing.T) {
	c, err := utils.AES(
		"1234567812345678",
		utils.WithIV("abcdefgh12345678"),
		utils.WithAllowUnsafeECB(true),
		utils.WithAllowUnsafeStreamMode(true),
	)
	if err != nil {
		t.Fatal(err)
	}

	modes := []struct {
		name string
		mode utils.CipherMode
	}{
		{name: "ECB", mode: utils.ECB},
		{name: "CBC", mode: utils.CBC},
		{name: "CTR", mode: utils.CTR},
		{name: "CFB", mode: utils.CFB},
		{name: "OFB", mode: utils.OFB},
	}
	for _, tt := range modes {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := c.EncryptBytes(nil, tt.mode, utils.NoPad)
			if err != nil {
				t.Fatalf("EncryptBytes() error = %v", err)
			}
			decrypted, err := c.DecryptBytes(encrypted, tt.mode, utils.NoUnpad)
			if err != nil {
				t.Fatalf("DecryptBytes() error = %v", err)
			}
			if len(decrypted) != 0 {
				t.Fatalf("DecryptBytes() length = %d, want 0", len(decrypted))
			}
		})
	}
}

func TestNilCipherStreamModesReturnError(t *testing.T) {
	var c *utils.Cipher
	cases := []struct {
		name string
		run  func() error
	}{
		{name: "CTR", run: func() error { _, err := c.EncryptCTR(nil, utils.NoPad); return err }},
		{name: "CFB", run: func() error { _, err := c.EncryptCFB(nil, utils.NoPad); return err }},
		{name: "OFB", run: func() error { _, err := c.EncryptOFB(nil, utils.NoPad); return err }},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("Encrypt%s() panicked: %v", tt.name, recovered)
				}
			}()
			if err := tt.run(); err == nil {
				t.Fatalf("Encrypt%s() error = nil, want key error", tt.name)
			}
		})
	}
}

func TestCipherRejectsImplicitKeyIVByDefault(t *testing.T) {
	c, err := utils.AES("1234567812345678")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.EncryptCBC([]byte("data"), utils.PKCS7Pad); err == nil {
		t.Fatal("EncryptCBC() expected error when IV is not configured")
	}
	ctr, err := utils.AES("1234567812345678", utils.WithAllowUnsafeStreamMode(true))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ctr.EncryptCTR([]byte("data"), utils.PKCS7Pad); err == nil {
		t.Fatal("EncryptCTR() expected error when IV is not configured")
	}
}

func TestCipherAllowsUnsafeStreamModesWhenEnabled(t *testing.T) {
	c, err := utils.AES("1234567812345678", utils.WithAllowUnsafeStreamMode(true), utils.WithAllowUnsafeKeyIV(true))
	if err != nil {
		t.Fatal(err)
	}

	ctrEncrypted, err := c.EncryptCTR([]byte("legacy-data"), utils.PKCS7Pad)
	if err != nil {
		t.Fatalf("EncryptCTR() error = %v", err)
	}
	ctrDecrypted, err := c.DecryptCTR(ctrEncrypted, utils.PKCS7Unpad)
	if err != nil {
		t.Fatalf("DecryptCTR() error = %v", err)
	}
	if string(ctrDecrypted) != "legacy-data" {
		t.Fatalf("DecryptCTR() = %q, want %q", ctrDecrypted, "legacy-data")
	}

	encrypted, err := c.EncryptCFB([]byte("legacy-data"), utils.PKCS7Pad)
	if err != nil {
		t.Fatalf("EncryptCFB() error = %v", err)
	}
	decrypted, err := c.DecryptCFB(encrypted, utils.PKCS7Unpad)
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
	encrypted, err := c.EncryptECB([]byte("legacy-data"), utils.PKCS7Pad)
	if err != nil {
		t.Fatalf("EncryptECB() error = %v", err)
	}
	decrypted, err := c.DecryptECB(encrypted, utils.PKCS7Unpad)
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
	encrypted, err := c.EncryptCBC([]byte("legacy-data"), utils.PKCS7Pad)
	if err != nil {
		t.Fatalf("EncryptCBC() error = %v", err)
	}
	decrypted, err := c.DecryptCBC(encrypted, utils.PKCS7Unpad)
	if err != nil {
		t.Fatalf("DecryptCBC() error = %v", err)
	}
	if string(decrypted) != "legacy-data" {
		t.Fatalf("DecryptCBC() = %q, want %q", decrypted, "legacy-data")
	}
}

func TestCipherCTRNoPad(t *testing.T) {
	c, err := utils.AES("1234567812345678", utils.WithRandIV(true), utils.WithAllowUnsafeStreamMode(true))
	if err != nil {
		t.Fatal(err)
	}

	raw := []byte("short")
	encrypted, err := c.EncryptCTR(raw, utils.NoPad)
	if err != nil {
		t.Fatalf("EncryptCTR() error = %v", err)
	}
	decrypted, err := c.DecryptCTR(encrypted, utils.NoUnpad)
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
		benchCipherBytes, err = c.EncryptBytes(data, utils.CBC, utils.PKCS7Pad)
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
	encrypted, err := c.EncryptBytes([]byte("benchmark payload"), utils.CBC, utils.PKCS7Pad)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchCipherBytes, err = c.DecryptBytes(encrypted, utils.CBC, utils.PKCS7Unpad)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCipherAESCTRNoPadEncrypt(b *testing.B) {
	c, err := utils.AES("1234567812345678", utils.WithRandIV(true), utils.WithAllowUnsafeStreamMode(true))
	if err != nil {
		b.Fatal(err)
	}
	data := []byte("benchmark payload")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchCipherBytes, err = c.EncryptCTR(data, utils.NoPad)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCipherAESCTRNoPadEncryptTo(b *testing.B) {
	c, err := utils.AES("1234567812345678", utils.WithRandIV(true), utils.WithAllowUnsafeStreamMode(true))
	if err != nil {
		b.Fatal(err)
	}
	// data 是流模式加密输入，NoPad 下 EncryptTo 可直接写入复用缓冲区。
	data := []byte("benchmark payload")
	// dst 是循环复用的密文缓冲区，用于衡量减少输出切片分配后的收益。
	dst := make([]byte, 0, len(data)+16)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dst = dst[:0]
		benchCipherBytes, err = c.EncryptTo(dst, data, utils.CTR, utils.NoPad)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCipherAESCTRNoPadDecrypt(b *testing.B) {
	c, err := utils.AES("1234567812345678", utils.WithRandIV(true), utils.WithAllowUnsafeStreamMode(true))
	if err != nil {
		b.Fatal(err)
	}
	encrypted, err := c.EncryptCTR([]byte("benchmark payload"), utils.NoPad)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchCipherBytes, err = c.DecryptCTR(encrypted, utils.NoUnpad)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCipherAESCTRNoPadDecryptTo(b *testing.B) {
	c, err := utils.AES("1234567812345678", utils.WithRandIV(true), utils.WithAllowUnsafeStreamMode(true))
	if err != nil {
		b.Fatal(err)
	}
	encrypted, err := c.EncryptCTR([]byte("benchmark payload"), utils.NoPad)
	if err != nil {
		b.Fatal(err)
	}
	// dst 是循环复用的明文缓冲区，用于衡量减少输出切片分配后的收益。
	dst := make([]byte, 0, len(encrypted))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dst = dst[:0]
		benchCipherBytes, err = c.DecryptTo(dst, encrypted, utils.CTR, utils.NoUnpad)
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
