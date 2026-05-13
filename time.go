package utils

import (
	"strconv"
	"strings"
	"time"

	"github.com/Is999/go-utils/errors"
)

// ============================ 时区配置 ============================

// Local 获取系统运行时区。
//
// 返回值：*time.Location 系统当前时区
func Local() *time.Location {
	return time.Now().Location()
}

// CST 获取中国标准时区（东八区）。
//
// 返回值：*time.Location UTC+8 时区
func CST() *time.Location {
	return time.FixedZone("CST", 8*3600)
}

// UTC 获取 UTC 时区。
//
// 返回值：*time.Location UTC 时区
func UTC() *time.Location {
	return time.FixedZone("UTC", 0)
}

// ============================ 时间格式模板 ============================

// patterns 时间格式化 layout 规则映射表。
// 将常用格式符转换为 Go 的时间格式化字符串。
//
// 格式符说明：
//   - Y: 4 位数字年份（如 2024）
//   - y: 2 位数字年份（如 24）
//   - m: 2 位数字月份（01-12）
//   - n: 1-2 位数字月份（1-12）
//   - d: 2 位数字日期（01-31）
//   - j: 1-2 位数字日期（1-31）
//   - H: 24 小时制小时（00-23）
//   - i: 分钟（00-59）
//   - s: 秒数（00-59）
var patterns = strings.NewReplacer(
	"ms", "000", // 毫秒数（3 位）
	"us", "000", // 微秒数（3 位）
	"ns", "000", // 纳秒数（3 位）

	"Y", "2006", // 4 位数字完整年份
	"y", "06", // 2 位数字年份

	"m", "01", // 数字月份，有前导零
	"n", "1", // 数字月份，无前导零
	"M", "Jan", // 英文月份缩写
	"F", "January", // 英文月份完整

	"d", "02", // 日期，有前导零
	"j", "2", // 日期，无前导零

	"D", "Mon", // 英文星期缩写
	"l", "Monday", // 英文星期完整

	"g", "3", // 12 小时制小时，无前导零
	"h", "03", // 12 小时制小时，有前导零
	"H", "15", // 24 小时制小时，有前导零

	"a", "pm", // 小写上午/下午
	"A", "PM", // 大写上午/下午

	"i", "04", // 分钟，有前导零
	"s", "05", // 秒数，有前导零

	"e", "MST", // 时区标识
	"P", "-07:00", // 时差（冒号分隔）
	"O", "-0700", // 时差（无冒号）
)

// ============================ 日期计算 ============================

// MonthDay 获取指定年份月份的天数。
//
// 参数说明：
//   - year：年份
//   - month：月份（1-12）
//
// 返回值：该月的天数
func MonthDay(year int, month int) (days int) {
	switch {
	case month != 2 && (month == 4 || month == 6 || month == 9 || month == 11):
		days = 30
	case month != 2:
		days = 31
	case ((year%4) == 0 && (year%100) != 0) || (year%400) == 0:
		days = 29
	default:
		days = 28
	}
	return
}

// CheckDate 验证日期是否合法。
//
// 参数说明：
//   - year：年份（1-32767）
//   - month：月份（1-12）
//   - day：日期（1-31）
//
// 返回值：true 表示日期合法
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

// ============================ 时间计算 ============================

// AddTime 对时间进行增减计算。
// 支持的时间单位：Y(年)、M(月)、D(日)、H(时)、I(分)、S(秒)、L(毫秒)、C(微秒)、N(纳秒)。
// 使用 + 或 - 前缀表示增加或减少。
//
// 参数说明：
//   - t：原始时间
//   - addTimes：增减时间列表，如 "-1D"、"+2H"、"1M"
//
// 返回值：计算后的时间，错误信息
//
// 示例：
//
//	AddTime(t, "-1D", "+2H", "30S") // 减1天，加2小时，加30秒
func AddTime(t time.Time, addTimes ...string) (time.Time, error) {
	for _, v := range addTimes {
		v = strings.TrimSpace(v)
		if len(v) < 2 {
			return t, errors.Errorf("addTimes parameter error: %q", v)
		}
		add, err := strconv.Atoi(strings.TrimSpace(v[:len(v)-1]))
		if err != nil {
			return t, errors.Tag(err)
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

// ============================ 日期信息提取 ============================

// DateInfo 获取时间的详细信息映射表。
//
// 参数说明：
//   - t：待解析的时间
//
// 返回值：包含年月日时分秒、周信息、时间戳等的 map
//
// 返回值字段说明：
//   - year: 年份
//   - month: 月份（1-12）
//   - monthEn: 英文月份
//   - day: 日期
//   - hour: 小时（0-23）
//   - minute: 分钟（0-59）
//   - second: 秒数（0-59）
//   - millisecond: 毫秒
//   - microsecond: 微秒
//   - nanosecond: 纳秒
//   - unix: 时间戳（秒）
//   - unixNano: 时间戳（纳秒）
//   - weekDay: 星期几（0-6，0 为周日）
//   - weekDayEn: 英文星期
//   - yearWeek: 一年中第几周
//   - yearDay: 一年中第几天
//   - date: 格式化日期 "2006-01-02 15:04:05"
//   - dateNs: 格式化日期（纳秒精度）"2006-01-02T15:04:05.999999999Z07:00"
func DateInfo(t time.Time) map[string]interface{} {
	param := make(map[string]interface{})

	param["year"] = t.Year()
	param["monthEn"] = t.Month().String()
	param["month"] = int(t.Month())
	param["day"] = t.Day()
	param["hour"] = t.Hour()
	param["minute"] = t.Minute()
	param["second"] = t.Second()
	param["millisecond"] = t.Nanosecond() / int(time.Millisecond)
	param["microsecond"] = t.Nanosecond() / int(time.Microsecond)
	param["nanosecond"] = t.Nanosecond()
	param["unix"] = t.Unix()
	param["unixNano"] = t.UnixNano()
	param["weekDayEn"] = t.Weekday().String()
	param["weekDay"] = int(t.Weekday())
	_, param["yearWeek"] = t.ISOWeek()
	param["yearDay"] = t.YearDay()
	param["date"] = t.Format(time.DateTime)
	param["dateNs"] = t.Format(time.RFC3339Nano)
	return param
}

// ============================ 时间格式化 ============================

// TimeFormat 将时间戳格式化为字符串。
//
// 参数说明：
//   - timeZone：目标时区
//   - layout：格式化模板，如 "2006-01-02 15:04:05"
//   - timestamp：可变参数，Unix 时间戳（秒），支持传入纳秒
//
// 返回值：格式化后的时间字符串
//
// 示例：
//
//	TimeFormat(CST(), "2006-01-02 15:04:05", 1700000000)
//	TimeFormat(CST(), "2006-01-02 15:04:05", 1700000000000000000) // 纳秒时间戳
func TimeFormat(timeZone *time.Location, layout string, timestamp ...int64) string {
	timeZone = normalizeLocation(timeZone)
	if len(timestamp) == 0 {
		return time.Now().In(timeZone).Format(layout)
	}

	sec, nsec := timestamp[0], int64(0)
	if len(timestamp) > 1 {
		nsec = timestamp[1]
	}

	if len(timestamp) == 1 && (sec >= 1e18 || sec <= -1e18) {
		sec = timestamp[0] / 1e9
		nsec = timestamp[0] % 1e9
	}
	return time.Unix(sec, nsec).In(timeZone).Format(layout)
}

// TimeParse 解析时间字符串为 time.Time。
//
// 参数说明：
//   - timeZone：目标时区
//   - layout：格式化模板
//   - timeStr：时间字符串
//
// 返回值：解析后的时间，错误信息
func TimeParse(timeZone *time.Location, layout, timeStr string) (time.Time, error) {
	return time.ParseInLocation(layout, timeStr, normalizeLocation(timeZone))
}

// Date 使用 patterns 规则格式化时间。
// patterns 支持 PHP 风格的格式化符（如 Y-m-d H:i:s）。
//
// 参数说明：
//   - timeZone：目标时区
//   - layout：格式化模板
//   - timestamp：可变参数，Unix 时间戳
//
// 返回值：格式化后的时间字符串
//
// 示例：
//
//	Date(CST(), "Y-m-d H:i:s", 1700000000) // "2023-11-15 01:46:40"
func Date(timeZone *time.Location, layout string, timestamp ...int64) string {
	return TimeFormat(timeZone, patterns.Replace(layout), timestamp...)
}

// Strtotime 解析时间字符串（智能解析）。
// 支持多种常见格式自动识别，也支持指定格式解析。
//
// 参数说明：
//   - timeZone：目标时区
//   - parse：可变参数，第一个为格式化模板，第二个为时间字符串
//
// 返回值：解析后的时间，错误信息
//
// 示例：
//
//	Strtotime(CST(), "2006-01-02 15:04:05") // 当前时间（如果解析失败）
//	Strtotime(CST(), "2006-01-02 15:04:05", "2024-01-01 12:00:00") // 指定时间
//	Strtotime(CST(), "Y-m-d H:i:s") // 当前时间
func Strtotime(timeZone *time.Location, parse ...string) (t time.Time, err error) {
	timeZone = normalizeLocation(timeZone)
	if len(parse) == 1 {
		layouts := []string{
			time.RFC3339Nano,
			time.DateTime, // "2006-01-02 15:04:05"
			time.DateOnly, // "2006-01-02"
		}

		for _, layout := range layouts {
			if layout != time.RFC3339Nano && len(layout) != len(parse[0]) {
				continue
			}
			t, err = TimeParse(timeZone, layout, parse[0])
			if err == nil {
				return t, nil
			}
		}
		err = errors.Errorf("Unparsable time format:%s", parse[0])
	} else if len(parse) > 1 {
		t, err = TimeParse(timeZone, parse[0], parse[1])
		if err == nil {
			return t, nil
		}

		t, err = TimeParse(timeZone, patterns.Replace(parse[0]), parse[1])
		if err == nil {
			return t, nil
		}

		err = errors.Tag(err)
	}
	return time.Now().In(timeZone), err
}

// normalizeLocation 将 nil 时区归一为 time.Local，避免 time.Time.In(nil) 或 ParseInLocation(nil) panic。
func normalizeLocation(location *time.Location) *time.Location {
	if location == nil {
		return time.Local
	}
	return location
}

// ============================ 时间比较 ============================

// Before 比较 t1 是否在 t2 之前。
//
// 参数说明：
//   - layout：时间格式化模板
//   - t1：第一个时间字符串
//   - t2：第二个时间字符串
//
// 返回值：true 表示 t1 在 t2 之前，错误信息
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

// After 比较 t1 是否在 t2 之后。
//
// 参数说明：
//   - layout：时间格式化模板
//   - t1：第一个时间字符串
//   - t2：第二个时间字符串
//
// 返回值：true 表示 t1 在 t2 之后，错误信息
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

// Equal 比较两个时间是否相等。
//
// 参数说明：
//   - layout：时间格式化模板
//   - t1：第一个时间字符串
//   - t2：第二个时间字符串
//
// 返回值：true 表示相等，错误信息
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

// Sub 计算两个时间的差值。
//
// 参数说明：
//   - layout：时间格式化模板
//   - t1：第一个时间字符串
//   - t2：第二个时间字符串
//
// 返回值：t1 - t2 的时间差，错误信息
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
