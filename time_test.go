package utils_test

import (
	"github.com/Is999/go-utils"
	"testing"
	"time"
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

func TestDateInfo(t *testing.T) {
	type args struct {
		s []string
		t time.Time
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "000", args: args{t: time.Now()}, wantErr: false},                                                                                    // 当前时间
		{name: "001", args: args{s: []string{"+4H", "20I"}, t: time.Now()}, wantErr: false},                                                         // 当前时间+ 4小时20分钟
		{name: "002", args: args{s: nil, t: time.Unix(1678718401, 0)}, wantErr: false},                                                              // 指定时间
		{name: "003", args: args{s: nil, t: time.Unix(1678718401, 124685000)}, wantErr: false},                                                      // 指定时间
		{name: "004", args: args{s: []string{"76N"}, t: time.Unix(1678718401, 124685000)}, wantErr: false},                                          // 指定时间+ 76纳秒
		{name: "005", args: args{s: []string{"-76N"}, t: time.Unix(1678718401, 124685000)}, wantErr: false},                                         // 指定时间- 76纳秒
		{name: "006", args: args{s: []string{"7600N"}, t: time.Unix(1678718401, 124685000)}, wantErr: false},                                        // 指定时间+ 7.6微秒
		{name: "007", args: args{s: []string{"7C", "600N"}, t: time.Unix(1678718401, 124685000)}, wantErr: false},                                   // 指定时间+ 7.6微秒
		{name: "008", args: args{s: []string{"-1D", "1M", "-1Y", "+4H", "20I", "30S", "76N"}, t: time.Unix(1678718401, 124685000)}, wantErr: false}, // 指定时间+
		{name: "009", args: args{s: []string{"-1d", "1m", "-1y", "+4h", "20i", "30s", "76n"}, t: time.Unix(1678718401, 124685000)}, wantErr: false}, // 指定时间+
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.AddTime(tt.args.t, tt.args.s...)
			if (err != nil) != tt.wantErr {
				t.Errorf("DateInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			utils.MapRange(utils.DateInfo(got), func(key string, value interface{}) bool {
				//t.Logf("%v %v\n", key, value)
				return true
			})
		})
	}
}

func TestDate(t *testing.T) {
	type args struct {
		format string
		ts     []int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "001", args: args{format: "Y-m-d H:i:s"}, want: ""},
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
			if got := utils.Date(utils.UTC(), tt.args.format, tt.args.ts...); tt.want != "" && got != tt.want {
				t.Errorf("Date() = %v, want %v", got, tt.want)
			} else {
				// t.Logf("Date() = %v\n", got)
			}
		})
	}
}

func TestTimeFormat(t *testing.T) {
	type args struct {
		format string
		ts     []int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "001", args: args{format: utils.SecondDash}, want: ""},
		{name: "002", args: args{format: utils.NanosecondDash, ts: []int64{1678718401124685076}}, want: "2023-03-13 22:40:01.124685076"}, // UnixNano纳秒
		{name: "003", args: args{format: utils.SecondDash, ts: []int64{1678718401}}, want: "2023-03-13 22:40:01"},                        // Unix秒
		{name: "004", args: args{format: utils.SecondDash, ts: []int64{1678718401, 124685076}}, want: "2023-03-13 22:40:01"},             // Unix秒 + 纳秒
		{name: "005", args: args{format: utils.MillisecondDash, ts: []int64{1678718401, 124685076}}, want: "2023-03-13 22:40:01.124"},
		{name: "006", args: args{format: utils.MicrosecondDash, ts: []int64{1678718401, 124685076}}, want: "2023-03-13 22:40:01.124685"},
		{name: "007", args: args{format: utils.NanosecondDash, ts: []int64{1678718401, 124685076}}, want: "2023-03-13 22:40:01.124685076"}, // Unix秒 + 纳秒
		{name: "008", args: args{format: utils.NanosecondSlash, ts: []int64{1678718401124685076}}, want: "2023/03/13 22:40:01.124685076"},  // UnixNano纳秒
		{name: "009", args: args{format: utils.NanosecondSeam, ts: []int64{1678718401, 124685076}}, want: "20230313224001.124685076"},
		{name: "010", args: args{format: utils.SecondDash + " Z07:00", ts: []int64{1678718401124685076}}, want: "2023-03-13 22:40:01 +08:00"},
		{name: "011", args: args{format: utils.SecondDash + " -0700", ts: []int64{1678718401124685076}}, want: "2023-03-13 22:40:01 +0800"},
		{name: "012", args: args{format: utils.SecondDash + " -0700 MST", ts: []int64{1678718401, 124685076}}, want: "2023-03-13 22:40:01 +0800 CST"},
		{name: "013", args: args{format: time.StampNano, ts: []int64{1678718401, 124685076}}, want: "Mar 13 22:40:01.124685076"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.TimeFormat(utils.CST(), tt.args.format, tt.args.ts...); tt.want != "" && got != tt.want {
				t.Errorf("TimeFormat() = %v, want %v", got, tt.want)
			} else {
				// t.Logf("TimeFormat() = %v\n", got)
			}
		})
	}
}

func TestStrtotime(t *testing.T) {
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
		{name: "008", args: args{[]string{"2023-03-13 14:40:01.124"}}, wantErr: false},
		{name: "009", args: args{[]string{"2023-03-13 14:40:01.124685"}}, wantErr: false},
		{name: "010", args: args{[]string{"2023-03-13 14:40:01.124685076"}}, wantErr: false},
		{name: "011", args: args{[]string{"Mar 13 22:40:01.124685076"}}, wantErr: true},
		{name: "012", args: args{[]string{"2023 Mar 13 22:40:01.124685076"}}, wantErr: true}, // 匹配不到
		{name: "013", args: args{[]string{"2006-01-02 15:04:05", "2023-03-13 14:40:01"}}, wantErr: false},
		{name: "014", args: args{[]string{"2006-01-02 15:04:05.000", "2023-03-13 14:40:01.124"}}, wantErr: false},
		{name: "015", args: args{[]string{"2006-01-02 15:04:05.000000", "2023-03-13 14:40:01.124685"}}, wantErr: false},
		{name: "016", args: args{[]string{"2006-01-02 15:04:05.000000000", "2023-03-13 14:40:01.124685076"}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := utils.Strtotime(utils.Local(), tt.args.e...); (err == nil) == tt.wantErr {
				t.Errorf("Strtotime() = %v, want %v, WrapError = %v", got.UnixNano(), tt.wantErr, err)
			} else if !tt.wantErr {
				//t.Logf("Strtotime() unxNano %v, time %v", got.UnixNano(), got.Format(utils.NanosecondDash))
			}
		})
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
		{name: "001", args: args{layout: utils.DayDash, t1: "2023-03-13", t2: "2023-03-13"}, want: true},
		{name: "002", args: args{layout: utils.DayDash, t1: "2023-03-13", t2: "2023-03-14"}, want: false},
		{name: "003", args: args{layout: utils.DayDash, t1: "2023-03-13", t2: "2023-04-13"}, want: false},
		{name: "004", args: args{layout: utils.DayDash, t1: "2023-03-13", t2: "2022-03-13"}, want: false},
		{name: "005", args: args{layout: utils.HourDash, t1: "2023-03-13 14", t2: "2023-03-13 14"}, want: true},
		{name: "006", args: args{layout: utils.HourDash, t1: "2023-03-13 14", t2: "2023-03-13 15"}, want: false},
		{name: "007", args: args{layout: utils.MinuteDash, t1: "2023-03-13 14:40", t2: "2023-03-13 14:40"}, want: true},
		{name: "008", args: args{layout: utils.MinuteDash, t1: "2023-03-13 14:40", t2: "2023-03-13 14:41"}, want: false},
		{name: "009", args: args{layout: utils.SecondDash, t1: "2023-03-13 14:40:01", t2: "2023-03-13 14:40:01"}, want: true},
		{name: "010", args: args{layout: utils.SecondDash, t1: "2023-03-13 14:40:01", t2: "2023-03-13 14:40:09"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := utils.Equal(tt.args.layout, tt.args.t1, tt.args.t2); err != nil && got != tt.want {
				t.Errorf("Equal() = %v, want %v, WrapError %v", got, tt.want, err)
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
		{name: "001", args: args{layout: utils.DayDash, t1: "2023-03-13", t2: "2023-03-13"}, want: false}, // t1 等于 t2
		{name: "002", args: args{layout: utils.DayDash, t1: "2023-03-13", t2: "2023-03-14"}, want: false}, // t1 小于 t2
		{name: "003", args: args{layout: utils.DayDash, t1: "2023-03-13", t2: "2023-04-13"}, want: false},
		{name: "004", args: args{layout: utils.DayDash, t1: "2023-03-14", t2: "2023-03-13"}, want: true}, // t1 大于 t2
		{name: "005", args: args{layout: utils.HourDash, t1: "2023-03-13 14", t2: "2023-03-13 14"}, want: false},
		{name: "006", args: args{layout: utils.HourDash, t1: "2023-03-13 14", t2: "2023-03-13 15"}, want: false},
		{name: "007", args: args{layout: utils.HourDash, t1: "2023-03-13 15", t2: "2023-03-13 14"}, want: true},
		{name: "008", args: args{layout: utils.SecondDash, t1: "2023-03-13 14:40:01", t2: "2023-03-13 14:40:01"}, want: false},
		{name: "009", args: args{layout: utils.SecondDash, t1: "2023-03-13 14:40:21", t2: "2023-03-13 14:40:01"}, want: true},
		{name: "010", args: args{layout: utils.SecondDash, t1: "2023-03-13 14:40:01", t2: "2023-03-13 14:40:09"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := utils.After(tt.args.layout, tt.args.t1, tt.args.t2); err != nil && got != tt.want {
				t.Errorf("After() = %v, want %v, WrapError %v", got, tt.want, err)
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
		{name: "001", args: args{layout: utils.DayDash, t1: "2023-03-13", t2: "2023-03-13"}, want: false}, // t1 等于 t2
		{name: "002", args: args{layout: utils.DayDash, t1: "2023-03-13", t2: "2023-03-14"}, want: true},  // t1 小于 t2
		{name: "003", args: args{layout: utils.DayDash, t1: "2023-03-13", t2: "2023-04-13"}, want: true},
		{name: "004", args: args{layout: utils.DayDash, t1: "2023-03-14", t2: "2023-03-13"}, want: false}, // t1 大于 t2
		{name: "005", args: args{layout: utils.HourDash, t1: "2023-03-13 14", t2: "2023-03-13 14"}, want: false},
		{name: "006", args: args{layout: utils.HourDash, t1: "2023-03-13 14", t2: "2023-03-13 15"}, want: true},
		{name: "007", args: args{layout: utils.HourDash, t1: "2023-03-13 15", t2: "2023-03-13 14"}, want: false},
		{name: "008", args: args{layout: utils.SecondDash, t1: "2023-03-13 14:40:01", t2: "2023-03-13 14:40:01"}, want: false},
		{name: "009", args: args{layout: utils.SecondDash, t1: "2023-03-13 14:40:21", t2: "2023-03-13 14:40:01"}, want: false},
		{name: "010", args: args{layout: utils.SecondDash, t1: "2023-03-13 14:40:01", t2: "2023-03-13 14:40:09"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := utils.Before(tt.args.layout, tt.args.t1, tt.args.t2); err != nil && got != tt.want {
				t.Errorf("Before() = %v, want %v, WrapError %v", got, tt.want, err)
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
		{name: "001", args: args{layout: utils.DayDash, t1: "2023-03-13", t2: "2023-03-13"}, want: 0},               // t1 > t2 结果等于 0
		{name: "002", args: args{layout: utils.DayDash, t1: "2023-03-13", t2: "2023-03-14"}, want: -86400000000000}, // t1 > t2 结果小于 0
		{name: "004", args: args{layout: utils.DayDash, t1: "2023-03-14", t2: "2023-03-13"}, want: 86400000000000},  // t1 > t2 结果大于 0
		{name: "005", args: args{layout: utils.SecondDash, t1: "2023-03-13 14:40:12", t2: "2023-03-13 14:40:01"}, want: 11000000000},
		{name: "006", args: args{layout: utils.NanosecondDash, t1: "2023-03-13 14:40:01.124685776", t2: "2023-03-13 14:40:01.124685076"}, want: 700},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := utils.Sub(tt.args.layout, tt.args.t1, tt.args.t2); err != nil && got.Nanoseconds() != tt.want {
				t.Errorf("Sub() = %v, want %v, WrapError %v", got, tt.want, err)
			}
		})
	}
}
