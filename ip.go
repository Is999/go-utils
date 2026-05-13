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
	return clientIPByTrustChecker(r, isTrustedProxyAddr)
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
//
// 参数说明：
//   - ip：待判断的 IP 地址
//
// 返回值：true 表示命中白名单
func (p *TrustedProxies) Contains(ip net.IP) bool {
	addr, ok := addrFromIP(ip)
	if p == nil || !ok {
		return false
	}
	return p.containsAddr(addr)
}

// containsAddr 判断 netip 地址是否命中可信代理白名单。
// 该方法服务于 ClientIP 热路径，避免把每个请求地址转换为 net.IP 切片。
//
// 参数说明：
//   - addr：待判断的客户端或代理地址，来源于请求 RemoteAddr / X-Forwarded-For。
//
// 返回值：true 表示命中白名单。
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
//
// 参数说明：
//   - r：HTTP 请求对象
//   - trustedProxies：可信代理白名单
//
// 返回值：客户端 IP 地址字符串
func ClientIPWithTrustedProxies(r *http.Request, trustedProxies *TrustedProxies) string {
	return clientIPByTrustChecker(r, func(addr netip.Addr) bool {
		return trustedProxies != nil && trustedProxies.containsAddr(addr)
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
	addr, ok := parseRequestAddr(raw)
	if !ok {
		return nil
	}
	return ipFromAddr(addr)
}

// parseRequestAddr 从请求相关字符串中解析值类型 IP 地址。
// 支持 RemoteAddr、X-Real-IP、X-Forwarded-For 单节点以及错误多值头；返回 netip.Addr 以减少热路径分配。
//
// 参数说明：
//   - raw：原始地址字符串，可能是 host:port、[ipv6]:port、单个 IP 或错误拼接的多值头。
//
// 返回值：
//   - netip.Addr：解析成功的 IP 地址，IPv4-mapped IPv6 会归一化为 IPv4。
//   - bool：true 表示解析成功，false 表示输入为空或不是合法 IP。
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
//
// 参数说明：
//   - value：单个 IP 或 CIDR 表达式
//
// 返回值：解析后的网段对象、错误信息
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
//
// 参数说明：
//   - ip：标准库 IP 切片，可能是 IPv4、IPv6 或 IPv4-mapped IPv6。
//
// 返回值：
//   - netip.Addr：归一化后的 IP 地址。
//   - bool：true 表示转换成功。
func addrFromIP(ip net.IP) (netip.Addr, bool) {
	if ip == nil {
		return netip.Addr{}, false
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return netip.AddrFrom4([4]byte{ipv4[0], ipv4[1], ipv4[2], ipv4[3]}), true
	}
	ipv6 := ip.To16()
	if ipv6 == nil {
		return netip.Addr{}, false
	}
	return netip.AddrFrom16([16]byte{
		ipv6[0], ipv6[1], ipv6[2], ipv6[3],
		ipv6[4], ipv6[5], ipv6[6], ipv6[7],
		ipv6[8], ipv6[9], ipv6[10], ipv6[11],
		ipv6[12], ipv6[13], ipv6[14], ipv6[15],
	}), true
}

// ipFromAddr 将 netip.Addr 转成标准库 net.IP。
// 仅用于兼容旧内部函数和全部代理可信时的历史回退语义，不参与常规 ClientIP 热路径返回。
//
// 参数说明：
//   - addr：值类型 IP 地址。
//
// 返回值：标准库 net.IP；无效地址返回 nil。
func ipFromAddr(addr netip.Addr) net.IP {
	if !addr.IsValid() {
		return nil
	}
	if addr.Is4() {
		ipv4 := addr.As4()
		return net.IPv4(ipv4[0], ipv4[1], ipv4[2], ipv4[3])
	}
	ipv6 := addr.As16()
	return net.IP{
		ipv6[0], ipv6[1], ipv6[2], ipv6[3],
		ipv6[4], ipv6[5], ipv6[6], ipv6[7],
		ipv6[8], ipv6[9], ipv6[10], ipv6[11],
		ipv6[12], ipv6[13], ipv6[14], ipv6[15],
	}
}

// isTrustedProxyIP 判断来源地址是否可被视为可信代理。
// 默认仅信任回环地址，避免把所有私网来源都视为可信代理。
func isTrustedProxyIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// isTrustedProxyAddr 判断来源地址是否可被视为默认可信代理。
// 默认只信任回环地址，避免内网机器伪造 X-Forwarded-For 影响客户端 IP 判断。
//
// 参数说明：
//   - addr：来源地址，通常来自请求 RemoteAddr。
//
// 返回值：true 表示可以信任转发头。
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
