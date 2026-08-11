package core

import (
	"context"
	testx "github.com/lcylpzls/testx"
	"testing"
	"time"
)

// BenchmarkAllowHit 基准:令牌充足时的 Allow 命中。
func BenchmarkAllowHit(b *testing.B) {
	l, err := NewTokenBucket(1e9, 1e9)
	testx.RequireNoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Allow()
	}
}

// BenchmarkWaitHit 基准:令牌充足时的 Wait 命中。
func BenchmarkWaitHit(b *testing.B) {
	l, err := NewTokenBucket(1e9, 1e9)
	testx.RequireNoError(b, err)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Wait(ctx)
	}
}

// BenchmarkAllowReject 基准:令牌耗尽时的拒绝。
func BenchmarkAllowReject(b *testing.B) {
	l, err := NewTokenBucket(0.001, 1)
	testx.RequireNoError(b, err)

	_ = l.Allow()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Allow()
	}
}

// BenchmarkCBAallow 基准:熔断器 Closed 状态放行。
func BenchmarkCBAallow(b *testing.B) {
	cb, err := NewCircuitBreaker()
	testx.RequireNoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Allow()
	}
}

// BenchmarkBulkheadTryAcquire 基准:舱壁非阻塞获取并释放。
func BenchmarkBulkheadTryAcquire(b *testing.B) {
	bh, err := NewBulkhead(1e6)
	testx.RequireNoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		release, ok := bh.TryAcquire()
		if ok {
			release()
		}
	}
}

// BenchmarkWindowAllow 基准:固定窗口放行。
func BenchmarkWindowAllow(b *testing.B) {
	w, err := NewFixedWindow(1e9, time.Second)
	testx.RequireNoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.Allow()
	}
}
