package utils

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"strings"
	"time"
)

type RSA struct {
	pubKey *rsa.PublicKey  //公钥
	priKey *rsa.PrivateKey //私钥
}

// NewRSA 实例化RSA并设置公钥和私钥
func NewRSA(pub, pri string, isFilePath ...bool) (*RSA, error) {
	r := &RSA{}
	if err := r.SetPublicKey(pub, isFilePath...); err != nil {
		return r, Wrap(err)
	}
	if err := r.SetPrivateKey(pri, isFilePath...); err != nil {
		return r, Wrap(err)
	}
	return r, nil
}

// NewPubRSA 实例化RSA并设置公钥，用于加密或验证签名
func NewPubRSA(pub string, isFilePath ...bool) (*RSA, error) {
	r := &RSA{}
	if err := r.SetPublicKey(pub, isFilePath...); err != nil {
		return r, Wrap(err)
	}
	return r, nil
}

// NewPriRSA 实例化RSA并设置私钥，用于解密或签名
func NewPriRSA(pri string, isFilePath ...bool) (*RSA, error) {
	r := &RSA{}
	if err := r.SetPrivateKey(pri, isFilePath...); err != nil {
		return r, Wrap(err)
	}
	return r, nil
}

// SetPublicKey 设置公钥
//
//	publicKey 公钥(路径)
//	isFilePath publicKey 传的是否是文件路径
func (r *RSA) SetPublicKey(publicKey string, isFilePath ...bool) error {
	var key []byte
	// 读取文件
	if isFilePath[0] {
		content, err := os.ReadFile(publicKey)
		if err != nil {
			return Wrap(err)
		}
		key = content
	} else {
		key = []byte(publicKey)
	}

	block, _ := pem.Decode(key) // 将密钥解析成公钥实例
	if block == nil || -1 == strings.Index(strings.ToUpper(block.Type), "PUBLIC") {
		return Error("Public key error ")
	}

	var (
		pubInterface interface{}
		err          error
	)

	// PKCS8
	pubInterface, err = x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil && -1 != strings.Index(err.Error(), "ParsePKCS1PublicKey") {
		// PKCS1
		pubInterface, err = x509.ParsePKCS1PublicKey(block.Bytes)
	}

	if err != nil {
		return Wrap(err)
	}

	var ok bool
	if r.pubKey, ok = pubInterface.(*rsa.PublicKey); !ok {
		return Error("PublicKey 类型错误 ")
	}

	return nil
}

// SetPrivateKey 设置私钥
//
//	privateKey 私钥(路径)
//	isFilePath publicKey 传的是否是文件路径
func (r *RSA) SetPrivateKey(privateKey string, isFilePath ...bool) error {
	var key []byte
	// 读取文件
	if isFilePath[0] {
		content, err := os.ReadFile(privateKey)
		if err != nil {
			return Wrap(err)
		}
		key = content
	} else {
		key = []byte(privateKey)
	}

	// 将密钥解析成私钥实例
	block, _ := pem.Decode(key)
	if block == nil || -1 == strings.Index(strings.ToUpper(block.Type), "PRIVATE") {
		return Error("Private key error ")
	}

	var (
		priInterface interface{}
		err          error
	)

	// PKCS1
	priInterface, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil && -1 != strings.Index(err.Error(), "ParsePKCS8PrivateKey") {
		// PKCS8
		priInterface, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	}

	if err != nil {
		return Wrap(err)
	}

	var ok bool
	if r.priKey, ok = priInterface.(*rsa.PrivateKey); !ok {
		return Error("PrivateKey 类型错误 ")
	}

	return nil
}

// IsSetPublicKey 是否正确设置 PublicKey
func (r *RSA) IsSetPublicKey() error {
	if r.pubKey == nil {
		return Error("Public Key is not set ")
	}
	return nil
}

// IsSetPrivateKey 是否正确设置 PrivateKey
func (r *RSA) IsSetPrivateKey() error {
	if r.priKey == nil {
		return Error("Private Key is not set ")
	}
	return nil
}

// Encrypt 加密(公钥)
//
//	data 待加密数据
//	encode 编码方法
func (r *RSA) Encrypt(data string, encode Encode) (string, error) {
	if err := r.IsSetPublicKey(); err != nil {
		return "", Wrap(err)
	}
	encrypt, err := rsa.EncryptPKCS1v15(rand.Reader, r.pubKey, []byte(data)) //RSA算法加密
	if err != nil {
		return "", Wrap(err)
	}
	return encode(encrypt), nil
}

// Decrypt 解密(私钥)
//
//	encrypt 代解密数据
//	decode 解码方法
func (r *RSA) Decrypt(encrypt string, decode Decode) (string, error) {
	if err := r.IsSetPrivateKey(); err != nil {
		return "", Wrap(err)
	}
	ciphertext, err := decode(encrypt)
	if err != nil {
		return "", Wrap(err)
	}
	decrypt, err := rsa.DecryptPKCS1v15(rand.Reader, r.priKey, ciphertext) //RSA算法解密
	return string(decrypt), nil
}

// Sign 签名(私钥)
//
//	data 待签名数据
//	hash 加密哈希函数标识:
//	 - crypto.SHA256 : Sign(data, crypto.SHA256, encode)
//	 - crypto.MD5 : Sign(data, crypto.MD5, encode)
//	encode - 编码方法
func (r *RSA) Sign(data string, hash crypto.Hash, encode Encode) (string, error) {
	if err := r.IsSetPrivateKey(); err != nil {
		return "", Wrap(err)
	}
	h := hash.New()
	h.Write([]byte(data))
	hashed := h.Sum(nil)
	sign, err := rsa.SignPKCS1v15(rand.Reader, r.priKey, hash, hashed)
	if err != nil {
		return "", Wrap(err)
	}
	return encode(sign), nil
}

// Verify 验证签名(公钥)
//
//	data 待验证数据
//	sign 签名串
//	hash 加密哈希函数标识:
//	 - crypto.SHA256 : Verify(data, signature, crypto.SHA256, decode)
//	 - crypto.MD5 : Verify(data, signature, crypto.MD5, decode)
//	decode 解码方法
func (r *RSA) Verify(data, sign string, hash crypto.Hash, decode Decode) error {
	if err := r.IsSetPublicKey(); err != nil {
		return Wrap(err)
	}
	signByte, err := decode(sign)
	if err != nil {
		return Wrap(err)
	}
	h := hash.New()
	h.Write([]byte(data))
	hashed := h.Sum(nil)
	return rsa.VerifyPKCS1v15(r.pubKey, hash, hashed, signByte)
}

// GenerateKeyRSA 生成秘钥(公钥PKCS8格式 私钥PKCS1格式)
//
//	path 秘钥存放地址
//	bits 生成秘钥位大小: 512、1024、2048、4096
//	pkcs 秘钥格式, 默认格式(公钥PKCS8格式 私钥PKCS1格式):
//	 - pkcs[0] isPubPKCS8 公钥是否是PKCS8格式: 默认 true
//	 - pkcs[1] isPriPKCS1 私钥是否是PKCS1格式: 默认 true
//
//	RETURN:
//	- []string 返回两个文件名, 第一个公钥文件名, 第二个私钥文件名
func GenerateKeyRSA(path string, bits int, pkcs ...bool) ([]string, error) {
	isPubPKCS8 := true // 默认 公钥PKCS8格式
	isPriPKCS1 := true // 默认 私钥PKCS1格式
	if len(pkcs) > 0 {
		isPubPKCS8 = pkcs[0]
		if len(pkcs) > 1 {
			isPriPKCS1 = pkcs[1]
		}
	}

	/*
		生成私钥
	*/
	// 使用RSA中的GenerateKey方法生成私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, Wrap(err)
	}

	var fileName = make([]string, 2)
	var privateStream []byte
	// 通过X509标准将得到的RAS私钥序列化为：ASN.1 的DER编码字符串
	if isPriPKCS1 {
		privateStream = x509.MarshalPKCS1PrivateKey(privateKey)
	} else {
		privateStream, err = x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return nil, Wrap(err)
		}
	}

	// 将私钥字符串设置到pem格式块中
	block1 := pem.Block{
		Type:  "private key",
		Bytes: privateStream,
	}

	// 创建私钥文件
	if isPriPKCS1 {
		fileName[1] = path + "private_pkcs1_" + time.Now().Format(SecondSeam) + ".pem"
	} else {
		fileName[1] = path + "private_pkcs8_" + time.Now().Format(SecondSeam) + ".pem"
	}
	fPrivate, err := os.Create(fileName[1])
	if err != nil {
		return nil, Wrap(err)
	}
	defer fPrivate.Close()

	// 通过pem将设置的数据进行编码
	err = pem.Encode(fPrivate, &block1)
	if err != nil {
		return nil, Wrap(err)
	}

	/*
		生成公钥
	*/
	var publicStream []byte
	// 公钥序列化
	if isPubPKCS8 {
		publicStream, err = x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		if err != nil {
			return nil, Wrap(err)
		}
	} else {
		publicStream = x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)
	}

	// 将公钥字符串设置到pem格式块中
	block2 := pem.Block{
		Type:  "public key",
		Bytes: publicStream,
	}

	// 创建公钥文件
	if isPubPKCS8 {
		fileName[0] = path + "public_pkcs8_" + time.Now().Format(`20060102150405`) + ".pem"
	} else {
		fileName[0] = path + "public_pkcs1_" + time.Now().Format(`20060102150405`) + ".pem"
	}
	fPublic, err := os.Create(fileName[0])
	if err != nil {
		return nil, Wrap(err)
	}
	defer fPublic.Close()

	// 通过pem将设置的数据进行编码
	err = pem.Encode(fPublic, &block2)
	if err != nil {
		return nil, Wrap(err)
	}

	return fileName, nil
}
