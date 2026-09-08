package utils

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Is999/go-utils/errors"
)

// TrustedProxies 保存调用方配置的可信代理 IP/CIDR；构造后只读，可供并发请求复用。
// 规则只比较 IP，不按 IPv6 zone 区分网卡。
type TrustedProxies struct {
	prefixes []netip.Prefix // NewTrustedProxies 解析后的代理地址规则。
}

const serverIPCacheTTL = time.Minute // 服务器 IP 缓存有效期

// serverIPCache 缓存服务器 IP，避免热路径重复探测网卡或外网连接。
var serverIPCache struct {
	mu        sync.RWMutex // 缓存读写锁
	ip        string       // 缓存的服务器 IP
	expiresAt time.Time    // 缓存过期时间
}

// ServerIP 按缓存、LocalIP、UDP 路由探测的顺序获取本机地址，可能是私网或回环地址。
// UDP 探测使用 1 秒超时，不查询 NAT 后的公网地址。
func ServerIP() string {
	// 缓存命中时无需创建超时上下文和定时器。
	if ip := loadServerIPCache(); ip != "" {
		return ip
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return ServerIPContext(ctx)
}

// ServerIPContext 沿用 ServerIP 的地址来源顺序，并缓存非空结果 1 分钟。
// ctx 仅控制最后的 UDP 探测，不能中止 LocalIP 中的主机名解析或网卡枚举。
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

// LocalIP 优先返回主机名解析出的首个 IPv4；没有结果时查找非回环 IPv4 网卡地址。
// 主机名解析结果可能是回环地址；未找到地址时返回空串。
func LocalIP() string {
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
// 仅回环 RemoteAddr 会启用转发头解析；其他代理规则使用 ClientIPWithTrustedProxies 配置。
func ClientIP(r *http.Request) string {
	return clientIPByTrustChecker(r, netip.Addr.IsLoopback)
}

// NewTrustedProxies 构造可信代理白名单。
// 接受 IP 或 CIDR，忽略空白项；任一非空项非法则返回错误，不交付部分规则。
func NewTrustedProxies(values ...string) (*TrustedProxies, error) {
	proxies := &TrustedProxies{
		prefixes: make([]netip.Prefix, 0, len(values)),
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		prefix, err := parseTrustedProxy(value)
		if err != nil {
			return nil, errors.Tag(err)
		}
		proxies.prefixes = append(proxies.prefixes, prefix)
	}
	return proxies, nil
}

// Contains 判断地址是否命中规则；nil 规则或无效 IP 均返回 false。
func (p *TrustedProxies) Contains(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	return ok && p.containsAddr(addr.Unmap())
}

// containsAddr 直接匹配已解析的地址，避免转发链中反复转换 net.IP。
func (p *TrustedProxies) containsAddr(addr netip.Addr) bool {
	if p == nil || !addr.IsValid() {
		return false
	}
	// Prefix 不携带 zone，匹配时忽略作用域，返回给调用方的地址仍保留原值。
	addr = addr.WithZone("")
	for _, prefix := range p.prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// ClientIPWithTrustedProxies 使用显式可信代理白名单解析客户端 IP。
// 仅 RemoteAddr 命中规则时读取转发头；nil 规则不信任任何代理。
// 优先使用 X-Forwarded-For，再尝试 X-Real-Ip，最后返回 RemoteAddr。
func ClientIPWithTrustedProxies(r *http.Request, trustedProxies *TrustedProxies) string {
	return clientIPByTrustChecker(r, trustedProxies.containsAddr)
}

// parseRequestAddr 接受 IP 或 host:port，IPv4 映射地址转为 IPv4，多值字符串只取首项。
func parseRequestAddr(raw string) (netip.Addr, bool) {
	raw, _, _ = strings.Cut(raw, ",")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return netip.Addr{}, false
	}

	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	} else {
		raw = strings.Trim(raw, "[]")
	}

	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

// clientIPByTrustChecker 根据可信代理判断函数解析客户端 IP。
// RemoteAddr 无法解析时返回空串；远端不可信时只返回其地址，不读取转发头。
func clientIPByTrustChecker(r *http.Request, isTrusted func(netip.Addr) bool) string {
	if r == nil {
		return ""
	}

	remoteAddr, ok := parseRequestAddr(r.RemoteAddr)
	if !ok {
		return ""
	}
	if isTrusted == nil || !isTrusted(remoteAddr) {
		return remoteAddr.String()
	}

	// 多个同名转发头按接收顺序组成一条链。
	xff := strings.Join(r.Header.Values("X-Forwarded-For"), ",")
	if forwardedAddr, ok := clientIPFromForwardedChain(xff, isTrusted); ok {
		return forwardedAddr.String()
	}
	if realAddr, ok := parseRequestAddr(r.Header.Get("X-Real-Ip")); ok {
		return realAddr.String()
	}
	return remoteAddr.String()
}

// clientIPFromForwardedChain 从右向左查找首个不可信地址，调用方须先确认直连来源可信。
func clientIPFromForwardedChain(xff string, isTrusted func(netip.Addr) bool) (netip.Addr, bool) {
	if xff == "" || isTrusted == nil {
		return netip.Addr{}, false
	}

	var leftmostValid netip.Addr // 全链可信时的候选，不保证它就是原始客户端。
	for rest := xff; rest != ""; {
		part := rest
		if comma := strings.LastIndexByte(rest, ','); comma >= 0 {
			part = rest[comma+1:]
			rest = rest[:comma]
		} else {
			rest = ""
		}

		addr, ok := parseRequestAddr(part)
		if !ok {
			// 非法节点不影响继续查找更左侧的合法地址。
			continue
		}
		leftmostValid = addr
		if !isTrusted(addr) {
			return addr, true
		}
	}

	// 全链可信时使用最左侧地址；没有合法节点则交给调用方继续回退。
	return leftmostValid, leftmostValid.IsValid()
}

// parseTrustedProxy 解析单个可信代理配置，支持 IP 与 CIDR 两种形式。
func parseTrustedProxy(value string) (netip.Prefix, error) {
	if strings.Contains(value, "/") {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return netip.Prefix{}, errors.Errorf("trusted proxy CIDR 非法: %s", value)
		}
		// 仅完整落在 IPv4 映射范围内的前缀随地址一起 Unmap，保留更宽的 IPv6 规则。
		if prefix.Addr().Is4In6() && prefix.Bits() >= 96 {
			prefix = netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits()-96)
		}
		return prefix.Masked(), nil
	}

	addr, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Prefix{}, errors.Errorf("trusted proxy IP 非法: %s", value)
	}
	addr = addr.Unmap()
	return netip.PrefixFrom(addr, addr.BitLen()), nil
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

// storeServerIPCache 保存调用方确认的非空地址；探测在锁外进行，并发未命中可能各自探测。
func storeServerIPCache(ip string) {
	serverIPCache.mu.Lock()
	serverIPCache.ip = ip
	serverIPCache.expiresAt = time.Now().Add(serverIPCacheTTL)
	serverIPCache.mu.Unlock()
}

// dialServerIP 读取到 8.8.8.8 的 UDP 路由所选本地地址，不发送应用数据或查询公网 IP。
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
