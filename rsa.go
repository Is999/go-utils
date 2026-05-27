package utils

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"hash"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Is999/go-utils/errors"
)

// RSA 封装 RSA 公钥、私钥及常用加解密/签名能力。
//
// 说明：
//   - RSA 自身适合加密小块数据；本实现会按密钥长度自动分块，便于加密较长字符串。
//   - 新系统优先推荐 EncryptOAEP/DecryptOAEP；Encrypt/Decrypt(PKCS#1 v1.5) 主要用于兼容旧系统。
//   - RSA 对象初始化后只读，公钥/私钥可被多个 goroutine 并发用于加解密和验签。
type RSA struct {
	pubKey *rsa.PublicKey  // 公钥
	priKey *rsa.PrivateKey // 私钥
}

// RSAOption RSA 配置项。
type RSAOption func(*rsaOptions)

// rsaOptions 保存 RSA 初始化选项。
type rsaOptions struct {
	isFilePath bool // 密钥字符串是否按文件路径读取。
}

// RSA 安全边界常量。
const (
	// minRSABits 定义生产环境建议的最小 RSA 密钥位数。
	minRSABits = 2048
)

// WithRSAFilePath 指定密钥参数是否为文件路径。
func WithRSAFilePath(isFilePath bool) RSAOption {
	return func(o *rsaOptions) {
		o.isFilePath = isFilePath
	}
}

// NewRSA 实例化 RSA，并同时设置公钥和私钥。
func NewRSA(pub, pri string, opts ...RSAOption) (*RSA, error) {
	cfg := parseRSAOptions(opts...)
	r := &RSA{}
	if err := r.SetPublicKey(pub, cfg.isFilePath); err != nil {
		return r, errors.Tag(err)
	}
	if err := r.SetPrivateKey(pri, cfg.isFilePath); err != nil {
		return r, errors.Tag(err)
	}
	return r, nil
}

// NewPubRSA 实例化只含公钥的 RSA，用于加密或验签。
func NewPubRSA(pub string, opts ...RSAOption) (*RSA, error) {
	cfg := parseRSAOptions(opts...)
	r := &RSA{}
	if err := r.SetPublicKey(pub, cfg.isFilePath); err != nil {
		return r, errors.Tag(err)
	}
	return r, nil
}

// NewPriRSA 实例化只含私钥的 RSA，用于解密或签名。
func NewPriRSA(pri string, opts ...RSAOption) (*RSA, error) {
	cfg := parseRSAOptions(opts...)
	r := &RSA{}
	if err := r.SetPrivateKey(pri, cfg.isFilePath); err != nil {
		return r, errors.Tag(err)
	}
	return r, nil
}

// parseRSAOptions 解析 RSA 选项，nil 选项会被忽略。
func parseRSAOptions(opts ...RSAOption) rsaOptions {
	cfg := rsaOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

// SetPublicKey 设置公钥。
//
// publicKey 可以是 PEM 文本、去掉头尾后的 base64 DER 文本，或文件路径。
func (r *RSA) SetPublicKey(publicKey string, isFilePath bool) error {
	key, err := readKeyData(publicKey, isFilePath)
	if err != nil {
		return errors.Tag(err)
	}
	der, err := decodeKeyDER(key, "PUBLIC")
	if err != nil {
		return errors.Tag(err)
	}
	pub, err := parseRSAPublicKey(der)
	if err != nil {
		return errors.Tag(err)
	}
	r.pubKey = pub
	return nil
}

// SetPrivateKey 设置私钥。
//
// privateKey 可以是 PEM 文本、去掉头尾后的 base64 DER 文本，或文件路径。
func (r *RSA) SetPrivateKey(privateKey string, isFilePath bool) error {
	key, err := readKeyData(privateKey, isFilePath)
	if err != nil {
		return errors.Tag(err)
	}
	der, err := decodeKeyDER(key, "PRIVATE")
	if err != nil {
		return errors.Tag(err)
	}
	pri, err := parseRSAPrivateKey(der)
	if err != nil {
		return errors.Tag(err)
	}
	r.priKey = pri
	return nil
}

// IsSetPublicKey 校验公钥是否已设置。
func (r *RSA) IsSetPublicKey() error {
	if r == nil || r.pubKey == nil {
		return errors.New("RSA 公钥未设置")
	}
	return nil
}

// IsSetPrivateKey 校验私钥是否已设置。
func (r *RSA) IsSetPrivateKey() error {
	if r == nil || r.priKey == nil {
		return errors.New("RSA 私钥未设置")
	}
	return nil
}

// Encrypt 使用公钥和 PKCS#1 v1.5 填充加密。
//
// 生产新协议更建议使用 EncryptOAEP。
func (r *RSA) Encrypt(data string, encode EncodeToString) (string, error) {
	if encode == nil {
		return "", errors.New("encode 不能为空")
	}
	if err := r.IsSetPublicKey(); err != nil {
		return "", errors.Tag(err)
	}

	keySize := r.pubKey.Size()
	maxPayload := keySize - 11
	encrypted, err := rsaEncryptChunks([]byte(data), keySize, maxPayload, func(chunk []byte) ([]byte, error) {
		return rsaEncryptPKCS1v15Legacy(r.pubKey, chunk)
	})
	if err != nil {
		return "", errors.Tag(err)
	}
	return encode(encrypted), nil
}

// Decrypt 使用私钥和 PKCS#1 v1.5 填充解密。
func (r *RSA) Decrypt(encrypt string, decode DecodeString) (string, error) {
	if decode == nil {
		return "", errors.New("decode 不能为空")
	}
	if err := r.IsSetPrivateKey(); err != nil {
		return "", errors.Tag(err)
	}

	ciphertext, err := decode(encrypt)
	if err != nil {
		return "", errors.Tag(err)
	}
	decrypted, err := rsaDecryptChunks(ciphertext, r.priKey.Size(), func(chunk []byte) ([]byte, error) {
		return rsaDecryptPKCS1v15Legacy(r.priKey, chunk)
	})
	if err != nil {
		return "", errors.Tag(err)
	}
	return string(decrypted), nil
}

// rsaEncryptPKCS1v15Legacy 使用 PKCS#1 v1.5 兼容旧密文协议。
func rsaEncryptPKCS1v15Legacy(pub *rsa.PublicKey, data []byte) ([]byte, error) {
	//lint:ignore SA1019 PKCS#1 v1.5 加密仅保留为历史协议兼容，新协议应使用 OAEP。
	return rsa.EncryptPKCS1v15(rand.Reader, pub, data)
}

// rsaDecryptPKCS1v15Legacy 使用 PKCS#1 v1.5 兼容旧密文协议。
func rsaDecryptPKCS1v15Legacy(pri *rsa.PrivateKey, data []byte) ([]byte, error) {
	//lint:ignore SA1019 PKCS#1 v1.5 解密仅保留为历史协议兼容，新协议应使用 OAEP。
	return rsa.DecryptPKCS1v15(rand.Reader, pri, data)
}

// Sign 使用私钥生成 PKCS#1 v1.5 签名。
func (r *RSA) Sign(data string, hash crypto.Hash, encode EncodeToString) (string, error) {
	if encode == nil {
		return "", errors.New("encode 不能为空")
	}
	if err := validateRSASignHash(hash); err != nil {
		return "", errors.Tag(err)
	}
	if err := r.IsSetPrivateKey(); err != nil {
		return "", errors.Tag(err)
	}
	hashed, err := hashBytes([]byte(data), hash)
	if err != nil {
		return "", errors.Tag(err)
	}
	sign, err := rsa.SignPKCS1v15(rand.Reader, r.priKey, hash, hashed)
	if err != nil {
		return "", errors.Tag(err)
	}
	return encode(sign), nil
}

// Verify 使用公钥验证 PKCS#1 v1.5 签名。
func (r *RSA) Verify(data, sign string, hash crypto.Hash, decode DecodeString) error {
	if decode == nil {
		return errors.New("decode 不能为空")
	}
	if err := validateRSASignHash(hash); err != nil {
		return errors.Tag(err)
	}
	if err := r.IsSetPublicKey(); err != nil {
		return errors.Tag(err)
	}
	signByte, err := decode(sign)
	if err != nil {
		return errors.Tag(err)
	}
	hashed, err := hashBytes([]byte(data), hash)
	if err != nil {
		return errors.Tag(err)
	}
	return rsa.VerifyPKCS1v15(r.pubKey, hash, hashed, signByte)
}

// EncryptOAEP 使用公钥和 OAEP 填充加密。
//
// 注意：hash.Hash 实例不是并发安全对象；并发场景推荐使用 EncryptOAEPHash，由方法内部创建摘要实例。
func (r *RSA) EncryptOAEP(data string, encode EncodeToString, hash hash.Hash) (string, error) {
	if encode == nil {
		return "", errors.New("encode 不能为空")
	}
	if err := validateRSAOAEPHash(hash); err != nil {
		return "", errors.Tag(err)
	}
	if err := r.IsSetPublicKey(); err != nil {
		return "", errors.Tag(err)
	}

	keySize := r.pubKey.Size()
	maxPayload := keySize - 2*hash.Size() - 2
	encrypted, err := rsaEncryptChunks([]byte(data), keySize, maxPayload, func(chunk []byte) ([]byte, error) {
		hash.Reset()
		return rsa.EncryptOAEP(hash, rand.Reader, r.pubKey, chunk, nil)
	})
	if err != nil {
		return "", errors.Tag(err)
	}
	return encode(encrypted), nil
}

// DecryptOAEP 使用私钥和 OAEP 填充解密。
//
// 注意：hash.Hash 实例不是并发安全对象；并发场景推荐使用 DecryptOAEPHash，由方法内部创建摘要实例。
func (r *RSA) DecryptOAEP(encrypt string, decode DecodeString, hash hash.Hash) (string, error) {
	if decode == nil {
		return "", errors.New("decode 不能为空")
	}
	if err := validateRSAOAEPHash(hash); err != nil {
		return "", errors.Tag(err)
	}
	if err := r.IsSetPrivateKey(); err != nil {
		return "", errors.Tag(err)
	}

	ciphertext, err := decode(encrypt)
	if err != nil {
		return "", errors.Tag(err)
	}
	decrypted, err := rsaDecryptChunks(ciphertext, r.priKey.Size(), func(chunk []byte) ([]byte, error) {
		hash.Reset()
		return rsa.DecryptOAEP(hash, rand.Reader, r.priKey, chunk, nil)
	})
	if err != nil {
		return "", errors.Tag(err)
	}
	return string(decrypted), nil
}

// EncryptOAEPHash 使用指定 crypto.Hash 创建独立摘要实例并执行 OAEP 加密。
//
// 该方法比直接传 hash.Hash 更适合高并发复用 RSA 对象，生产代码建议优先使用 SHA256 及以上摘要算法。
func (r *RSA) EncryptOAEPHash(data string, encode EncodeToString, hashID crypto.Hash) (string, error) {
	if encode == nil {
		return "", errors.New("encode 不能为空")
	}
	if err := validateRSAOAEPHashID(hashID); err != nil {
		return "", errors.Tag(err)
	}
	if err := r.IsSetPublicKey(); err != nil {
		return "", errors.Tag(err)
	}

	keySize := r.pubKey.Size()
	maxPayload := keySize - 2*hashID.Size() - 2
	oaepHash := hashID.New()
	encrypted, err := rsaEncryptChunks([]byte(data), keySize, maxPayload, func(chunk []byte) ([]byte, error) {
		// 单次方法调用内复用摘要对象，避免每个 RSA 分块重复分配。
		oaepHash.Reset()
		return rsa.EncryptOAEP(oaepHash, rand.Reader, r.pubKey, chunk, nil)
	})
	if err != nil {
		return "", errors.Tag(err)
	}
	return encode(encrypted), nil
}

// DecryptOAEPHash 使用指定 crypto.Hash 创建独立摘要实例并执行 OAEP 解密。
//
// 该方法比直接传 hash.Hash 更适合高并发复用 RSA 对象，生产代码建议优先使用 SHA256 及以上摘要算法。
func (r *RSA) DecryptOAEPHash(encrypt string, decode DecodeString, hashID crypto.Hash) (string, error) {
	if decode == nil {
		return "", errors.New("decode 不能为空")
	}
	if err := validateRSAOAEPHashID(hashID); err != nil {
		return "", errors.Tag(err)
	}
	if err := r.IsSetPrivateKey(); err != nil {
		return "", errors.Tag(err)
	}

	ciphertext, err := decode(encrypt)
	if err != nil {
		return "", errors.Tag(err)
	}
	oaepHash := hashID.New()
	decrypted, err := rsaDecryptChunks(ciphertext, r.priKey.Size(), func(chunk []byte) ([]byte, error) {
		// 单次方法调用内复用摘要对象，避免每个 RSA 分块重复分配。
		oaepHash.Reset()
		return rsa.DecryptOAEP(oaepHash, rand.Reader, r.priKey, chunk, nil)
	})
	if err != nil {
		return "", errors.Tag(err)
	}
	return string(decrypted), nil
}

// SignPSS 使用私钥生成 PSS 签名。
func (r *RSA) SignPSS(data string, hash crypto.Hash, encode EncodeToString, opts *rsa.PSSOptions) (string, error) {
	if encode == nil {
		return "", errors.New("encode 不能为空")
	}
	if err := validateRSASignHash(hash); err != nil {
		return "", errors.Tag(err)
	}
	if err := r.IsSetPrivateKey(); err != nil {
		return "", errors.Tag(err)
	}
	hashed, err := hashBytes([]byte(data), hash)
	if err != nil {
		return "", errors.Tag(err)
	}
	sign, err := rsa.SignPSS(rand.Reader, r.priKey, hash, hashed, opts)
	if err != nil {
		return "", errors.Tag(err)
	}
	return encode(sign), nil
}

// VerifyPSS 使用公钥验证 PSS 签名。
func (r *RSA) VerifyPSS(data, sign string, hash crypto.Hash, decode DecodeString, opts *rsa.PSSOptions) error {
	if decode == nil {
		return errors.New("decode 不能为空")
	}
	if err := validateRSASignHash(hash); err != nil {
		return errors.Tag(err)
	}
	if err := r.IsSetPublicKey(); err != nil {
		return errors.Tag(err)
	}
	signByte, err := decode(sign)
	if err != nil {
		return errors.Tag(err)
	}
	hashed, err := hashBytes([]byte(data), hash)
	if err != nil {
		return errors.Tag(err)
	}
	return rsa.VerifyPSS(r.pubKey, hash, hashed, signByte, opts)
}

// GenerateKeyRSA 生成 RSA 密钥文件。
//
// path 为密钥存放目录；bits 为密钥位数；生产环境要求至少 2048。
// pkcs[0] 控制公钥格式是否为 PKCS8，默认 true；pkcs[1] 控制私钥格式是否为 PKCS1，默认 true。
func GenerateKeyRSA(path string, bits int, pkcs ...bool) ([]string, error) {
	if bits < minRSABits {
		return nil, errors.Errorf("RSA 密钥位数不能低于 %d，当前位数: %d", minRSABits, bits)
	}
	if strings.TrimSpace(path) != "" {
		// 拒绝通过符号链接目录写入密钥文件，避免把私钥写到调用方预期目录之外。
		if info, err := os.Lstat(path); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return nil, errors.Errorf("GenerateKeyRSA() 不允许密钥目录为符号链接: path=%s", path)
			}
		} else if !os.IsNotExist(err) {
			return nil, errors.Tag(err)
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			return nil, errors.Tag(err)
		}
	}

	isPubPKCS8 := true
	isPriPKCS1 := true
	if len(pkcs) > 0 {
		isPubPKCS8 = pkcs[0]
		if len(pkcs) > 1 {
			isPriPKCS1 = pkcs[1]
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, errors.Tag(err)
	}

	privateStream, err := marshalPrivateKey(privateKey, isPriPKCS1)
	if err != nil {
		return nil, errors.Tag(err)
	}
	publicStream, err := marshalPublicKey(&privateKey.PublicKey, isPubPKCS8)
	if err != nil {
		return nil, errors.Tag(err)
	}

	now := time.Now()
	ts := now.Format(SecondTime) + "_" + strconv.FormatInt(now.UnixNano(), 36)
	fileName := make([]string, 2)
	fileName[0] = filepath.Join(path, Ternary(isPubPKCS8, "public_pkcs8_", "public_pkcs1_")+ts+".pem")
	fileName[1] = filepath.Join(path, Ternary(isPriPKCS1, "private_pkcs1_", "private_pkcs8_")+ts+".pem")

	publicType := Ternary(isPubPKCS8, "PUBLIC KEY", "RSA PUBLIC KEY")
	if err = writePEMFile(fileName[0], &pem.Block{Type: publicType, Bytes: publicStream}, 0o644); err != nil {
		return nil, errors.Tag(err)
	}

	// 保持旧版本 PEM 头兼容：PKCS8 私钥也使用 RSA PRIVATE KEY 头，解析时按 DER 自动识别。
	if err = writePEMFile(fileName[1], &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateStream}, 0o600); err != nil {
		return nil, errors.Tag(err)
	}
	return fileName, nil
}

// RemovePEMHeaders 去掉 PEM 头尾标记和空白字符。
func RemovePEMHeaders(pemText string) string {
	var b strings.Builder
	b.Grow(len(pemText))
	for line := range strings.SplitSeq(pemText, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "-----BEGIN ") || strings.HasPrefix(upper, "-----END ") {
			continue
		}
		b.WriteString(line)
	}
	return strings.TrimSpace(b.String())
}

// AddPEMHeaders 为 RSA 密钥串添加 PEM 头尾标记。
//
// keyType 支持 public/private。
func AddPEMHeaders(key, keyType string) (string, error) {
	var header, footer string
	switch {
	case strings.EqualFold(keyType, "public"):
		header = "-----BEGIN PUBLIC KEY-----"
		footer = "-----END PUBLIC KEY-----"
	case strings.EqualFold(keyType, "private"):
		header = "-----BEGIN RSA PRIVATE KEY-----"
		footer = "-----END RSA PRIVATE KEY-----"
	default:
		return "", errors.New("密钥类型错误")
	}

	body := RemovePEMHeaders(key)
	var b strings.Builder
	b.Grow(len(body) + len(header) + len(footer) + len(body)/64 + 4)
	b.WriteString(header)
	for i := 0; i < len(body); i += 64 {
		end := min(i+64, len(body))
		b.WriteByte('\n')
		b.WriteString(body[i:end])
	}
	b.WriteByte('\n')
	b.WriteString(footer)
	return b.String(), nil
}

// readKeyData 根据配置读取密钥数据。
func readKeyData(key string, isFilePath bool) ([]byte, error) {
	if isFilePath {
		return os.ReadFile(key)
	}
	return []byte(key), nil
}

// decodeKeyDER 将 PEM、base64 DER 或原始 DER 密钥统一转换为 DER 字节。
func decodeKeyDER(key []byte, wantType string) ([]byte, error) {
	key = bytes.TrimSpace(key)
	if len(key) == 0 {
		return nil, errors.New("密钥不能为空")
	}
	if block, _ := pem.Decode(key); block != nil {
		if wantType != "" && !strings.Contains(strings.ToUpper(block.Type), wantType) {
			return nil, errors.Errorf("%s类型错误", rsaKeyTypeName(wantType))
		}
		return block.Bytes, nil
	}

	body := RemovePEMHeaders(string(key))
	if body == "" {
		return nil, errors.New("密钥内容为空")
	}
	if der, err := base64.StdEncoding.DecodeString(body); err == nil {
		return der, nil
	}
	return key, nil
}

// rsaKeyTypeName 返回适合错误信息展示的密钥类型名称。
func rsaKeyTypeName(wantType string) string {
	switch strings.ToUpper(wantType) {
	case "PUBLIC":
		return "公钥"
	case "PRIVATE":
		return "私钥"
	default:
		return "密钥"
	}
}

// parseRSAPublicKey 解析 PKIX 或 PKCS#1 公钥并校验安全位数。
func parseRSAPublicKey(der []byte) (*rsa.PublicKey, error) {
	if pubAny, err := x509.ParsePKIXPublicKey(der); err == nil {
		pub, ok := pubAny.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("公钥类型错误")
		}
		if err = validateRSAPublicKey(pub); err != nil {
			return nil, errors.Tag(err)
		}
		return pub, nil
	}
	if pub, err := x509.ParsePKCS1PublicKey(der); err == nil {
		if err = validateRSAPublicKey(pub); err != nil {
			return nil, errors.Tag(err)
		}
		return pub, nil
	}
	return nil, errors.New("公钥解析失败")
}

// parseRSAPrivateKey 解析 PKCS#1 或 PKCS#8 私钥并校验安全位数。
func parseRSAPrivateKey(der []byte) (*rsa.PrivateKey, error) {
	if pri, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		if err = validateRSAPrivateKey(pri); err != nil {
			return nil, errors.Tag(err)
		}
		return pri, nil
	}
	if priAny, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		pri, ok := priAny.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("私钥类型错误")
		}
		if err = validateRSAPrivateKey(pri); err != nil {
			return nil, errors.Tag(err)
		}
		return pri, nil
	}
	return nil, errors.New("私钥解析失败")
}

// validateRSAPublicKey 校验 RSA 公钥是否满足生产安全下限。
//
// 参数说明：
//   - pub：待校验公钥。
//
// 返回值：错误信息。
func validateRSAPublicKey(pub *rsa.PublicKey) error {
	if pub == nil || pub.N == nil {
		return errors.New("RSA 公钥不能为空")
	}
	if bits := pub.N.BitLen(); bits < minRSABits {
		return errors.Errorf("RSA 公钥位数不能低于 %d，当前位数: %d", minRSABits, bits)
	}
	return nil
}

// validateRSAPrivateKey 校验 RSA 私钥结构和安全位数。
//
// 参数说明：
//   - pri：待校验私钥。
//
// 返回值：错误信息。
func validateRSAPrivateKey(pri *rsa.PrivateKey) error {
	if pri == nil || pri.N == nil {
		return errors.New("RSA 私钥不能为空")
	}
	if bits := pri.N.BitLen(); bits < minRSABits {
		return errors.Errorf("RSA 私钥位数不能低于 %d，当前位数: %d", minRSABits, bits)
	}
	if err := pri.Validate(); err != nil {
		return errors.Tag(err)
	}
	// 预计算 CRT 参数，加快后续私钥解密和签名操作。
	pri.Precompute()
	return nil
}

// rsaEncryptChunks 按 RSA 最大载荷分块加密数据。
func rsaEncryptChunks(data []byte, keySize, maxPayload int, encrypt func([]byte) ([]byte, error)) ([]byte, error) {
	if maxPayload <= 0 {
		return nil, errors.New("加密失败：最大分块长度小于等于 0")
	}
	if len(data) == 0 {
		return []byte{}, nil
	}
	chunks := (len(data) + maxPayload - 1) / maxPayload
	out := make([]byte, 0, chunks*keySize)
	for start := 0; start < len(data); start += maxPayload {
		end := min(start+maxPayload, len(data))
		encrypted, err := encrypt(data[start:end])
		if err != nil {
			return nil, errors.Tag(err)
		}
		out = append(out, encrypted...)
	}
	return out, nil
}

// rsaDecryptChunks 按 RSA 密钥长度分块解密数据。
func rsaDecryptChunks(ciphertext []byte, keySize int, decrypt func([]byte) ([]byte, error)) ([]byte, error) {
	if keySize <= 0 {
		return nil, errors.New("解密失败：密钥长度异常")
	}
	if len(ciphertext) == 0 {
		return []byte{}, nil
	}
	if len(ciphertext)%keySize != 0 {
		return nil, errors.New("密文长度必须是 RSA 密钥字节长度的整数倍")
	}
	out := make([]byte, 0, len(ciphertext))
	for start := 0; start < len(ciphertext); start += keySize {
		decrypted, err := decrypt(ciphertext[start : start+keySize])
		if err != nil {
			return nil, errors.Tag(err)
		}
		out = append(out, decrypted...)
	}
	return out, nil
}

// hashBytes 使用指定摘要算法计算数据摘要。
func hashBytes(data []byte, hash crypto.Hash) ([]byte, error) {
	if !hash.Available() {
		return nil, errors.New("hash 不可用")
	}
	h := hash.New()
	_, _ = h.Write(data)
	return h.Sum(nil), nil
}

// validateRSASignHash 校验 RSA 签名使用的摘要算法。
func validateRSASignHash(hash crypto.Hash) error {
	switch hash {
	case crypto.MD5, crypto.SHA1:
		return errors.Errorf("不安全的 RSA 签名摘要算法: %s", hash.String())
	}
	if !hash.Available() {
		return errors.New("hash 不可用")
	}
	return nil
}

// validateRSAOAEPHash 校验 OAEP 摘要实例是否满足生产安全下限。
func validateRSAOAEPHash(hash hash.Hash) error {
	if hash == nil {
		return errors.New("hash 不能为空")
	}
	if hash.Size() < 32 {
		return errors.Errorf("OAEP 摘要长度不能低于 32 字节，当前长度: %d", hash.Size())
	}
	return nil
}

// validateRSAOAEPHashID 校验 OAEP 摘要算法标识是否可用且安全。
func validateRSAOAEPHashID(hashID crypto.Hash) error {
	switch hashID {
	case crypto.MD5, crypto.SHA1:
		return errors.Errorf("不安全的 RSA OAEP 摘要算法: %s", hashID.String())
	}
	if !hashID.Available() {
		return errors.New("hash 不可用")
	}
	if hashID.Size() < 32 {
		return errors.Errorf("OAEP 摘要长度不能低于 32 字节，当前长度: %d", hashID.Size())
	}
	return nil
}

// marshalPrivateKey 按 PKCS#1 或 PKCS#8 格式序列化私钥。
func marshalPrivateKey(privateKey *rsa.PrivateKey, isPKCS1 bool) ([]byte, error) {
	if isPKCS1 {
		return x509.MarshalPKCS1PrivateKey(privateKey), nil
	}
	return x509.MarshalPKCS8PrivateKey(privateKey)
}

// marshalPublicKey 按 PKCS#1 或 PKIX 格式序列化公钥。
func marshalPublicKey(publicKey *rsa.PublicKey, isPKCS8 bool) ([]byte, error) {
	if isPKCS8 {
		return x509.MarshalPKIXPublicKey(publicKey)
	}
	return x509.MarshalPKCS1PublicKey(publicKey), nil
}

// writePEMFile 以指定权限写入 PEM 文件。
func writePEMFile(name string, block *pem.Block, perm os.FileMode) error {
	return writeFileAtomic(name, perm, func(file *os.File) error {
		if err := pem.Encode(file, block); err != nil {
			return errors.Tag(err)
		}
		return nil
	})
}
