package core

import (
	"context"
	testx "github.com/lcylpzls/testx"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lcylpzls/errx"
)

func TestNewTokenBucketInvalid(t *testing.T) {
	cases := []struct {
		rate  float64
		burst int
	}{
		{0, 1},
		{-1, 1},
		{math.NaN(), 1},
		{math.Inf(1), 1},
		{1, 0},
		{1, -1},
	}
	for _, c := range cases {
		_, err := NewTokenBucket(c.rate, c.burst)
		if err == nil {
			t.Errorf("rate=%v burst=%d 应报错", c.rate, c.burst)
			continue
		}
		if code, _ := errx.CodeOf(err); code != CodeInvalidConfig {
			t.Errorf("错误码 = %s,want %s", code, CodeInvalidConfig)
		}
	}
}

func TestRateBurst(t *testing.T) {
	l, err := NewTokenBucket(10, 5)
	testx.RequireNoError(t, err)

	if l.Rate() != 10 || l.Burst() != 5 {
		t.Errorf("Rate=%v Burst=%d", l.Rate(), l.Burst())
	}
}

func TestAllowBurst(t *testing.T) {
	l, err := NewTokenBucket(1, 3)
	testx.RequireNoError(t, err)

	for i := 0; i < 3; i++ {
		if !l.Allow() {
			t.Fatalf("第 %d 次应通过(突发)", i+1)
		}
	}
	if l.Allow() {
		t.Fatal("桶耗尽后应拒绝")
	}
}

func TestAllowRefill(t *testing.T) {
	l, err := NewTokenBucket(1, 1)
	testx.RequireNoError(t, err)

	t0 := time.Now()
	var now atomic.Value
	now.Store(t0)
	l.now = func() time.Time { return now.Load().(time.Time) }

	if !l.Allow() {
		t.Fatal("初始应通过")
	}
	if l.Allow() {
		t.Fatal("未补充应拒绝")
	}
	now.Store(t0.Add(time.Second))
	if !l.Allow() {
		t.Fatal("补充 1 秒后应通过")
	}
	// 长时间补充不超过 burst
	now.Store(t0.Add(100 * time.Second))
	if !l.Allow() {
		t.Fatal("长时补充应通过")
	}
	if l.Allow() {
		t.Fatal("补充后仍受 burst 限制")
	}
}

func TestRetryAfter(t *testing.T) {
	l, err := NewTokenBucket(100, 1)
	testx.RequireNoError(t, err)
	// 满桶时无需等待。
	testx.RequireEqual(t, l.RetryAfter(), time.Duration(0))
	// 消耗后需要等待约 10ms 补回 1 枚令牌。
	testx.RequireTrue(t, l.Allow())
	wait := l.RetryAfter()
	testx.RequireTrue(t, wait > 0 && wait <= 20*time.Millisecond)
	// 补充后恢复 0 等待。
	time.Sleep(50 * time.Millisecond)
	testx.RequireEqual(t, l.RetryAfter(), time.Duration(0))
}

func TestAllowN(t *testing.T) {
	l, err := NewTokenBucket(1, 5)
	testx.RequireNoError(t, err)

	if !l.AllowN(5) {
		t.Fatal("5 个令牌应通过")
	}
	if l.AllowN(1) {
		t.Fatal("耗尽后应拒绝")
	}
	if !l.AllowN(0) {
		t.Fatal("n<=0 应视为通过")
	}
	if !l.AllowN(-3) {
		t.Fatal("负 n 应视为通过")
	}
}

func TestWaitImmediate(t *testing.T) {
	l, err := NewTokenBucket(10, 10)
	testx.RequireNoError(t, err)

	if err := l.Wait(context.Background()); err != nil {
		t.Fatalf("满桶应立即通过:%v", err)
	}
	if err := l.WaitN(context.Background(), 0); err != nil {
		t.Fatalf("n<=0 应通过:%v", err)
	}
}

func TestWaitNBlocks(t *testing.T) {
	l, err := NewTokenBucket(100, 1)
	testx.RequireNoError(t, err)

	if !l.Allow() {
		t.Fatal("初始应通过")
	}
	start := time.Now()
	if err := l.WaitN(context.Background(), 1); err != nil {
		t.Fatalf("等待应通过:%v", err)
	}
	if elapsed := time.Since(start); elapsed < 5*time.Millisecond {
		t.Errorf("等待过短:%v", elapsed)
	}
	// 等待后令牌已消耗,再次 Allow 应拒绝
	if l.Allow() {
		t.Fatal("WaitN 消耗后应拒绝")
	}
}

func TestWaitCanceled(t *testing.T) {
	l, err := NewTokenBucket(1, 1)
	testx.RequireNoError(t, err)

	t0 := time.Now()
	var now atomic.Value
	now.Store(t0)
	l.now = func() time.Time { return now.Load().(time.Time) }

	if !l.Allow() {
		t.Fatal("初始应通过")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = l.WaitN(ctx, 1)
	testx.RequireError(t, err)

	if code, _ := errx.CodeOf(err); code != CodeWaitCanceled {
		t.Errorf("错误码 = %s,want %s", code, CodeWaitCanceled)
	}
	if kind := errx.KindOf(err); kind != errx.KindCancelled {
		t.Errorf("分类 = %s,want cancelled", kind)
	}
	// 取消应归还预定,桶恢复到调用前状态(tokens=0)。
	if l.Allow() {
		t.Fatal("取消后应恢复到调用前状态")
	}
	now.Store(t0.Add(time.Second))
	if !l.Allow() {
		t.Fatal("补充后应可继续通过")
	}
}

func TestWaitNExceedsBurst(t *testing.T) {
	l, err := NewTokenBucket(10, 2)
	testx.RequireNoError(t, err)

	err = l.WaitN(context.Background(), 3)
	testx.RequireError(t, err)

	if code, _ := errx.CodeOf(err); code != CodeInvalidConfig {
		t.Errorf("错误码 = %s,want %s", code, CodeInvalidConfig)
	}
}

func TestWaitNilContext(t *testing.T) {
	l, err := NewTokenBucket(100, 1)
	testx.RequireNoError(t, err)

	if !l.Allow() {
		t.Fatal("初始应通过")
	}
	//lint:ignore SA1012 有意覆盖 nil context 防护逻辑
	if err := l.Wait(nil); err != nil {
		t.Fatalf("nil ctx 应视为 Background:%v", err)
	}
}

func TestLimiterMetrics(t *testing.T) {
	m := newFakeMetrics()
	l, err := NewTokenBucket(1, 2, WithMetrics(m))
	testx.RequireNoError(t, err)

	l.Allow()
	l.Allow()
	l.Allow() // 拒绝
	if m.counter(metricLimiterAccepted) != 2 {
		t.Errorf("accepted = %d,want 2", m.counter(metricLimiterAccepted))
	}
	if m.counter(metricLimiterRejected) != 1 {
		t.Errorf("rejected = %d,want 1", m.counter(metricLimiterRejected))
	}
}

func TestLimiterWaitDurationMetric(t *testing.T) {
	m := newFakeMetrics()
	l, err := NewTokenBucket(100, 1, WithMetrics(m))
	testx.RequireNoError(t, err)

	if !l.Allow() {
		t.Fatal("初始应通过")
	}
	if err := l.WaitN(context.Background(), 1); err != nil {
		t.Fatalf("等待应通过:%v", err)
	}
	if m.durationCount(metricLimiterWaitDur) != 1 {
		t.Errorf("wait_duration 未记录:%d", m.durationCount(metricLimiterWaitDur))
	}
}

func TestErrRateLimited(t *testing.T) {
	err := ErrRateLimited()
	if code, _ := errx.CodeOf(err); code != CodeRateLimited {
		t.Errorf("错误码 = %s,want %s", code, CodeRateLimited)
	}
	if kind := errx.KindOf(err); kind != errx.KindRateLimited {
		t.Errorf("分类 = %s", kind)
	}
}

func TestErrCircuitOpen(t *testing.T) {
	err := ErrCircuitOpen()
	if code, _ := errx.CodeOf(err); code != CodeCircuitOpen {
		t.Errorf("错误码 = %s,want %s", code, CodeCircuitOpen)
	}
	if kind := errx.KindOf(err); kind != errx.KindUnavailable {
		t.Errorf("分类 = %s", kind)
	}
}

func TestVersion(t *testing.T) {
	testx.Equal(t, Version, "v1.5.1")

}

func TestSetRate(t *testing.T) {
	l, err := NewTokenBucket(1, 1)
	testx.RequireNoError(t, err)

	if err := l.SetRate(10); err != nil {
		t.Fatalf("SetRate 失败:%v", err)
	}
	if l.Rate() != 10 {
		t.Errorf("Rate = %v,want 10", l.Rate())
	}
	// 非法速率不改变原值
	if err := l.SetRate(0); err == nil {
		t.Error("rate=0 应非法")
	}
	if err := l.SetRate(math.NaN()); err == nil {
		t.Error("NaN 应非法")
	}
	if l.Rate() != 10 {
		t.Error("非法 SetRate 不应改变速率")
	}
	// 速率变化生效:注入时钟验证补充量
	t0 := time.Now()
	var now atomic.Value
	now.Store(t0)
	l.now = func() time.Time { return now.Load().(time.Time) }
	if !l.Allow() {
		t.Fatal("初始应通过")
	}
	now.Store(t0.Add(100 * time.Millisecond))
	if !l.Allow() {
		t.Fatal("rate=10 时 100ms 应补充 1 个令牌")
	}
}

func TestLimiterOptions(t *testing.T) {
	logger := &fakeLogger{}
	m := newFakeMetrics()
	l, err := NewTokenBucket(1, 1, WithMetrics(m), WithLogger(logger), nil)
	testx.RequireNoError(t, err)

	testx.Equal(t, l.metrics, m)

	_ = logger
}

func TestLimiterConcurrent(t *testing.T) {
	l, err := NewTokenBucket(100000, 100)
	testx.RequireNoError(t, err)

	var accepted, rejected atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				if l.Allow() {
					accepted.Add(1)
				} else {
					rejected.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	if accepted.Load()+rejected.Load() != 8000 {
		t.Errorf("总调用数不符:%d+%d", accepted.Load(), rejected.Load())
	}
}

func TestErrorCodesRegistered(t *testing.T) {
	codes := []errx.Code{CodeInvalidConfig, CodeRateLimited, CodeCircuitOpen, CodeBulkheadFull, CodeWaitCanceled}
	for _, c := range codes {
		if errx.Describe(c) == "" {
			t.Errorf("错误码 %s 未注册", c)
		}
	}
}
