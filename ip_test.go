package utils_test

import (
	"net"
	"net/http"
	"testing"

	"github.com/Is999/go-utils"
)

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
