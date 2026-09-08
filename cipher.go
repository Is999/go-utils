package utils

import (
	"crypto/cipher"
	"crypto/rand"
	"io"

	"github.com/Is999/go-utils/errors"
)

// Cipher 是 AES/DES/3DES 的通用分组加密器。
//
// 构造后密钥和 IV 只读；并发复用时，自定义分组算法和回调也须支持并发调用。
// 随机 IV 写入密文头部；固定 IV 由双方另行配置。ECB、流模式和密钥派生 IV 默认禁用。
type Cipher struct {
	key                   []byte       // AES: 16/24/32 字节；DES: 8 字节；3DES: 24 字节。
	iv                    []byte       // 固定 IV；为空时表示未显式配置固定 IV。
	isRandIV              bool         // true 表示每次加密生成随机 IV，并把 IV 放在密文头部。
	allowUnsafeECB        bool         // true 表示允许使用 ECB 模式，仅兼容旧系统时开启。
	allowUnsafeKeyIV      bool         // true 表示允许未设置 IV 时退回到 key 派生 IV，仅兼容旧系统时开启。
	allowUnsafeStreamMode bool         // true 表示允许使用 CTR/CFB/OFB 等非认证流模式，仅兼容旧系统时开启。
	block                 cipher.Block // 构造函数提供的分组算法，所有操作共用该实例。
}

// CipherOption 仅在构造时生效，按传入顺序应用。
type CipherOption func(*cipherOptions)

// cipherOptions 仅在构造阶段使用，固定 IV 优先于随机 IV。
type cipherOptions struct {
	randIV                bool    // 是否启用随机 IV。
	iv                    *string // 固定 IV 配置。
	allowUnsafeECB        bool    // 是否允许使用 ECB。
	allowUnsafeKeyIV      bool    // 是否允许使用 key 派生 IV。
	allowUnsafeStreamMode bool    // 是否允许使用非认证流模式。
}

// WithRandIV 设置是否随机生成 IV；同时设置 WithIV 时固定 IV 优先。
func WithRandIV(isRand bool) CipherOption {
	return func(o *cipherOptions) {
		o.randIV = isRand
	}
}

// WithIV 按原始字节设置固定 IV，长度须等于分组大小：AES 为 16 字节，DES/3DES 为 8 字节。
func WithIV(iv string) CipherOption {
	return func(o *cipherOptions) {
		o.iv = &iv
	}
}

// WithAllowUnsafeECB 控制默认禁用的 ECB，仅用于兼容会暴露重复明文模式的旧协议。
func WithAllowUnsafeECB(allow bool) CipherOption {
	return func(o *cipherOptions) {
		o.allowUnsafeECB = allow
	}
}

// WithAllowUnsafeKeyIV 允许未配置 IV 时使用密钥前一个分组的字节，默认禁用。
func WithAllowUnsafeKeyIV(allow bool) CipherOption {
	return func(o *cipherOptions) {
		o.allowUnsafeKeyIV = allow
	}
}

// WithAllowUnsafeStreamMode 控制默认禁用的 CTR/CFB/OFB；这些模式不验证密文完整性。
func WithAllowUnsafeStreamMode(allow bool) CipherOption {
	return func(o *cipherOptions) {
		o.allowUnsafeStreamMode = allow
	}
}

// NewCipher 创建通用分组加密器。
//
// key 按原始字节使用；block 通常传 aes.NewCipher、des.NewCipher 或 des.NewTripleDESCipher。
// nil 选项被忽略；自定义 block 负责校验其算法约束。
func NewCipher(key string, block CipherBlock, opts ...CipherOption) (*Cipher, error) {
	cfg := cipherOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	c := &Cipher{
		isRandIV:              cfg.randIV,
		allowUnsafeECB:        cfg.allowUnsafeECB,
		allowUnsafeKeyIV:      cfg.allowUnsafeKeyIV,
		allowUnsafeStreamMode: cfg.allowUnsafeStreamMode,
	}
	// IV 长度取决于分组大小，须先创建底层算法。
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

// setKey 在构造阶段创建分组算法，成功后才保存密钥和算法实例。
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

	k := []byte(key)
	b, err := block(k)
	if err != nil {
		return errors.Tag(err)
	}
	c.key = k
	c.block = b
	return nil
}

// check 拒绝 nil 接收者及尚未创建分组算法的实例。
func (c *Cipher) check() error {
	// setKey 成功后才保存密钥和算法，block 足以标识初始化状态。
	if c == nil || c.block == nil {
		return errors.New("请先设置密钥")
	}
	return nil
}

// setIV 在构造阶段保存固定 IV，长度须与已创建的分组算法一致。
func (c *Cipher) setIV(iv string) error {
	if err := c.check(); err != nil {
		return err
	}
	if len(iv) != c.block.BlockSize() {
		return errors.Errorf("IV 长度必须是 %d 字节，当前长度: %d", c.block.BlockSize(), len(iv))
	}
	c.iv = []byte(iv)
	return nil
}

// EncryptECB 返回独立密文；须显式开启 ECB，填充后长度须为分组大小的整数倍。
func (c *Cipher) EncryptECB(data []byte, padding Pad) ([]byte, error) {
	if err := c.check(); err != nil {
		return nil, err
	}
	if err := c.checkUnsafeECB(); err != nil {
		return nil, err
	}
	paddingData, err := c.pad(data, padding)
	if err != nil {
		return nil, err
	}

	encrypted := make([]byte, len(paddingData))
	if err := c.cryptBlockLoop(encrypted, paddingData, c.block.Encrypt); err != nil {
		return nil, err
	}
	return encrypted, nil
}

// DecryptECB 解密整块密文后调用 unpad，原样返回其结果和错误；仅 NoUnpad 接受空密文。
func (c *Cipher) DecryptECB(data []byte, unpad Unpad) ([]byte, error) {
	if err := c.check(); err != nil {
		return nil, err
	}
	if err := c.checkUnsafeECB(); err != nil {
		return nil, err
	}
	if err := c.validateBlockCiphertext(data, isNoUnpadFunc(unpad)); err != nil {
		return nil, err
	}
	if unpad == nil {
		return nil, errors.New("unpad 不能为空")
	}

	decrypted := make([]byte, len(data))
	if err := c.cryptBlockLoop(decrypted, data, c.block.Decrypt); err != nil {
		return nil, err
	}
	return unpad(decrypted)
}

// EncryptCBC 返回独立密文，填充后长度须为分组大小的整数倍；随机 IV 位于密文头部。
func (c *Cipher) EncryptCBC(data []byte, padding Pad) ([]byte, error) {
	paddingData, out, dst, iv, err := c.prepareBlockEncrypt(data, padding)
	if err != nil {
		return nil, errors.Tag(err)
	}
	if err = c.validateBlockPlaintext(paddingData); err != nil {
		return nil, err
	}
	cipher.NewCBCEncrypter(c.block, iv).CryptBlocks(dst, paddingData)
	return out, nil
}

// DecryptCBC 按当前 IV 配置解密整块密文，再原样返回 unpad 的结果和错误。
func (c *Cipher) DecryptCBC(data []byte, unpad Unpad) ([]byte, error) {
	if err := c.check(); err != nil {
		return nil, err
	}
	body, iv, err := c.splitCiphertextIV(data)
	if err != nil {
		return nil, errors.Tag(err)
	}
	// 去掉随机 IV 后再检查正文；仅 NoUnpad 接受空密文。
	if err := c.validateBlockCiphertext(body, isNoUnpadFunc(unpad)); err != nil {
		return nil, err
	}
	if unpad == nil {
		return nil, errors.New("unpad 不能为空")
	}

	decrypted := make([]byte, len(body))
	cipher.NewCBCDecrypter(c.block, iv).CryptBlocks(decrypted, body)
	return unpad(decrypted)
}

// EncryptCTR 使用已显式开启的 CTR 模式加密；NoPad 允许任意明文长度。
func (c *Cipher) EncryptCTR(data []byte, padding Pad) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("CTR"); err != nil {
		return nil, err
	}
	return c.encryptStream(data, padding, cipher.NewCTR)
}

// DecryptCTR 解密后调用 unpad；内置 NoUnpad 直接返回独立明文缓冲区。
func (c *Cipher) DecryptCTR(data []byte, unpad Unpad) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("CTR"); err != nil {
		return nil, err
	}
	return c.decryptStream(data, unpad, cipher.NewCTR)
}

// EncryptCFB 使用已显式开启的 CFB 模式加密；NoPad 允许任意明文长度。
func (c *Cipher) EncryptCFB(data []byte, padding Pad) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("CFB"); err != nil {
		return nil, err
	}
	//lint:ignore SA1019 CFB 仅在显式开启后用于兼容旧协议。
	return c.encryptStream(data, padding, cipher.NewCFBEncrypter)
}

// DecryptCFB 解密后调用 unpad；内置 NoUnpad 直接返回独立明文缓冲区。
func (c *Cipher) DecryptCFB(data []byte, unpad Unpad) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("CFB"); err != nil {
		return nil, err
	}
	//lint:ignore SA1019 CFB 仅在显式开启后用于兼容旧协议。
	return c.decryptStream(data, unpad, cipher.NewCFBDecrypter)
}

// EncryptOFB 使用已显式开启的 OFB 模式加密；NoPad 允许任意明文长度。
func (c *Cipher) EncryptOFB(data []byte, padding Pad) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("OFB"); err != nil {
		return nil, err
	}
	//lint:ignore SA1019 OFB 仅在显式开启后用于兼容旧协议。
	return c.encryptStream(data, padding, cipher.NewOFB)
}

// DecryptOFB 解密后调用 unpad；内置 NoUnpad 直接返回独立明文缓冲区。
func (c *Cipher) DecryptOFB(data []byte, unpad Unpad) ([]byte, error) {
	if err := c.checkUnsafeStreamMode("OFB"); err != nil {
		return nil, err
	}
	//lint:ignore SA1019 OFB 仅在显式开启后用于兼容旧协议。
	return c.decryptStream(data, unpad, cipher.NewOFB)
}

// EncryptBytes 加密字节数据并返回原始密文字节。
//
// 密文使用独立缓冲区；padding 直接接收 data，自定义回调决定是否修改输入。
// 除内置 NoPad 外，填充结果须非空且按分组大小对齐，包括流模式。
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

// DecryptBytes 解密原始密文字节；unpad 接收独立明文缓冲区，返回值及其别名关系由回调决定。
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
// 流模式配合 NoPad 可直接复用 dst；输入须与密文写入区完全重叠或不重叠，随机 IV 写入区须避开输入。
func (c *Cipher) EncryptTo(dst, data []byte, mode CipherMode, padding Pad) ([]byte, error) {
	switch mode {
	case CTR:
		if err := c.checkUnsafeStreamMode("CTR"); err != nil {
			return nil, err
		}
		return c.encryptStreamTo(dst, data, padding, cipher.NewCTR)
	case CFB:
		if err := c.checkUnsafeStreamMode("CFB"); err != nil {
			return nil, err
		}
		//lint:ignore SA1019 CFB 仅在显式开启后用于兼容旧协议。
		return c.encryptStreamTo(dst, data, padding, cipher.NewCFBEncrypter)
	case OFB:
		if err := c.checkUnsafeStreamMode("OFB"); err != nil {
			return nil, err
		}
		//lint:ignore SA1019 OFB 仅在显式开启后用于兼容旧协议。
		return c.encryptStreamTo(dst, data, padding, cipher.NewOFB)
	default:
		// 分组模式使用独立输出，成功后才向 dst 追加。
		encrypted, err := c.EncryptBytes(data, mode, padding)
		if err != nil {
			return nil, errors.Tag(err)
		}
		return append(dst, encrypted...), nil
	}
}

// DecryptTo 将解密结果追加到 dst 并返回结果切片。
// 流模式配合 NoUnpad 可直接复用 dst，写入区须与去掉 IV 后的密文完全重叠或不重叠。
func (c *Cipher) DecryptTo(dst, data []byte, mode CipherMode, unpad Unpad) ([]byte, error) {
	switch mode {
	case CTR:
		if err := c.checkUnsafeStreamMode("CTR"); err != nil {
			return nil, err
		}
		return c.decryptStreamTo(dst, data, unpad, cipher.NewCTR)
	case CFB:
		if err := c.checkUnsafeStreamMode("CFB"); err != nil {
			return nil, err
		}
		//lint:ignore SA1019 CFB 仅在显式开启后用于兼容旧协议。
		return c.decryptStreamTo(dst, data, unpad, cipher.NewCFBDecrypter)
	case OFB:
		if err := c.checkUnsafeStreamMode("OFB"); err != nil {
			return nil, err
		}
		//lint:ignore SA1019 OFB 仅在显式开启后用于兼容旧协议。
		return c.decryptStreamTo(dst, data, unpad, cipher.NewOFB)
	default:
		// 分组模式先完成解密和去填充，失败时不追加明文。
		decrypted, err := c.DecryptBytes(data, mode, unpad)
		if err != nil {
			return nil, errors.Tag(err)
		}
		return append(dst, decrypted...), nil
	}
}

// Encrypt 加密字符串并编码输出。
//
// encode 和 padding 均须非 nil；密文格式和填充约束与 EncryptBytes 相同。
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
// decode 和 unpad 均须非 nil，解码或去填充失败时返回空字符串及错误。
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

// pad 校验填充结果；自定义回调是否修改输入由调用方约定。
func (c *Cipher) pad(data []byte, padding Pad) ([]byte, error) {
	if padding == nil {
		return nil, errors.New("padding 不能为空")
	}
	if isNoPadFunc(padding) {
		// 加密结果使用独立缓冲区，直接读取明文即可；公开 NoPad 仍保留复制语义。
		return data, nil
	}
	paddingData := padding(data, c.block.BlockSize())
	// 自定义填充在所有模式下都须返回非空、按分组对齐的结果。
	if len(paddingData) == 0 || len(paddingData)%c.block.BlockSize() != 0 {
		return nil, errors.New("padding 后数据长度必须是分组大小的倍数")
	}
	return paddingData, nil
}

// validateBlockPlaintext 仅用于分组模式，流模式允许任意明文长度。
func (c *Cipher) validateBlockPlaintext(data []byte) error {
	blockSize := c.block.BlockSize()
	// NoPad 不补齐长度，须在调用底层 CryptBlocks 前拒绝残块。
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

// prepareBlockEncrypt 为 CBC/CTR/CFB/OFB 分配输出，dst 和随机 IV 均借用 out 的存储。
func (c *Cipher) prepareBlockEncrypt(data []byte, padding Pad) (paddingData, out, dst, iv []byte, err error) {
	if err = c.check(); err != nil {
		return nil, nil, nil, nil, err
	}
	paddingData, err = c.pad(data, padding)
	if err != nil {
		return nil, nil, nil, nil, err
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
		return nil, nil, nil, nil, err
	}
	out = make([]byte, len(paddingData))
	return paddingData, out, out, iv, nil
}

// prepareStreamDecrypt 返回借用输入的正文，IV 取自密文前缀或当前配置。
func (c *Cipher) prepareStreamDecrypt(data []byte, allowEmpty bool) (body, iv []byte, err error) {
	if err = c.check(); err != nil {
		return nil, nil, err
	}
	body, iv, err = c.splitCiphertextIV(data)
	if err != nil {
		return nil, nil, errors.Tag(err)
	}
	// 只有内置 NoUnpad 允许解密空正文。
	if len(body) == 0 && !allowEmpty {
		return nil, nil, errors.New("密文不能为空")
	}
	return body, iv, nil
}

// encryptStream 为每次调用创建独立流状态，密文使用新缓冲区。
func (c *Cipher) encryptStream(data []byte, padding Pad, newStream func(cipher.Block, []byte) cipher.Stream) ([]byte, error) {
	paddingData, out, dst, iv, err := c.prepareBlockEncrypt(data, padding)
	if err != nil {
		return nil, errors.Tag(err)
	}
	newStream(c.block, iv).XORKeyStream(dst, paddingData)
	return out, nil
}

// decryptStream 在独立缓冲区中解密，再将结果交给去填充策略。
func (c *Cipher) decryptStream(data []byte, unpad Unpad, newStream func(cipher.Block, []byte) cipher.Stream) ([]byte, error) {
	// NoUnpad 同时允许空明文并跳过去填充，两个分支复用同一次策略判断。
	skipUnpad := isNoUnpadFunc(unpad)
	body, iv, err := c.prepareStreamDecrypt(data, skipUnpad)
	if err != nil {
		return nil, errors.Tag(err)
	}
	if unpad == nil {
		return nil, errors.New("unpad 不能为空")
	}

	decrypted := make([]byte, len(body))
	newStream(c.block, iv).XORKeyStream(decrypted, body)
	if skipUnpad {
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
		// 自定义填充可能改变数据和长度，完成回调及校验后再追加。
		encrypted, err := c.encryptStream(data, padding, newStream)
		if err != nil {
			return nil, errors.Tag(err)
		}
		return append(dst, encrypted...), nil
	}
	if err := c.check(); err != nil {
		return nil, err
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
		return nil, err
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
		// 自定义去填充可能改变结果或返回错误，成功后才追加明文。
		decrypted, err := c.decryptStream(data, unpad, newStream)
		if err != nil {
			return nil, errors.Tag(err)
		}
		return append(dst, decrypted...), nil
	}

	// 自定义去填充已在上方返回；NoUnpad 路径允许空明文。
	body, iv, err := c.prepareStreamDecrypt(data, true)
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

// fixedIV 返回只读 IV 视图；未配置时，仅兼容开关允许借用密钥前一个分组。
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

// splitCiphertextIV 返回借用的密文和 IV 视图；随机 IV 占输入的前一个分组。
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
		return nil, nil, err
	}
	return data, iv, nil
}

// validateBlockCiphertext 校验整块长度，仅 NoUnpad 路径允许空密文。
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

// cryptBlockLoop 用于 ECB，dst 由调用方按 src 长度分配。
func (c *Cipher) cryptBlockLoop(dst, src []byte, crypt func(dst, src []byte)) error {
	blockSize := c.block.BlockSize()
	// NoPad 可返回未对齐明文，须在首次读写分组前拒绝。
	if len(src)%blockSize != 0 {
		return errors.Errorf("输入长度必须是分组大小的倍数: length=%d, blockSize=%d", len(src), blockSize)
	}
	for start := 0; start < len(src); start += blockSize {
		end := start + blockSize
		crypt(dst[start:end], src[start:end])
	}
	return nil
}
