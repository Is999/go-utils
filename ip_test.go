package utils_test

import (
	"net"
	"net/http"
	"testing"

	"github.com/Is999/go-utils"
)

// benchmarkClientIP 保存 ClientIP 基准测试结果，避免编译器消除解析调用。
var benchmarkClientIP string

// TestServerIP 不固定本机网络地址，只校验非空结果是合法 IP。
func TestServerIP(t *testing.T) {
	if got := utils.ServerIP(); got != "" && net.ParseIP(got) == nil {
		t.Fatalf("ServerIP() = %q, want IP or empty", got)
	}
}

// TestLocalIP 无可用网卡或主机名记录时允许空结果，非空结果必须是 IPv4。
func TestLocalIP(t *testing.T) {
	if got := utils.LocalIP(); got != "" && net.ParseIP(got).To4() == nil {
		t.Fatalf("LocalIP() = %q, want IPv4 or empty", got)
	}
}

// TestClientIP 验证普通提取不依赖直连来源的信任规则，并按头优先级与链中顺序返回地址。
func TestClientIP(t *testing.T) {
	// 用不同连接来源、转发链和无效输入覆盖代理后的真实调用方式。
	tests := []struct {
		name   string   // 场景名称，标明本例覆盖的来源或回退规则。
		remote string   // 连接来源，允许缺失或非法以验证头部独立解析。
		xff    []string // 按接收顺序提供的 XFF 头，每项可包含多个代理节点。
		realIP []string // XFF 不可用时读取的 X-Real-IP 头。
		want   string   // 期望提取并规范化后的客户端地址。
	}{
		{name: "empty request"},
		{name: "IPv4 connection", remote: "203.0.113.10:443", want: "203.0.113.10"},
		{name: "IPv6 connection", remote: "[2001:db8::10]:443", want: "2001:db8::10"},
		{name: "loopback connection", remote: "127.0.0.1:80", want: "127.0.0.1"},
		{name: "invalid connection", remote: "not-an-address"},
		{name: "private proxy XFF", remote: "10.0.0.10:443", xff: []string{"203.0.113.10"}, want: "203.0.113.10"},
		{name: "public proxy XFF", remote: "8.8.8.8:443", xff: []string{"175.176.32.112"}, want: "175.176.32.112"},
		{name: "private proxy real IP", remote: "192.168.1.1:443", realIP: []string{"203.0.113.10"}, want: "203.0.113.10"},
		{name: "loopback proxy real IP", remote: "127.0.0.1:80", realIP: []string{"192.168.47.142"}, want: "192.168.47.142"},
		{name: "XFF before real IP", remote: "10.0.0.10:443", xff: []string{"203.0.113.10"}, realIP: []string{"198.51.100.8"}, want: "203.0.113.10"},
		{name: "client before proxies", remote: "10.0.0.10:443", xff: []string{"203.0.113.10, 192.168.1.1, 10.1.1.1"}, want: "203.0.113.10"},
		{name: "private client retained", remote: "127.0.0.1:80", xff: []string{"192.168.47.141, 175.176.32.112, 192.168.47.143"}, want: "192.168.47.141"},
		{name: "multiple XFF headers", remote: "10.0.0.10:443", xff: []string{"203.0.113.10", "198.51.100.8, 10.1.1.1"}, want: "203.0.113.10"},
		{name: "invalid XFF header before valid", remote: "10.0.0.10:443", xff: []string{"invalid-ip", "203.0.113.10"}, want: "203.0.113.10"},
		{name: "invalid XFF nodes skipped", remote: "10.0.0.10:443", xff: []string{" , unknown, 203.0.113.10 , 10.1.1.1"}, want: "203.0.113.10"},
		{name: "IPv6 forwarded client", remote: "10.0.0.10:443", xff: []string{"2001:db8::10, 10.1.1.1"}, want: "2001:db8::10"},
		{name: "forwarded IPv6 port", remote: "10.0.0.10:443", xff: []string{"[2001:db8::10]:8443"}, want: "2001:db8::10"},
		{name: "mapped IPv4 forwarded client", remote: "10.0.0.10:443", xff: []string{"::ffff:203.0.113.10"}, want: "203.0.113.10"},
		{name: "IPv4 real IP port", remote: "10.0.0.10:443", realIP: []string{"203.0.113.10:8443"}, want: "203.0.113.10"},
		{name: "invalid XFF falls back to real IP", remote: "10.0.0.10:443", xff: []string{"unknown"}, realIP: []string{"203.0.113.10"}, want: "203.0.113.10"},
		{name: "invalid real IP header before valid", remote: "10.0.0.10:443", realIP: []string{"unknown", "203.0.113.10"}, want: "203.0.113.10"},
		{name: "headers fall back to connection", remote: "127.0.0.1:8080", xff: []string{"not-an-ip"}, realIP: []string{"invalid-ip"}, want: "127.0.0.1"},
		{name: "all addresses invalid", remote: "not-an-address", xff: []string{"unknown"}, realIP: []string{"invalid-ip"}},
		{name: "XFF without connection", xff: []string{"203.0.113.10"}, want: "203.0.113.10"},
		{name: "XFF with invalid connection", remote: "not-an-address", xff: []string{"203.0.113.10"}, want: "203.0.113.10"},
		{name: "real IP without connection", realIP: []string{"203.0.113.10"}, want: "203.0.113.10"},
	}

	// 经过公开入口断言结果，防止连接来源再次阻断头部提取。
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				RemoteAddr: tt.remote,
				Header: http.Header{
					"X-Forwarded-For": tt.xff,
					"X-Real-Ip":       tt.realIP,
				},
			}
			if got := utils.ClientIP(req); got != tt.want {
				t.Fatalf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}

	// nil 请求与空请求都没有地址，不能因读取头部而触发 panic。
	if got := utils.ClientIP(nil); got != "" {
		t.Fatalf("ClientIP(nil) = %q, want empty", got)
	}
}

// TestClientIPWithTrustedProxiesWithoutRules 保留显式安全入口对伪造头部的拒绝规则。
func TestClientIPWithTrustedProxiesWithoutRules(t *testing.T) {
	// 空名单与 nil 名单均不信任代理，包含回环、内网和公网连接来源。
	emptyProxies, err := utils.NewTrustedProxies()
	if err != nil {
		t.Fatal(err)
	}
	for _, proxies := range []*utils.TrustedProxies{nil, emptyProxies} {
		for _, remote := range []string{"127.0.0.1", "10.0.0.10", "8.8.8.8"} {
			req := &http.Request{
				RemoteAddr: net.JoinHostPort(remote, "443"),
				Header: http.Header{
					"X-Forwarded-For": {"175.176.32.112"},
					"X-Real-Ip":       {"198.51.100.8"},
				},
			}
			if got := utils.ClientIPWithTrustedProxies(req, proxies); got != remote {
				t.Fatalf("ClientIPWithTrustedProxies() = %q, want %q", got, remote)
			}
		}
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

func TestTrustedProxiesMappedCIDR(t *testing.T) {
	// IPv4 映射前缀与对应 IPv4 网段等价，更宽的 IPv6 前缀仍只匹配 IPv6。
	for _, tt := range []struct {
		name string
		rule string
		addr string
		want bool
	}{
		{name: "subnet first", rule: "::ffff:192.0.2.129/120", addr: "192.0.2.0", want: true},
		{name: "subnet last", rule: "::ffff:192.0.2.129/120", addr: "192.0.2.255", want: true},
		{name: "mapped address", rule: "::ffff:192.0.2.129/120", addr: "::ffff:192.0.2.1", want: true},
		{name: "before subnet", rule: "::ffff:192.0.2.129/120", addr: "192.0.1.255"},
		{name: "after subnet", rule: "::ffff:192.0.2.129/120", addr: "192.0.3.0"},
		{name: "all IPv4 first", rule: "::ffff:0:0/96", addr: "0.0.0.0", want: true},
		{name: "all IPv4 last", rule: "::ffff:0:0/96", addr: "255.255.255.255", want: true},
		{name: "all IPv4 excludes IPv6", rule: "::ffff:0:0/96", addr: "::1"},
		{name: "single IPv4", rule: "::ffff:192.0.2.1/128", addr: "192.0.2.1", want: true},
		{name: "single mapped IPv4", rule: "::ffff:192.0.2.1/128", addr: "::ffff:192.0.2.1", want: true},
		{name: "before single IPv4", rule: "::ffff:192.0.2.1/128", addr: "192.0.2.0"},
		{name: "after single IPv4", rule: "::ffff:192.0.2.1/128", addr: "192.0.2.2"},
		{name: "IPv6 all", rule: "::/0", addr: "2001:db8::1", want: true},
		{name: "IPv6 all excludes IPv4", rule: "::/0", addr: "192.0.2.1"},
		{name: "IPv6 all excludes mapped IPv4", rule: "::/0", addr: "::ffff:192.0.2.1"},
		{name: "wide prefix IPv6", rule: "::ffff:0:0/80", addr: "::1", want: true},
		{name: "wide prefix excludes IPv4", rule: "::ffff:0:0/80", addr: "192.0.2.1"},
		{name: "wide prefix excludes mapped IPv4", rule: "::ffff:0:0/80", addr: "::ffff:192.0.2.1"},
		{name: "before mapped prefix IPv6", rule: "::ffff:0:0/95", addr: "::fffe:c000:201", want: true},
		{name: "before mapped prefix excludes IPv4", rule: "::ffff:0:0/95", addr: "192.0.2.1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			proxies, err := utils.NewTrustedProxies(tt.rule)
			if err != nil {
				t.Fatal(err)
			}
			ip := net.ParseIP(tt.addr)
			if got := proxies.Contains(ip); got != tt.want {
				t.Errorf("Contains(%q) for %q = %v, want %v", tt.addr, tt.rule, got, tt.want)
			}

			// 同时经过请求入口，确认地址规范化与公开 Contains 使用同一信任边界。
			req := &http.Request{
				RemoteAddr: net.JoinHostPort(tt.addr, "443"),
				Header:     http.Header{"X-Forwarded-For": []string{"198.51.100.7"}},
			}
			wantIP := ip.String()
			if tt.want {
				wantIP = "198.51.100.7"
			}
			if got := utils.ClientIPWithTrustedProxies(req, proxies); got != wantIP {
				t.Errorf("ClientIPWithTrustedProxies() = %q, want %q", got, wantIP)
			}
		})
	}
}

// TestTrustedProxiesScopedIPv6 验证网卡作用域不影响 IP 规则匹配，返回的地址仍保留作用域。
func TestTrustedProxiesScopedIPv6(t *testing.T) {
	for _, tt := range []struct {
		name      string
		rule      string // 空串使用 nil 白名单，不能启用转发头。
		remote    string
		forwarded string
		realIP    string
		want      string
	}{
		{name: "CIDR", rule: "fe80::/10", remote: "fe80::1%en0", forwarded: "198.51.100.1", want: "198.51.100.1"},
		{name: "single IP", rule: "fe80::1", remote: "fe80::1%en0", forwarded: "198.51.100.1", want: "198.51.100.1"},
		{name: "another interface", rule: "fe80::1", remote: "fe80::1%en1", forwarded: "198.51.100.1", want: "198.51.100.1"},
		{name: "trusted chain", rule: "fe80::/10", remote: "fe80::1%en0", forwarded: "198.51.100.1, fe80::2%en1", want: "198.51.100.1"},
		{name: "untrusted hop", rule: "fe80::/10", remote: "fe80::1%en0", forwarded: "198.51.100.1, 2001:db8::1%en1, fe80::2%en1", want: "2001:db8::1%en1"},
		{name: "real IP fallback", rule: "fe80::/10", remote: "fe80::1%en0", forwarded: "invalid", realIP: "fe80::2%en1", want: "fe80::2%en1"},
		{name: "remote fallback", rule: "fe80::/10", remote: "fe80::1%en0", want: "fe80::1%en0"},
		{name: "outside single IP", rule: "fe80::1", remote: "fe80::2%en0", forwarded: "198.51.100.1", want: "fe80::2%en0"},
		{name: "outside CIDR", rule: "fe80::/10", remote: "2001:db8::1%en0", forwarded: "198.51.100.1", want: "2001:db8::1%en0"},
		{name: "nil rules", remote: "fe80::1%en0", forwarded: "198.51.100.1", want: "fe80::1%en0"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var proxies *utils.TrustedProxies
			if tt.rule != "" {
				var err error
				proxies, err = utils.NewTrustedProxies(tt.rule)
				if err != nil {
					t.Fatal(err)
				}
			}
			req := &http.Request{
				RemoteAddr: net.JoinHostPort(tt.remote, "443"),
				Header: http.Header{
					"X-Forwarded-For": {tt.forwarded},
					"X-Real-Ip":       {tt.realIP},
				},
			}
			if got := utils.ClientIPWithTrustedProxies(req, proxies); got != tt.want {
				t.Fatalf("ClientIPWithTrustedProxies() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientIPWithMultipleForwardedHeaders(t *testing.T) {
	proxies, err := utils.NewTrustedProxies("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	// 同名 X-Forwarded-For 字段按接收顺序组成同一条链，不能丢弃后续字段。
	for _, tt := range []struct {
		name    string
		remote  string
		headers []string
		want    string
	}{
		{name: "single header", remote: "10.0.0.1", headers: []string{"198.51.100.1, 203.0.113.5, 10.0.0.2"}, want: "203.0.113.5"},
		{name: "split header", remote: "10.0.0.1", headers: []string{"198.51.100.1", "203.0.113.5, 10.0.0.2"}, want: "203.0.113.5"},
		{name: "trusted chain across headers", remote: "10.0.0.1", headers: []string{"198.51.100.1, 10.0.0.2", "10.0.0.3"}, want: "198.51.100.1"},
		{name: "invalid nodes", remote: "10.0.0.1", headers: []string{"invalid", "203.0.113.5, invalid, 10.0.0.2"}, want: "203.0.113.5"},
		{name: "empty first header", remote: "10.0.0.1", headers: []string{"", "203.0.113.5, 10.0.0.2"}, want: "203.0.113.5"},
		{name: "all trusted", remote: "10.0.0.1", headers: []string{"10.0.0.4", "invalid, 10.0.0.2"}, want: "10.0.0.4"},
		{name: "real IP fallback", remote: "10.0.0.1", headers: []string{"invalid", " "}, want: "198.51.100.8"},
		{name: "untrusted remote", remote: "8.8.8.8", headers: []string{"198.51.100.1", "203.0.113.5, 10.0.0.2"}, want: "8.8.8.8"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				RemoteAddr: net.JoinHostPort(tt.remote, "443"),
				Header: http.Header{
					"X-Forwarded-For": tt.headers,
					"X-Real-Ip":       []string{"198.51.100.8"},
				},
			}
			if got := utils.ClientIPWithTrustedProxies(req, proxies); got != tt.want {
				t.Fatalf("ClientIPWithTrustedProxies() = %q, want %q", got, tt.want)
			}
		})
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

// BenchmarkClientIPRemoteOnly 计量不读取转发头的直连路径。
func BenchmarkClientIPRemoteOnly(b *testing.B) {
	req := &http.Request{RemoteAddr: "8.8.8.8:443"}
	b.ReportAllocs()
	for b.Loop() {
		benchmarkClientIP = utils.ClientIP(req)
	}
}

// BenchmarkClientIPLoopbackForwardedChain 计量回环连接下从 XFF 链首提取客户端地址的路径。
func BenchmarkClientIPLoopbackForwardedChain(b *testing.B) {
	req := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"192.168.47.141, 192.168.47.142, 175.176.32.112, 192.168.47.143"},
		},
		RemoteAddr: "127.0.0.1:80",
	}
	b.ReportAllocs()
	for b.Loop() {
		benchmarkClientIP = utils.ClientIP(req)
	}
}

// BenchmarkClientIPTrustedProxies 计量跳过链尾可信代理后的地址解析。
func BenchmarkClientIPTrustedProxies(b *testing.B) {
	proxies, err := utils.NewTrustedProxies("10.0.0.0/8", "192.168.0.0/16")
	if err != nil {
		b.Fatal(err)
	}
	req := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"203.0.113.10, 10.1.1.1, 192.168.1.1"},
		},
		RemoteAddr: "10.0.0.10:443",
	}
	b.ReportAllocs()
	for b.Loop() {
		benchmarkClientIP = utils.ClientIPWithTrustedProxies(req, proxies)
	}
}

// BenchmarkClientIPInvalidForwardFallback 计量非法 XFF 回退到 X-Real-Ip 的路径。
func BenchmarkClientIPInvalidForwardFallback(b *testing.B) {
	proxies, err := utils.NewTrustedProxies("10.0.0.0/8")
	if err != nil {
		b.Fatal(err)
	}
	req := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"invalid-ip"},
			"X-Real-Ip":       []string{"203.0.113.11"},
		},
		RemoteAddr: "10.0.0.10:443",
	}
	b.ReportAllocs()
	for b.Loop() {
		benchmarkClientIP = utils.ClientIPWithTrustedProxies(req, proxies)
	}
}
