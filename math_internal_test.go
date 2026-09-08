package utils

import "testing"

func TestRandUint64NRejection(t *testing.T) {
	for _, n := range []uint64{1, 3, 1 << 63, ^uint64(0)} {
		calls := 0
		got := randUint64N(n, func() uint64 {
			calls++
			// 最大值落在拒绝区，第二次采样才产生有效结果。
			if calls == 1 {
				return ^uint64(0)
			}
			return n - 1
		})
		if got != n-1 || calls != 2 {
			t.Fatalf("randUint64N(%d) = %d after %d calls", n, got, calls)
		}
	}
}
