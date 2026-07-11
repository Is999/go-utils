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

// TrustedProxies 保存可信代理 IP/CIDR 列表。
// 生产环境建议按网关、负载均衡或 Ingress 的固定地址显式配置，避免过宽地信任所有私网来源。
type TrustedProxies struct {
	prefixes []netip.Prefix // prefixes 是可信代理 IP/CIDR 规则，数据来源于 NewTrustedProxies 的业务配置。
}

const serverIPCacheTTL = time.Minute // 服务器 IP 缓存有效期

// serverIPCache 缓存服务器 IP，避免热路径重复探测网卡或外网连接。
var serverIPCache struct {
	mu        sync.RWMutex // 缓存读写锁
	ip        string       // 缓存的服务器 IP
	expiresAt time.Time    // 缓存过期时间
}

// ServerIP 获取服务器对外 IP 地址。
// 默认优先返回缓存值或本地网卡 IP，避免在热路径上频繁拨号外网地址。
// 如需自定义超时控制，可使用 ServerIPContext。
func ServerIP() string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return ServerIPContext(ctx)
}

// ServerIPContext 获取服务器出站 IP 地址，并允许调用方控制超时。
// 默认优先返回缓存值或本地网卡 IP，仅在本地 IP 不可用时才回退到 UDP 探测。
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
func ClientIP(r *http.Request) string {
	return clientIPByTrustChecker(r, isTrustedProxyAddr)
}

// NewTrustedProxies 构造可信代理白名单。
// 支持传入单个 IP 或 CIDR，例如 `10.0.0.10`、`10.0.0.0/8`、`fd00::/8`。
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

// Contains 判断指定 IP 是否命中可信代理白名单。
func (p *TrustedProxies) Contains(ip net.IP) bool {
	addr, ok := addrFromIP(ip)
	if p == nil || !ok {
		return false
	}
	return p.containsAddr(addr)
}

// containsAddr 判断 netip 地址是否命中可信代理白名单。
// 该方法服务于 ClientIP 热路径，避免把每个请求地址转换为 net.IP 切片。
func (p *TrustedProxies) containsAddr(addr netip.Addr) bool {
	if p == nil || !addr.IsValid() {
		return false
	}
	for _, prefix := range p.prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// ClientIPWithTrustedProxies 使用显式可信代理白名单解析客户端 IP。
// 仅当 RemoteAddr 命中 trustedProxies 时才信任 `X-Forwarded-For` / `X-Real-Ip`。
func ClientIPWithTrustedProxies(r *http.Request, trustedProxies *TrustedProxies) string {
	return clientIPByTrustChecker(r, func(addr netip.Addr) bool {
		return trustedProxies != nil && trustedProxies.containsAddr(addr)
	})
}

// parseRequestAddr 从请求相关字符串中解析值类型 IP 地址。
// 支持 RemoteAddr、X-Real-IP、X-Forwarded-For 单节点以及错误多值头；返回 netip.Addr 以减少热路径分配。
func parseRequestAddr(raw string) (netip.Addr, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return netip.Addr{}, false
	}

	// 请求头中可能错误地带了多个值，这里只取第一个。
	if comma := strings.IndexByte(raw, ','); comma >= 0 {
		// comma 是第一个逗号位置，截断后可避免 strings.Split 为异常多值头创建临时切片。
		raw = raw[:comma]
	}
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
// 当远端地址不可信时，直接回退为 `RemoteAddr`，不读取任何转发头。
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

	if forwardedAddr, ok := clientIPFromForwardedChain(r.Header.Get("X-Forwarded-For"), remoteAddr, isTrusted); ok {
		return forwardedAddr.String()
	}
	if realAddr, ok := parseRequestAddr(r.Header.Get("X-Real-Ip")); ok {
		return realAddr.String()
	}
	return remoteAddr.String()
}

// clientIPFromForwardedChain 按代理链从右向左回溯客户端地址。
// 会跳过所有可信代理，返回第一个不在可信代理列表中的地址。
func clientIPFromForwardedChain(xff string, remoteAddr netip.Addr, isTrusted func(netip.Addr) bool) (netip.Addr, bool) {
	if xff == "" || isTrusted == nil {
		return netip.Addr{}, false
	}

	var leftmostValid netip.Addr // leftmostValid 记录最左侧合法转发 IP，全部代理可信时按历史语义返回原始客户端地址。
	for rest := xff; rest != ""; {
		part := rest // part 是当前从右侧切出的 X-Forwarded-For 节点，可能包含空格或非法 IP。
		if comma := strings.LastIndexByte(rest, ','); comma >= 0 {
			// comma 是当前剩余字符串中的最后一个分隔符，用于无分配地从右向左回溯代理链。
			part = rest[comma+1:]
			rest = rest[:comma]
		} else {
			rest = ""
		}

		// X-Forwarded-For 按代理追加顺序从左到右增长；从右向左扫描可跳过末端可信代理链。
		addr, ok := parseRequestAddr(part) // addr 是当前节点解析出的候选客户端地址，非法节点直接跳过以支持降级。
		if !ok {
			continue
		}
		leftmostValid = addr
		if !isTrusted(addr) {
			return addr, true
		}
	}

	if !leftmostValid.IsValid() {
		return netip.Addr{}, false
	}
	if remoteAddr.IsValid() && !isTrusted(remoteAddr) {
		return remoteAddr, true
	}
	// 全部转发节点都可信时，按历史行为返回最左侧原始客户端地址。
	return leftmostValid, true
}

// parseTrustedProxy 解析单个可信代理配置，支持 IP 与 CIDR 两种形式。
func parseTrustedProxy(value string) (netip.Prefix, error) {
	if strings.Contains(value, "/") {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return netip.Prefix{}, errors.Errorf("trusted proxy CIDR 非法: %s", value)
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

// addrFromIP 将标准库 net.IP 转成 netip.Addr。
// 该函数用于兼容公开的 TrustedProxies.Contains(net.IP)，同时让内部热路径使用零分配值类型。
func addrFromIP(ip net.IP) (netip.Addr, bool) {
	addr, ok := netip.AddrFromSlice(ip)
	return addr.Unmap(), ok
}

// isTrustedProxyAddr 判断来源地址是否可被视为默认可信代理。
// 默认只信任回环地址，避免内网机器伪造 X-Forwarded-For 影响客户端 IP 判断。
func isTrustedProxyAddr(addr netip.Addr) bool {
	return addr.IsValid() && addr.IsLoopback()
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
