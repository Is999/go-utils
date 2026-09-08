package utils_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"testing"

	"github.com/Is999/go-utils"
)

// TestConfigureDefaults 验证空调用和无效选项都会消耗首次配置机会，后续调用不能覆盖默认值。
func TestConfigureDefaults(t *testing.T) {
	for name, opts := range map[string][]utils.Option{
		"empty": nil,
		"ignored options": {
			nil, utils.WithJSON(nil, nil), utils.WithJSON(json.Marshal, nil),
			utils.WithJSON(nil, json.Unmarshal), utils.WithLogger(nil),
		},
	} {
		t.Run(name, func(t *testing.T) {
			runConfigTest(t, func(t *testing.T) {
				utils.Configure(opts...)
				logger := utils.Log() // 首次配置发布的实例应一直保留。
				utils.Configure(utils.WithJSON(func(any) ([]byte, error) {
					return []byte("replaced"), nil
				}, json.Unmarshal), utils.WithLogger(&slogAdapter{slog.Default()}))

				got, err := utils.Marshal("value")
				if err != nil || string(got) != `"value"` {
					t.Fatalf("Marshal() = (%q, %v), want standard JSON", got, err)
				}
				var decoded string
				if err := utils.Unmarshal([]byte(`"value"`), &decoded); err != nil || decoded != "value" {
					t.Fatalf("Unmarshal() = (%q, %v), want value", decoded, err)
				}
				response := utils.Response{Body: utils.Body{Code: 42, Data: "value"}}
				want, err := json.Marshal(response.Body)
				if err != nil {
					t.Fatal(err)
				}
				got, err = response.Encode()
				if err != nil || string(got) != string(want) {
					t.Fatalf("Response.Encode() = (%q, %v), want %q", got, err, want)
				}
				if logger == nil || utils.Log() != logger {
					t.Fatal("later Configure replaced the default logger")
				}
			})
		})
	}
}

// TestConfigureOptions 验证选项顺序、成对编解码器及自定义错误的原样传播。
func TestConfigureOptions(t *testing.T) {
	runConfigTest(t, func(t *testing.T) {
		cause := errors.New("custom codec failure")
		encode := func(any) ([]byte, error) { return []byte("partial"), cause }
		decode := func(_ []byte, target any) error {
			*target.(*string) = "partial"
			return cause
		}
		logger := &slogAdapter{slog.Default()} // 独立适配器用于核对选项顺序，日志实现复用现有示例。
		utils.Configure(
			utils.WithJSON(json.Marshal, json.Unmarshal), utils.WithLogger(&slogAdapter{slog.Default()}),
			utils.WithJSON(encode, decode), utils.WithLogger(logger),
			utils.WithJSON(nil, nil), utils.WithJSON(json.Marshal, nil),
			utils.WithJSON(nil, json.Unmarshal), utils.WithLogger(nil), nil,
		)
		utils.Configure(utils.WithJSON(json.Marshal, json.Unmarshal), utils.WithLogger(&slogAdapter{slog.Default()}))

		// 这里要求同一个错误对象，使用 Is 会漏掉意外增加的包装层。
		got, err := utils.Marshal(nil)
		if err != cause || string(got) != "partial" {
			t.Fatalf("Marshal() = (%q, %v), want partial result and original error", got, err)
		}
		var decoded string
		if err := utils.Unmarshal(nil, &decoded); err != cause || decoded != "partial" {
			t.Fatalf("Unmarshal() = (%q, %v), want partial result and original error", decoded, err)
		}
		got, err = (&utils.Response{}).Encode()
		if err != cause || string(got) != "partial" {
			t.Fatalf("Response.Encode() = (%q, %v), want partial result and original error", got, err)
		}
		if utils.Log() != logger {
			t.Fatal("Log() did not use the last valid logger option")
		}
	})
}

// runConfigTest 在独立进程执行配置断言，避免 Configure 的一次性状态影响其他测试。
func runConfigTest(t *testing.T, test func(*testing.T)) {
	t.Helper()
	const childEnv = "GO_UTILS_CONFIG_TEST" // 子进程只执行指定用例，防止递归启动测试。
	if os.Getenv(childEnv) == t.Name() {
		test(t)
		return
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), binary, "-test.run=^"+regexp.QuoteMeta(t.Name())+"$")
	cmd.Env = append(os.Environ(), childEnv+"="+t.Name())
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("config subprocess: %v\n%s", err, output)
	}
}

func TestMarshal(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name string
		data any
		want string // 默认编码器的完整 JSON 输出。
	}{
		{name: "struct", data: testStruct{Name: "test", Age: 18}, want: `{"name":"test","age":18}`},
		{name: "map", data: map[string]any{"key": "value"}, want: `{"key":"value"}`},
		{name: "slice", data: []int{1, 2, 3}, want: `[1,2,3]`},
		{name: "string", data: "string", want: `"string"`},
		{name: "number", data: 123, want: `123`},
		{name: "nil", data: nil, want: `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.Marshal(tt.data)
			if err != nil || string(got) != tt.want {
				t.Fatalf("Marshal() = (%q, %v), want %q", got, err, tt.want)
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name    string
		data    string
		target  any
		want    any // 默认解码器应写入的完整目标值。
		wantErr bool
	}{
		{name: "struct", data: `{"name":"test","age":18}`, target: &testStruct{}, want: &testStruct{Name: "test", Age: 18}},
		{name: "map", data: `{"key":"value"}`, target: &map[string]any{}, want: &map[string]any{"key": "value"}},
		{name: "slice", data: `[1,2,3]`, target: &[]int{}, want: &[]int{1, 2, 3}},
		{name: "invalid JSON", data: `invalid json`, target: &testStruct{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.Unmarshal([]byte(tt.data), tt.target)
			if tt.wantErr {
				if _, ok := errors.AsType[*json.SyntaxError](err); !ok {
					t.Fatalf("Unmarshal() error = %v, want json.SyntaxError", err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(tt.target, tt.want) {
				t.Fatalf("Unmarshal() target = %#v, error = %v; want %#v", tt.target, err, tt.want)
			}
		})
	}
}
