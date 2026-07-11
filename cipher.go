package utils

import (
	"crypto/cipher"
	"crypto/rand"
	"io"

	"github.com/Is999/go-utils/errors"
)

// Cipher 是 AES/DES/3DES 的通用分组加密器。
//
// 架构说明：
//   - key 和固定 IV 在构造后只读，Encrypt/Decrypt 过程中不再修改对象状态，便于并发复用。
//   - WithRandIV(true) 时，加密会把随机 IV 写入密文头部，解密会从密文头部读取 IV。
//   - ECB、非认证流模式与“未显式设置 IV 时使用 key 派生 IV”都属于兼容旧系统的保留能力，默认禁用。
//   - 公开 API 同时支持传统 padding 语义和 `NoPad/NoUnpad` 零额外拷贝路径。
type Cipher struct {
	key                   []byte       // AES: 16/24/32 字节；DES: 8 字节；3DES: 24 字节。
	iv                    []byte       // 固定 IV；为空时表示未显式配置固定 IV。
	isRandIV              bool         // true 表示每次加密生成随机 IV，并把 IV 放在密文头部。
	allowUnsafeECB        bool         // true 表示允许使用 ECB 模式，仅兼容旧系统时开启。
	allowUnsafeKeyIV      bool         // true 表示允许未设置 IV 时退回到 key 派生 IV，仅兼容旧系统时开启。
	allowUnsafeStreamMode bool         // true 表示允许使用 CTR/CFB/OFB 等非认证流模式，仅兼容旧系统时开启。
	block                 cipher.Block // Go 标准库分组密码实现。
}

// CipherOption 加密器配置项。
type CipherOption func(*cipherOptions)

// cipherOptions 是加密器构造阶段的内部配置。
//
// 字段含义与 Cipher 的安全开关保持一致，仅在构造阶段使用。
type cipherOptions struct {
	randIV                bool    // 是否启用随机 IV。
	iv                    *string // 固定 IV 配置。
	allowUnsafeECB        bool    // 是否允许使用 ECB。
	allowUnsafeKeyIV      bool    // 是否允许使用 key 派生 IV。
	allowUnsafeStreamMode bool    // 是否允许使用非认证流模式。
}

// WithRandIV 设置是否随机生成 IV。
//
// 如果同时设置 WithIV，则固定 IV 优先，随机 IV 自动关闭。
func WithRandIV(isRand bool) CipherOption {
	return func(o *cipherOptions) {
		o.randIV = isRand
	}
}

// WithIV 设置固定 IV。
//
// IV 长度必须等于算法分组长度：AES 为 16 字节，DES/3DES 为 8 字节。
func WithIV(iv string) CipherOption {
	return func(o *cipherOptions) {
		o.iv = &iv
	}
}

// WithAllowUnsafeECB 设置是否允许使用 ECB 模式。
//
// 安全说明：
//   - 默认不允许，避免在生产环境中误用会泄露明文模式特征的 ECB。
//   - 仅在兼容旧系统密文协议时才建议显式开启。
func WithAllowUnsafeECB(allow bool) CipherOption {
	return func(o *cipherOptions) {
		o.allowUnsafeECB = allow
	}
}

// WithAllowUnsafeKeyIV 设置是否允许在未显式配置 IV 时退回到 key 派生 IV。
//
// 安全说明：
//   - 默认不允许，避免把固定且与密钥相关的 IV 当作生产默认值。
//   - 仅在兼容历史密文或旧系统协议时才建议显式开启。
func WithAllowUnsafeKeyIV(allow bool) CipherOption {
	return func(o *cipherOptions) {
		o.allowUnsafeKeyIV = allow
	}
}

// WithAllowUnsafeStreamMode 设置是否允许使用 CTR/CFB/OFB 等非认证流模式。
//
// 安全说明：
//   - 默认不允许，避免在生产环境中误用已不推荐的非认证模式。
//   - 仅在兼容旧系统密文协议时才建议显式开启。
func WithAllowUnsafeStreamMode(allow bool) CipherOption {
	return func(o *cipherOptions) {
		o.allowUnsafeStreamMode = allow
	}
}

// NewCipher 创建通用分组加密器。
//
// key 为原始密钥字符串；block 通常传 aes.NewCipher、des.NewCipher 或 des.NewTripleDESCipher。
func NewCipher(key string, block CipherBlock, opts ...CipherOption) (*Cipher, error) {
	// 聚合可选配置，保证构造入口统一。
	cfg := cipherOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	// 先初始化密钥和底层分组算法，再处理 IV 配置。
	c := &Cipher{
		isRandIV:              cfg.randIV,
		allowUnsafeECB:        cfg.allowUnsafeECB,
		allowUnsafeKeyIV:      cfg.allowUnsafeKeyIV,
		allowUnsafeStreamMode: cfg.allowUnsafeStreamMode,
	}
	if err := c.setKey(key, block); err != nil {
		return nil, errors.Tag(err)
	}
	if cfg.iv != nil {
		// 显式设置固定 IV 时，固定 IV 优先于随机 IV。
		c.isRandIV = false
		if err := c.setIV(*cfg.iv); err != nil {
			return nil, errors.Tag(err)
		}
	}
	return c, nil
}

// setKey 设置密钥并创建底层分组密码。
func (c *Cipher) setKey(key string, block CipherBlock) error {
	if block == nil {
		return errors.New("CipherBlock 不能为空")
	}
	// 当前工具库统一限制为 DES/AES/3DES 可接受的密钥长度。
	switch len(key) {
	default:
		return errors.Errorf("密钥长度必须是 8、16、24 或 32 字节，当前长度: %d", len(key))
	case 8, 16, 24, 32:
	}

	// 先构造底层 block，成功后再写入实例字段，避免部分状态污染对象。
	k := []byte(key)
	b, err := block(k)
	if err != nil {
		return errors.Tag(err)
	}
	c.key = k
	c.block = b
	return nil
}

// check 校验加密器是否可用。
func (c *Cipher) check() error {
	if c == nil || len(c.key) == 0 || c.block == nil {
		return errors.New("请先设置密钥")
	}
	return nil
}

// setIV 设置固定 IV。
func (c *Cipher) setIV(iv string) error {
	if err := c.check(); err != nil {
		return errors.Tag(err)
	}
	if len(iv) != c.block.BlockSize() {
		return errors.Errorf("IV 长度必须是 %d 字节，当前长度: %d", c.block.BlockSize(), len(iv))
	}
	c.iv = []byte(iv)
	return nil
}

// EncryptECB 使用 ECB 模式加密。
func (c *Cipher) EncryptECB(data []byte, padding Pad) ([]byte, error) {
	if err := c.check(); err != nil {
		return nil, errors.Tag(err)
	}
	if err := c.checkUnsafeECB(); err != nil {
		return nil, errors.Tag(err)
	}
	// ECB 需要保证输入长度是分组大小的整数倍，因此先执行填充。
	paddingData, err := c.pad(data, padding)
	if err != nil {
		return nil, errors.Tag(err)
	}

	// 逐块独立加密，保持 ECB 原始语义。
	encrypted := make([]byte, len(paddingData))
	if err := c.cryptBlockLoop(encrypted, paddingData, c.block.Encrypt); err != nil {
		return nil, errors.Tag(err)
	}
	return encrypted, nil
}

// DecryptECB 使用 ECB 模式解密。
func (c *Cipher) DecryptECB(data []byte, unpad Unpad) ([]byte, error) {
	if err := c.check(); err != nil {
		return nil, errors.Tag(err)
	}
	if err := c.checkUnsafeECB(); err != nil {
		return nil, errors.Tag(err)
	}
	if err := c.validateBlockCiphertext(data, isNoUnpadFunc(unpad)); err != nil {
		return nil, errors.Tag(err)
	}
	if unpad == nil {
		return nil, errors.New("unpad 不能为空")
	}

	decrypted := make([]byte, len(data))
	if err := c.cryptBlockLoop(decrypted, data, c.block.Decrypt); err != nil {
		return nil, errors.Tag(err)
	}
	return unpad(decrypted)
}

// EncryptCBC 使用 CBC 模式加密。
func (c *Cipher) EncryptCBC(data []byte, padding Pad) ([]byte, error) {
	paddingData, out, dst, iv, err := c.prepareBlockEncrypt(data, padding)
	if err != nil {
		return nil, errors.Tag(err)
	}
	if err = c.validateBlockPlaintext(paddingData); err != nil {
		return nil, errors.Tag(err)
	}
	cipher.NewCBCEncrypter(c.block, iv).CryptBlocks(dst, paddingData)
	return out, nil
}

// DecryptCBC 使用 CBC 模式解密。
func (c *Cipher) DecryptCBC(data []byte, unpad Unpad) ([]byte, error) {
	body, iv, err := c.prepareBlockDecrypt(data, isNoUnpadFunc(unpad))
	if err != nil {
		return nil, errors.Tag(err)
	}
	if unpad == nil {
		return nil, errors.New("unpad 不能为空")
	}

	decrypted := make([]byte, len(body))
	cipher.NewCBCDecrypter(c.block, iv).CryptBlocks(decrypted, body)
	return unpad(decrypted)
}

// EncryptCTR 使用 CTR 模式加密。
func (c *Cipher) EncryptCTR(data []byte, padding Pad) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("CTR"); err != nil {
		return nil, errors.Tag(err)
	}
	return c.encryptStream(data, padding, cipher.NewCTR)
}

// DecryptCTR 使用 CTR 模式解密。
func (c *Cipher) DecryptCTR(data []byte, unpad Unpad) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("CTR"); err != nil {
		return nil, errors.Tag(err)
	}
	return c.decryptStream(data, unpad, cipher.NewCTR)
}

// EncryptCFB 使用 CFB 模式加密。
func (c *Cipher) EncryptCFB(data []byte, padding Pad) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("CFB"); err != nil {
		return nil, errors.Tag(err)
	}
	return c.encryptStream(data, padding, newLegacyCFBEncrypter)
}

// DecryptCFB 使用 CFB 模式解密。
func (c *Cipher) DecryptCFB(data []byte, unpad Unpad) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("CFB"); err != nil {
		return nil, errors.Tag(err)
	}
	return c.decryptStream(data, unpad, newLegacyCFBDecrypter)
}

// EncryptOFB 使用 OFB 模式加密。
func (c *Cipher) EncryptOFB(data []byte, padding Pad) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("OFB"); err != nil {
		return nil, errors.Tag(err)
	}
	return c.encryptStream(data, padding, newLegacyOFBStream)
}

// DecryptOFB 使用 OFB 模式解密。
func (c *Cipher) DecryptOFB(data []byte, unpad Unpad) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("OFB"); err != nil {
		return nil, errors.Tag(err)
	}
	return c.decryptStream(data, unpad, newLegacyOFBStream)
}

// newLegacyCFBEncrypter 创建兼容旧协议的 CFB 加密流。
func newLegacyCFBEncrypter(block cipher.Block, iv []byte) cipher.Stream {
	//lint:ignore SA1019 CFB 仅在 WithAllowUnsafeStreamMode 显式开启后用于兼容旧系统。
	return cipher.NewCFBEncrypter(block, iv)
}

// newLegacyCFBDecrypter 创建兼容旧协议的 CFB 解密流。
func newLegacyCFBDecrypter(block cipher.Block, iv []byte) cipher.Stream {
	//lint:ignore SA1019 CFB 仅在 WithAllowUnsafeStreamMode 显式开启后用于兼容旧系统。
	return cipher.NewCFBDecrypter(block, iv)
}

// newLegacyOFBStream 创建兼容旧协议的 OFB 流。
func newLegacyOFBStream(block cipher.Block, iv []byte) cipher.Stream {
	//lint:ignore SA1019 OFB 仅在 WithAllowUnsafeStreamMode 显式开启后用于兼容旧系统。
	return cipher.NewOFB(block, iv)
}

// EncryptBytes 加密字节数据并返回原始密文字节。
//
// 该方法适合业务层自行选择编码方式，可避免 string 与 []byte 的额外转换。
func (c *Cipher) EncryptBytes(data []byte, mode CipherMode, padding Pad) ([]byte, error) {
	switch mode {
	case ECB:
		return c.EncryptECB(data, padding)
	case CBC:
		return c.EncryptCBC(data, padding)
	case CTR:
		return c.EncryptCTR(data, padding)
	case CFB:
		return c.EncryptCFB(data, padding)
	case OFB:
		return c.EncryptOFB(data, padding)
	default:
		return nil, errors.New("错误的加密模式")
	}
}

// DecryptBytes 解密原始密文字节。
func (c *Cipher) DecryptBytes(data []byte, mode CipherMode, unpad Unpad) ([]byte, error) {
	switch mode {
	case ECB:
		return c.DecryptECB(data, unpad)
	case CBC:
		return c.DecryptCBC(data, unpad)
	case CTR:
		return c.DecryptCTR(data, unpad)
	case CFB:
		return c.DecryptCFB(data, unpad)
	case OFB:
		return c.DecryptOFB(data, unpad)
	default:
		return nil, errors.New("错误的解密模式")
	}
}

// EncryptTo 将加密结果追加到 dst 并返回结果切片。
// 高频加密场景可复用调用方缓冲区，减少密文字节切片分配；CTR/CFB/OFB 且 NoPad 时走零填充复制快路径。
func (c *Cipher) EncryptTo(dst, data []byte, mode CipherMode, padding Pad) ([]byte, error) {
	switch mode {
	case CTR:
		if err := c.checkUnsafeStreamMode("CTR"); err != nil {
			return nil, errors.Tag(err)
		}
		return c.encryptStreamTo(dst, data, padding, cipher.NewCTR)
	case CFB:
		if err := c.checkUnsafeStreamMode("CFB"); err != nil {
			return nil, errors.Tag(err)
		}
		return c.encryptStreamTo(dst, data, padding, newLegacyCFBEncrypter)
	case OFB:
		if err := c.checkUnsafeStreamMode("OFB"); err != nil {
			return nil, errors.Tag(err)
		}
		return c.encryptStreamTo(dst, data, padding, newLegacyOFBStream)
	default:
		// 分组模式和自定义 padding 可能改变长度或返回新切片，统一复用现有路径保证兼容性。
		encrypted, err := c.EncryptBytes(data, mode, padding)
		if err != nil {
			return nil, errors.Tag(err)
		}
		return append(dst, encrypted...), nil
	}
}

// DecryptTo 将解密结果追加到 dst 并返回结果切片。
// 高频解密场景可复用调用方缓冲区，减少明文字节切片分配；CTR/CFB/OFB 且 NoUnpad 时走直接写入快路径。
func (c *Cipher) DecryptTo(dst, data []byte, mode CipherMode, unpad Unpad) ([]byte, error) {
	switch mode {
	case CTR:
		if err := c.checkUnsafeStreamMode("CTR"); err != nil {
			return nil, errors.Tag(err)
		}
		return c.decryptStreamTo(dst, data, unpad, cipher.NewCTR)
	case CFB:
		if err := c.checkUnsafeStreamMode("CFB"); err != nil {
			return nil, errors.Tag(err)
		}
		return c.decryptStreamTo(dst, data, unpad, newLegacyCFBDecrypter)
	case OFB:
		if err := c.checkUnsafeStreamMode("OFB"); err != nil {
			return nil, errors.Tag(err)
		}
		return c.decryptStreamTo(dst, data, unpad, newLegacyOFBStream)
	default:
		// 分组模式和自定义 unpad 可能改变长度或返回新切片，统一复用现有路径保证兼容性。
		decrypted, err := c.DecryptBytes(data, mode, unpad)
		if err != nil {
			return nil, errors.Tag(err)
		}
		return append(dst, decrypted...), nil
	}
}

// Encrypt 加密字符串并编码输出。
//
// data 为待加密数据；mode 为加密模式；encode 为编码方法；padding 为填充方法。
func (c *Cipher) Encrypt(data string, mode CipherMode, encode EncodeToString, padding Pad) (string, error) {
	if encode == nil {
		return "", errors.New("encode 不能为空")
	}
	encrypted, err := c.EncryptBytes([]byte(data), mode, padding)
	if err != nil {
		return "", errors.Tag(err)
	}
	return encode(encrypted), nil
}

// Decrypt 解密编码后的密文字符串。
//
// encrypt 为待解密数据；decode 为解码方法；unpad 为去填充方法。
func (c *Cipher) Decrypt(encrypt string, mode CipherMode, decode DecodeString, unpad Unpad) (string, error) {
	if decode == nil {
		return "", errors.New("decode 不能为空")
	}
	ciphertext, err := decode(encrypt)
	if err != nil {
		return "", errors.Tag(err)
	}
	decrypted, err := c.DecryptBytes(ciphertext, mode, unpad)
	if err != nil {
		return "", errors.Tag(err)
	}
	return string(decrypted), nil
}

// pad 对原始数据执行填充。
func (c *Cipher) pad(data []byte, padding Pad) ([]byte, error) {
	if padding == nil {
		return nil, errors.New("padding 不能为空")
	}
	if isNoPadFunc(padding) {
		return NoPad(data, c.block.BlockSize()), nil
	}
	// 填充后的数据必须满足分组大小要求，否则后续块加密一定失败。
	paddingData := padding(data, c.block.BlockSize())
	if len(paddingData) == 0 || len(paddingData)%c.block.BlockSize() != 0 {
		return nil, errors.New("padding 后数据长度必须是分组大小的倍数")
	}
	return paddingData, nil
}

// validateBlockPlaintext 校验块模式明文是否可以直接分组加密。
//
// NoPad 用于 CBC/ECB 时不会补齐长度，因此必须在调用 CryptBlocks 或逐块加密前显式返回错误，
// 避免标准库或切片边界检查触发 panic；CTR/CFB/OFB 等流模式不走该校验。
func (c *Cipher) validateBlockPlaintext(data []byte) error {
	blockSize := c.block.BlockSize()
	if len(data)%blockSize != 0 {
		return errors.Errorf("明文长度必须是分组大小的倍数: length=%d, blockSize=%d", len(data), blockSize)
	}
	return nil
}

// checkUnsafeStreamMode 校验是否允许使用非认证流模式。
func (c *Cipher) checkUnsafeStreamMode(modeName string) error {
	if c == nil {
		return errors.New("请先设置密钥")
	}
	if c.allowUnsafeStreamMode {
		return nil
	}
	return errors.Errorf("%s 模式属于非认证加密模式，默认已禁用；如需兼容旧系统，请显式开启 WithAllowUnsafeStreamMode(true)", modeName)
}

// checkUnsafeECB 校验是否允许使用 ECB 模式。
func (c *Cipher) checkUnsafeECB() error {
	if c.allowUnsafeECB {
		return nil
	}
	return errors.New("ECB 模式会泄露明文模式特征，默认已禁用；如需兼容旧系统，请显式开启 WithAllowUnsafeECB(true)")
}

// prepareBlockEncrypt 为 CBC/CTR/CFB/OFB 等模式准备加密数据。
func (c *Cipher) prepareBlockEncrypt(data []byte, padding Pad) (paddingData, out, dst, iv []byte, err error) {
	if err = c.check(); err != nil {
		return nil, nil, nil, nil, errors.Tag(err)
	}
	paddingData, err = c.pad(data, padding)
	if err != nil {
		return nil, nil, nil, nil, errors.Tag(err)
	}

	blockSize := c.block.BlockSize()
	if c.isRandIV {
		// 随机 IV 模式下，将 IV 放在密文头部，便于解密端直接解析。
		out = make([]byte, blockSize+len(paddingData))
		if _, err = io.ReadFull(rand.Reader, out[:blockSize]); err != nil {
			return nil, nil, nil, nil, errors.Tag(err)
		}
		return paddingData, out, out[blockSize:], out[:blockSize], nil
	}

	iv, err = c.fixedIV()
	if err != nil {
		return nil, nil, nil, nil, errors.Tag(err)
	}
	out = make([]byte, len(paddingData))
	return paddingData, out, out, iv, nil
}

// prepareBlockDecrypt 为 CBC 模式准备解密数据。
func (c *Cipher) prepareBlockDecrypt(data []byte, allowEmpty bool) (body, iv []byte, err error) {
	if err = c.check(); err != nil {
		return nil, nil, errors.Tag(err)
	}
	body, iv, err = c.splitCiphertextIV(data)
	if err != nil {
		return nil, nil, errors.Tag(err)
	}
	if err = c.validateBlockCiphertext(body, allowEmpty); err != nil {
		return nil, nil, errors.Tag(err)
	}
	return body, iv, nil
}

// prepareStreamDecrypt 为 CTR/CFB/OFB 等流模式准备解密数据。
func (c *Cipher) prepareStreamDecrypt(data []byte, allowEmpty bool) (body, iv []byte, err error) {
	if err = c.check(); err != nil {
		return nil, nil, errors.Tag(err)
	}
	body, iv, err = c.splitCiphertextIV(data)
	if err != nil {
		return nil, nil, errors.Tag(err)
	}
	if len(body) == 0 && !allowEmpty {
		return nil, nil, errors.New("密文不能为空")
	}
	return body, iv, nil
}

// encryptStream 使用流模式执行加密。
func (c *Cipher) encryptStream(data []byte, padding Pad, newStream func(cipher.Block, []byte) cipher.Stream) ([]byte, error) {
	paddingData, out, dst, iv, err := c.prepareBlockEncrypt(data, padding)
	if err != nil {
		return nil, errors.Tag(err)
	}
	newStream(c.block, iv).XORKeyStream(dst, paddingData)
	return out, nil
}

// decryptStream 使用流模式执行解密。
func (c *Cipher) decryptStream(data []byte, unpad Unpad, newStream func(cipher.Block, []byte) cipher.Stream) ([]byte, error) {
	body, iv, err := c.prepareStreamDecrypt(data, isNoUnpadFunc(unpad))
	if err != nil {
		return nil, errors.Tag(err)
	}
	if unpad == nil {
		return nil, errors.New("unpad 不能为空")
	}

	decrypted := make([]byte, len(body))
	newStream(c.block, iv).XORKeyStream(decrypted, body)
	if isNoUnpadFunc(unpad) {
		return decrypted, nil
	}
	return unpad(decrypted)
}

// encryptStreamTo 使用流模式将密文追加写入 dst。
func (c *Cipher) encryptStreamTo(dst, data []byte, padding Pad, newStream func(cipher.Block, []byte) cipher.Stream) ([]byte, error) {
	if padding == nil {
		return nil, errors.New("padding 不能为空")
	}
	if !isNoPadFunc(padding) {
		// 自定义 padding 可能返回新数据或改变长度，回退到旧路径以保留调用方定义的边界语义。
		encrypted, err := c.encryptStream(data, padding, newStream)
		if err != nil {
			return nil, errors.Tag(err)
		}
		return append(dst, encrypted...), nil
	}
	if err := c.check(); err != nil {
		return nil, errors.Tag(err)
	}

	blockSize := c.block.BlockSize()
	if c.isRandIV {
		// 随机 IV 协议要求密文头部携带 IV，输出长度比明文多一个分组。
		out, tail := appendCipherOutput(dst, blockSize+len(data))
		iv := tail[:blockSize]
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			return nil, errors.Tag(err)
		}
		newStream(c.block, iv).XORKeyStream(tail[blockSize:], data)
		return out, nil
	}

	iv, err := c.fixedIV()
	if err != nil {
		return nil, errors.Tag(err)
	}
	out, tail := appendCipherOutput(dst, len(data))
	newStream(c.block, iv).XORKeyStream(tail, data)
	return out, nil
}

// decryptStreamTo 使用流模式将明文追加写入 dst。
func (c *Cipher) decryptStreamTo(dst, data []byte, unpad Unpad, newStream func(cipher.Block, []byte) cipher.Stream) ([]byte, error) {
	if unpad == nil {
		return nil, errors.New("unpad 不能为空")
	}
	if !isNoUnpadFunc(unpad) {
		// 自定义去填充可能裁剪或校验明文，回退到旧路径以保留错误与边界行为。
		decrypted, err := c.decryptStream(data, unpad, newStream)
		if err != nil {
			return nil, errors.Tag(err)
		}
		return append(dst, decrypted...), nil
	}

	body, iv, err := c.prepareStreamDecrypt(data, isNoUnpadFunc(unpad))
	if err != nil {
		return nil, errors.Tag(err)
	}
	out, tail := appendCipherOutput(dst, len(body))
	newStream(c.block, iv).XORKeyStream(tail, body)
	return out, nil
}

// appendCipherOutput 为密文或明文结果扩展 dst，并返回本次写入窗口。
// 调用方负责保证 dst 可写区域不与输入数据发生不安全重叠。
func appendCipherOutput(dst []byte, size int) ([]byte, []byte) {
	start := len(dst)
	if size <= 0 {
		return dst, dst[start:]
	}
	if start+size <= cap(dst) {
		out := dst[:start+size]
		return out, out[start:]
	}
	out := make([]byte, start+size)
	copy(out, dst)
	return out, out[start:]
}

// fixedIV 获取固定 IV。
// 默认要求业务显式设置固定 IV 或启用随机 IV；只有开启兼容开关时才允许退回到 key 派生 IV。
func (c *Cipher) fixedIV() ([]byte, error) {
	blockSize := c.block.BlockSize()
	if len(c.iv) > 0 {
		if len(c.iv) != blockSize {
			return nil, errors.Errorf("IV 长度必须是 %d 字节，当前长度: %d", blockSize, len(c.iv))
		}
		return c.iv, nil
	}
	if !c.allowUnsafeKeyIV {
		return nil, errors.New("当前模式必须显式设置 WithIV(...) 或开启 WithRandIV(true)；如需兼容旧系统，请显式开启 WithAllowUnsafeKeyIV(true)")
	}
	if len(c.key) < blockSize {
		return nil, errors.New("密钥长度小于分组大小，无法生成默认 IV")
	}
	return c.key[:blockSize], nil
}

// splitCiphertextIV 从密文中拆分真实密文和 IV。
// 随机 IV 模式下，密文前 blockSize 字节为 IV。
func (c *Cipher) splitCiphertextIV(data []byte) ([]byte, []byte, error) {
	blockSize := c.block.BlockSize()
	if c.isRandIV {
		if len(data) < blockSize {
			return nil, nil, errors.New("密文太短，无法读取 IV")
		}
		return data[blockSize:], data[:blockSize], nil
	}
	iv, err := c.fixedIV()
	if err != nil {
		return nil, nil, errors.Tag(err)
	}
	return data, iv, nil
}

// validateBlockCiphertext 校验分组密文是否合法。
func (c *Cipher) validateBlockCiphertext(data []byte, allowEmpty bool) error {
	blockSize := c.block.BlockSize()
	if len(data) == 0 && !allowEmpty {
		return errors.New("密文不能为空")
	}
	if len(data)%blockSize != 0 {
		return errors.New("密文长度必须是分组大小的倍数")
	}
	return nil
}

// cryptBlockLoop 逐个分组执行加解密。
func (c *Cipher) cryptBlockLoop(dst, src []byte, crypt func(dst, src []byte)) error {
	blockSize := c.block.BlockSize()
	if len(src)%blockSize != 0 {
		return errors.Errorf("输入长度必须是分组大小的倍数: length=%d, blockSize=%d", len(src), blockSize)
	}
	if len(dst) < len(src) {
		return errors.Errorf("输出缓冲区长度不足: dst=%d, src=%d", len(dst), len(src))
	}
	for start := 0; start < len(src); start += blockSize {
		end := start + blockSize
		crypt(dst[start:end], src[start:end])
	}
	return nil
}
