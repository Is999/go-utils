package utils

import (
	"testing"
	"time"
)

// BenchmarkServerIPCacheHit 只衡量有效缓存命中，不依赖本机 DNS 或网卡配置。
func BenchmarkServerIPCacheHit(b *testing.B) {
	serverIPCache.mu.Lock()
	previousIP, previousExpiry := serverIPCache.ip, serverIPCache.expiresAt
	serverIPCache.ip = "192.0.2.1"
	serverIPCache.expiresAt = time.Now().Add(time.Hour)
	serverIPCache.mu.Unlock()
	b.Cleanup(func() {
		serverIPCache.mu.Lock()
		serverIPCache.ip, serverIPCache.expiresAt = previousIP, previousExpiry
		serverIPCache.mu.Unlock()
	})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if got := ServerIP(); got != "192.0.2.1" {
			b.Fatalf("ServerIP() = %q", got)
		}
	}
}
