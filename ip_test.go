package utils

import (
	"net/http"
	"testing"
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
			gotIp := ServerIP()
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
			gotIp := LocalIP()
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
		}, want: "175.176.32.112"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClientIP(tt.args.req); got != tt.want {
				t.Errorf("ClientIp() = %v, want %v", got, tt.want)
			}
		})
	}
}
