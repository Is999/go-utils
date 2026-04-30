package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestCipherConcurrentRandIV(t *testing.T) {
	c, err := AES("1234567812345678", WithRandIV(true))
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
				encrypted, err := c.Encrypt(data, CBC, base64.StdEncoding.EncodeToString, Pkcs7Padding)
				if err != nil {
					t.Errorf("Encrypt() error = %v", err)
					return
				}
				decrypted, err := c.Decrypt(encrypted, CBC, base64.StdEncoding.DecodeString, Pkcs7UnPadding)
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

func TestPkcs7UnPaddingRejectsInvalidPadding(t *testing.T) {
	tests := [][]byte{
		{},
		{1, 2, 3, 0},
		{1, 2, 3, 5},
		{1, 2, 3, 2, 3},
	}
	for _, tt := range tests {
		if _, err := Pkcs7UnPadding(tt); err == nil {
			t.Fatalf("Pkcs7UnPadding(%v) expected error", tt)
		}
	}
}

func TestPaddingDoesNotMutateInput(t *testing.T) {
	src := make([]byte, 2, 16)
	copy(src, "ab")

	padded := Pkcs7Padding(src, 8)
	padded[0] = 'x'
	if string(src) != "ab" {
		t.Fatalf("Pkcs7Padding mutated input: %q", src)
	}

	zeroPadded := ZeroPadding(src, 8)
	zeroPadded[0] = 'y'
	if string(src) != "ab" {
		t.Fatalf("ZeroPadding mutated input: %q", src)
	}
}

func TestCipherRejectsNilCallbacks(t *testing.T) {
	c, err := AES("1234567812345678")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Encrypt("data", CBC, nil, Pkcs7Padding); err == nil {
		t.Fatal("Encrypt() with nil encode expected error")
	}
	if _, err := c.Encrypt("data", CBC, base64.StdEncoding.EncodeToString, nil); err == nil {
		t.Fatal("Encrypt() with nil padding expected error")
	}
	if _, err := c.Decrypt("data", CBC, nil, Pkcs7UnPadding); err == nil {
		t.Fatal("Decrypt() with nil decode expected error")
	}
	if _, err := c.DecryptBytes([]byte("data"), CBC, nil); err == nil {
		t.Fatal("DecryptBytes() with nil unPadding expected error")
	}
}

func TestRSAWithoutPEMHeaders(t *testing.T) {
	dir := t.TempDir()
	files, err := GenerateKeyRSA(dir, 1024)
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

	r, err := NewRSA(RemovePEMHeaders(string(pub)), RemovePEMHeaders(string(pri)))
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
	c, err := AES("1234567812345678", WithIV("1234567812345678"))
	if err != nil {
		b.Fatal(err)
	}
	data := []byte("benchmark payload")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchCipherBytes, err = c.EncryptBytes(data, CBC, Pkcs7Padding)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCipherAESCBCDecrypt(b *testing.B) {
	c, err := AES("1234567812345678", WithIV("1234567812345678"))
	if err != nil {
		b.Fatal(err)
	}
	encrypted, err := c.EncryptBytes([]byte("benchmark payload"), CBC, Pkcs7Padding)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchCipherBytes, err = c.DecryptBytes(encrypted, CBC, Pkcs7UnPadding)
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
