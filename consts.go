package utils

// 加密模式枚举。
// 用于指定分组密码的工作模式。
const (
	ECB McryptMode = iota // 0 电码本模式（Electronic Codebook Book，ECB），ECB 无须设置初始化向量 IV
	CBC                   // 1 密码分组链接模式（Cipher Block Chaining，CBC），如果明文长度不是分组长度 16 字节（DES 8 字节）的整数倍需要进行填充
	CTR                   // 2 计算器模式（Counter，CTR）
	CFB                   // 3 密码反馈模式（Cipher FeedBack，CFB）
	OFB                   // 4 输出反馈模式（Output FeedBack，OFB）
)

// 计算机存储单位：Byte、KB、MB、GB、TB、PB、EB。
// int64 最大支持到 EB 级别。
const (
	Byte int64 = 1 << (10 * iota) // 1Byte（字节）
	KB                            // 1024Byte = 1KB（千字节）
	MB                            // 1048576Byte = 1024KB = 1MB（兆字节）
	GB                            // 1073741824Byte = 1048576KB = 1024MB = 1GB（吉字节）
	TB                            // 1099511627776Byte = ...（太字节）
	PB                            // 1125899906842624Byte（拍字节）
	EB                            // 1152921504606846976Byte（艾字节）
)

// 时间格式化模板常量。
// 用于 time.Time.Format 方法，格式遵循 Go 的参考时间 "2006-01-02 15:04:05"。
const (
	YearTime   = "2006"            // 年份格式：4 位数字
	MonthTime  = YearTime + "01"   // 年月格式：YYYYMM
	DayTime    = MonthTime + "02"  // 年月日格式：YYYYMMDD
	HourTime   = DayTime + " 15"   // 年月日时格式：YYYYMMDDHH
	MinuteTime = HourTime + "04"   // 年月日时分格式：YYYYMMDDHHMM
	SecondTime = MinuteTime + "05" // 年月日时分秒格式：YYYYMMDDHHMMSS
)
