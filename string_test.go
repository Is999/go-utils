package utils_test

import (
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/Is999/go-utils"
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

// TestNewReplacer 验证可复用替换器会固定构造时的 map 规则，并支持 nil 接收者安全返回原字符串。
func TestNewReplacer(t *testing.T) {
	rules := map[string]string{
		"blue": "red",
		".":    "!",
	}
	replacer := utils.NewReplacer(rules)
	rules["blue"] = "green"

	got := replacer.Replace("blue car.")
	if got != "red car!" {
		t.Fatalf("Replacer.Replace() = %q, want %q", got, "red car!")
	}

	var nilReplacer *utils.Replacer
	if got = nilReplacer.Replace("keep"); got != "keep" {
		t.Fatalf("nil Replacer.Replace() = %q, want keep", got)
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

// TestSubstrASCII 覆盖纯 ASCII 快路径的正负索引边界，保证优化后仍保持历史截取语义。
func TestSubstrASCII(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		start  int
		length int
		want   string
	}{
		{name: "middle", value: "abcdef", start: 1, length: 3, want: "bcd"},
		{name: "negative_start", value: "abcdef", start: -3, length: 2, want: "de"},
		{name: "negative_end_last", value: "abcdef", start: 2, length: -1, want: "cdef"},
		{name: "negative_end_before_last", value: "abcdef", start: 2, length: -2, want: "cde"},
		{name: "start_before_head", value: "abcdef", start: -99, length: 3, want: "abc"},
		{name: "start_after_tail", value: "abcdef", start: 99, length: 3, want: ""},
		{name: "start_at_tail", value: "abcdef", start: 6, length: 1, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Substr(tt.value, tt.start, tt.length); got != tt.want {
				t.Fatalf("Substr() = %q, want %q", got, tt.want)
			}
		})
	}
}

// BenchmarkReplace 衡量兼容函数每次构造替换规则的成本，作为可复用替换器的对照组。
func BenchmarkReplace(b *testing.B) {
	rules := map[string]string{
		"blue": "red",
		".":    "!",
	}
	for i := 0; i < b.N; i++ {
		_ = utils.Replace("Mr Blue has a blue house and a blue car.", rules)
	}
}

// BenchmarkReplacerReuse 衡量同一批规则反复替换时的复用收益。
func BenchmarkReplacerReuse(b *testing.B) {
	replacer := utils.NewReplacer(map[string]string{
		"blue": "red",
		".":    "!",
	})
	for i := 0; i < b.N; i++ {
		_ = replacer.Replace("Mr Blue has a blue house and a blue car.")
	}
}

// BenchmarkSubstrASCII 衡量纯 ASCII 字符串截取的零分配快路径。
func BenchmarkSubstrASCII(b *testing.B) {
	value := "And know that you do not have to be perfect, you can be good."
	for i := 0; i < b.N; i++ {
		_ = utils.Substr(value, 4, 32)
	}
}

// BenchmarkSubstrUnicode 衡量非 ASCII 字符串继续按 rune 截取的兼容路径。
func BenchmarkSubstrUnicode(b *testing.B) {
	value := "你可以很好，但你无需完美。"
	for i := 0; i < b.N; i++ {
		_ = utils.Substr(value, 2, 8)
	}
}

func TestReverseString(t *testing.T) {
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
			if got := utils.ReverseString(tt.args.str); got != tt.want {
				t.Errorf("ReverseString() = %v, want %v", got, tt.want)
			} else {
				// t.Logf("ReverseString() = %v|%v", tt.args.str, got)
			}
		})
	}
}

// go test -fuzz=ReverseString -fuzztime 10s
func FuzzReverseString(f *testing.F) {
	testcases := []string{"Hello, world", " ", "!12345", "反转一个字符串\"A￥%&cd=L8217\""}
	for _, tc := range testcases {
		f.Add(tc) // Use f.Add to provide a seed corpus
	}
	f.Fuzz(func(t *testing.T, orig string) {
		if !utf8.ValidString(orig) {
			return
		}
		rev := utils.ReverseString(orig)

		doubleRev := utils.ReverseString(rev)
		if orig != doubleRev {
			t.Errorf("Before: %q, after: %q", orig, doubleRev)
		}
		if !utf8.ValidString(rev) {
			t.Errorf("Reverse produced invalid UTF-8 string %q", rev)
		}
	})
}

func TestRandomLetters(t *testing.T) {
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
			if got := utils.RandomLetters(tt.args.n); len(got) != tt.args.n {
				t.Errorf("RandomLetters() = %v, wantSize %v", got, tt.args.n)
			} else {
				//t.Logf("RandomLetters() = %v, size %v", got, tt.args.n)
			}
		})
	}
}

func TestRandomID(t *testing.T) {
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
	r := utils.RandSource
	wg := &sync.WaitGroup{}
	for _, tt := range tests {
		wg.Add(1)
		go t.Run(tt.name, func(t *testing.T) {
			defer wg.Done()
			if got := utils.RandomID(tt.args.n, r); len(got) != tt.args.n {
				t.Errorf("%v RandomID() = %v, size=%v, wantSize %v", tt.name, got, len(got), tt.args.n)
			} else {
				// t.Logf("%v RandomID() = %v, size %v", tt.name, got, tt.args.n)
			}
		})
	}
	wg.Wait()
}

// go test -bench=RandomID$ -run ^$  -count 5 -benchmem
func BenchmarkRandomID(b *testing.B) {
	type args struct {
		n int
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "001", args: args{10}},
	}
	r := utils.RandSource
	b.ResetTimer()
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if got := utils.RandomID(tt.args.n, r); len(got) != tt.args.n {
					b.Errorf("RandomID() = %v, wantSize %v", got, tt.args.n)
				}

				/*if got := RandomID(tt.args.n); len(got) != tt.args.n {
					b.Errorf("RandomID() = %v, wantSize %v", got, tt.args.n)
				}*/
			}
		})
	}
}

func TestRandomString(t *testing.T) {
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
			if got := utils.RandomString(tt.args.n, tt.args.alpha); len(got) != tt.want {
				t.Errorf("RandomString() = %v, lenth %v, wantSize %v", got, tt.args.n, tt.want)
			} else {
				//t.Logf("RandomString() = %v, size %v", got, tt.args.n)
			}
		})
	}
}

func TestSecureRandomLetters(t *testing.T) {
	got, err := utils.SecureRandomLetters(32)
	if err != nil {
		t.Fatalf("SecureRandomLetters() error = %v", err)
	}
	if len(got) != 32 {
		t.Fatalf("SecureRandomLetters() len = %d, want 32", len(got))
	}
	if !allCharsInAlphabet(got, utils.ALPHA) {
		t.Fatalf("SecureRandomLetters() = %q, want chars in %q", got, utils.ALPHA)
	}
}

func TestSecureRandomID(t *testing.T) {
	got, err := utils.SecureRandomID(32)
	if err != nil {
		t.Fatalf("SecureRandomID() error = %v", err)
	}
	if len(got) != 32 {
		t.Fatalf("SecureRandomID() len = %d, want 32", len(got))
	}
	if !allCharsInAlphabet(got[:1], utils.ALPHA) {
		t.Fatalf("SecureRandomID() first char = %q, want alpha", got[:1])
	}
	if !allCharsInAlphabet(got[1:], utils.ALNUM) {
		t.Fatalf("SecureRandomID() tail = %q, want alnum", got[1:])
	}
}

func TestSecureRandomString(t *testing.T) {
	got, err := utils.SecureRandomString(24, "abc123")
	if err != nil {
		t.Fatalf("SecureRandomString() error = %v", err)
	}
	if len(got) != 24 {
		t.Fatalf("SecureRandomString() len = %d, want 24", len(got))
	}
	if !allCharsInAlphabet(got, "abc123") {
		t.Fatalf("SecureRandomString() = %q, want chars in %q", got, "abc123")
	}
}

func TestSecureUniqueID(t *testing.T) {
	got, err := utils.SecureUniqueID(8)
	if err != nil {
		t.Fatalf("SecureUniqueID() error = %v", err)
	}
	if len(got) != 16 {
		t.Fatalf("SecureUniqueID() len = %d, want 16", len(got))
	}
	if !allCharsInAlphabet(got[:1], utils.ALPHA) || !allCharsInAlphabet(got[1:], utils.ALNUM) {
		t.Fatalf("SecureUniqueID() = %q, want leading alpha and tail alnum", got)
	}
}

func TestUniqueID(t *testing.T) {
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

	r := utils.RandSource
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utils.UniqueID(tt.args.l, r)
		})
	}
}

// go test -bench=UniqueID$ -run ^$  -count 5 -benchmem
func BenchmarkUniqueID(t *testing.B) {
	type args struct {
		l uint8
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "018", args: args{32}},
	}
	r := utils.RandSource
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.B) {
			for n := 0; n < t.N; n++ {
				utils.UniqueID(tt.args.l, r)
			}
		})
	}
}

func allCharsInAlphabet(s, alpha string) bool {
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(alpha, rune(s[i])) {
			return false
		}
	}
	return true
}
