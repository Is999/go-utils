package utils_test

import (
	"net"
	"net/http"
	"testing"

	"github.com/Is999/go-utils"
)

// benchmarkClientIP 保存 ClientIP 基准测试结果，避免编译器消除解析调用。
var benchmarkClientIP string

func TestServerIP(t *testing.T) {
	tests := []struct {
		name   string
		wantIp string
	}{
		{name: "001", wantIp: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIp := utils.ServerIP()
			if tt.wantIp != "" && gotIp != tt.wantIp {
				t.Errorf("ServerIP() gotIp = %v, want %v", gotIp, tt.wantIp)
			} else {
				t.Logf("ServerIP() gotIp = %v", gotIp)
			}
		})
	}
}

func TestLocalIP(t *testing.T) {
	tests := []struct {
		name   string
		wantIp string
	}{
		{name: "001", wantIp: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIp := utils.LocalIP()
			if tt.wantIp != "" && gotIp != tt.wantIp {
				t.Errorf("LocalIP() gotIp = %v, want %v", gotIp, tt.wantIp)
			} else {
				t.Logf("LocalIP() gotIp = %v", gotIp)
			}
		})
	}
}

func TestClientIP(t *testing.T) {
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "001", args: args{
			req: &http.Request{
				RemoteAddr: "127.0.0.1:80",
			},
		}, want: "127.0.0.1"},
		{name: "002", args: args{
			req: &http.Request{
				Header: http.Header{
					"X-Real-Ip": []string{"192.168.47.142"},
				},
				RemoteAddr: "127.0.0.1:80",
			},
		}, want: "192.168.47.142"},
		{name: "003", args: args{
			req: &http.Request{
				Header: http.Header{
					"X-Real-Ip":       []string{"192.168.47.141"},
					"X-Forwarded-For": []string{"192.168.47.141, 192.168.47.142, 175.176.32.112, 192.168.47.143"},
				},
				RemoteAddr: "127.0.0.1:80",
			},
		}, want: "192.168.47.143"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.ClientIP(tt.args.req); got != tt.want {
				t.Errorf("ClientIp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientIPRejectsSpoofedForwardHeader(t *testing.T) {
	req := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"175.176.32.112"},
		},
		RemoteAddr: "8.8.8.8:443",
	}
	if got := utils.ClientIP(req); got != "8.8.8.8" {
		t.Fatalf("ClientIP() = %v, want %v", got, "8.8.8.8")
	}
}

func TestClientIPDoesNotTrustPrivateRemoteByDefault(t *testing.T) {
	req := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"203.0.113.10"},
			"X-Real-Ip":       []string{"198.51.100.8"},
		},
		RemoteAddr: "10.0.0.10:443",
	}
	if got := utils.ClientIP(req); got != "10.0.0.10" {
		t.Fatalf("ClientIP() = %v, want %v", got, "10.0.0.10")
	}
}

func TestClientIPIgnoresInvalidProxyHeader(t *testing.T) {
	req := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"not-an-ip"},
			"X-Real-Ip":       []string{"invalid-ip"},
		},
		RemoteAddr: "127.0.0.1:8080",
	}
	if got := utils.ClientIP(req); got != "127.0.0.1" {
		t.Fatalf("ClientIP() = %v, want %v", got, "127.0.0.1")
	}
}

func TestNewTrustedProxies(t *testing.T) {
	proxies, err := utils.NewTrustedProxies("10.0.0.10", "10.0.0.0/8", "fd00::/8")
	if err != nil {
		t.Fatalf("NewTrustedProxies() error = %v", err)
	}

	for _, tt := range []struct {
		ip   string
		want bool
	}{
		{ip: "10.0.0.10", want: true},
		{ip: "10.1.1.1", want: true},
		{ip: "fd00::1", want: true},
		{ip: "192.168.1.1", want: false},
	} {
		if got := proxies.Contains(net.ParseIP(tt.ip)); got != tt.want {
			t.Fatalf("TrustedProxies.Contains(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestNewTrustedProxiesRejectsInvalidValue(t *testing.T) {
	if _, err := utils.NewTrustedProxies("bad-value"); err == nil {
		t.Fatal("NewTrustedProxies() expected error")
	}
}

func TestClientIPWithTrustedProxies(t *testing.T) {
	proxies, err := utils.NewTrustedProxies("10.0.0.0/8", "192.168.0.0/16")
	if err != nil {
		t.Fatalf("NewTrustedProxies() error = %v", err)
	}

	req := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"203.0.113.10, 10.1.1.1, 192.168.1.1"},
		},
		RemoteAddr: "10.0.0.10:443",
	}
	if got := utils.ClientIPWithTrustedProxies(req, proxies); got != "203.0.113.10" {
		t.Fatalf("ClientIPWithTrustedProxies() = %v, want %v", got, "203.0.113.10")
	}
}

func TestClientIPWithTrustedProxiesIgnoresHeaderWhenRemoteUntrusted(t *testing.T) {
	proxies, err := utils.NewTrustedProxies("10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewTrustedProxies() error = %v", err)
	}

	req := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"203.0.113.10"},
			"X-Real-Ip":       []string{"198.51.100.8"},
		},
		RemoteAddr: "8.8.8.8:443",
	}
	if got := utils.ClientIPWithTrustedProxies(req, proxies); got != "8.8.8.8" {
		t.Fatalf("ClientIPWithTrustedProxies() = %v, want %v", got, "8.8.8.8")
	}
}

func TestClientIPWithTrustedProxiesFallsBackToXRealIP(t *testing.T) {
	proxies, err := utils.NewTrustedProxies("10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewTrustedProxies() error = %v", err)
	}

	req := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"invalid-ip"},
			"X-Real-Ip":       []string{"203.0.113.11"},
		},
		RemoteAddr: "10.0.0.10:443",
	}
	if got := utils.ClientIPWithTrustedProxies(req, proxies); got != "203.0.113.11" {
		t.Fatalf("ClientIPWithTrustedProxies() = %v, want %v", got, "203.0.113.11")
	}
}

// BenchmarkClientIPRemoteOnly 覆盖未信任转发头的普通请求。
// 数据来源只有 RemoteAddr，应直接返回远端地址，不扫描 Header。
func BenchmarkClientIPRemoteOnly(b *testing.B) {
	req := &http.Request{RemoteAddr: "8.8.8.8:443"} // req 模拟公网直连请求，默认策略不信任任何转发头。
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchmarkClientIP = utils.ClientIP(req)
	}
}

// BenchmarkClientIPLoopbackForwardedChain 覆盖默认可信回环代理下的 X-Forwarded-For 链。
// 数据来源是多级转发头，期望从右向左找到第一个非可信代理地址。
func BenchmarkClientIPLoopbackForwardedChain(b *testing.B) {
	req := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"192.168.47.141, 192.168.47.142, 175.176.32.112, 192.168.47.143"},
		},
		RemoteAddr: "127.0.0.1:80",
	} // req 模拟本地反向代理转发请求，默认只信任回环 RemoteAddr。
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchmarkClientIP = utils.ClientIP(req)
	}
}

// BenchmarkClientIPTrustedProxies 覆盖显式可信代理白名单场景。
// 数据来源是业务配置的 CIDR 列表和 X-Forwarded-For 链，需跳过链尾可信代理。
func BenchmarkClientIPTrustedProxies(b *testing.B) {
	proxies, err := utils.NewTrustedProxies("10.0.0.0/8", "192.168.0.0/16") // proxies 表示业务侧显式信任的网关网段。
	if err != nil {
		b.Fatal(err)
	}
	req := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"203.0.113.10, 10.1.1.1, 192.168.1.1"},
		},
		RemoteAddr: "10.0.0.10:443",
	} // req 模拟请求经过多层可信网关后进入服务。
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchmarkClientIP = utils.ClientIPWithTrustedProxies(req, proxies)
	}
}

// BenchmarkClientIPInvalidForwardFallback 覆盖转发头非法时的降级策略。
// X-Forwarded-For 全部非法时，应退回 X-Real-Ip，避免把不可解析数据作为客户端地址。
func BenchmarkClientIPInvalidForwardFallback(b *testing.B) {
	proxies, err := utils.NewTrustedProxies("10.0.0.0/8") // proxies 表示仅信任内网网关来源。
	if err != nil {
		b.Fatal(err)
	}
	req := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"invalid-ip"},
			"X-Real-Ip":       []string{"203.0.113.11"},
		},
		RemoteAddr: "10.0.0.10:443",
	} // req 模拟上游转发头脏数据，验证降级到 X-Real-Ip。
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchmarkClientIP = utils.ClientIPWithTrustedProxies(req, proxies)
	}
}
