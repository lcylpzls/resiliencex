package resiliencex_test

import (
	"testing"
	"time"

	"github.com/lcylpzls/resiliencex"
)

// TestPublicAPI 黑盒冒烟测试：覆盖根包全部转发函数、类型别名与常量。
func TestPublicAPI(t *testing.T) {
	if resiliencex.Version != "v1.4.1" {
		t.Fatalf("Version = %s", resiliencex.Version)
	}

	fw, err := resiliencex.NewFixedWindow(10, time.Second,
		resiliencex.WithMetrics(nil),
		resiliencex.WithLogger(nil),
		resiliencex.WithTraceHook(nil),
	)
	if err != nil || fw == nil {
		t.Fatalf("NewFixedWindow 失败：%v", err)
	}
	sw, err := resiliencex.NewSlidingWindow(10, time.Second)
	if err != nil || sw == nil {
		t.Fatalf("NewSlidingWindow 失败：%v", err)
	}
	lb, err := resiliencex.NewTokenBucket(1, 1)
	if err != nil || lb == nil {
		t.Fatalf("NewTokenBucket 失败：%v", err)
	}
	bh, err := resiliencex.NewBulkhead(2)
	if err != nil || bh == nil {
		t.Fatalf("NewBulkhead 失败：%v", err)
	}
	cb, err := resiliencex.NewCircuitBreaker(
		resiliencex.WithFailureThreshold(0.5),
		resiliencex.WithMinRequests(5),
		resiliencex.WithOpenTimeout(time.Second),
		resiliencex.WithHalfOpenMax(2),
		resiliencex.WithWindow(time.Second, 10),
		resiliencex.WithOnStateChange(func(from, to resiliencex.State) {}),
	)
	if err != nil || cb == nil {
		t.Fatalf("NewCircuitBreaker 失败：%v", err)
	}

	_ = resiliencex.ErrRateLimited()
	_ = resiliencex.ErrCircuitOpen()
	_ = resiliencex.StateClosed
	_ = resiliencex.StateOpen
	_ = resiliencex.StateHalfOpen
	_ = resiliencex.CodeInvalidConfig
	_ = resiliencex.CodeRateLimited
	_ = resiliencex.CodeCircuitOpen
	_ = resiliencex.CodeBulkheadFull
	_ = resiliencex.CodeWaitCanceled

	var _ resiliencex.Window
	var _ resiliencex.Option
	var _ resiliencex.Metrics
	var _ resiliencex.Limiter
	var _ resiliencex.State
	var _ resiliencex.Counts
	var _ resiliencex.CircuitBreaker
	var _ resiliencex.Bulkhead
	var _ resiliencex.TraceAttr
	var _ resiliencex.TraceHook
}
