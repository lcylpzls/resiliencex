package resiliencex

import (
	"context"
	"testing"
	"time"
)

// FuzzLimiter 保证任意参数与操作序列下限流器不 panic。
func FuzzLimiter(f *testing.F) {
	f.Add(float64(1), 1, 0)
	f.Add(float64(100), 10, 5)
	f.Add(float64(-1), 0, 1)
	f.Fuzz(func(t *testing.T, rate float64, burst int, n int) {
		l, err := NewTokenBucket(rate, burst)
		if err != nil {
			return
		}
		_ = l.AllowN(n)
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		_ = l.WaitN(ctx, n)
		_ = l.Rate()
		_ = l.Burst()
	})
}
