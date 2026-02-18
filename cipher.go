package utils

import (
	"crypto/cipher"
	"crypto/rand"
	"io"

	"github.com/Is999/go-utils/errors"
)

// 1. Cipher(AES | DES) 加密模式：
//	（a）电码本模式（Electronic Codebook Book，ECB），ECB无须设置初始化向量IV；
// 	（b）密码分组链接模式（Cipher Block Chaining ，CBC），如果明文长度不是分组长度16字节的整数倍需要进行填充；
// 	（c）计算器模式（Counter，CTR）；
// 	（d）密码反馈模式（Cipher FeedBack，CFB）；
// 	（e）输出反馈模式（Output FeedBack，OFB）。
// 2. AES|DES是对称分组加密算法。AES每组长度为128bits，即16字节；DES每组长度为64bits，即8字节。
// 3. AES秘钥的长度只能是16、24或32字节，分别对应三种AES，即AES-128, AES-192和AES-256，三者的区别是加密的轮数不同；DES秘钥的长度只能是8字节；3DES秘钥的长度只能是24字节。
// 4. IV长度: AES的IV长度只能是16字节, DES的IV长度只能是8字节。

type Cipher struct {
	key      []byte // AES秘钥的长度只能是16、24或32字节，分别对应三种AES，即AES-128, AES-192和AES-256；DES秘钥的长度只能是8字节；3DES秘钥的长度只能是24字节。
	iv       []byte // 初始化向量IV, AES的IV长度只能是16字节, DES的IV长度只能是8字节。
	isRandIV bool   // isRandIV 为true每次加密随机生成IV, IV值会放在密文头部, 解密时根据秘钥长度获取IV;
	block    cipher.Block
}

// CipherOption 加密器配置项
type CipherOption func(*cipherOptions)

type cipherOptions struct {
	randIV bool
	iv     *string
}

// WithRandIV 设置是否随机生成IV, 如果设置了IV则以IV为准, 不随机生成IV
func WithRandIV(isRand bool) CipherOption {
	return func(o *cipherOptions) {
		o.randIV = isRand
	}
}

// WithIV 设置固定IV, 如果设置了IV则以IV为准, 不随机生成IV
func WithIV(iv string) CipherOption {
	return func(o *cipherOptions) {
		o.iv = &iv
	}
}

// NewCipher Cipher
//
//	key 秘钥
//	block 密码: (AES | DES).NewCipher
func NewCipher(key string, block CipherBlock, opts ...CipherOption) (*Cipher, error) {
	cfg := cipherOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	c := &Cipher{isRandIV: cfg.randIV}
	if err := c.setKey(key, block); err != nil {
		return nil, errors.Wrap(err)
	}
	if cfg.iv != nil {
		c.isRandIV = false
		if err := c.setIV(*cfg.iv); err != nil {
			return nil, errors.Wrap(err)
		}
	}
	return c, nil
}

// setKey 设置秘钥
func (c *Cipher) setKey(key string, block CipherBlock) (err error) {
	switch len(key) {
	default:
		return errors.Errorf("AES秘钥的长度只能是16、24或32字节，DES秘钥的长度只能是8字节，3DES秘钥的长度只能是24字节。 当前预设置的秘钥[%s]长度: %d", key, len(key))
	case 8, 16, 24, 32:
	}

	k := []byte(key)
	//创建实例
	c.block, err = block(k)
	if err != nil {
		return errors.Wrap(err)
	}
	c.key = k
	return
}

// IsSetKey 是否设置了key
func (c *Cipher) isSetKey() bool {
	if len(c.key) == 0 || c.block == nil {
		return false
	}
	return true
}

// check 校验key 和 iv
func (c *Cipher) check() error {
	if !c.isSetKey() {
		return errors.New("请先设置秘钥")
	}

	// 随机生成 IV
	if c.isRandIV {
		return nil
	}

	if len(c.iv) == 0 {
		return c.setIV(string(c.key[:c.block.BlockSize()]))
	}

	return nil
}

// setIV 设置加密偏移量
func (c *Cipher) setIV(iv string) error {
	if !c.isSetKey() {
		return errors.New("请先设置秘钥")
	}

	// 判断IV 长度
	if len(iv) != c.block.BlockSize() {
		return errors.Errorf("iv的长度只能是%d个字节, 当前预设置的iv[%s]长度: %d", c.block.BlockSize(), iv, len(iv))
	}

	c.iv = []byte(iv)
	return nil
}

// EncryptECB 加密
func (c *Cipher) EncryptECB(data []byte, padding Padding) ([]byte, error) {
	// 填充
	paddingData := padding(data, c.block.BlockSize())

	// 初始化加密数据接收切片
	encrypt := make([]byte, len(paddingData))

	// 获取密钥长度
	blockSize := c.block.BlockSize()

	// CEB是把整个明文分成若干段相同的小段，然后对每一小段进行加密
	for bs, be := 0, blockSize; bs < len(paddingData); bs, be = bs+blockSize, be+blockSize {
		//执行加密
		c.block.Encrypt(encrypt[bs:be], paddingData[bs:be])
	}

	return encrypt, nil
}

// DecryptECB 解密
func (c *Cipher) DecryptECB(data []byte, unPadding UnPadding) ([]byte, error) {
	// 初始化解密数据接收切片
	decrypt := make([]byte, len(data))

	// 获取密钥长度
	blockSize := c.block.BlockSize()

	// 执行分段解密
	for bs, be := 0, blockSize; bs < len(data); bs, be = bs+blockSize, be+blockSize {
		// 执行解密
		c.block.Decrypt(decrypt[bs:be], data[bs:be])
	}

	// 去除填充
	return unPadding(decrypt)
}

// EncryptCBC 加密
func (c *Cipher) EncryptCBC(data []byte, padding Padding) ([]byte, error) {
	// 校验设置
	if err := c.check(); err != nil {
		return nil, errors.Wrap(err)
	}

	// 大小
	blockSize := c.block.BlockSize()

	// 填充
	paddingData := padding(data, blockSize)

	// 初始化加密数据接收切片
	encrypt := make([]byte, Ternary(c.isRandIV, blockSize+len(paddingData), len(paddingData)))

	// 判断是否随机生成 IV
	if c.isRandIV {
		// 随机生成IV, 将IV值添加到密文开头
		if _, err := io.ReadFull(rand.Reader, encrypt[:blockSize]); err != nil {
			return nil, errors.Wrap(err)
		}
		c.iv = encrypt[:blockSize]
	}

	// 使用 CBC加密模式
	blockMode := cipher.NewCBCEncrypter(c.block, c.iv)

	// 执行加密
	blockMode.CryptBlocks(Ternary(c.isRandIV, encrypt[blockSize:], encrypt), paddingData)

	return encrypt, nil
}

// DecryptCBC 解密
func (c *Cipher) DecryptCBC(data []byte, unPadding UnPadding) ([]byte, error) {
	// 校验设置
	if err := c.check(); err != nil {
		return nil, errors.Wrap(err)
	}

	// 大小
	blockSize := c.block.BlockSize()

	// 判断是否是随机生成 IV
	if c.isRandIV {
		if len(data) < blockSize {
			return nil, errors.New("密文太短")
		}
		c.iv = data[:blockSize]
		data = data[blockSize:]
	}

	// 判断密文长度
	if len(data)%blockSize != 0 {
		return nil, errors.New("密文不是块大小的倍数")
	}

	// 使用 CBC
	blockMode := cipher.NewCBCDecrypter(c.block, c.iv)

	// 初始化解密数据接收切片
	decrypt := make([]byte, len(data))

	// 执行解密
	blockMode.CryptBlocks(decrypt, data)

	// 去除填充
	return unPadding(decrypt)
}

// EncryptCTR 加密
func (c *Cipher) EncryptCTR(data []byte, padding Padding) ([]byte, error) {
	// 校验设置
	if err := c.check(); err != nil {
		return nil, errors.Wrap(err)
	}

	// 大小
	blockSize := c.block.BlockSize()

	// 填充
	paddingData := padding(data, blockSize)

	// 初始化加密数据接收切片
	encrypt := make([]byte, Ternary(c.isRandIV, blockSize+len(paddingData), len(paddingData)))

	// 判断是否随机生成 IV
	if c.isRandIV {
		// 随机生成IV, 将IV值添加到密文开头
		if _, err := io.ReadFull(rand.Reader, encrypt[:blockSize]); err != nil {
			return nil, errors.Wrap(err)
		}
		c.iv = encrypt[:blockSize]
	}

	// 使用 CTR加密模式
	stream := cipher.NewCTR(c.block, c.iv)

	// 执行加密
	stream.XORKeyStream(Ternary(c.isRandIV, encrypt[blockSize:], encrypt), paddingData)

	return encrypt, nil
}

// DecryptCTR 解密
func (c *Cipher) DecryptCTR(data []byte, unPadding UnPadding) ([]byte, error) {
	// 校验设置
	if err := c.check(); err != nil {
		return nil, errors.Wrap(err)
	}

	// 大小
	blockSize := c.block.BlockSize()

	// 判断是否是随机生成 IV
	if c.isRandIV {
		if len(data) < blockSize {
			return nil, errors.New("密文太短")
		}
		c.iv = data[:blockSize]
		data = data[blockSize:]
	}

	// 判断密文长度
	if len(data)%blockSize != 0 {
		return nil, errors.New("密文不是块大小的倍数")
	}

	// 使用 CTR
	stream := cipher.NewCTR(c.block, c.iv)

	// 初始化解密数据接收切片
	decrypt := make([]byte, len(data))

	// 执行解密
	stream.XORKeyStream(decrypt, data)

	// 去除填充
	return unPadding(decrypt)
}

// EncryptCFB 加密
func (c *Cipher) EncryptCFB(data []byte, padding Padding) ([]byte, error) {
	// 校验设置
	if err := c.check(); err != nil {
		return nil, errors.Wrap(err)
	}

	// 大小
	blockSize := c.block.BlockSize()

	// 填充
	paddingData := padding(data, blockSize)

	// 初始化加密数据接收切片
	encrypt := make([]byte, Ternary(c.isRandIV, blockSize+len(paddingData), len(paddingData)))

	// 判断是否随机生成 IV
	if c.isRandIV {
		// 随机生成IV, 将IV值添加到密文开头
		if _, err := io.ReadFull(rand.Reader, encrypt[:blockSize]); err != nil {
			return nil, errors.Wrap(err)
		}
		c.iv = encrypt[:blockSize]
	}

	// 使用 CFB加密模式
	stream := cipher.NewCFBEncrypter(c.block, c.iv)

	// 执行加密
	stream.XORKeyStream(Ternary(c.isRandIV, encrypt[blockSize:], encrypt), paddingData)

	return encrypt, nil
}

// DecryptCFB 解密
func (c *Cipher) DecryptCFB(data []byte, unPadding UnPadding) ([]byte, error) {
	// 校验设置
	if err := c.check(); err != nil {
		return nil, errors.Wrap(err)
	}

	// 大小
	blockSize := c.block.BlockSize()

	// 判断是否是随机生成 IV
	if c.isRandIV {
		if len(data) < blockSize {
			return nil, errors.New("密文太短")
		}
		c.iv = data[:blockSize]
		data = data[blockSize:]
	}

	// 判断密文长度
	if len(data)%blockSize != 0 {
		return nil, errors.New("密文不是块大小的倍数")
	}

	// 使用 CFB
	stream := cipher.NewCFBDecrypter(c.block, c.iv)

	// 初始化解密数据接收切片
	decrypt := make([]byte, len(data))

	// 执行解密
	stream.XORKeyStream(decrypt, data)

	// 去除填充
	return unPadding(decrypt)
}

// EncryptOFB 加密
func (c *Cipher) EncryptOFB(data []byte, padding Padding) ([]byte, error) {
	// 校验设置
	if err := c.check(); err != nil {
		return nil, errors.Wrap(err)
	}

	// 大小
	blockSize := c.block.BlockSize()

	// 填充
	paddingData := padding(data, blockSize)

	// 初始化加密数据接收切片
	encrypt := make([]byte, Ternary(c.isRandIV, blockSize+len(paddingData), len(paddingData)))

	// 判断是否随机生成 IV
	if c.isRandIV {
		// 随机生成IV, 将IV值添加到密文开头
		if _, err := io.ReadFull(rand.Reader, encrypt[:blockSize]); err != nil {
			return nil, errors.Wrap(err)
		}
		c.iv = encrypt[:blockSize]
	}

	// 使用 OFB加密模式
	stream := cipher.NewOFB(c.block, c.iv)

	// 执行加密
	stream.XORKeyStream(Ternary(c.isRandIV, encrypt[blockSize:], encrypt), paddingData)

	return encrypt, nil
}

// DecryptOFB 解密
func (c *Cipher) DecryptOFB(data []byte, unPadding UnPadding) ([]byte, error) {
	// 校验设置
	if err := c.check(); err != nil {
		return nil, errors.Wrap(err)
	}

	// 大小
	blockSize := c.block.BlockSize()

	// 判断是否是随机生成 IV
	if c.isRandIV {
		if len(data) < blockSize {
			return nil, errors.New("密文太短")
		}
		c.iv = data[:blockSize]
		data = data[blockSize:]
	}

	// 判断密文长度
	if len(data)%blockSize != 0 {
		return nil, errors.New("密文不是块大小的倍数")
	}

	// 使用 OFB
	stream := cipher.NewOFB(c.block, c.iv)

	// 初始化解密数据接收切片
	decrypt := make([]byte, len(data))

	// 执行解密
	stream.XORKeyStream(decrypt, data)

	// 去除填充
	return unPadding(decrypt)
}

// Encrypt 加密
//
//	data 待加密数据
//	mode 加密模式:
//	 - ECB: Encrypt(data, ECB, encode, padding)
//	 - CBC: Encrypt(data, CBC, encode, padding)
//	 - CTR: Encrypt(data, CTR, encode, padding)
//	 - CFB: Encrypt(data, CFB, encode, padding)
//	 - OFB: Encrypt(data, OFB, encode, padding)
//	encode 编码方法
//	padding 填充数据方法
func (c *Cipher) Encrypt(data string, mode McryptMode, encode EncodeToString, padding Padding) (string, error) {
	var (
		encrypt []byte
		err     error
	)

	switch mode {
	case ECB:
		encrypt, err = c.EncryptECB([]byte(data), padding)
	case CBC:
		encrypt, err = c.EncryptCBC([]byte(data), padding)
	case CTR:
		encrypt, err = c.EncryptCTR([]byte(data), padding)
	case CFB:
		encrypt, err = c.EncryptCFB([]byte(data), padding)
	case OFB:
		encrypt, err = c.EncryptOFB([]byte(data), padding)
	default:
		return "", errors.New("错误的加密模式")
	}

	if err != nil {
		return "", errors.Wrap(err)
	}

	return encode(encrypt), nil
}

// Decrypt 解密
//
//	encrypt 待解密数据
//	mode 加密模式:
//	 - ECB: Decrypt(encrypt, ECB, decode, unPadding)
//	 - CBC: Decrypt(encrypt, CBC, decode, unPadding)
//	 - CTR: Decrypt(encrypt, CTR, decode, unPadding)
//	 - CFB: Decrypt(encrypt, CFB, decode, unPadding)
//	 - OFB: Decrypt(encrypt, OFB, decode, unPadding)
//	decode 解码方法
//	unPadding 去除填充数据方法
func (c *Cipher) Decrypt(encrypt string, mode McryptMode, decode DecodeString, unPadding UnPadding) (string, error) {
	ciphertext, err := decode(encrypt)
	if err != nil {
		return "", errors.Wrap(err)
	}

	var decrypt []byte

	switch mode {
	case ECB:
		decrypt, err = c.DecryptECB(ciphertext, unPadding)
	case CBC:
		decrypt, err = c.DecryptCBC(ciphertext, unPadding)
	case CTR:
		decrypt, err = c.DecryptCTR(ciphertext, unPadding)
	case CFB:
		decrypt, err = c.DecryptCFB(ciphertext, unPadding)
	case OFB:
		decrypt, err = c.DecryptOFB(ciphertext, unPadding)
	default:
		return "", errors.New("错误的解密模式")
	}

	return string(decrypt), errors.Wrap(err)
}
