package utils_test

import (
	"testing"
	"time"

	"github.com/Is999/go-utils"
)

func TestMonthDay(t *testing.T) {
	type args struct {
		year  int
		month int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "001", args: args{year: 2020, month: 1}, want: 31},
		{name: "002", args: args{year: 2020, month: 2}, want: 29},
		{name: "003", args: args{year: 2000, month: 2}, want: 29},
		{name: "004", args: args{year: 1900, month: 2}, want: 28},
		{name: "005", args: args{year: 2020, month: 3}, want: 31},
		{name: "006", args: args{year: 2020, month: 4}, want: 30},
		{name: "007", args: args{year: 2020, month: 5}, want: 31},
		{name: "008", args: args{year: 2020, month: 6}, want: 30},
		{name: "009", args: args{year: 2020, month: 7}, want: 31},
		{name: "010", args: args{year: 2020, month: 8}, want: 31},
		{name: "011", args: args{year: 2020, month: 9}, want: 30},
		{name: "012", args: args{year: 2020, month: 10}, want: 31},
		{name: "013", args: args{year: 2020, month: 11}, want: 30},
		{name: "014", args: args{year: 2020, month: 12}, want: 31},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotDays := utils.MonthDay(tt.args.year, tt.args.month); gotDays != tt.want {
				t.Errorf("MonthDay() = %v, want %v", gotDays, tt.want)
			}
		})
	}
}

func TestCheckDate(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		month int
		day   int
		want  bool
	}{
		{name: "valid leap day", year: 2024, month: 2, day: 29, want: true},
		{name: "invalid common year leap day", year: 2023, month: 2, day: 29, want: false},
		{name: "invalid small month day", year: 2024, month: 4, day: 31, want: false},
		{name: "invalid month", year: 2024, month: 13, day: 1, want: false},
		{name: "invalid day", year: 2024, month: 1, day: 0, want: false},
		{name: "invalid year", year: 0, month: 1, day: 1, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.CheckDate(tt.year, tt.month, tt.day); got != tt.want {
				t.Fatalf("CheckDate(%d, %d, %d) = %v, want %v", tt.year, tt.month, tt.day, got, tt.want)
			}
		})
	}
}

// TestAddTime 用固定 UTC 时间校验单位精度与参数累加顺序。
func TestAddTime(t *testing.T) {
	// 固定纳秒分量，使小时到纳秒的增量都能独立核对。
	base := time.Date(2023, 3, 13, 14, 40, 1, 124685000, time.UTC)
	tests := []struct {
		name   string    // 单位或组合场景。
		deltas []string  // 按给定顺序传入的增量。
		want   time.Time // 保留 UTC 时区的精确结果。
	}{
		{name: "empty", want: base},
		{name: "hours_minutes", deltas: []string{"+4H", "20I"}, want: time.Date(2023, 3, 13, 19, 0, 1, 124685000, time.UTC)},
		{name: "positive_nanoseconds", deltas: []string{"76N"}, want: time.Date(2023, 3, 13, 14, 40, 1, 124685076, time.UTC)},
		{name: "negative_nanoseconds", deltas: []string{"-76N"}, want: time.Date(2023, 3, 13, 14, 40, 1, 124684924, time.UTC)},
		{name: "nanoseconds", deltas: []string{"7600N"}, want: time.Date(2023, 3, 13, 14, 40, 1, 124692600, time.UTC)},
		{name: "microseconds_nanoseconds", deltas: []string{"7C", "600N"}, want: time.Date(2023, 3, 13, 14, 40, 1, 124692600, time.UTC)},
		{name: "milliseconds", deltas: []string{"2L"}, want: time.Date(2023, 3, 13, 14, 40, 1, 126685000, time.UTC)},
		{name: "uppercase_units", deltas: []string{"-1D", "1M", "-1Y", "+4H", "20I", "30S", "76N"}, want: time.Date(2022, 4, 12, 19, 0, 31, 124685076, time.UTC)},
		{name: "lowercase_units", deltas: []string{"-1d", "1m", "-1y", "+4h", "20i", "30s", "76n"}, want: time.Date(2022, 4, 12, 19, 0, 31, 124685076, time.UTC)},
		{name: "argument_order", deltas: []string{"1M", "-13D"}, want: time.Date(2023, 3, 31, 14, 40, 1, 124685000, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.AddTime(base, tt.deltas...)
			if err != nil {
				t.Fatalf("AddTime() error = %v", err)
			}
			if !got.Equal(tt.want) || got.Location() != time.UTC {
				t.Errorf("AddTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddTimeRejectsInvalidInputWithoutPanic(t *testing.T) {
	tests := []string{"", "   ", "1"}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("AddTime(%q) panicked: %v", tt, recovered)
				}
			}()
			if _, err := utils.AddTime(time.Unix(0, 0), tt); err == nil {
				t.Fatalf("AddTime(%q) expected error", tt)
			}
		})
	}
}

func TestDate(t *testing.T) {
	t.Run("current", func(t *testing.T) {
		// 调用可能跨秒，按输出精度检查前后时间区间。
		before := time.Now().Truncate(time.Second)
		got := utils.Date(utils.UTC(), "Y-m-d H:i:s")
		after := time.Now()
		parsed, err := time.ParseInLocation(time.DateTime, got, time.UTC)
		if err != nil || parsed.Before(before) || parsed.After(after) {
			t.Fatalf("Date() = %q, error = %v, want time in [%v, %v]", got, err, before, after)
		}
	})

	type args struct {
		format string
		ts     []int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "002", args: args{format: "Y-m-d H:i:s", ts: []int64{1678718401124685076}}, want: "2023-03-13 14:40:01"},
		{name: "003", args: args{format: "Y-m-d H:i:s", ts: []int64{1678718401}}, want: "2023-03-13 14:40:01"},
		{name: "004", args: args{format: "F-d/Y l Ah:i:s Pe", ts: []int64{1678718401}}, want: "March-13/2023 Monday PM02:40:01 +00:00UTC"},
		{name: "005", args: args{format: "Y-m-d H:i:s", ts: []int64{1678718401, 124685076}}, want: "2023-03-13 14:40:01"},
		{name: "006", args: args{format: "Y-m-d H:i:s.ms", ts: []int64{1678718401, 124685076}}, want: "2023-03-13 14:40:01.124"},
		{name: "007", args: args{format: "Y-m-d H:i:s.msus", ts: []int64{1678718401, 124685076}}, want: "2023-03-13 14:40:01.124685"},
		{name: "008", args: args{format: "Y-m-d H:i:s.msusns", ts: []int64{1678718401, 124685076}}, want: "2023-03-13 14:40:01.124685076"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Date(utils.UTC(), tt.args.format, tt.args.ts...); got != tt.want {
				t.Errorf("Date() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeFormat(t *testing.T) {
	t.Run("current", func(t *testing.T) {
		// 按固定 UTC+8 解析输出，区间判断允许调用跨秒。
		before := time.Now().Truncate(time.Second)
		got := utils.TimeFormat(utils.CST(), time.DateTime)
		after := time.Now()
		parsed, err := time.ParseInLocation(time.DateTime, got, time.FixedZone("CST", 8*60*60))
		if err != nil || parsed.Before(before) || parsed.After(after) {
			t.Fatalf("TimeFormat() = %q, error = %v, want time in [%v, %v]", got, err, before, after)
		}
	})

	type args struct {
		format string
		ts     []int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "002", args: args{format: "2006-01-02 15:04:05.000000000", ts: []int64{1678718401124685076}}, want: "2023-03-13 22:40:01.124685076"}, // UnixNano纳秒
		{name: "003", args: args{format: time.DateTime, ts: []int64{1678718401}}, want: "2023-03-13 22:40:01"},                                      // Unix秒
		{name: "004", args: args{format: time.DateTime, ts: []int64{1678718401, 124685076}}, want: "2023-03-13 22:40:01"},                           // Unix秒 + 纳秒
		{name: "005", args: args{format: "2006-01-02 15:04:05.000", ts: []int64{1678718401, 124685076}}, want: "2023-03-13 22:40:01.124"},
		{name: "006", args: args{format: "2006-01-02 15:04:05.000000", ts: []int64{1678718401, 124685076}}, want: "2023-03-13 22:40:01.124685"},
		{name: "007", args: args{format: "2006-01-02 15:04:05.000000000", ts: []int64{1678718401, 124685076}}, want: "2023-03-13 22:40:01.124685076"}, // Unix秒 + 纳秒
		{name: "008", args: args{format: "2006/01/02 15:04:05.000000000", ts: []int64{1678718401124685076}}, want: "2023/03/13 22:40:01.124685076"},   // UnixNano纳秒
		{name: "009", args: args{format: "20060102150405.000000000", ts: []int64{1678718401, 124685076}}, want: "20230313224001.124685076"},
		{name: "010", args: args{format: time.DateTime + " Z07:00", ts: []int64{1678718401124685076}}, want: "2023-03-13 22:40:01 +08:00"},
		{name: "011", args: args{format: time.DateTime + " -0700", ts: []int64{1678718401124685076}}, want: "2023-03-13 22:40:01 +0800"},
		{name: "012", args: args{format: time.DateTime + " -0700 MST", ts: []int64{1678718401, 124685076}}, want: "2023-03-13 22:40:01 +0800 CST"},
		{name: "013", args: args{format: time.StampNano, ts: []int64{1678718401, 124685076}}, want: "Mar 13 22:40:01.124685076"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.TimeFormat(utils.CST(), tt.args.format, tt.args.ts...); got != tt.want {
				t.Errorf("TimeFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeFormatNilLocationAndNegativeUnixNano(t *testing.T) {
	got := utils.TimeFormat(nil, time.RFC3339, 0)
	want := time.Unix(0, 0).In(time.Local).Format(time.RFC3339)
	if got != want {
		t.Fatalf("TimeFormat(nil) = %q, want %q", got, want)
	}

	const ts int64 = -1234567890123456789
	got = utils.TimeFormat(utils.UTC(), "2006-01-02 15:04:05.000000000", ts)
	want = time.Unix(ts/1e9, ts%1e9).In(utils.UTC()).Format("2006-01-02 15:04:05.000000000")
	if got != want {
		t.Fatalf("TimeFormat(negative unix nano) = %q, want %q", got, want)
	}
}

func TestParseTime(t *testing.T) {
	type args struct {
		e []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{}, wantErr: false},
		{name: "002", args: args{[]string{"Y-m-d H:i:s", "2023-03-13 14:40:01"}}, wantErr: false},
		{name: "003", args: args{[]string{"F-d/Y l Ah:i:s Pe", "March-13/2023 Monday PM02:40:01 +00:00UTC"}}, wantErr: false},
		{name: "004", args: args{[]string{"Y-m-d H:i:s.ms", "2023-03-13 14:40:01.124"}}, wantErr: false},
		{name: "005", args: args{[]string{"Y-m-d H:i:s.msus", "2023-03-13 14:40:01.124685"}}, wantErr: false},
		{name: "006", args: args{[]string{"Y-m-d H:i:s.msusns", "2023-03-13 14:40:01.124685076"}}, wantErr: false},
		{name: "007", args: args{[]string{"2023-03-13 14:40:01"}}, wantErr: false},
		{name: "008", args: args{[]string{"Mar 13 22:40:01.124685076"}}, wantErr: true},
		{name: "009", args: args{[]string{"2023 Mar 13 22:40:01.124685076"}}, wantErr: true}, // 匹配不到
		{name: "010", args: args{[]string{"2006-01-02 15:04:05", "2023-03-13 14:40:01"}}, wantErr: false},
		{name: "011", args: args{[]string{"2006-01-02 15:04:05.000", "2023-03-13 14:40:01.124"}}, wantErr: false},
		{name: "012", args: args{[]string{"2006-01-02 15:04:05.000000", "2023-03-13 14:40:01.124685"}}, wantErr: false},
		{name: "013", args: args{[]string{"2006-01-02 15:04:05.000000000", "2023-03-13 14:40:01.124685076"}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := utils.ParseTime(utils.Local(), tt.args.e...); (err == nil) == tt.wantErr {
				t.Errorf("ParseTime() = %v, want %v, WrapError = %v", got.UnixNano(), tt.wantErr, err)
			}
		})
	}
}

func TestParseTimeRFC3339NanoVariableLength(t *testing.T) {
	tests := []string{
		"2023-03-13T14:40:01Z",
		"2023-03-13T14:40:01.124685076Z",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			if _, err := utils.ParseTime(utils.UTC(), tt); err != nil {
				t.Fatalf("ParseTime(%q) error = %v", tt, err)
			}
		})
	}
}

func TestParseTimeAutoLayouts(t *testing.T) {
	// 无时区日期使用调用方时区；RFC3339 自带的 Z/偏移不能被自动匹配顺序覆盖。
	location := time.FixedZone("test", 8*60*60)
	for _, tt := range []struct {
		input string
		want  time.Time
	}{
		{input: "2024-02-29", want: time.Date(2024, 2, 29, 0, 0, 0, 0, location)},
		{input: "2024-02-29 13:14:15", want: time.Date(2024, 2, 29, 13, 14, 15, 0, location)},
		{input: "2024-02-29T13:14:15Z", want: time.Date(2024, 2, 29, 13, 14, 15, 0, time.UTC)},
		{input: "2024-02-29T1:14:15Z", want: time.Date(2024, 2, 29, 1, 14, 15, 0, time.UTC)},
		{input: "2024-02-29T13:14:15.123+08:00", want: time.Date(2024, 2, 29, 13, 14, 15, 123000000, location)},
	} {
		t.Run(tt.input, func(t *testing.T) {
			got, err := utils.ParseTime(location, tt.input)
			_, gotOffset := got.Zone()
			_, wantOffset := tt.want.Zone()
			if err != nil || !got.Equal(tt.want) || gotOffset != wantOffset {
				t.Fatalf("ParseTime(%q) = (%v, %v), want %v", tt.input, got, err, tt.want)
			}
		})
	}
	for _, input := range []string{"2024-02-30", "2024-02-29 25:14:15", "2024-02-29T13:14:15", "invalid"} {
		if _, err := utils.ParseTime(location, input); err == nil || err.Error() != "Unparsable time format:"+input {
			t.Fatalf("ParseTime(%q) error = %v, want unchanged parse error", input, err)
		}
	}
}

func TestTimeDetailsCalendarFields(t *testing.T) {
	// 使用闰日和非零时分秒，验证返回字段保持 int 类型及调用方时区中的日历值。
	value := time.Date(2024, 2, 29, 13, 14, 15, 123456789, time.FixedZone("test", 8*60*60))
	details := utils.TimeDetails(value)
	for key, want := range map[string]any{
		"year": 2024, "month": 2, "day": 29, "monthEn": "February",
		"hour": 13, "minute": 14, "second": 15,
		"millisecond": 123, "microsecond": 123456, "nanosecond": 123456789,
		"date": "2024-02-29 13:14:15", "dateNs": "2024-02-29T13:14:15.123456789+08:00",
	} {
		if got := details[key]; got != want {
			t.Fatalf("TimeDetails()[%q] = %#v, want %#v", key, got, want)
		}
	}
}

func BenchmarkParseTimeAutoLayouts(b *testing.B) {
	// 各自动匹配格式单独计量，时区构造放在计时范围之外。
	location := time.FixedZone("test", 8*60*60)
	for _, input := range []string{"2024-02-29", "2024-02-29 13:14:15", "2024-02-29T13:14:15Z"} {
		b.Run(input, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := utils.ParseTime(location, input); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTimeDetails(b *testing.B) {
	// 保留完整公开返回值的构建成本，不单测日期拆分 helper。
	value := time.Date(2024, 2, 29, 13, 14, 15, 123456789, time.FixedZone("test", 8*60*60))
	b.ReportAllocs()
	for b.Loop() {
		_ = utils.TimeDetails(value)
	}
}

func TestEqual(t *testing.T) {
	type args struct {
		layout string
		t1     string
		t2     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "001", args: args{layout: time.DateOnly, t1: "2023-03-13", t2: "2023-03-13"}, want: true},
		{name: "002", args: args{layout: time.DateOnly, t1: "2023-03-13", t2: "2023-03-14"}, want: false},
		{name: "003", args: args{layout: time.DateOnly, t1: "2023-03-13", t2: "2023-04-13"}, want: false},
		{name: "004", args: args{layout: time.DateOnly, t1: "2023-03-13", t2: "2022-03-13"}, want: false},
		{name: "005", args: args{layout: "2006-01-02 15", t1: "2023-03-13 14", t2: "2023-03-13 14"}, want: true},
		{name: "006", args: args{layout: "2006-01-02 15", t1: "2023-03-13 14", t2: "2023-03-13 15"}, want: false},
		{name: "007", args: args{layout: "2006-01-02 15:04", t1: "2023-03-13 14:40", t2: "2023-03-13 14:40"}, want: true},
		{name: "008", args: args{layout: "2006-01-02 15:04", t1: "2023-03-13 14:40", t2: "2023-03-13 14:41"}, want: false},
		{name: "009", args: args{layout: time.DateTime, t1: "2023-03-13 14:40:01", t2: "2023-03-13 14:40:01"}, want: true},
		{name: "010", args: args{layout: time.DateTime, t1: "2023-03-13 14:40:01", t2: "2023-03-13 14:40:09"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.Equal(tt.args.layout, tt.args.t1, tt.args.t2)
			if err != nil {
				t.Fatalf("Equal() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAfter(t *testing.T) {
	type args struct {
		layout string
		t1     string
		t2     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "001", args: args{layout: time.DateOnly, t1: "2023-03-13", t2: "2023-03-13"}, want: false}, // t1 等于 t2
		{name: "002", args: args{layout: time.DateOnly, t1: "2023-03-13", t2: "2023-03-14"}, want: false}, // t1 小于 t2
		{name: "003", args: args{layout: time.DateOnly, t1: "2023-03-13", t2: "2023-04-13"}, want: false},
		{name: "004", args: args{layout: time.DateOnly, t1: "2023-03-14", t2: "2023-03-13"}, want: true}, // t1 大于 t2
		{name: "005", args: args{layout: "2006-01-02 15", t1: "2023-03-13 14", t2: "2023-03-13 14"}, want: false},
		{name: "006", args: args{layout: "2006-01-02 15", t1: "2023-03-13 14", t2: "2023-03-13 15"}, want: false},
		{name: "007", args: args{layout: "2006-01-02 15", t1: "2023-03-13 15", t2: "2023-03-13 14"}, want: true},
		{name: "008", args: args{layout: time.DateTime, t1: "2023-03-13 14:40:01", t2: "2023-03-13 14:40:01"}, want: false},
		{name: "009", args: args{layout: time.DateTime, t1: "2023-03-13 14:40:21", t2: "2023-03-13 14:40:01"}, want: true},
		{name: "010", args: args{layout: time.DateTime, t1: "2023-03-13 14:40:01", t2: "2023-03-13 14:40:09"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.After(tt.args.layout, tt.args.t1, tt.args.t2)
			if err != nil {
				t.Fatalf("After() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("After() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBefore(t *testing.T) {
	type args struct {
		layout string
		t1     string
		t2     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "001", args: args{layout: time.DateOnly, t1: "2023-03-13", t2: "2023-03-13"}, want: false}, // t1 等于 t2
		{name: "002", args: args{layout: time.DateOnly, t1: "2023-03-13", t2: "2023-03-14"}, want: true},  // t1 小于 t2
		{name: "003", args: args{layout: time.DateOnly, t1: "2023-03-13", t2: "2023-04-13"}, want: true},
		{name: "004", args: args{layout: time.DateOnly, t1: "2023-03-14", t2: "2023-03-13"}, want: false}, // t1 大于 t2
		{name: "005", args: args{layout: "2006-01-02 15", t1: "2023-03-13 14", t2: "2023-03-13 14"}, want: false},
		{name: "006", args: args{layout: "2006-01-02 15", t1: "2023-03-13 14", t2: "2023-03-13 15"}, want: true},
		{name: "007", args: args{layout: "2006-01-02 15", t1: "2023-03-13 15", t2: "2023-03-13 14"}, want: false},
		{name: "008", args: args{layout: time.DateTime, t1: "2023-03-13 14:40:01", t2: "2023-03-13 14:40:01"}, want: false},
		{name: "009", args: args{layout: time.DateTime, t1: "2023-03-13 14:40:21", t2: "2023-03-13 14:40:01"}, want: false},
		{name: "010", args: args{layout: time.DateTime, t1: "2023-03-13 14:40:01", t2: "2023-03-13 14:40:09"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.Before(tt.args.layout, tt.args.t1, tt.args.t2)
			if err != nil {
				t.Fatalf("Before() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Before() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSub(t *testing.T) {
	type args struct {
		layout string
		t1     string
		t2     string
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{name: "001", args: args{layout: time.DateOnly, t1: "2023-03-13", t2: "2023-03-13"}, want: 0},               // t1 == t2，结果等于 0
		{name: "002", args: args{layout: time.DateOnly, t1: "2023-03-13", t2: "2023-03-14"}, want: -86400000000000}, // t1 < t2，结果小于 0
		{name: "004", args: args{layout: time.DateOnly, t1: "2023-03-14", t2: "2023-03-13"}, want: 86400000000000},  // t1 > t2 结果大于 0
		{name: "005", args: args{layout: time.DateTime, t1: "2023-03-13 14:40:12", t2: "2023-03-13 14:40:01"}, want: 11000000000},
		{name: "006", args: args{layout: "2006-01-02 15:04:05.000000000", t1: "2023-03-13 14:40:01.124685776", t2: "2023-03-13 14:40:01.124685076"}, want: 700},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.Sub(tt.args.layout, tt.args.t1, tt.args.t2)
			if err != nil {
				t.Fatalf("Sub() error = %v", err)
			}
			if got.Nanoseconds() != tt.want {
				t.Errorf("Sub() = %v, want %v", got.Nanoseconds(), tt.want)
			}
		})
	}
}
