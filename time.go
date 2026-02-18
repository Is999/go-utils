package utils

import (
	"strconv"
	"strings"
	"time"

	"github.com/Is999/go-utils/errors"
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
	"ms", "000", // 毫秒数
	"us", "000", // 微秒数
	"ns", "000", // 纳秒数

	"Y", "2006", // 4位数字完整表示的年份
	"y", "06", // 2 位数字表示的年份

	"m", "01", // 数字表示的月份，有前导零
	"n", "1", // 数字表示的月份，没有前导零
	"M", "Jan", // 三个字母缩写表示的月份
	"F", "January", // 月份，完整的文本格式，例如 January 或者 March

	"d", "02", // 月份中的第几天，有前导零的2位数字01到31
	"j", "2", // 月份中的第几天，没有前导零，1到31

	"D", "Mon", // 星期几，显示3个字母 Mon 到 Sun
	"l", "Monday", // 星期几，完整的文本格式，Sunday 到 Saturday

	// 时间格式
	"g", "3", // 小时，12 小时格式，没有前导零
	"h", "03", // 小时，12 小时格式，有前导零
	"H", "15", // 小时，24 小时格式，有前导零

	"a", "pm", // 小写的上午和下午值
	"A", "PM", // 大写的上午和下午值

	"i", "04", // 分钟数，有前导零
	"s", "05", // 秒数，有前导零

	// 时区
	"e", "MST", // 时区标识
	"P", "-07:00", // 时差
	"O", "-0700", // 时差
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

// AddTime 增加时间
//
//	addTimes 增加时间（Y年，M月，D日，H时，I分，S秒，L毫秒，C微妙，N纳秒)
//
//	示例：DateInfo(time, "-1D", "1M", "-1Y", "+4H", "20I", "30S", "76N")
func AddTime(t time.Time, addTimes ...string) (time.Time, error) {
	for _, v := range addTimes {
		v = strings.TrimSpace(v)
		add, err := strconv.Atoi(strings.TrimSpace(v[:len(v)-1]))
		if err != nil {
			return t, errors.Wrap(err)
		}

		switch strings.ToUpper(v[len(v)-1:]) {
		case "Y":
			t = t.AddDate(add, 0, 0)
		case "M":
			t = t.AddDate(0, add, 0)
		case "D":
			t = t.AddDate(0, 0, add)
		case "H":
			t = t.Add(time.Hour * time.Duration(add))
		case "I":
			t = t.Add(time.Minute * time.Duration(add))
		case "S":
			t = t.Add(time.Second * time.Duration(add))
		case "L":
			t = t.Add(time.Millisecond * time.Duration(add))
		case "C":
			t = t.Add(time.Microsecond * time.Duration(add))
		case "N":
			t = t.Add(time.Nanosecond * time.Duration(add))
		default:
			return t, errors.New("addTimes parameter error!")
		}
	}
	return t, nil
}

// DateInfo 获取日期信息
//
//	返回：year int - 年，
//		month int - 月，monthEn string - 英文月，
//		day int - 日，yearDay int - 一年中第几日， weekDay int - 一周中第几日，
//		hour int - 时，hour int - 分，second int - 秒，
//		millisecond int - 毫秒，microsecond int - 微妙，nanosecond int - 纳秒，
//		unix int64 - 时间戳-秒，unixNano int64 - 时间戳-纳秒，
//		weekDay int - 星期几，weekDayEn string - 星期几英文， yearWeek int - 一年中第几周，
//		date string - 格式化日期，dateNs string - 格式化日期（纳秒)
func DateInfo(t time.Time) map[string]interface{} {
	param := make(map[string]interface{})

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
	// 格式化日期
	param["date"] = t.Format(time.DateTime)
	// 格式化日期（纳秒)
	param["dateNs"] = t.Format(time.RFC3339Nano)
	return param
}

// TimeFormat 时间戳格式化为时间字符串
//
//	timeZone 时区
//	layout 格式化，例：
//	 - TimeFormat(timeZone, "2006-01-02 15:04:05", unix)
//	timestamp Unix时间戳：sec秒和nsec纳秒
func TimeFormat(timeZone *time.Location, layout string, timestamp ...int64) string {
	if len(timestamp) == 0 {
		return time.Now().In(timeZone).Format(layout)
	}

	var sec, nsec int64 = timestamp[0], 0
	if len(timestamp) > 1 {
		nsec = timestamp[1]
	}

	if len(timestamp) == 1 && sec >= 1e18 {
		sec = timestamp[0] / 1e9
		nsec = timestamp[0] % 1e9
	}
	return time.Unix(sec, nsec).In(timeZone).Format(layout)
}

// TimeParse 解析时间字符串
//
//	timeZone 时区
//	layout 格式化，例：
//	 - TimeFormat(timeZone, "2006-01-02 15:04:05", timeStr)
//	timeStr 时间字符串
func TimeParse(timeZone *time.Location, layout, timeStr string) (time.Time, error) {
	return time.ParseInLocation(layout, timeStr, timeZone)
}

// Date 使用 patterns 定义的格式对时间格式化
//
//	timeZone 时区
//	layout 格式化，例：
//	 - Date(timeZone, "Y-m-d H:i:s", unix)
//	timestamp Unix时间戳：sec秒和nsec纳秒
func Date(timeZone *time.Location, layout string, timestamp ...int64) string {
	return TimeFormat(timeZone, patterns.Replace(layout), timestamp...)
}

// Strtotime 解析时间字符串（不确定的时间格式，尽可能解析时间字符串）
//
//	timeZone 时区
//	parse 解析时间字符串包含两个参数：
//	 - layout 格式化样式字符串
//	 - timeStr 时间字符串
//	例：
//	 - Strtotime(timeZone, "2006-01-02 15:04:05") // 当前时间
//	 - Strtotime(timeZone, "2006-01-02 15:04:05", timeStr) // 指定时间timeStr
//	 - Strtotime(timeZone, "Y-m-d H:i:s") // 当前时间
//	 - Strtotime(timeZone, "Y-m-d H:i:s", timeStr) // 指定时间timeStr
func Strtotime(timeZone *time.Location, parse ...string) (t time.Time, err error) {
	if len(parse) == 1 {
		layouts := []string{
			time.RFC3339Nano,
			time.DateTime, // 2006-01-02 15:04:05
			time.DateOnly, // 2006-01-02
		}

		for _, layout := range layouts {
			if len(layout) != len(parse[0]) {
				continue
			}
			t, err = TimeParse(timeZone, layout, parse[0])
			if err == nil {
				return t, nil
			}
		}
		err = errors.Errorf("Unparsable time format:%s", parse[0])
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
		return false, errors.Errorf("t1[%v] time parsing error: %s", t1, err.Error())
	}
	tt2, err := time.Parse(layout, t2)
	if err != nil {
		return false, errors.Errorf("t2[%v] time parsing error: %s", t2, err.Error())
	}
	return tt1.Before(tt2), nil
}

// After 返回true: t1在t2之后（t1大于t2），返回false: t1小于等于t2。
func After(layout string, t1, t2 string) (bool, error) {
	tt1, err := time.Parse(layout, t1)
	if err != nil {
		return false, errors.Errorf("t1[%v] time parsing error: %s", t1, err.Error())
	}
	tt2, err := time.Parse(layout, t2)
	if err != nil {
		return false, errors.Errorf("t2[%v] time parsing error: %s", t2, err.Error())
	}
	return tt1.After(tt2), nil
}

// Equal 判断t1是否与t2相等
func Equal(layout string, t1, t2 string) (bool, error) {
	tt1, err := time.Parse(layout, t1)
	if err != nil {
		return false, errors.Errorf("t1[%v] time parsing error: %s", t1, err.Error())
	}
	tt2, err := time.Parse(layout, t2)
	if err != nil {
		return false, errors.Errorf("t2[%v] time parsing error: %s", t2, err.Error())
	}
	return tt1.Equal(tt2), nil
}

// Sub t1与t2的时间差，t1>t2 结果大于0，否则结果小于等于0
func Sub(layout string, t1, t2 string) (time.Duration, error) {
	tt1, err := time.Parse(layout, t1)
	if err != nil {
		return 0, errors.Errorf("t1[%v] time parsing error: %s", t1, err.Error())
	}
	tt2, err := time.Parse(layout, t2)
	if err != nil {
		return 0, errors.Errorf("t2[%v] time parsing error: %s", t2, err.Error())
	}
	return tt1.Sub(tt2), nil
}
