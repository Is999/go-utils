package utils_test

import (
	"os"
	"testing"

	"github.com/Is999/go-utils"
)

func TestGetEnv(t *testing.T) {
	type args struct {
		key        string
		defaultVal []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "001", args: args{key: "GCCGO", defaultVal: []string{"gccgo"}}, want: "gccgo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.GetEnv(tt.args.key, tt.args.defaultVal...); got != tt.want {
				t.Errorf("GetEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnv_ExistingKey(t *testing.T) {
	os.Setenv("GO_UTILS_TEST_KEY", "test_value")
	defer os.Unsetenv("GO_UTILS_TEST_KEY")

	got := utils.GetEnv("GO_UTILS_TEST_KEY", "default")
	if got != "test_value" {
		t.Errorf("GetEnv() = %v, want test_value", got)
	}
}

func TestGetEnv_EmptyValue(t *testing.T) {
	got := utils.GetEnv("GO_UTILS_NONEXISTENT_KEY_12345", "fallback")
	if got != "fallback" {
		t.Errorf("GetEnv() = %v, want fallback", got)
	}
}

func TestGetEnv_NoDefault(t *testing.T) {
	got := utils.GetEnv("GO_UTILS_NONEXISTENT_KEY_12345")
	if got != "" {
		t.Errorf("GetEnv() = %v, want empty string", got)
	}
}
