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

// 时间格式化格式
const (
	Year   string = "2006" // 年
	Month  string = "01"   // 月
	Day    string = "02"   // 日
	Hour   string = "15"   // 时
	Minute string = "04"   // 分
	Second string = "05"   // 秒

	MonthSeam       = Year + Month            // 200601
	DaySeam         = MonthSeam + Day         // 20060102
	HourSeam        = DaySeam + Hour          // 2006010215
	MinuteSeam      = HourSeam + Minute       // 200601021504
	SecondSeam      = MinuteSeam + Second     // 20060102150405
	MillisecondSeam = SecondSeam + ".000"     // 20060102150405.000
	MicrosecondSeam = MillisecondSeam + "000" // 20060102150405.000000
	NanosecondSeam  = MicrosecondSeam + "000" // 20060102150405.000000000

	MonthDash       = Year + "-" + Month        // 2006-01
	DayDash         = MonthDash + "-" + Day     // 2006-01-02
	HourDash        = DayDash + " " + Hour      // 2006-01-02 15
	MinuteDash      = HourDash + ":" + Minute   // 2006-01-02 15:04
	SecondDash      = MinuteDash + ":" + Second // 2006-01-02 15:04:05
	MillisecondDash = SecondDash + ".000"       // 2006-01-02 15:04:05.000
	MicrosecondDash = MillisecondDash + "000"   // 2006-01-02 15:04:05.000000
	NanosecondDash  = MicrosecondDash + "000"   // 2006-01-02 15:04:05.000000000

	MonthSlash       = Year + "/" + Month         // 2006/01
	DaySlash         = MonthSlash + "/" + Day     // 2006/01/02
	HourSlash        = DaySlash + " " + Hour      // 2006/01/02 15
	MinuteSlash      = HourSlash + ":" + Minute   // 2006/01/02 15:04
	SecondSlash      = MinuteSlash + ":" + Second // 2006/01/02 15:04:05
	MillisecondSlash = SecondSlash + ".000"       // 2006/01/02 15:04:05.000
	MicrosecondSlash = MillisecondSlash + "000"   // 2006/01/02 15:04:05.000000
	NanosecondSlash  = MicrosecondSlash + "000"   // 2006/01/02 15:04:05.000000000
)

// 日志级别
const (
	DEBUG   = Level(iota) // Debug 级别调试
	INFO                  // Info 级别信息
	WARNING               // Warning 级别警告
	ERROR                 // Error 级别错误
	FATAL                 // Fatal 级别致命错误，将调用 os.Exit(1) 退出程序。
	DISABLE               // 禁用日志打印
)
