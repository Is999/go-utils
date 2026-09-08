package utils

const (
	// ECB 表示电码本模式，无须设置初始化向量 IV。
	ECB CipherMode = iota
	// CBC 表示密码分组链接模式，明文长度不是分组长度整数倍时需要填充。
	CBC
	// CTR 表示计数器流模式，可使用 NoPad/NoUnpad。
	CTR
	// CFB 表示密码反馈流模式，可使用 NoPad/NoUnpad。
	CFB
	// OFB 表示输出反馈流模式，可使用 NoPad/NoUnpad。
	OFB
)

const (
	// Byte 表示 1 字节。
	Byte int64 = 1 << (10 * iota)
	// KB 表示 1024 字节。
	KB
	// MB 表示 1024 KB。
	MB
	// GB 表示 1024 MB。
	GB
	// TB 表示 1024 GB。
	TB
	// PB 表示 1024 TB。
	PB
	// EB 表示 1024 PB，int64 最大可安全表达到该级别。
	EB
)

const (
	// YearTime 是 4 位年份格式。
	YearTime = "2006"
	// MonthTime 是无分隔符的年月 layout：200601。
	MonthTime = YearTime + "01"
	// DayTime 是无分隔符的年月日 layout：20060102。
	DayTime = MonthTime + "02"
	// HourTime 是日期与小时之间带空格的 layout：20060102 15。
	HourTime = DayTime + " 15"
	// MinuteTime 是时分之间无分隔符的 layout：20060102 1504。
	MinuteTime = HourTime + "04"
	// SecondTime 是时分秒之间无分隔符的 layout：20060102 150405。
	SecondTime = MinuteTime + "05"
)
