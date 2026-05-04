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
//   - ECB 与“未显式设置 IV 时使用 key 派生 IV”都属于兼容旧系统的保留能力，默认禁用。
//   - 公开 API 同时支持传统 padding 语义和 `NoPadding/NoUnPadding` 零额外拷贝路径。
type Cipher struct {
	key                   []byte       // AES: 16/24/32 字节；DES: 8 字节；3DES: 24 字节。
	iv                    []byte       // 固定 IV；为空时表示未显式配置固定 IV。
	isRandIV              bool         // true 表示每次加密生成随机 IV，并把 IV 放在密文头部。
	allowUnsafeECB        bool         // true 表示允许使用 ECB 模式，仅兼容旧系统时开启。
	allowUnsafeKeyIV      bool         // true 表示允许未设置 IV 时退回到 key 派生 IV，仅兼容旧系统时开启。
	allowUnsafeStreamMode bool         // true 表示允许使用 CFB/OFB 等非认证流模式，仅兼容旧系统时开启。
	block                 cipher.Block // Go 标准库分组密码实现。
}

// CipherOption 加密器配置项。
type CipherOption func(*cipherOptions)

// cipherOptions 是加密器构造阶段的内部配置。
//
// 字段说明：
//   - randIV：是否启用随机 IV。
//   - iv：固定 IV，优先级高于随机 IV。
//   - allowUnsafeECB：是否允许使用 ECB。
//   - allowUnsafeKeyIV：是否允许未配置 IV 时回退到 key 派生 IV。
//   - allowUnsafeStreamMode：是否允许使用 CFB/OFB 等不推荐模式。
type cipherOptions struct {
	randIV                bool    // 是否启用随机 IV。
	iv                    *string // 固定 IV 配置。
	allowUnsafeECB        bool    // 是否允许使用 ECB。
	allowUnsafeKeyIV      bool    // 是否允许使用 key 派生 IV。
	allowUnsafeStreamMode bool    // 是否允许使用不安全流模式。
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

// WithAllowUnsafeStreamMode 设置是否允许使用 CFB/OFB 等非认证流模式。
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
//
// 参数说明：
//   - key：原始密钥字符串。
//   - block：底层分组算法构造函数。
//
// 返回值：错误信息。
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

// isSetKey 判断密钥和分组密码是否已经设置。
//
// 返回值：true 表示加密器已经完成密钥初始化。
func (c *Cipher) isSetKey() bool {
	return c != nil && len(c.key) > 0 && c.block != nil
}

// check 校验加密器是否可用。
//
// 返回值：错误信息。
func (c *Cipher) check() error {
	if !c.isSetKey() {
		return errors.New("请先设置密钥")
	}
	return nil
}

// setIV 设置固定 IV。
//
// 参数说明：
//   - iv：固定 IV 字符串。
//
// 返回值：错误信息。
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
//
// 参数说明：
//   - data：待加密的原始数据。
//   - padding：填充函数。
//
// 返回值：密文字节，错误信息。
func (c *Cipher) EncryptECB(data []byte, padding Padding) ([]byte, error) {
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
	c.cryptBlockLoop(encrypted, paddingData, c.block.Encrypt)
	return encrypted, nil
}

// DecryptECB 使用 ECB 模式解密。
//
// 参数说明：
//   - data：待解密密文。
//   - unPadding：去填充函数。
//
// 返回值：明文字节，错误信息。
func (c *Cipher) DecryptECB(data []byte, unPadding UnPadding) ([]byte, error) {
	if err := c.check(); err != nil {
		return nil, errors.Tag(err)
	}
	if err := c.checkUnsafeECB(); err != nil {
		return nil, errors.Tag(err)
	}
	if err := c.validateBlockCiphertext(data); err != nil {
		return nil, errors.Tag(err)
	}
	if unPadding == nil {
		return nil, errors.New("unPadding 不能为空")
	}

	decrypted := make([]byte, len(data))
	c.cryptBlockLoop(decrypted, data, c.block.Decrypt)
	return unPadding(decrypted)
}

// EncryptCBC 使用 CBC 模式加密。
//
// 参数说明：
//   - data：待加密原始数据。
//   - padding：填充函数。
//
// 返回值：密文字节，错误信息。
func (c *Cipher) EncryptCBC(data []byte, padding Padding) ([]byte, error) {
	paddingData, out, dst, iv, err := c.prepareBlockEncrypt(data, padding)
	if err != nil {
		return nil, errors.Tag(err)
	}
	cipher.NewCBCEncrypter(c.block, iv).CryptBlocks(dst, paddingData)
	return out, nil
}

// DecryptCBC 使用 CBC 模式解密。
//
// 参数说明：
//   - data：待解密密文。
//   - unPadding：去填充函数。
//
// 返回值：明文字节，错误信息。
func (c *Cipher) DecryptCBC(data []byte, unPadding UnPadding) ([]byte, error) {
	body, iv, err := c.prepareBlockDecrypt(data)
	if err != nil {
		return nil, errors.Tag(err)
	}
	if unPadding == nil {
		return nil, errors.New("unPadding 不能为空")
	}

	decrypted := make([]byte, len(body))
	cipher.NewCBCDecrypter(c.block, iv).CryptBlocks(decrypted, body)
	return unPadding(decrypted)
}

// EncryptCTR 使用 CTR 模式加密。
//
// 参数说明：
//   - data：待加密原始数据。
//   - padding：填充函数。
//
// 返回值：密文字节，错误信息。
func (c *Cipher) EncryptCTR(data []byte, padding Padding) ([]byte, error) {
	return c.encryptStream(data, padding, cipher.NewCTR)
}

// DecryptCTR 使用 CTR 模式解密。
//
// 参数说明：
//   - data：待解密密文。
//   - unPadding：去填充函数。
//
// 返回值：明文字节，错误信息。
func (c *Cipher) DecryptCTR(data []byte, unPadding UnPadding) ([]byte, error) {
	return c.decryptStream(data, unPadding, cipher.NewCTR)
}

// EncryptCFB 使用 CFB 模式加密。
//
// 参数说明：
//   - data：待加密原始数据。
//   - padding：填充函数。
//
// 返回值：密文字节，错误信息。
func (c *Cipher) EncryptCFB(data []byte, padding Padding) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("CFB"); err != nil {
		return nil, errors.Tag(err)
	}
	return c.encryptStream(data, padding, cipher.NewCFBEncrypter)
}

// DecryptCFB 使用 CFB 模式解密。
//
// 参数说明：
//   - data：待解密密文。
//   - unPadding：去填充函数。
//
// 返回值：明文字节，错误信息。
func (c *Cipher) DecryptCFB(data []byte, unPadding UnPadding) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("CFB"); err != nil {
		return nil, errors.Tag(err)
	}
	return c.decryptStream(data, unPadding, cipher.NewCFBDecrypter)
}

// EncryptOFB 使用 OFB 模式加密。
//
// 参数说明：
//   - data：待加密原始数据。
//   - padding：填充函数。
//
// 返回值：密文字节，错误信息。
func (c *Cipher) EncryptOFB(data []byte, padding Padding) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("OFB"); err != nil {
		return nil, errors.Tag(err)
	}
	return c.encryptStream(data, padding, cipher.NewOFB)
}

// DecryptOFB 使用 OFB 模式解密。
//
// 参数说明：
//   - data：待解密密文。
//   - unPadding：去填充函数。
//
// 返回值：明文字节，错误信息。
func (c *Cipher) DecryptOFB(data []byte, unPadding UnPadding) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("OFB"); err != nil {
		return nil, errors.Tag(err)
	}
	return c.decryptStream(data, unPadding, cipher.NewOFB)
}

// EncryptBytes 加密字节数据并返回原始密文字节。
//
// 该方法适合业务层自行选择编码方式，可避免 string 与 []byte 的额外转换。
//
// 参数说明：
//   - data：待加密原始数据。
//   - mode：加密模式。
//   - padding：填充函数。
//
// 返回值：密文字节，错误信息。
func (c *Cipher) EncryptBytes(data []byte, mode McryptMode, padding Padding) ([]byte, error) {
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
//
// 参数说明：
//   - data：待解密密文。
//   - mode：解密模式。
//   - unPadding：去填充函数。
//
// 返回值：明文字节，错误信息。
func (c *Cipher) DecryptBytes(data []byte, mode McryptMode, unPadding UnPadding) ([]byte, error) {
	switch mode {
	case ECB:
		return c.DecryptECB(data, unPadding)
	case CBC:
		return c.DecryptCBC(data, unPadding)
	case CTR:
		return c.DecryptCTR(data, unPadding)
	case CFB:
		return c.DecryptCFB(data, unPadding)
	case OFB:
		return c.DecryptOFB(data, unPadding)
	default:
		return nil, errors.New("错误的解密模式")
	}
}

// Encrypt 加密字符串并编码输出。
//
// data 为待加密数据；mode 为加密模式；encode 为编码方法；padding 为填充方法。
func (c *Cipher) Encrypt(data string, mode McryptMode, encode EncodeToString, padding Padding) (string, error) {
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
// encrypt 为待解密数据；decode 为解码方法；unPadding 为去填充方法。
func (c *Cipher) Decrypt(encrypt string, mode McryptMode, decode DecodeString, unPadding UnPadding) (string, error) {
	if decode == nil {
		return "", errors.New("decode 不能为空")
	}
	ciphertext, err := decode(encrypt)
	if err != nil {
		return "", errors.Tag(err)
	}
	decrypted, err := c.DecryptBytes(ciphertext, mode, unPadding)
	if err != nil {
		return "", errors.Tag(err)
	}
	return string(decrypted), nil
}

// pad 对原始数据执行填充。
//
// 参数说明：
//   - data：原始数据。
//   - padding：填充函数。
//
// 返回值：填充后的数据，错误信息。
func (c *Cipher) pad(data []byte, padding Padding) ([]byte, error) {
	if padding == nil {
		return nil, errors.New("padding 不能为空")
	}
	if isNoPaddingFunc(padding) {
		return NoPadding(data, c.block.BlockSize()), nil
	}
	// 填充后的数据必须满足分组大小要求，否则后续块加密一定失败。
	paddingData := padding(data, c.block.BlockSize())
	if len(paddingData) == 0 || len(paddingData)%c.block.BlockSize() != 0 {
		return nil, errors.New("padding 后数据长度必须是分组大小的倍数")
	}
	return paddingData, nil
}

// checkUnsafeStreamMode 校验是否允许使用不安全流模式。
//
// 参数说明：
//   - modeName：模式名称，如 CFB、OFB。
//
// 返回值：错误信息。
func (c *Cipher) checkUnsafeStreamMode(modeName string) error {
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
//
// 参数说明：
//   - data：原始数据。
//   - padding：填充函数。
//
// 返回值：
//   - paddingData：填充后的原始数据。
//   - out：最终输出缓冲区。
//   - dst：真正写入密文的位置。
//   - iv：本次加密使用的 IV。
//   - err：错误信息。
func (c *Cipher) prepareBlockEncrypt(data []byte, padding Padding) (paddingData, out, dst, iv []byte, err error) {
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

// prepareBlockDecrypt 为 CBC/CTR/CFB/OFB 等模式准备解密数据。
//
// 参数说明：
//   - data：待解密密文。
//
// 返回值：
//   - body：实际密文主体。
//   - iv：解密时使用的 IV。
//   - err：错误信息。
func (c *Cipher) prepareBlockDecrypt(data []byte) (body, iv []byte, err error) {
	if err = c.check(); err != nil {
		return nil, nil, errors.Tag(err)
	}
	body, iv, err = c.splitCiphertextIV(data)
	if err != nil {
		return nil, nil, errors.Tag(err)
	}
	if err = c.validateBlockCiphertext(body); err != nil {
		return nil, nil, errors.Tag(err)
	}
	return body, iv, nil
}

// prepareStreamDecrypt 为 CTR/CFB/OFB 等流模式准备解密数据。
func (c *Cipher) prepareStreamDecrypt(data []byte) (body, iv []byte, err error) {
	if err = c.check(); err != nil {
		return nil, nil, errors.Tag(err)
	}
	body, iv, err = c.splitCiphertextIV(data)
	if err != nil {
		return nil, nil, errors.Tag(err)
	}
	if len(body) == 0 {
		return nil, nil, errors.New("密文不能为空")
	}
	return body, iv, nil
}

// encryptStream 使用流模式执行加密。
//
// 参数说明：
//   - data：待加密原始数据。
//   - padding：填充函数。
//   - newStream：流模式构造函数。
//
// 返回值：密文字节，错误信息。
func (c *Cipher) encryptStream(data []byte, padding Padding, newStream func(cipher.Block, []byte) cipher.Stream) ([]byte, error) {
	paddingData, out, dst, iv, err := c.prepareBlockEncrypt(data, padding)
	if err != nil {
		return nil, errors.Tag(err)
	}
	newStream(c.block, iv).XORKeyStream(dst, paddingData)
	return out, nil
}

// decryptStream 使用流模式执行解密。
//
// 参数说明：
//   - data：待解密密文。
//   - unPadding：去填充函数。
//   - newStream：流模式构造函数。
//
// 返回值：明文字节，错误信息。
func (c *Cipher) decryptStream(data []byte, unPadding UnPadding, newStream func(cipher.Block, []byte) cipher.Stream) ([]byte, error) {
	body, iv, err := c.prepareStreamDecrypt(data)
	if err != nil {
		return nil, errors.Tag(err)
	}
	if unPadding == nil {
		return nil, errors.New("unPadding 不能为空")
	}

	decrypted := make([]byte, len(body))
	newStream(c.block, iv).XORKeyStream(decrypted, body)
	if isNoUnPaddingFunc(unPadding) {
		return decrypted, nil
	}
	return unPadding(decrypted)
}

// fixedIV 获取固定 IV。
// 默认要求业务显式设置固定 IV 或启用随机 IV；只有开启兼容开关时才允许退回到 key 派生 IV。
//
// 返回值：固定 IV，错误信息。
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
//
// 参数说明：
//   - data：原始密文字节。
//
// 返回值：真实密文、IV、错误信息。
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
//
// 参数说明：
//   - data：待校验密文。
//
// 返回值：错误信息。
func (c *Cipher) validateBlockCiphertext(data []byte) error {
	blockSize := c.block.BlockSize()
	if len(data) == 0 {
		return errors.New("密文不能为空")
	}
	if len(data)%blockSize != 0 {
		return errors.New("密文长度必须是分组大小的倍数")
	}
	return nil
}

// cryptBlockLoop 逐个分组执行加解密。
//
// 参数说明：
//   - dst：目标缓冲区。
//   - src：源缓冲区。
//   - crypt：单个分组的加解密函数。
func (c *Cipher) cryptBlockLoop(dst, src []byte, crypt func(dst, src []byte)) {
	blockSize := c.block.BlockSize()
	for start := 0; start < len(src); start += blockSize {
		end := start + blockSize
		crypt(dst[start:end], src[start:end])
	}
}
