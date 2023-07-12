package utils

import (
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// ServerIP 服务器对外IP
func ServerIP() string {
	// 连接一个外部服务（谷歌）获取出站 IP 地址
	conn, err := net.DialTimeout("udp", "8.8.8.8:80", 5*time.Second)
	if err != nil {
		return LocalIP()
	}
	defer conn.Close()

	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	return ""
}

// LocalIP 本地IP
func LocalIP() string {
	// 获取主机名
	hostname, err := os.Hostname()
	if err == nil {
		ips, err := net.LookupIP(hostname)
		if err == nil {
			// 遍历 IP 地址，选择第一个 IPv4 类型的地址
			for _, ip := range ips {
				if ipv4 := ip.To4(); ipv4 != nil {
					return ipv4.String()
				}
			}
		}
	}

	addr, err := net.InterfaceAddrs()
	if err == nil {
		for _, v := range addr {
			// 检查ip地址判断是否回环地址
			if inet, ok := v.(*net.IPNet); ok && !inet.IP.IsLoopback() && inet.IP.To4() != nil {
				return inet.IP.String()
			}
		}
	}
	return ""
}

// ClientIP 获取客户端IP
func ClientIP(r *http.Request) string {
	// 获取请求头中的 X-Forwarded-For 字段值
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// 通过逗号分隔的多个 IP 地址，返回第一个非内网 IP
		ips := strings.Split(xForwardedFor, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if !isPrivateIP(ip) {
				return ip
			}
		}
	}

	// 获取请求头中的 X-Real-Ip 字段值
	xRealIp := strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	ip := strings.TrimSpace(strings.Split(xRealIp, ",")[0])
	if ip != "" {
		return ip
	}

	// 获取 RemoteAddr 字段的值
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
		return ip
	}

	return r.RemoteAddr
}

func isPrivateIP(ip string) bool {
	// 检查是否是私有 IP 地址
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && (parsedIP.IsLoopback() || parsedIP.IsPrivate())
}
