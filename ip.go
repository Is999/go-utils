package utils

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Is999/go-utils/errors"
)

// TrustedProxies 保存可信代理 IP/CIDR 列表。
// 生产环境建议按网关、负载均衡或 Ingress 的固定地址显式配置，避免过宽地信任所有私网来源。
type TrustedProxies struct {
	nets []*net.IPNet
}

const serverIPCacheTTL = time.Minute

var serverIPCache struct {
	mu        sync.RWMutex
	ip        string
	expiresAt time.Time
}

// ServerIP 获取服务器对外 IP 地址。
// 默认优先返回缓存值或本地网卡 IP，避免在热路径上频繁拨号外网地址。
// 如需自定义超时控制，可使用 ServerIPContext。
//
// 返回值：服务器对外 IP 地址字符串，获取失败返回空字符串
func ServerIP() string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return ServerIPContext(ctx)
}

// ServerIPContext 获取服务器出站 IP 地址，并允许调用方控制超时。
// 默认优先返回缓存值或本地网卡 IP，仅在本地 IP 不可用时才回退到 UDP 探测。
//
// 参数说明：
//   - ctx：上下文，可用于控制超时或取消
//
// 返回值：服务器出站 IP 地址字符串，获取失败返回空字符串
func ServerIPContext(ctx context.Context) string {
	if ip := loadServerIPCache(); ip != "" {
		return ip
	}
	if ip := LocalIP(); ip != "" {
		storeServerIPCache(ip)
		return ip
	}
	ip := dialServerIP(ctx)
	if ip != "" {
		storeServerIPCache(ip)
	}
	return ip
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
// 默认仅在请求来自回环地址时信任转发头，生产环境建议使用 ClientIPWithTrustedProxies 显式配置白名单。
//
// 参数说明：
//   - r：HTTP 请求对象
//
// 返回值：客户端 IP 地址字符串
func ClientIP(r *http.Request) string {
	return clientIPByTrustChecker(r, isTrustedProxyIP)
}

// NewTrustedProxies 构造可信代理白名单。
// 支持传入单个 IP 或 CIDR，例如 `10.0.0.10`、`10.0.0.0/8`、`fd00::/8`。
//
// 参数说明：
//   - values：可信代理 IP 或 CIDR 列表
//
// 返回值：可信代理配置、错误信息
func NewTrustedProxies(values ...string) (*TrustedProxies, error) {
	proxies := &TrustedProxies{
		nets: make([]*net.IPNet, 0, len(values)),
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		ipNet, err := parseTrustedProxy(value)
		if err != nil {
			return nil, errors.Tag(err)
		}
		proxies.nets = append(proxies.nets, ipNet)
	}
	return proxies, nil
}

// Contains 判断指定 IP 是否命中可信代理白名单。
//
// 参数说明：
//   - ip：待判断的 IP 地址
//
// 返回值：true 表示命中白名单
func (p *TrustedProxies) Contains(ip net.IP) bool {
	if p == nil || ip == nil {
		return false
	}
	for _, ipNet := range p.nets {
		if ipNet != nil && ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIPWithTrustedProxies 使用显式可信代理白名单解析客户端 IP。
// 仅当 RemoteAddr 命中 trustedProxies 时才信任 `X-Forwarded-For` / `X-Real-Ip`。
//
// 参数说明：
//   - r：HTTP 请求对象
//   - trustedProxies：可信代理白名单
//
// 返回值：客户端 IP 地址字符串
func ClientIPWithTrustedProxies(r *http.Request, trustedProxies *TrustedProxies) string {
	return clientIPByTrustChecker(r, func(ip net.IP) bool {
		return trustedProxies != nil && trustedProxies.Contains(ip)
	})
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

// clientIPByTrustChecker 根据可信代理判断函数解析客户端 IP。
// 当远端地址不可信时，直接回退为 `RemoteAddr`，不读取任何转发头。
func clientIPByTrustChecker(r *http.Request, isTrusted func(net.IP) bool) string {
	if r == nil {
		return ""
	}

	remoteIP := parseRequestIP(r.RemoteAddr)
	if remoteIP == nil {
		return ""
	}
	if isTrusted == nil || !isTrusted(remoteIP) {
		return remoteIP.String()
	}

	if forwardedIP := clientIPFromForwardedChain(r.Header.Get("X-Forwarded-For"), remoteIP, isTrusted); forwardedIP != nil {
		return forwardedIP.String()
	}
	if realIP := parseRequestIP(r.Header.Get("X-Real-Ip")); realIP != nil {
		return realIP.String()
	}
	return remoteIP.String()
}

// clientIPFromForwardedChain 按代理链从右向左回溯客户端地址。
// 会跳过所有可信代理，返回第一个不在可信代理列表中的地址。
func clientIPFromForwardedChain(xff string, remoteIP net.IP, isTrusted func(net.IP) bool) net.IP {
	forwardedIPs := forwardedIPs(xff)
	if len(forwardedIPs) == 0 {
		return nil
	}

	for i := len(forwardedIPs) - 1; i >= 0; i-- {
		if !isTrusted(forwardedIPs[i]) {
			return forwardedIPs[i]
		}
	}

	if remoteIP != nil && !isTrusted(remoteIP) {
		return remoteIP
	}
	return forwardedIPs[0]
}

// forwardedIPs 解析 X-Forwarded-For 中的所有合法 IP，保持原始顺序。
//
// 参数说明：
//   - xff：X-Forwarded-For 原始值
//
// 返回值：合法 IP 列表
func forwardedIPs(xff string) []net.IP {
	parts := strings.Split(xff, ",")
	ips := make([]net.IP, 0, len(parts))
	for _, part := range parts {
		if ip := parseRequestIP(part); ip != nil {
			ips = append(ips, ip)
		}
	}
	return ips
}

// parseTrustedProxy 解析单个可信代理配置，支持 IP 与 CIDR 两种形式。
//
// 参数说明：
//   - value：单个 IP 或 CIDR 表达式
//
// 返回值：解析后的网段对象、错误信息
func parseTrustedProxy(value string) (*net.IPNet, error) {
	if strings.Contains(value, "/") {
		_, ipNet, err := net.ParseCIDR(value)
		if err != nil {
			return nil, errors.Errorf("trusted proxy CIDR 非法: %s", value)
		}
		return ipNet, nil
	}

	ip := net.ParseIP(value)
	if ip == nil {
		return nil, errors.Errorf("trusted proxy IP 非法: %s", value)
	}

	bits := 32
	if ip.To4() == nil {
		bits = 128
	}
	return &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(bits, bits),
	}, nil
}

// isTrustedProxyIP 判断来源地址是否可被视为可信代理。
// 默认仅信任回环地址，避免把所有私网来源都视为可信代理。
func isTrustedProxyIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// loadServerIPCache 读取未过期的 ServerIP 缓存值。
func loadServerIPCache() string {
	serverIPCache.mu.RLock()
	defer serverIPCache.mu.RUnlock()
	if serverIPCache.ip == "" || time.Now().After(serverIPCache.expiresAt) {
		return ""
	}
	return serverIPCache.ip
}

// storeServerIPCache 写入 ServerIP 缓存值。
func storeServerIPCache(ip string) {
	if strings.TrimSpace(ip) == "" {
		return
	}
	serverIPCache.mu.Lock()
	serverIPCache.ip = ip
	serverIPCache.expiresAt = time.Now().Add(serverIPCacheTTL)
	serverIPCache.mu.Unlock()
}

// dialServerIP 通过 UDP 探测获取当前出站 IP。
func dialServerIP(ctx context.Context) string {
	if ctx == nil {
		ctx = context.Background()
	}
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	return ""
}
