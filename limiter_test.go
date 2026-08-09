package resiliencex

import (
	"context"
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
	if err != nil {
		t.Fatal(err)
	}
	if l.Rate() != 10 || l.Burst() != 5 {
		t.Errorf("Rate=%v Burst=%d", l.Rate(), l.Burst())
	}
}

func TestAllowBurst(t *testing.T) {
	l, err := NewTokenBucket(1, 3)
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}
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

func TestAllowN(t *testing.T) {
	l, err := NewTokenBucket(1, 5)
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Wait(context.Background()); err != nil {
		t.Fatalf("满桶应立即通过:%v", err)
	}
	if err := l.WaitN(context.Background(), 0); err != nil {
		t.Fatalf("n<=0 应通过:%v", err)
	}
}

func TestWaitNBlocks(t *testing.T) {
	l, err := NewTokenBucket(100, 1)
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}
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
	if err == nil {
		t.Fatal("取消应返回错误")
	}
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
	if err != nil {
		t.Fatal(err)
	}
	err = l.WaitN(context.Background(), 3)
	if err == nil {
		t.Fatal("超过桶容量应报错")
	}
	if code, _ := errx.CodeOf(err); code != CodeInvalidConfig {
		t.Errorf("错误码 = %s,want %s", code, CodeInvalidConfig)
	}
}

func TestLimiterMetrics(t *testing.T) {
	m := newFakeMetrics()
	l, err := NewTokenBucket(1, 2, WithMetrics(m))
	if err != nil {
		t.Fatal(err)
	}
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

func TestLimiterOptions(t *testing.T) {
	logger := &fakeLogger{}
	m := newFakeMetrics()
	l, err := NewTokenBucket(1, 1, WithMetrics(m), WithLogger(logger), nil)
	if err != nil {
		t.Fatal(err)
	}
	if l.metrics != m {
		t.Error("Metrics 选项未生效")
	}
	_ = logger
}

func TestLimiterConcurrent(t *testing.T) {
	l, err := NewTokenBucket(100000, 100)
	if err != nil {
		t.Fatal(err)
	}
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
