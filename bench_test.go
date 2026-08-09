package resiliencex

import (
	"context"
	"testing"
)

// BenchmarkAllowHit 基准:令牌充足时的 Allow 命中。
func BenchmarkAllowHit(b *testing.B) {
	l, err := NewTokenBucket(1e9, 1e9)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Allow()
	}
}

// BenchmarkWaitHit 基准:令牌充足时的 Wait 命中。
func BenchmarkWaitHit(b *testing.B) {
	l, err := NewTokenBucket(1e9, 1e9)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Wait(ctx)
	}
}

// BenchmarkAllowReject 基准:令牌耗尽时的拒绝。
func BenchmarkAllowReject(b *testing.B) {
	l, err := NewTokenBucket(0.001, 1)
	if err != nil {
		b.Fatal(err)
	}
	_ = l.Allow()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Allow()
	}
}
