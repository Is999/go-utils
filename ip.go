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
	conn, err := net.DialTimeout("udp", "8.8.8.8:80", time.Second)
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
// 默认仅在请求来自可信代理（回环、私网、链路本地地址）时信任转发头，避免被客户端伪造。
//
// 参数说明：
//   - r：HTTP 请求对象
//
// 返回值：客户端 IP 地址字符串
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	remoteIP := parseRequestIP(r.RemoteAddr)
	if isTrustedProxyIP(remoteIP) {
		if forwardedIP := firstForwardedIP(r.Header.Get("X-Forwarded-For")); forwardedIP != nil {
			return forwardedIP.String()
		}
		if realIP := parseRequestIP(r.Header.Get("X-Real-Ip")); realIP != nil {
			return realIP.String()
		}
	}

	if remoteIP != nil {
		return remoteIP.String()
	}
	return ""
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

// parseRequestIP 从请求相关字符串中解析 IP。
// 支持 RemoteAddr、X-Real-IP 以及单个 IP 字符串。
func parseRequestIP(raw string) net.IP {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	// 请求头中可能错误地带了多个值，这里只取第一个。
	raw = strings.TrimSpace(strings.Split(raw, ",")[0])
	if raw == "" {
		return nil
	}

	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	} else {
		raw = strings.Trim(raw, "[]")
	}

	return net.ParseIP(raw)
}

// firstForwardedIP 获取 X-Forwarded-For 中最适合作为客户端地址的 IP。
// 优先返回第一个合法公网 IP；如果全是内网地址，则回退到第一个合法 IP。
func firstForwardedIP(xff string) net.IP {
	var firstValidIP net.IP
	for _, ipText := range strings.Split(xff, ",") {
		if ip := parseRequestIP(ipText); ip != nil {
			if firstValidIP == nil {
				firstValidIP = ip
			}
			if !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() {
				return ip
			}
		}
	}
	return firstValidIP
}

// isTrustedProxyIP 判断来源地址是否可被视为可信代理。
// 默认信任回环、私网和链路本地地址，以兼容常见内网反向代理部署。
func isTrustedProxyIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}
