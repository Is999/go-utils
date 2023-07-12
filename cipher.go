package utils

import (
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
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

// NewCipher Cipher
//
//	key 秘钥
//	block 密码: (AES | DES).NewCipher
//	isRandIV 随机生成IV: true 随机生成的IV会在加密后的密文开头
func NewCipher(key string, block CipherBlock, isRandIV bool) (*Cipher, error) {
	a := &Cipher{
		isRandIV: isRandIV,
	}
	if err := a.setKey(key, block); err != nil {
		return nil, err
	}
	return a, nil
}

// setKey 设置秘钥
func (a *Cipher) setKey(key string, block CipherBlock) (err error) {
	switch len(key) {
	default:
		return fmt.Errorf("AES秘钥的长度只能是16、24或32字节，DES秘钥的长度只能是8字节，3DES秘钥的长度只能是24字节。 当前预设置的秘钥[%s]长度: %d", key, len(key))
	case 8, 16, 24, 32:
	}

	k := []byte(key)
	//创建实例
	a.block, err = block(k)
	if err != nil {
		return err
	}
	a.key = k
	return
}

// IsSetKey 是否设置了key
func (a *Cipher) IsSetKey() bool {
	if len(a.key) == 0 || a.block == nil {
		return false
	}
	return true
}

// Check 校验key 和 iv
func (a *Cipher) Check() error {
	if !a.IsSetKey() {
		return errors.New("请先设置秘钥")
	}

	// 随机生成IV
	if a.isRandIV {
		return nil
	}

	// IV 校验和设置
	if len(a.iv) != a.block.BlockSize() {
		if len(a.iv) != 0 {
			return fmt.Errorf("iv的长度只能是%d个字节, 当前设置的iv[%s]长度: %d", a.block.BlockSize(), string(a.iv), len(a.iv))
		}
		a.iv = a.key[:a.block.BlockSize()]
	}
	return nil
}

// SetIv 加密偏移量
func (a *Cipher) SetIv(iv string) error {
	if !a.IsSetKey() {
		return errors.New("请先设置秘钥")
	}

	// 判断IV 长度
	if len(iv) != a.block.BlockSize() {
		return fmt.Errorf("iv的长度只能是%d个字节, 当前预设置的iv[%s]长度: %d", a.block.BlockSize(), iv, len(iv))
	}

	a.iv = []byte(iv)
	return nil
}

// EncryptECB 加密
func (a *Cipher) EncryptECB(data []byte, padding Padding) ([]byte, error) {
	// 校验设置
	if err := a.Check(); err != nil {
		return nil, err
	}

	// 填充
	paddingData := padding(data, a.block.BlockSize())

	// 初始化加密数据接收切片
	encrypt := make([]byte, len(paddingData))

	// 获取密钥长度
	blockSize := a.block.BlockSize()

	// CEB是把整个明文分成若干段相同的小段，然后对每一小段进行加密
	for bs, be := 0, blockSize; bs < len(paddingData); bs, be = bs+blockSize, be+blockSize {
		//执行加密
		a.block.Encrypt(encrypt[bs:be], paddingData[bs:be])
	}

	return encrypt, nil
}

// DecryptECB 解密
func (a *Cipher) DecryptECB(data []byte, unPadding UnPadding) ([]byte, error) {
	// 校验设置
	if err := a.Check(); err != nil {
		return nil, err
	}

	// 初始化解密数据接收切片
	decrypt := make([]byte, len(data))

	// 获取密钥长度
	blockSize := a.block.BlockSize()
	// 执行分段解密
	for bs, be := 0, blockSize; bs < len(data); bs, be = bs+blockSize, be+blockSize {
		// 执行解密
		a.block.Decrypt(decrypt[bs:be], data[bs:be])
	}

	// 去除填充
	return unPadding(decrypt)
}

// EncryptCBC 加密
func (a *Cipher) EncryptCBC(data []byte, padding Padding) ([]byte, error) {
	// 校验设置
	if err := a.Check(); err != nil {
		return nil, err
	}

	// 大小
	blockSize := a.block.BlockSize()

	// 填充
	paddingData := padding(data, blockSize)

	// 初始化加密数据接收切片
	encrypt := make([]byte, Ternary(a.isRandIV, blockSize+len(paddingData), len(paddingData)))

	// 判断是否随机生成IV
	if a.isRandIV {
		// 随机生成IV, 将IV值添加到密文开头
		if _, err := io.ReadFull(rand.Reader, encrypt[:blockSize]); err != nil {
			return nil, err
		}
		a.iv = encrypt[:blockSize]
	}

	// 使用CBC加密模式
	blockMode := cipher.NewCBCEncrypter(a.block, a.iv)

	// 执行加密
	blockMode.CryptBlocks(Ternary(a.isRandIV, encrypt[blockSize:], encrypt), paddingData)

	return encrypt, nil
}

// DecryptCBC 解密
func (a *Cipher) DecryptCBC(data []byte, unPadding UnPadding) ([]byte, error) {
	// 校验设置
	if err := a.Check(); err != nil {
		return nil, err
	}

	// 大小
	blockSize := a.block.BlockSize()

	// 判断是否是随机生成IV
	if a.isRandIV {
		if len(data) < blockSize {
			return nil, errors.New("密文太短")
		}
		a.iv = data[:blockSize]
		data = data[blockSize:]
	}

	// 判断密文长度
	if len(data)%blockSize != 0 {
		return nil, errors.New("密文不是块大小的倍数")
	}

	// 使用CBC
	blockMode := cipher.NewCBCDecrypter(a.block, a.iv)

	// 初始化解密数据接收切片
	decrypt := make([]byte, len(data))

	// 执行解密
	blockMode.CryptBlocks(decrypt, data)

	// 去除填充
	return unPadding(decrypt)
}

// EncryptCTR 加密
func (a *Cipher) EncryptCTR(data []byte, padding Padding) ([]byte, error) {
	// 校验设置
	if err := a.Check(); err != nil {
		return nil, err
	}

	// 大小
	blockSize := a.block.BlockSize()

	// 填充
	paddingData := padding(data, blockSize)

	// 初始化加密数据接收切片
	encrypt := make([]byte, Ternary(a.isRandIV, blockSize+len(paddingData), len(paddingData)))

	// 判断是否随机生成IV
	if a.isRandIV {
		// 随机生成IV, 将IV值添加到密文开头
		if _, err := io.ReadFull(rand.Reader, encrypt[:blockSize]); err != nil {
			return nil, err
		}
		a.iv = encrypt[:blockSize]
	}

	// 使用CTR加密模式
	stream := cipher.NewCTR(a.block, a.iv)

	// 执行加密
	stream.XORKeyStream(Ternary(a.isRandIV, encrypt[blockSize:], encrypt), paddingData)

	return encrypt, nil
}

// DecryptCTR 解密
func (a *Cipher) DecryptCTR(data []byte, unPadding UnPadding) ([]byte, error) {
	// 校验设置
	if err := a.Check(); err != nil {
		return nil, err
	}

	// 大小
	blockSize := a.block.BlockSize()

	// 判断是否是随机生成IV
	if a.isRandIV {
		if len(data) < blockSize {
			return nil, errors.New("密文太短")
		}
		a.iv = data[:blockSize]
		data = data[blockSize:]
	}

	// 判断密文长度
	if len(data)%blockSize != 0 {
		return nil, errors.New("密文不是块大小的倍数")
	}

	// 使用CTR
	stream := cipher.NewCTR(a.block, a.iv)

	// 初始化解密数据接收切片
	decrypt := make([]byte, len(data))

	// 执行解密
	stream.XORKeyStream(decrypt, data)

	// 去除填充
	return unPadding(decrypt)
}

// EncryptCFB 加密
func (a *Cipher) EncryptCFB(data []byte, padding Padding) ([]byte, error) {
	// 校验设置
	if err := a.Check(); err != nil {
		return nil, err
	}

	// 大小
	blockSize := a.block.BlockSize()

	// 填充
	paddingData := padding(data, blockSize)

	// 初始化加密数据接收切片
	encrypt := make([]byte, Ternary(a.isRandIV, blockSize+len(paddingData), len(paddingData)))

	// 判断是否随机生成IV
	if a.isRandIV {
		// 随机生成IV, 将IV值添加到密文开头
		if _, err := io.ReadFull(rand.Reader, encrypt[:blockSize]); err != nil {
			return nil, err
		}
		a.iv = encrypt[:blockSize]
	}

	// 使用CFB加密模式
	stream := cipher.NewCFBEncrypter(a.block, a.iv)

	// 执行加密
	stream.XORKeyStream(Ternary(a.isRandIV, encrypt[blockSize:], encrypt), paddingData)

	return encrypt, nil
}

// DecryptCFB 解密
func (a *Cipher) DecryptCFB(data []byte, unPadding UnPadding) ([]byte, error) {
	// 校验设置
	if err := a.Check(); err != nil {
		return nil, err
	}

	// 大小
	blockSize := a.block.BlockSize()

	// 判断是否是随机生成IV
	if a.isRandIV {
		if len(data) < blockSize {
			return nil, errors.New("密文太短")
		}
		a.iv = data[:blockSize]
		data = data[blockSize:]
	}

	// 判断密文长度
	if len(data)%blockSize != 0 {
		return nil, errors.New("密文不是块大小的倍数")
	}

	// 使用CFB
	stream := cipher.NewCFBDecrypter(a.block, a.iv)

	// 初始化解密数据接收切片
	decrypt := make([]byte, len(data))

	// 执行解密
	stream.XORKeyStream(decrypt, data)

	// 去除填充
	return unPadding(decrypt)
}

// EncryptOFB 加密
func (a *Cipher) EncryptOFB(data []byte, padding Padding) ([]byte, error) {
	// 校验设置
	if err := a.Check(); err != nil {
		return nil, err
	}

	// 大小
	blockSize := a.block.BlockSize()

	// 填充
	paddingData := padding(data, blockSize)

	// 初始化加密数据接收切片
	encrypt := make([]byte, Ternary(a.isRandIV, blockSize+len(paddingData), len(paddingData)))

	// 判断是否随机生成IV
	if a.isRandIV {
		// 随机生成IV, 将IV值添加到密文开头
		if _, err := io.ReadFull(rand.Reader, encrypt[:blockSize]); err != nil {
			return nil, err
		}
		a.iv = encrypt[:blockSize]
	}

	// 使用OFB加密模式
	stream := cipher.NewOFB(a.block, a.iv)

	// 执行加密
	stream.XORKeyStream(Ternary(a.isRandIV, encrypt[blockSize:], encrypt), paddingData)

	return encrypt, nil
}

// DecryptOFB 解密
func (a *Cipher) DecryptOFB(data []byte, unPadding UnPadding) ([]byte, error) {
	// 校验设置
	if err := a.Check(); err != nil {
		return nil, err
	}

	// 大小
	blockSize := a.block.BlockSize()

	// 判断是否是随机生成IV
	if a.isRandIV {
		if len(data) < blockSize {
			return nil, errors.New("密文太短")
		}
		a.iv = data[:blockSize]
		data = data[blockSize:]
	}

	// 判断密文长度
	if len(data)%blockSize != 0 {
		return nil, errors.New("密文不是块大小的倍数")
	}

	// 使用OFB
	stream := cipher.NewOFB(a.block, a.iv)

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
func (a *Cipher) Encrypt(data string, mode McryptMode, encode Encode, padding Padding) (string, error) {
	var (
		encrypt []byte
		err     error
	)

	switch mode {
	case ECB:
		encrypt, err = a.EncryptECB([]byte(data), padding)
	case CBC:
		encrypt, err = a.EncryptCBC([]byte(data), padding)
	case CTR:
		encrypt, err = a.EncryptCTR([]byte(data), padding)
	case CFB:
		encrypt, err = a.EncryptCFB([]byte(data), padding)
	case OFB:
		encrypt, err = a.EncryptOFB([]byte(data), padding)
	default:
		return "", errors.New("错误的加密模式")
	}

	if err != nil {
		return "", err
	}
	return encode(encrypt), nil
}

// Decrypt 加密
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
func (a *Cipher) Decrypt(encrypt string, mode McryptMode, decode Decode, unPadding UnPadding) (string, error) {
	ciphertext, err := decode(encrypt)
	if err != nil {
		return "", err
	}

	var decrypt []byte

	switch mode {
	case ECB:
		decrypt, err = a.DecryptECB(ciphertext, unPadding)
	case CBC:
		decrypt, err = a.DecryptCBC(ciphertext, unPadding)
	case CTR:
		decrypt, err = a.DecryptCTR(ciphertext, unPadding)
	case CFB:
		decrypt, err = a.DecryptCFB(ciphertext, unPadding)
	case OFB:
		decrypt, err = a.DecryptOFB(ciphertext, unPadding)
	default:
		return "", errors.New("错误的解密模式")
	}

	return string(decrypt), err
}
