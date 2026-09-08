package utils_test

import (
	"os"
	"testing"

	"github.com/Is999/go-utils"
)

func TestGetEnv(t *testing.T) {
	const key = "GO_UTILS_TEST_KEY" // 测试只改专用变量，结束后恢复调用环境。
	for _, tt := range []struct {
		name       string
		value      string
		unset      bool // 区分未设置与空字符串，两者都应使用默认值。
		defaultVal []string
		want       string
	}{
		{name: "unset", unset: true, defaultVal: []string{"fallback"}, want: "fallback"},
		{name: "empty", defaultVal: []string{"fallback"}, want: "fallback"},
		{name: "existing", value: "value", defaultVal: []string{"fallback"}, want: "value"},
		{name: "no default", unset: true},
		{name: "first default", defaultVal: []string{"first", "second"}, want: "first"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(key, tt.value)
			if tt.unset {
				if err := os.Unsetenv(key); err != nil {
					t.Fatal(err)
				}
			}
			if got := utils.GetEnv(key, tt.defaultVal...); got != tt.want {
				t.Errorf("GetEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
