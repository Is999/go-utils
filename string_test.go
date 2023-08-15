package utils_test

import (
	"github.com/Is999/go-utils"
	"sync"
	"testing"
	"unicode/utf8"
)

func TestReplace(t *testing.T) {
	type args struct {
		str   string
		pairs map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "001", args: args{str: "Mr Blue has a blue house and a blue car.", pairs: map[string]string{
			"blue": " red",
			".":    "!",
		}}, want: "Mr Blue has a  red house and a  red car!"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Replace(tt.args.str, tt.args.pairs); got != tt.want {
				t.Errorf("Replace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubstr(t *testing.T) {
	type args struct {
		str    string
		start  int
		length int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "001", args: args{str: "And know that you don’t have to be perfect, you can be good.\n\n你可以很好，但你无需完美。", start: 0, length: 60}, want: "And know that you don’t have to be perfect, you can be good."},
		{name: "002", args: args{str: "And know that you don’t have to be perfect, you can be good.\n\n你可以很好，但你无需完美。", start: 62, length: 12}, want: "你可以很好，但你无需完美"},
		{name: "003", args: args{str: "And know that you don’t have to be perfect, you can be good.\n\n你可以很好，但你无需完美。", start: -13, length: 12}, want: "你可以很好，但你无需完美"},
		{name: "004", args: args{str: "And know that you don’t have to be perfect, you can be good.\n\n你可以很好，但你无需完美。", start: 62, length: -2}, want: "你可以很好，但你无需完美"},
		{name: "005", args: args{str: "And know that you don’t have to be perfect, you can be good.\n\n你可以很好，但你无需完美。", start: -13, length: -2}, want: "你可以很好，但你无需完美"},
		{name: "006", args: args{str: "And know that you don’t have to be perfect, you can be good.\n\n你可以很好，但你无需完美。", start: 62, length: -1}, want: "你可以很好，但你无需完美。"}, // 截取到最后一位
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Substr(tt.args.str, tt.args.start, tt.args.length); got != tt.want {
				t.Errorf("Substr() = %d %v, want %d %v", utf8.RuneCountInString(got), got, utf8.RuneCountInString(tt.want), tt.want)
			}
		})
	}
}

func TestStrRev(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "001", args: args{"大A和小a，是否相同？"}, want: "？同相否是，a小和A大"},
		{name: "002", args: args{"反转一个字符串\"A￥%&cd=L8217\""}, want: "\"7128L=dc&%￥A\"串符字个一转反"},
		{name: "003", args: args{"Golang"}, want: "gnaloG"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !utf8.ValidString(tt.args.str) {
				t.Errorf("str[%q] is not valid UTF-8", tt.args.str)
				return
			}
			if got := utils.StrRev(tt.args.str); got != tt.want {
				t.Errorf("StrRev() = %v, want %v", got, tt.want)
			} else {
				// t.Logf("StrRev() = %v|%v", tt.args.str, got)
			}
		})
	}
}

// go test -fuzz=StrRev -fuzztime 10s
func FuzzStrRev(f *testing.F) {
	testcases := []string{"Hello, world", " ", "!12345", "反转一个字符串\"A￥%&cd=L8217\""}
	for _, tc := range testcases {
		f.Add(tc) // Use f.Add to provide a seed corpus
	}
	f.Fuzz(func(t *testing.T, orig string) {
		if !utf8.ValidString(orig) {
			return
		}
		rev := utils.StrRev(orig)

		doubleRev := utils.StrRev(rev)
		if orig != doubleRev {
			t.Errorf("Before: %q, after: %q", orig, doubleRev)
		}
		if !utf8.ValidString(rev) {
			t.Errorf("Reverse produced invalid UTF-8 string %q", rev)
		}
	})
}

func TestRandStr(t *testing.T) {
	type args struct {
		n int // 长度
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "001", args: args{0}},
		{name: "002", args: args{1}},
		{name: "003", args: args{2}},
		{name: "004", args: args{10}},
		{name: "005", args: args{20}},
	}
	// r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.RandStr(tt.args.n); len(got) != tt.args.n {
				t.Errorf("RandStr() = %v, wantSize %v", got, tt.args.n)
			} else {
				//t.Logf("RandStr() = %v, size %v", got, tt.args.n)
			}
		})
	}
}

func TestRandStr2(t *testing.T) {
	type args struct {
		n int // 长度
	}
	type test struct {
		name string
		args args
	}
	tests := []test{
		{name: "001", args: args{0}},
		{name: "002", args: args{1}},
		{name: "003", args: args{2}},
		{name: "004", args: args{10}},
		{name: "005", args: args{20}},
		{name: "006", args: args{20}},
		{name: "007", args: args{20}},
		{name: "008", args: args{20}},
		{name: "009", args: args{20}},
		{name: "010", args: args{20}},
		{name: "011", args: args{20}},
	}
	r := utils.Source()
	wg := &sync.WaitGroup{}
	for _, tt := range tests {
		wg.Add(1)
		go func(tt test, t *testing.T) {
			defer wg.Done()
			t.Run(tt.name, func(t *testing.T) {
				if got := utils.RandStr2(tt.args.n, r); len(got) != tt.args.n {
					t.Errorf("RandStr2() = %v, wantSize %v", got, tt.args.n)
				} else {
					// t.Logf("RandStr2() = %v, size %v", got, tt.args.n)
				}
			})
		}(tt, t)
	}
	wg.Wait()
}

// go test -bench=RandStr2$ -run ^$  -count 5 -benchmem
func BenchmarkRandStr2(b *testing.B) {
	type args struct {
		n int
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "001", args: args{10}},
	}
	r := utils.Source()
	b.ResetTimer()
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if got := utils.RandStr2(tt.args.n, r); len(got) != tt.args.n {
					b.Errorf("RandStr2() = %v, wantSize %v", got, tt.args.n)
				}

				/*if got := RandStr2(tt.args.n); len(got) != tt.args.n {
					b.Errorf("RandStr2() = %v, wantSize %v", got, tt.args.n)
				}*/
			}
		})
	}
}

func TestRandStr3(t *testing.T) {
	type args struct {
		n     int
		alpha string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "001", args: args{0, "0123456789abcdefghijklmn"}, want: 0},
		{name: "002", args: args{1, "0123456789abcdefghijklmn"}, want: 1},
		{name: "003", args: args{2, "0123456789abcdefghijklmn"}, want: 2},
		{name: "004", args: args{10, "0123456789abcdefghijklmn"}, want: 10},
		{name: "005", args: args{20, "0123456789abcdefghijklmn"}, want: 20},
		{name: "006", args: args{10, ""}, want: 0},
		{name: "007", args: args{10, "a"}, want: 10},
		{name: "008", args: args{10, "ab"}, want: 10},
		{name: "009", args: args{10, "abc"}, want: 10},
	}
	// r := Source()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.RandStr3(tt.args.n, tt.args.alpha); len(got) != tt.want {
				t.Errorf("RandStr3() = %v, lenth %v, wantSize %v", got, tt.args.n, tt.want)
			} else {
				//t.Logf("RandStr3() = %v, size %v", got, tt.args.n)
			}
		})
	}
}

func TestUniqId(t *testing.T) {
	type args struct {
		l uint8
	}
	type test struct {
		name string
		args args
	}
	tests := []test{
		{name: "001", args: args{11}},
		{name: "002", args: args{16}},
		{name: "003", args: args{17}},
		{name: "004", args: args{18}},
		{name: "005", args: args{19}},
		{name: "006", args: args{20}},
		{name: "007", args: args{21}},
		{name: "008", args: args{22}},
		{name: "009", args: args{23}},
		{name: "010", args: args{24}},
		{name: "011", args: args{25}},
		{name: "012", args: args{26}},
		{name: "013", args: args{27}},
		{name: "014", args: args{28}},
		{name: "015", args: args{29}},
		{name: "016", args: args{30}},
		{name: "017", args: args{31}},
		{name: "018", args: args{32}},
		{name: "019", args: args{33}},
	}

	r := utils.Source()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utils.UniqId(tt.args.l, r)
		})
	}
}

// go test -bench=UniqId$ -run ^$  -count 5 -benchmem
func BenchmarkUniqId(t *testing.B) {
	type args struct {
		l uint8
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "018", args: args{32}},
	}
	r := utils.Source()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.B) {
			for n := 0; n < t.N; n++ {
				utils.UniqId(tt.args.l, r)
			}
		})
	}
}
