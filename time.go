package utils

import (
	"strconv"
	"strings"
	"time"

	"github.com/Is999/go-utils/errors"
)

// ============================ 时区配置 ============================

// Local 获取系统运行时区。
func Local() *time.Location {
	return time.Local
}

// CST 获取中国标准时区（东八区）。
func CST() *time.Location {
	return time.FixedZone("CST", 8*3600)
}

// UTC 获取 UTC 时区。
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
func MonthDay(year int, month int) (days int) {
	if month == 2 {
		if year%4 == 0 && year%100 != 0 || year%400 == 0 {
			return 29
		}
		return 28
	}
	if month == 4 || month == 6 || month == 9 || month == 11 {
		return 30
	}
	return 31
}

// CheckDate 验证日期是否合法。
func CheckDate(year, month, day int) bool {
	if year < 1 || year > 32767 || month < 1 || month > 12 || day < 1 {
		return false
	}
	return day <= MonthDay(year, month)
}

// ============================ 时间计算 ============================

// AddTime 对时间进行增减计算。
// 支持的时间单位：Y(年)、M(月)、D(日)、H(时)、I(分)、S(秒)、L(毫秒)、C(微秒)、N(纳秒)。
// 使用 + 或 - 前缀表示增加或减少。
//
// 例如：
//
//	AddTime(t, "-1D", "+2H", "30S") // 减1天，加2小时，加30秒
func AddTime(t time.Time, addTimes ...string) (time.Time, error) {
	for _, v := range addTimes {
		add, unit, err := parseAddTimeDelta(v)
		if err != nil {
			return t, errors.Tag(err)
		}

		switch unit {
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
			return t, errors.New("addTimes parameter error!")
		}
	}
	return t, nil
}

// parseAddTimeDelta 解析 AddTime 的单个增减表达式。
func parseAddTimeDelta(value string) (int, byte, error) {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return 0, 0, errors.Errorf("addTimes parameter error: %q", value)
	}

	add, err := strconv.Atoi(strings.TrimSpace(value[:len(value)-1]))
	if err != nil {
		return 0, 0, errors.Tag(err)
	}
	unit := value[len(value)-1]
	if unit >= 'a' && unit <= 'z' {
		unit -= 'a' - 'A'
	}
	return add, unit, nil
}

// ============================ 日期信息提取 ============================

// TimeDetails 获取时间的详细信息映射表。
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
func TimeDetails(t time.Time) map[string]any {
	month := t.Month()
	weekday := t.Weekday()
	nanosecond := t.Nanosecond()
	_, yearWeek := t.ISOWeek()
	return map[string]any{
		"year":        t.Year(),
		"monthEn":     month.String(),
		"month":       int(month),
		"day":         t.Day(),
		"hour":        t.Hour(),
		"minute":      t.Minute(),
		"second":      t.Second(),
		"millisecond": nanosecond / int(time.Millisecond),
		"microsecond": nanosecond / int(time.Microsecond),
		"nanosecond":  nanosecond,
		"unix":        t.Unix(),
		"unixNano":    t.UnixNano(),
		"weekDayEn":   weekday.String(),
		"weekDay":     int(weekday),
		"yearWeek":    yearWeek,
		"yearDay":     t.YearDay(),
		"date":        t.Format(time.DateTime),
		"dateNs":      t.Format(time.RFC3339Nano),
	}
}

// ============================ 时间格式化 ============================

// TimeFormat 将时间戳格式化为字符串。
//
// 例如：
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
func TimeParse(timeZone *time.Location, layout, timeStr string) (time.Time, error) {
	return time.ParseInLocation(layout, timeStr, normalizeLocation(timeZone))
}

// Date 使用 patterns 规则格式化时间。
// patterns 支持 PHP 风格的格式化符（如 Y-m-d H:i:s）。
//
// 例如：
//
//	Date(CST(), "Y-m-d H:i:s", 1700000000) // "2023-11-15 01:46:40"
func Date(timeZone *time.Location, layout string, timestamp ...int64) string {
	return TimeFormat(timeZone, patterns.Replace(layout), timestamp...)
}

// ParseTime 解析时间字符串（智能解析）。
// 支持多种常见格式自动识别，也支持指定格式解析。
//
// 例如：
//
//	ParseTime(CST(), "2006-01-02 15:04:05") // 当前时间（如果解析失败）
//	ParseTime(CST(), "2006-01-02 15:04:05", "2024-01-01 12:00:00") // 指定时间
//	ParseTime(CST(), "Y-m-d H:i:s") // 当前时间
func ParseTime(timeZone *time.Location, parse ...string) (t time.Time, err error) {
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

// parseCompareTimes 解析时间比较函数的两个输入。
func parseCompareTimes(layout, t1, t2 string) (time.Time, time.Time, error) {
	tt1, err := time.Parse(layout, t1)
	if err != nil {
		return time.Time{}, time.Time{}, errors.Wrapf(err, "t1[%v] time parsing error", t1)
	}
	tt2, err := time.Parse(layout, t2)
	if err != nil {
		return time.Time{}, time.Time{}, errors.Wrapf(err, "t2[%v] time parsing error", t2)
	}
	return tt1, tt2, nil
}

// Before 比较 t1 是否在 t2 之前。
func Before(layout string, t1, t2 string) (bool, error) {
	tt1, tt2, err := parseCompareTimes(layout, t1, t2)
	if err != nil {
		return false, errors.Tag(err)
	}
	return tt1.Before(tt2), nil
}

// After 比较 t1 是否在 t2 之后。
func After(layout string, t1, t2 string) (bool, error) {
	tt1, tt2, err := parseCompareTimes(layout, t1, t2)
	if err != nil {
		return false, errors.Tag(err)
	}
	return tt1.After(tt2), nil
}

// Equal 比较两个时间是否相等。
func Equal(layout string, t1, t2 string) (bool, error) {
	tt1, tt2, err := parseCompareTimes(layout, t1, t2)
	if err != nil {
		return false, errors.Tag(err)
	}
	return tt1.Equal(tt2), nil
}

// Sub 计算两个时间的差值。
func Sub(layout string, t1, t2 string) (time.Duration, error) {
	tt1, tt2, err := parseCompareTimes(layout, t1, t2)
	if err != nil {
		return 0, errors.Tag(err)
	}
	return tt1.Sub(tt2), nil
}
