package utils

// 加密模式
const (
	ECB McryptMode = iota // 0 电码本模式（Electronic Codebook Book，ECB），ECB无须设置初始化向量IV
	CBC                   // 1 密码分组链接模式（Cipher Block Chaining ，CBC），如果明文长度不是分组长度16字节(des 8字节)的整数倍需要进行填充
	CTR                   // 2 计算器模式（Counter，CTR）
	CFB                   // 3 密码反馈模式（Cipher FeedBack，CFB）
	OFB                   // 4 输出反馈模式（Output FeedBack，OFB）
)

// 计算机存储单位：Byte、KB、MB、GB、TB、PB、EB
// int64最大支持EB
const (
	Byte int64 = 1 << (10 * iota) // 1Byte
	KB                            // 1024Byte = 1KB
	MB                            // 1048576Byte = 1024KB = 1MB
	GB                            // 1073741824Byte = 1048576KB = 1024MB = 1GB
	TB                            // 1099511627776Byte = ...
	PB                            // 1125899906842624Byte
	EB                            // 1152921504606846976Byte
)

const (
	YearTime   = "2006"
	MonthTime  = YearTime + "01"
	DayTime    = MonthTime + "02"
	HourTime   = DayTime + "15"
	MinuteTime = HourTime + "04"
	SecondTime = MinuteTime + "05"
)
