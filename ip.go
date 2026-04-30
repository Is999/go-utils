package utils

import (
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// ServerIP 获取服务器对外 IP 地址。
// 通过连接外部服务（8.8.8.8:80）获取出站 IP 地址。
// 如果连接失败，则回退到获取本地 IP。
//
// 返回值：服务器对外 IP 地址字符串，获取失败返回空字符串
func ServerIP() string {
	// 连接外部服务获取出站 IP
	conn, err := net.DialTimeout("udp", "8.8.8.8:80", 5*time.Second)
	if err != nil {
		return LocalIP()
	}
	defer conn.Close()

	// 提取本地 IP 地址
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	return ""
}

// LocalIP 获取本机 IP 地址。
// 优先返回主机名对应的 IPv4 地址，如果获取失败则遍历网络接口。
//
// 返回值：本地 IP 地址字符串，获取失败返回空字符串
func LocalIP() string {
	// 优先通过主机名获取 IP
	hostname, err := os.Hostname()
	if err == nil {
		ips, err := net.LookupIP(hostname)
		if err == nil {
			for _, ip := range ips {
				if ipv4 := ip.To4(); ipv4 != nil {
					return ipv4.String()
				}
			}
		}
	}

	// 遍历网络接口获取 IP
	addr, err := net.InterfaceAddrs()
	if err == nil {
		for _, v := range addr {
			if inet, ok := v.(*net.IPNet); ok && !inet.IP.IsLoopback() && inet.IP.To4() != nil {
				return inet.IP.String()
			}
		}
	}
	return ""
}

// ClientIP 获取客户端 IP 地址。
// 优先从请求头中获取，依次检查 X-Forwarded-For、X-Real-IP、RemoteAddr。
//
// 参数说明：
//   - r：HTTP 请求对象
//
// 返回值：客户端 IP 地址字符串
func ClientIP(r *http.Request) string {
	// 优先从 X-Forwarded-For 获取
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if !isPrivateIP(ip) {
				return ip
			}
		}
	}

	// 从 X-Real-IP 获取
	xri := strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	ip := strings.TrimSpace(strings.Split(xri, ",")[0])
	if ip != "" {
		return ip
	}

	// 从 RemoteAddr 获取
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
		return ip
	}
	return r.RemoteAddr
}

// isPrivateIP 检查 IP 地址是否为私有地址或回环地址。
//
// 参数说明：
//   - ip：IP 地址字符串
//
// 返回值：true 表示是私有地址或回环地址
func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && (parsedIP.IsLoopback() || parsedIP.IsPrivate())
}
