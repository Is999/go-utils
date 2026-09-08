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
// 加解密按密钥容量分块并拼接结果；PKCS#1 v1.5 保留用于旧协议，OAEP 用于新协议。
// SetPublicKey/SetPrivateKey 仅在配置阶段调用；配置后可并发使用，回调和外部摘要实例由调用方同步。
type RSA struct {
	pubKey *rsa.PublicKey  // 加密和验签使用；未配置时为 nil。
	priKey *rsa.PrivateKey // 解密和签名使用；设置时完成 CRT 预计算。
}

// RSAOption 仅在构造时生效，按传入顺序应用。
type RSAOption func(*rsaOptions)

// rsaOptions 在构造阶段确定密钥内容的读取方式。
type rsaOptions struct {
	isFilePath bool // 密钥字符串是否按文件路径读取。
}

const (
	// minRSABits 是密钥生成和解析共同执行的最小位数。
	minRSABits = 2048
)

// WithRSAFilePath 为 true 时按文件路径读取密钥，默认直接解析参数内容。
func WithRSAFilePath(isFilePath bool) RSAOption {
	return func(o *rsaOptions) {
		o.isFilePath = isFilePath
	}
}

// NewRSA 实例化 RSA，并同时设置公钥和私钥。
// 失败也返回实例，先成功设置的密钥会保留；使用前须检查 error。
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

// NewPubRSA 创建用于加密或验签的实例，失败时仍返回未配置公钥的实例。
func NewPubRSA(pub string, opts ...RSAOption) (*RSA, error) {
	cfg := parseRSAOptions(opts...)
	r := &RSA{}
	if err := r.SetPublicKey(pub, cfg.isFilePath); err != nil {
		return r, errors.Tag(err)
	}
	return r, nil
}

// NewPriRSA 创建用于解密或签名的实例，失败时仍返回未配置私钥的实例。
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

// SetPublicKey 在配置阶段设置公钥，解析失败保留原公钥。
//
// publicKey 可以是 PEM 文本、base64 DER 文本、原始 DER 数据，或文件路径。
// 文本允许首尾空白；原始 DER 必须按完整二进制字节传入。
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

// SetPrivateKey 在配置阶段设置私钥，解析失败保留原私钥。
//
// privateKey 可以是 PEM 文本、base64 DER 文本、原始 DER 数据，或文件路径。
// 文本允许首尾空白；原始 DER 必须按完整二进制字节传入。
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

// IsSetPublicKey 在 nil 接收者或尚未设置公钥时返回错误。
func (r *RSA) IsSetPublicKey() error {
	if r == nil || r.pubKey == nil {
		return errors.New("RSA 公钥未设置")
	}
	return nil
}

// IsSetPrivateKey 在 nil 接收者或尚未设置私钥时返回错误。
func (r *RSA) IsSetPrivateKey() error {
	if r == nil || r.priKey == nil {
		return errors.New("RSA 私钥未设置")
	}
	return nil
}

// Encrypt 使用公钥和 PKCS#1 v1.5 填充加密。
//
// 每块最多容纳密钥字节数减 11 的明文，密文拼接后仅调用一次 encode；空输入不生成 RSA 分块。
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
		//lint:ignore SA1019 PKCS#1 v1.5 保留为旧协议入口，新协议使用 OAEP。
		return rsa.EncryptPKCS1v15(rand.Reader, r.pubKey, chunk)
	})
	if err != nil {
		return "", errors.Tag(err)
	}
	return encode(encrypted), nil
}

// Decrypt 解码并按密钥字节数拆块，解密失败不返回部分明文；空密文返回空字符串。
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
		//lint:ignore SA1019 PKCS#1 v1.5 保留为旧协议入口，新协议使用 OAEP。
		return rsa.DecryptPKCS1v15(rand.Reader, r.priKey, chunk)
	})
	if err != nil {
		return "", errors.Tag(err)
	}
	return string(decrypted), nil
}

// Sign 先按 hash 计算整个 data 的摘要，再生成 PKCS#1 v1.5 签名；data 无需预先摘要。
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

// Verify 按 hash 重新计算整个 data 的摘要，验证解码后的 PKCS#1 v1.5 签名。
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
// 每块明文上限为密钥字节数 - 2*hash.Size() - 2，label 固定为空。
// 调用会重置 hash，调用方不得并发共享该实例；EncryptOAEPHash 可在内部创建独立摘要。
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
// hash 须与加密时一致，label 固定为空；调用会重置 hash，不得与其他调用并发共享。
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

// EncryptOAEPHash 为本次 OAEP 加密创建独立摘要，hashID 须已注册且摘要至少为 32 字节。
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

// DecryptOAEPHash 为本次 OAEP 解密创建独立摘要，hashID 须与加密时一致。
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

// SignPSS 按 hash 计算 data 的摘要后签名；opts 可为 nil，非零 opts.Hash 须与 hash 一致。
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

// VerifyPSS 按 hash 计算 data 的摘要并验签；opts 可为 nil，沿用标准库规则忽略 opts.Hash。
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
// bits 至少为 2048；pkcs[0] 选择公钥 PKIX/PKCS1，pkcs[1] 选择私钥 PKCS1/PKCS8，均默认 true。
// 返回路径按公钥、私钥排列，权限分别为 0644、0600；私钥写入失败时已生成的公钥文件会保留。
func GenerateKeyRSA(path string, bits int, pkcs ...bool) ([]string, error) {
	if bits < minRSABits {
		return nil, errors.Errorf("RSA 密钥位数不能低于 %d，当前位数: %d", minRSABits, bits)
	}
	if err := prepareRSAKeyDir(path); err != nil {
		return nil, errors.Tag(err)
	}
	publicPKIX, privatePKCS1 := true, true // 未提供的格式开关保留默认值。
	if len(pkcs) > 0 {
		publicPKIX = pkcs[0]
	}
	if len(pkcs) > 1 {
		privatePKCS1 = pkcs[1]
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, errors.Tag(err)
	}

	// 公私钥均成功编码后才开始写文件，避免编码失败时留下半对密钥。
	privateStream, err := marshalPrivateKey(privateKey, privatePKCS1)
	if err != nil {
		return nil, errors.Tag(err)
	}
	publicStream, err := marshalPublicKey(&privateKey.PublicKey, publicPKIX)
	if err != nil {
		return nil, errors.Tag(err)
	}

	// 同一对密钥共用时间后缀，便于对应公私钥文件。
	now := time.Now()
	suffix := now.Format(SecondTime) + "_" + strconv.FormatInt(now.UnixNano(), 36) + ".pem"
	// PKIX 公钥沿用 public_pkcs8_ 文件名前缀。
	fileName := []string{
		filepath.Join(path, Ternary(publicPKIX, "public_pkcs8_", "public_pkcs1_")+suffix),
		filepath.Join(path, Ternary(privatePKCS1, "private_pkcs1_", "private_pkcs8_")+suffix),
	}
	publicType := Ternary(publicPKIX, "PUBLIC KEY", "RSA PUBLIC KEY")
	if err = writePEMFile(fileName[0], &pem.Block{Type: publicType, Bytes: publicStream}, 0o644); err != nil {
		return nil, errors.Tag(err)
	}

	// 保持旧版本 PEM 头兼容：PKCS8 私钥也使用 RSA PRIVATE KEY 头，解析时按 DER 自动识别。
	if err = writePEMFile(fileName[1], &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateStream}, 0o600); err != nil {
		return nil, errors.Tag(err)
	}
	return fileName, nil
}

// prepareRSAKeyDir 在生成密钥前准备目录，已有路径须通过符号链接检查。
func prepareRSAKeyDir(path string) error {
	// 空白路径不预建目录，后续写入仍使用调用方传入的原路径。
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := assertNoSymlinkPath(filepath.Dir(path), path); err != nil {
		return errors.Tag(err)
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.Errorf("GenerateKeyRSA() 不允许密钥目录为符号链接: path=%s", path)
		}
	} else if !os.IsNotExist(err) {
		return errors.Tag(err)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return errors.Tag(err)
	}
	if err := assertNoSymlinkPath(filepath.Dir(path), path); err != nil {
		return errors.Tag(err)
	}
	return nil
}

// RemovePEMHeaders 去掉标记行及各行首尾空白，再拼接正文；行内空白和大小写保持原样。
func RemovePEMHeaders(pemText string) string {
	var b strings.Builder
	b.Grow(len(pemText))
	for line := range strings.SplitSeq(pemText, "\n") {
		line = strings.TrimSpace(line)
		// 只有标记行需要转换大小写，base64 正文按原样拼接。
		if strings.HasPrefix(line, "-----") {
			upper := strings.ToUpper(line)
			if strings.HasPrefix(upper, "-----BEGIN ") || strings.HasPrefix(upper, "-----END ") {
				continue
			}
		}
		b.WriteString(line)
	}
	return strings.TrimSpace(b.String())
}

// AddPEMHeaders 将正文按每行 64 字节折行并添加标记，keyType 支持 public/private，忽略大小写。
// 只重排文本，不解析或验证密钥内容。
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

// readKeyData 按配置读取文件或参数内容，不裁剪二进制密钥字节。
func readKeyData(key string, isFilePath bool) ([]byte, error) {
	if isFilePath {
		return os.ReadFile(key)
	}
	return []byte(key), nil
}

// decodeKeyDER 将 PEM、base64 DER 或原始 DER 密钥统一转换为 DER 字节。
func decodeKeyDER(key []byte, wantType string) ([]byte, error) {
	// 文本格式允许首尾空白，探测时使用裁剪视图。
	trimmed := bytes.TrimSpace(key)
	if len(trimmed) == 0 {
		return nil, errors.New("密钥不能为空")
	}
	if block, _ := pem.Decode(trimmed); block != nil {
		if wantType != "" && !strings.Contains(strings.ToUpper(block.Type), wantType) {
			return nil, errors.Errorf("%s类型错误", rsaKeyTypeName(wantType))
		}
		return block.Bytes, nil
	}

	body := RemovePEMHeaders(string(trimmed))
	if body == "" {
		return nil, errors.New("密钥内容为空")
	}
	if der, err := base64.StdEncoding.DecodeString(body); err == nil {
		return der, nil
	}
	// 未识别为文本时保留全部 DER 字节，尾部空白也可能属于密钥。
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

// validateRSAPublicKey 检查公钥及模数非空，并执行最小位数限制。
func validateRSAPublicKey(pub *rsa.PublicKey) error {
	if pub == nil || pub.N == nil {
		return errors.New("RSA 公钥不能为空")
	}
	if bits := pub.N.BitLen(); bits < minRSABits {
		return errors.Errorf("RSA 公钥位数不能低于 %d，当前位数: %d", minRSABits, bits)
	}
	return nil
}

// validateRSAPrivateKey 在配置阶段校验私钥结构和最小位数。
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
	// 在配置阶段完成 CRT 预计算，供后续解密和签名复用。
	pri.Precompute()
	return nil
}

// rsaEncryptChunks 按载荷上限分块加密，并按输入顺序拼接密文。
func rsaEncryptChunks(data []byte, keySize, maxPayload int, encrypt func([]byte) ([]byte, error)) ([]byte, error) {
	if maxPayload <= 0 {
		return nil, errors.New("加密失败：最大分块长度小于等于 0")
	}
	// 空输入不调用分块函数，保持非 nil 空结果。
	if len(data) == 0 {
		return []byte{}, nil
	}
	chunks := (len(data) + maxPayload - 1) / maxPayload
	out := make([]byte, 0, chunks*keySize)
	for start := 0; start < len(data); start += maxPayload {
		end := min(start+maxPayload, len(data))
		encrypted, err := encrypt(data[start:end])
		if err != nil {
			// 任一分块失败都丢弃已收集的密文。
			return nil, errors.Tag(err)
		}
		out = append(out, encrypted...)
	}
	return out, nil
}

// rsaDecryptChunks 按密钥字节数拆块解密，并按输入顺序拼接明文。
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
			// 任一分块失败都丢弃已收集的明文。
			return nil, errors.Tag(err)
		}
		out = append(out, decrypted...)
	}
	return out, nil
}

// hashBytes 为每次计算创建独立摘要实例，算法须已注册。
func hashBytes(data []byte, hash crypto.Hash) ([]byte, error) {
	if !hash.Available() {
		return nil, errors.New("hash 不可用")
	}
	h := hash.New()
	// hash.Hash 约定 Write 不返回错误。
	_, _ = h.Write(data)
	return h.Sum(nil), nil
}

// validateRSASignHash 拒绝 MD5/SHA1，并要求摘要实现已注册。
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

// validateRSAOAEPHash 要求非 nil 摘要实例，输出长度至少为 32 字节。
func validateRSAOAEPHash(hash hash.Hash) error {
	if hash == nil {
		return errors.New("hash 不能为空")
	}
	if hash.Size() < 32 {
		return errors.Errorf("OAEP 摘要长度不能低于 32 字节，当前长度: %d", hash.Size())
	}
	return nil
}

// validateRSAOAEPHashID 拒绝 MD5/SHA1，要求已注册且输出长度至少为 32 字节。
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
func marshalPublicKey(publicKey *rsa.PublicKey, usePKIX bool) ([]byte, error) {
	if usePKIX {
		return x509.MarshalPKIXPublicKey(publicKey)
	}
	return x509.MarshalPKCS1PublicKey(publicKey), nil
}

// writePEMFile 写入完整 PEM 后按指定权限替换目标，不负责密钥对之间的回滚。
func writePEMFile(name string, block *pem.Block, perm os.FileMode) error {
	return writeFileAtomic(name, perm, func(file *os.File) error {
		if err := pem.Encode(file, block); err != nil {
			return errors.Tag(err)
		}
		return nil
	})
}
