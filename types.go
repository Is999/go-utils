package utils

import "crypto/cipher"

// Signed 有符合整数
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Unsigned 无符号整数
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Integer 整数
type Integer interface {
	Signed | Unsigned
}

// Float 浮点数
type Float interface {
	~float32 | ~float64
}

// Number 数字
type Number interface {
	Integer | Float
}

// Ordered 数字或字符串
type Ordered interface {
	Number | ~string
}

// Slice 数字或字符串类型slice
//
//	实现了排序接口, 可用sort.Sort(Slice) 排序
type Slice[T Ordered] []T

func (s Slice[T]) Len() int           { return len(s) }
func (s Slice[T]) Less(i, j int) bool { return s[i] < s[j] }
func (s Slice[T]) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// 密码
type (
	// McryptMode 密码模式
	McryptMode int8

	// Encode 加密方法
	//	 - hex.EncodeToString
	//	 - base64.StdEncoding.EncodeToString
	Encode func([]byte) string

	// Decode 解密方法
	//	 - hex.DecodeString
	//	 - base64.StdEncoding.DecodeString
	Decode func(string) ([]byte, error)

	// Padding 填充数据方法
	//	 - Pkcs7Padding
	//	 - ZeroPadding
	Padding func([]byte, int) []byte

	// UnPadding 去除填充数据方法
	//	 - Pkcs7UnPadding
	//	 - ZeroUnPadding
	UnPadding func([]byte) ([]byte, error)

	// CipherBlock 密码(AES | DES)
	//	 - aes.NewCipher
	//	 - des.NewCipher
	//	 - des.NewTripleDESCipher
	CipherBlock func([]byte) (cipher.Block, error)
)

// 文件
type (
	// ReadBlock 处理读取的数据块
	//	 - size 读取的数据块大小
	//	 - block 读取的数据块
	//	返回值 - error 处理错误信息: 返回的 error == DONE 代表正确处理完数据并终止扫描
	ReadBlock func(size int, block []byte) error

	// ReadScan 处理scan扫描的行数据
	//	 - num 行号: 当前扫描到第几行
	//	 - line 行数据: 当前扫描的行数据
	//	 - WrapError 扫描错误信息
	//	返回值 - error 处理错误信息: 返回的 error == DONE 代表正确处理完数据并终止扫描
	ReadScan func(num int, line []byte, err error) error

	// ReadLine 处理scan扫描的行数据
	//	 - num 行号: 当前扫描到第几行
	//	 - line 行数据: 当前扫描的行数据
	//	 - lineDone 当前行(num)数据是否读取完毕: true 当前行(num)数据读取完毕; false 当前行(num)数据未读完
	//	返回值 - error 处理错误信息: 返回的 error == DONE 代表正确处理完数据并终止扫描
	ReadLine func(num int, line []byte, lineDone bool) error
)
