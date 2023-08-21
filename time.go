package utils

import (
	"github.com/Is999/go-utils/errors"
	"strconv"
	"strings"
	"time"
)

// Local 系统运行时区
func Local() *time.Location {
	return time.Now().Location()
}

// CST 东八时区
func CST() *time.Location {
	return time.FixedZone("CST", 8*3600)
}

// UTC UTC时区
func UTC() *time.Location {
	return time.FixedZone("UTC", 0)
}

// patterns 时间格式化layout规则
var patterns = strings.NewReplacer(
	// ms us ns 必须放在最前面
	"ms", "000", // 毫秒数
	"us", "000", // 微秒数
	"ns", "000", // 纳秒数

	// 年
	"Y", "2006", // 4 位数字完整表示的年份
	"y", "06", // 2 位数字表示的年份

	// 月
	"m", "01", // 数字表示的月份，有前导零
	"n", "1", // 数字表示的月份，没有前导零
	"M", "Jan", // 三个字母缩写表示的月份
	"F", "January", // 月份，完整的文本格式，例如 January 或者 March

	// 日
	"d", "02", // 月份中的第几天，有前导零的 2 位数字
	"j", "2", // 月份中的第几天，没有前导零

	"D", "Mon", // 星期几，文本表示，3 个字母
	"l", "Monday", // 星期几，完整的文本格式;L的小写字母

	// 时间
	"g", "3", // 小时，12 小时格式，没有前导零
	"G", "15", // 小时，24 小时格式，没有前导零
	"h", "03", // 小时，12 小时格式，有前导零
	"H", "15", // 小时，24 小时格式，有前导零

	"a", "pm", // 小写的上午和下午值
	"A", "PM", // 大写的上午和下午值

	"i", "04", // 有前导零的分钟数
	"s", "05", // 秒数，有前导零

	// time zone
	"T", "MST", // 时区
	"P", "-07:00", // 时差
	"O", "-0700", // 时差

	// RFC 2822
	"r", time.RFC1123Z,
)

// MonthDay 获取指定月份有多少天
//
//	year 年
//	month 月
func MonthDay(year int, month int) (days int) {
	if month != 2 {
		if month == 4 || month == 6 || month == 9 || month == 11 {
			days = 30
		} else {
			days = 31
		}
	} else {
		if ((year%4) == 0 && (year%100) != 0) || (year%400) == 0 {
			days = 29
		} else {
			days = 28
		}
	}
	return
}

// CheckDate 验证日期：年、月、日
func CheckDate(year, month, day int) bool {
	if month < 1 || month > 12 || day < 1 || day > 31 || year < 1 || year > 32767 {
		return false
	}
	switch month {
	case 4, 6, 9, 11:
		if day > 30 {
			return false
		}
	case 2:
		// 闰年
		if ((year%4) == 0 && (year%100) != 0) || (year%400) == 0 {
			if day > 29 {
				return false
			}
		} else if day > 28 {
			return false
		}
	}
	return true
}

// DateInfo 获取日期信息
//
//	addTimes 增加时间（Y年，M月，D日，H时，I分，S秒，L毫秒，C微妙，N纳秒)，多个参数英文逗号分割
//	unix Unix 时间sec秒和nsec纳秒
//
//	示例：DateInfo("-1D, 1M, -1Y, +4H, 20I, 30S, 76N",1299826006,152714700)
//
//	返回：year int - 年，
//		month int - 月，monthEn string - 英文月，
//		day int - 日，yearDay int - 一年中第几日， weekDay int - 一周中第几日，
//		hour int - 时，hour int - 分，second int - 秒，
//		millisecond int - 毫秒，microsecond int - 微妙，nanosecond int - 纳秒，
//		unix int64 - 时间戳-秒，unixNano int64 - 时间戳-纳秒，
//		weekDay int - 星期几，weekDayEn string - 星期几英文， yearWeek int - 一年中第几周，
//		date string - 格式化日期，dateNs string - 格式化日期（纳秒)
func DateInfo(addTimes string, unix ...int64) (map[string]interface{}, error) {
	param := make(map[string]interface{})
	t := time.Now()
	if len(unix) == 1 {
		var sec, nsec int64 = unix[0], 0
		if sec >= 1e18 {
			sec = unix[0] / 1e9
			nsec = unix[0] % 1e9
		}
		t = time.Unix(sec, nsec)
	} else if len(unix) > 1 {
		t = time.Unix(unix[0], unix[1])
	}
	if addTimes != "" {
		for _, v := range strings.Split(addTimes, ",") {
			v = strings.TrimSpace(v)
			add, err := strconv.Atoi(strings.TrimSpace(v[:len(v)-1]))
			if err != nil {
				return nil, errors.Wrap(err)
			}
			switch v[len(v)-1] {
			case 'Y':
				t = t.AddDate(add, 0, 0)
			case 'M':
				t = t.AddDate(0, add, 0)
			case 'D':
				t = t.AddDate(0, 0, add)
			case 'H':
				t = t.Add(time.Hour * time.Duration(add))
			case 'I':
				t = t.Add(time.Minute * time.Duration(add))
			case 'S':
				t = t.Add(time.Second * time.Duration(add))
			case 'L':
				t = t.Add(time.Millisecond * time.Duration(add))
			case 'C':
				t = t.Add(time.Microsecond * time.Duration(add))
			case 'N':
				t = t.Add(time.Nanosecond * time.Duration(add))
			default:
				return nil, errors.New("addTimes 参数错误！")
			}
		}
	}

	// 获取年
	param["year"] = t.Year()
	// 获取当前月(英文)
	param["monthEn"] = t.Month().String()
	// 获取月(1-12)
	param["month"] = int(t.Month())
	// 获取当前日
	param["day"] = t.Day()
	//获取当前小时
	param["hour"] = t.Hour()
	// 获取当前分钟
	param["minute"] = t.Minute()
	// 获取当前秒
	param["second"] = t.Second()
	// 获取当前时间戳-毫秒
	param["millisecond"] = t.Nanosecond() / int(time.Millisecond)
	// 获取当前时间戳-微妙
	param["microsecond"] = t.Nanosecond() / int(time.Microsecond)
	// 获取当前时间戳-纳秒
	param["nanosecond"] = t.Nanosecond()

	// 获取当前时间戳-秒
	param["unix"] = t.Unix()
	// 获取当前时间戳-纳秒
	param["unixNano"] = t.UnixNano()
	// 获取当前周几(英文)
	param["weekDayEn"] = t.Weekday().String()
	// 获取当前周几(0-6)
	param["weekDay"] = int(t.Weekday())
	// 一年中的第几周
	_, param["yearWeek"] = t.ISOWeek()
	// 一年中的第几天
	param["yearDay"] = t.YearDay()
	param["date"] = t.Format(SecondDash)
	param["dateNs"] = t.Format(NanosecondDash)
	return param, nil
}

// TimeFormat 时间格式化
//
//	timeZone 时区
//	layout 格式化，例：
//	 - TimeFormat(timeZone, "2006-01-02 15:04:05", unix)
//	unix Unix 时间sec秒和nsec纳秒
func TimeFormat(timeZone *time.Location, layout string, unix ...int64) string {
	if len(unix) == 0 {
		return time.Now().In(timeZone).Format(layout)
	}

	var sec, nsec int64 = unix[0], 0
	if len(unix) > 1 {
		nsec = unix[1]
	}

	if len(unix) == 1 && sec >= 1e18 {
		sec = unix[0] / 1e9
		nsec = unix[0] % 1e9
	}
	return time.Unix(sec, nsec).In(timeZone).Format(layout)
}

// TimeParse 解析时间字符串
//
//	timeZone 时区
//	layout 格式化，例：
//	 - TimeFormat(timeZone, "2006-01-02 15:04:05", timestamp)
//	timestamp 时间字符串
func TimeParse(timeZone *time.Location, layout, timestamp string) (time.Time, error) {
	return time.ParseInLocation(layout, timestamp, timeZone)
}

// Date 使用 patterns 定义的格式对时间格式化
//
//	timeZone 时区
//	layout 格式化，例：
//	 - Date(timeZone, "Y-m-d H:i:s", unix)
//	unix Unix 时间sec秒和nsec纳秒
func Date(timeZone *time.Location, layout string, unix ...int64) string {
	return TimeFormat(timeZone, patterns.Replace(layout), unix...)
}

// Strtotime 解析时间字符串（不确定的时间格式，尽可能解析时间字符串）
//
//	timeZone 时区
//	parse 解析时间字符串包含两个参数：
//	 - layout 格式化：
//	 - timestamp 时间字符串
//	例：
//	 - Strtotime(timeZone, "2006-01-02 15:04:05") // 当前时间
//	 - Strtotime(timeZone, "2006-01-02 15:04:05", timestamp) // timestamp 时间
//	 - Strtotime(timeZone, "Y-m-d H:i:s") // 当前时间
//	 - Strtotime(timeZone, "Y-m-d H:i:s", timestamp) // timestamp 时间
func Strtotime(timeZone *time.Location, parse ...string) (t time.Time, err error) {
	if len(parse) == 1 {
		// 使用频率高地放前面
		layouts := []string{
			MonthDash,  // 2006-01
			DayDash,    // 2006-01-02
			HourDash,   // 2006-01-02 15
			MinuteDash, // 2006-01-02 15:04
			SecondDash, // 2006-01-02 15:04:05

			Year,       // 2006
			MonthSeam,  // 200601
			DaySeam,    // 20060102
			HourSeam,   // 2006010215
			MinuteSeam, // 200601021504
			SecondSeam, // 20060102150405

			MonthSlash,  // 2006/01
			DaySlash,    // 2006/01/02
			HourSlash,   // 2006/01/02 15
			MinuteSlash, // 2006/01/02 15:04
			SecondSlash, // 2006/01/02 15:04:05

			Month,  // 01
			Day,    // 02
			Hour,   // 15
			Minute, // 04
			Second, // 05

			MillisecondDash,  // 2006-01-02 15:04:05.000
			MicrosecondDash,  // 2006-01-02 15:04:05.000000
			NanosecondDash,   // 2006-01-02 15:04:05.000000000
			MillisecondSeam,  // 20060102150405.000
			MicrosecondSeam,  // 20060102150405.000000
			NanosecondSeam,   // 20060102150405.000000000
			MillisecondSlash, // 2006/01/02 15:04:05.000
			MicrosecondSlash, // 2006/01/02 15:04:05.000000
			NanosecondSlash,  // 2006/01/02 15:04:05.000000000

			SecondDash + " -07:00",      // 2006-01-02 15:04:05 -07:00
			SecondSlash + " -07:00",     // 2006/01/02 15:04:05 -07:00
			SecondDash + " -07:00 MST",  // 2006-01-02 15:04:05 -07:00 MST
			SecondSlash + " -07:00 MST", // 2006/01/02 15:04:05 -07:00 MST

			MillisecondDash + " -07:00",  // 2006-01-02 15:04:05.000 -07:00
			MicrosecondDash + " -07:00",  // 2006-01-02 15:04:05.000000 -07:00
			NanosecondDash + " -07:00",   // 2006-01-02 15:04:05.000000000 -07:00
			MillisecondSlash + " -07:00", // 2006/01/02 15:04:05.000 -07:00
			MicrosecondSlash + " -07:00", // 2006/01/02 15:04:05.000000 -07:00
			NanosecondSlash + " -07:00",  // 2006/01/02 15:04:05.000000000 -07:00

			MillisecondDash + " -07:00 MST",  // 2006-01-02 15:04:05.000 -07:00 MST
			MicrosecondDash + " -07:00 MST",  // 2006-01-02 15:04:05.000000 -07:00 MST
			NanosecondDash + " -07:00 MST",   // 2006-01-02 15:04:05.000000000 -07:00 MST
			MillisecondSlash + " -07:00 MST", // 2006/01/02 15:04:05.000 -07:00 MST
			MicrosecondSlash + " -07:00 MST", // 2006/01/02 15:04:05.000000 -07:00 MST
			NanosecondSlash + " -07:00 MST",  // 2006/01/02 15:04:05.000000000 -07:00 MST

			time.ANSIC,       // "Mon Jan _2 15:04:05 2006"
			time.UnixDate,    // "Mon Jan _2 15:04:05 MST 2006"
			time.RubyDate,    // "Mon Jan 02 15:04:05 -0700 2006"
			time.RFC822,      // "02 Jan 06 15:04 MST"
			time.RFC822Z,     // "02 Jan 06 15:04 -0700" // RFC822 with numeric zone
			time.RFC850,      // "Monday, 02-Jan-06 15:04:05 MST"
			time.RFC1123,     // "Mon, 02 Jan 2006 15:04:05 MST"
			time.RFC1123Z,    // "Mon, 02 Jan 2006 15:04:05 -0700" // RFC1123 with numeric zone
			time.RFC3339,     // "2006-01-02T15:04:05Z07:00"
			time.RFC3339Nano, // "2006-01-02T15:04:05.999999999Z07:00"
			time.Kitchen,     // "3:04PM"
			time.Stamp,       // "Jan _2 15:04:05"
			time.StampMilli,  // "Jan _2 15:04:05.000"
			time.StampMicro,  // "Jan _2 15:04:05.000000"
			time.StampNano,   // "Jan _2 15:04:05.000000000"
		}
		for _, layout := range layouts {
			t, err = TimeParse(timeZone, layout, parse[0])
			if err == nil {
				return t, nil
			}
		}
		err = errors.Errorf("无法解析的时间格式：" + parse[0])
	} else if len(parse) > 1 {
		// 直接解析
		t, err = TimeParse(timeZone, parse[0], parse[1])
		if err == nil {
			return t, nil
		}

		// 使用patterns解析
		t, err = TimeParse(timeZone, patterns.Replace(parse[0]), parse[1])
		if err == nil {
			return t, nil
		}

		err = errors.Wrap(err)
	}
	return time.Now().In(timeZone), err
}

// Before 返回true: t1在t2之前（t1小于t2），返回false: t1大于等于t2。
func Before(layout string, t1, t2 string) (bool, error) {
	tt1, err := time.Parse(layout, t1)
	if err != nil {
		return false, errors.Errorf("t1[%v]时间解析错误：%v", t1, err.Error())
	}
	tt2, err := time.Parse(layout, t2)
	if err != nil {
		return false, errors.Errorf("t2[%v]时间解析错误：%v", t2, err.Error())
	}
	return tt1.Before(tt2), nil
}

// After 返回true: t1在t2之后（t1大于t2），返回false: t1小于等于t2。
func After(layout string, t1, t2 string) (bool, error) {
	tt1, err := time.Parse(layout, t1)
	if err != nil {
		return false, errors.Errorf("t1[%v]时间解析错误：%v", t1, err.Error())
	}
	tt2, err := time.Parse(layout, t2)
	if err != nil {
		return false, errors.Errorf("t2[%v]时间解析错误：%v", t2, err.Error())
	}
	return tt1.After(tt2), nil
}

// Equal 判断t1是否与t2相等
func Equal(layout string, t1, t2 string) (bool, error) {
	tt1, err := time.Parse(layout, t1)
	if err != nil {
		return false, errors.Errorf("t1[%v]时间解析错误：%v", t1, err.Error())
	}
	tt2, err := time.Parse(layout, t2)
	if err != nil {
		return false, errors.Errorf("t2[%v]时间解析错误：%v", t2, err.Error())
	}
	return tt1.Equal(tt2), nil
}

// Sub t1与t2的时间差，t1>t2 结果大于0，否则结果小于等于0
func Sub(layout string, t1, t2 string) (time.Duration, error) {
	tt1, err := time.Parse(layout, t1)
	if err != nil {
		return 0, errors.Errorf("t1[%v]时间解析错误：%v", t1, err.Error())
	}
	tt2, err := time.Parse(layout, t2)
	if err != nil {
		return 0, errors.Errorf("t2[%v]时间解析错误：%v", t2, err.Error())
	}
	return tt1.Sub(tt2), nil
}
