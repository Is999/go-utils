package utils

import (
	"strconv"
	"strings"
	"time"

	"github.com/Is999/go-utils/errors"
)

// ============================ 时区配置 ============================

// Local 返回 time.Local，使用进程的本地时区设置。
func Local() *time.Location {
	return time.Local
}

// CST 返回固定 UTC+8 时区，不应用夏令时规则。
func CST() *time.Location {
	return time.FixedZone("CST", 8*3600)
}

// UTC 返回名称为 UTC、偏移为 0 的固定时区。
func UTC() *time.Location {
	return time.FixedZone("UTC", 0)
}

// ============================ 时间格式模板 ============================

// patterns 将下列格式符替换为 Go layout，不实现 PHP 的全部格式符或转义规则。
// ms/us/ns 各贡献三位小数占位；.ms、.msus、.msusns 分别表示毫秒、微秒、纳秒精度。
var patterns = strings.NewReplacer(
	"ms", "000",
	"us", "000",
	"ns", "000",

	"Y", "2006", // 4 位数字完整年份
	"y", "06", // 2 位数字年份

	"m", "01", // 数字月份，有前导零（01-12）
	"n", "1", // 数字月份，无前导零（1-12）
	"M", "Jan", // 英文月份缩写
	"F", "January", // 英文月份完整

	"d", "02", // 日期，有前导零（01-31）
	"j", "2", // 日期，无前导零（1-31）

	"D", "Mon", // 英文星期缩写
	"l", "Monday", // 英文星期完整

	"g", "3", // 12 小时制小时，无前导零
	"h", "03", // 12 小时制小时，有前导零
	"H", "15", // 24 小时制小时，有前导零（00-23）

	"a", "pm", // 小写上午/下午
	"A", "PM", // 大写上午/下午

	"i", "04", // 分钟，有前导零（00-59）
	"s", "05", // 秒数，有前导零（00-59）

	"e", "MST", // 时区标识
	"P", "-07:00", // 时差（冒号分隔）
	"O", "-0700", // 时差（无冒号）
)

// ============================ 日期计算 ============================

// MonthDay 返回公历月份天数；不校验年月范围，非法月份同样返回 31。
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

// CheckDate 校验公历日期，年份限 1-32767，并检查月份天数与闰年。
func CheckDate(year, month, day int) bool {
	if year < 1 || year > 32767 || month < 1 || month > 12 || day < 1 {
		return false
	}
	return day <= MonthDay(year, month)
}

// ============================ 时间计算 ============================

// AddTime 对时间进行增减计算。
// 支持的时间单位：Y(年)、M(月)、D(日)、H(时)、I(分)、S(秒)、L(毫秒)、C(微秒)、N(纳秒)。
// 单位不区分大小写，可用 + 或 - 前缀；按参数顺序累加，失败时返回已完成部分和错误。
// 年月日使用 AddDate 的日历规则，其他单位使用固定时长。
//
// 例如：
//
//	AddTime(t, "-1D", "+2H", "30S") // 减1天，加2小时，加30秒
func AddTime(t time.Time, addTimes ...string) (time.Time, error) {
	for _, v := range addTimes {
		add, unit, err := parseAddTimeDelta(v)
		if err != nil {
			return t, err
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

// parseAddTimeDelta 接受数值两侧及表达式末尾的空白。
func parseAddTimeDelta(value string) (int, byte, error) {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return 0, 0, errors.Errorf("addTimes parameter error: %q", value)
	}

	add, err := strconv.Atoi(strings.TrimSpace(value[:len(value)-1]))
	if err != nil {
		return 0, 0, errors.Tag(err)
	}
	// 只归一化单位，未知单位仍由 AddTime 报告。
	unit := value[len(value)-1]
	if unit >= 'a' && unit <= 'z' {
		unit -= 'a' - 'A'
	}
	return add, unit, nil
}

// ============================ 日期信息提取 ============================

// TimeDetails 按 t 的时区返回日历字段；毫秒、微秒、纳秒字段均为当前秒内的偏移。
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
//   - yearWeek: ISO 周数，跨年时不一定属于 year 字段所示年份
//   - yearDay: 一年中第几天
//   - date: 格式化日期 "2006-01-02 15:04:05"
//   - dateNs: 格式化日期（纳秒精度）"2006-01-02T15:04:05.999999999Z07:00"
func TimeDetails(t time.Time) map[string]any {
	// 同一时区中的日历与时钟字段各拆分一次，避免逐字段重复换算。
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	weekday := t.Weekday()
	nanosecond := t.Nanosecond()
	_, yearWeek := t.ISOWeek()
	return map[string]any{
		"year":        year,
		"monthEn":     month.String(),
		"month":       int(month),
		"day":         day,
		"hour":        hour,
		"minute":      minute,
		"second":      second,
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

// TimeFormat 使用 Go layout 格式化时间，nil 时区按 time.Local 处理。
// 不传 timestamp 时取当前时间；传两项时按秒和纳秒解释，其余项忽略。
// 单项绝对值达到 1e18 时按纳秒解释，否则按秒解释；不自动识别毫秒或微秒。
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

// TimeParse 按 Go layout 解析；无时区信息时使用 timeZone，nil 时区按 time.Local 处理。
func TimeParse(timeZone *time.Location, layout, timeStr string) (time.Time, error) {
	return time.ParseInLocation(layout, timeStr, normalizeLocation(timeZone))
}

// Date 使用 patterns 中的格式符（如 Y-m-d H:i:s）；时间戳与时区规则同 TimeFormat。
//
// 例如：
//
//	Date(CST(), "Y-m-d H:i:s", 1700000000) // "2023-11-15 01:46:40"
func Date(timeZone *time.Location, layout string, timestamp ...int64) string {
	return TimeFormat(timeZone, patterns.Replace(layout), timestamp...)
}

// ParseTime 不传 parse 时返回当前时间和 nil 错误，nil 时区按 time.Local 处理。
// 单项为待解析文本，依次尝试 DateTime、DateOnly、RFC3339Nano；两项为 layout 和文本，忽略其余项。
// 自动匹配中 DateTime/DateOnly 按固定长度预筛选，变长文本需显式传 layout。
// 指定 layout 时先按 Go 格式解析，再尝试 patterns 替换；失败返回当前时间和解析错误。
func ParseTime(timeZone *time.Location, parse ...string) (t time.Time, err error) {
	timeZone = normalizeLocation(timeZone)
	if len(parse) == 1 {
		// 固定长度格式先匹配，普通日期无需先为 RFC3339 构造一次解析错误。
		layouts := []string{
			time.DateTime, // "2006-01-02 15:04:05"
			time.DateOnly, // "2006-01-02"
			time.RFC3339Nano,
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

		// Go layout 失败后才尝试兼容格式，避免改变可同时匹配时的优先级。
		t, err = TimeParse(timeZone, patterns.Replace(parse[0]), parse[1])
		if err == nil {
			return t, nil
		}

		err = errors.Tag(err)
	}
	// 无输入或解析失败都返回当前时间，由 err 区分是否成功。
	return time.Now().In(timeZone), err
}

// normalizeLocation 统一时间工具的 nil 时区约定。
func normalizeLocation(location *time.Location) *time.Location {
	if location == nil {
		return time.Local
	}
	return location
}

// ============================ 时间比较 ============================

// parseCompareTimes 按 Go layout 解析两个输入，无时区信息时使用 UTC。
func parseCompareTimes(layout, t1, t2 string) (time.Time, time.Time, error) {
	// 两个输入都错误时，优先报告 t1。
	tt1, err := time.Parse(layout, t1)
	if err != nil {
		// 在解析失败处附栈，比较入口直接传回同一错误。
		return time.Time{}, time.Time{}, errors.Wrapf(err, "t1[%v] time parsing error", t1)
	}
	tt2, err := time.Parse(layout, t2)
	if err != nil {
		return time.Time{}, time.Time{}, errors.Wrapf(err, "t2[%v] time parsing error", t2)
	}
	return tt1, tt2, nil
}

// Before 判断 t1 是否早于 t2；使用 Go layout，无时区信息时按 UTC 解析。
func Before(layout string, t1, t2 string) (bool, error) {
	tt1, tt2, err := parseCompareTimes(layout, t1, t2)
	if err != nil {
		return false, err
	}
	return tt1.Before(tt2), nil
}

// After 判断 t1 是否晚于 t2；使用 Go layout，无时区信息时按 UTC 解析。
func After(layout string, t1, t2 string) (bool, error) {
	tt1, tt2, err := parseCompareTimes(layout, t1, t2)
	if err != nil {
		return false, err
	}
	return tt1.After(tt2), nil
}

// Equal 判断两个文本是否代表同一时刻；使用 Go layout，无时区信息时按 UTC 解析。
func Equal(layout string, t1, t2 string) (bool, error) {
	tt1, tt2, err := parseCompareTimes(layout, t1, t2)
	if err != nil {
		return false, err
	}
	return tt1.Equal(tt2), nil
}

// Sub 返回 t1-t2 的时长；使用 Go layout，无时区信息时按 UTC 解析。
func Sub(layout string, t1, t2 string) (time.Duration, error) {
	tt1, tt2, err := parseCompareTimes(layout, t1, t2)
	if err != nil {
		return 0, err
	}
	return tt1.Sub(tt2), nil
}
