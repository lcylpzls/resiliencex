package core

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/lcylpzls/errx"
)

// Limiter 是令牌桶限流器:
// 以 rate 匀速补充令牌,burst 允许瞬时突发,惰性计算零后台任务。
type Limiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	tokens  float64
	last    time.Time
	now     func() time.Time // 时钟注入,测试用
	metrics Metrics
}

// NewTokenBucket 创建令牌桶限流器。
// rate 为每秒补充令牌数(>0),burst 为桶容量(>=1)。
func NewTokenBucket(rate float64, burst int, opts ...Option) (*Limiter, error) {
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate <= 0 {
		return nil, errx.NewCode(CodeInvalidConfig, "rate 必须为正数")
	}
	if burst < 1 {
		return nil, errx.NewCode(CodeInvalidConfig, "burst 必须大于等于 1")
	}
	cfg := &limiterConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	now := time.Now()
	return &Limiter{
		rate:    rate,
		burst:   float64(burst),
		tokens:  float64(burst),
		last:    now,
		now:     time.Now,
		metrics: cfg.metrics,
	}, nil
}

// Rate 返回每秒补充速率(不可变)。
func (l *Limiter) Rate() float64 {
	return l.rate
}

// SetRate 动态调整补充速率(支持运行期基于下游健康度调整)。
// rate 必须为正数,非法返回 RESX_INVALID_CONFIG 且不改变原值。
func (l *Limiter) SetRate(rate float64) error {
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate <= 0 {
		return errx.NewCode(CodeInvalidConfig, "rate 必须为正数")
	}
	l.mu.Lock()
	l.rate = rate
	l.mu.Unlock()
	return nil
}

// Burst 返回桶容量(不可变)。
func (l *Limiter) Burst() int {
	return int(l.burst)
}

// RetryAfter 返回补满 1 枚令牌所需的等待时间；当前余量足够时返回 0。
// 用于生成 HTTP 429 响应头 Retry-After 等场景。
func (l *Limiter) RetryAfter() time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill(l.now())
	if l.tokens >= 1 {
		return 0
	}
	return time.Duration((1 - l.tokens) / l.rate * float64(time.Second))
}

// Allow 等价于 AllowN(1)。
func (l *Limiter) Allow() bool {
	return l.AllowN(1)
}

// AllowN 非阻塞地尝试消耗 n 个令牌;桶余量不足时返回 false。
func (l *Limiter) AllowN(n int) bool {
	if n <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill(l.now())
	if float64(n) > l.tokens {
		l.rejected()
		return false
	}
	l.tokens -= float64(n)
	l.accepted()
	return true
}

// Wait 等价于 WaitN(ctx, 1)。
func (l *Limiter) Wait(ctx context.Context) error {
	return l.WaitN(ctx, 1)
}

// WaitN 阻塞地等待 n 个令牌;ctx 取消时立即返回并归还预定。
// 单次请求超过桶容量时返回参数错误。
func (l *Limiter) WaitN(ctx context.Context, n int) error {
	if n <= 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	l.mu.Lock()
	if float64(n) > l.burst {
		l.mu.Unlock()
		return errx.NewCodef(CodeInvalidConfig,
			"单次请求 %d 超过桶容量 %d", n, int(l.burst))
	}
	wait := l.reserveN(n, l.now())
	l.mu.Unlock()
	if wait <= 0 {
		l.accepted()
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	start := time.Now()
	select {
	case <-ctx.Done():
		// 归还预定的令牌。
		l.mu.Lock()
		l.tokens += float64(n)
		l.mu.Unlock()
		return errx.WrapCode(ctx.Err(), CodeWaitCanceled, "等待限流许可被取消")
	case <-timer.C:
		if l.metrics != nil {
			l.metrics.ObserveDuration(metricLimiterWaitDur, time.Since(start).Seconds(), nil)
		}
		l.accepted()
		return nil
	}
}

// accepted 输出放行指标。
func (l *Limiter) accepted() {
	if l.metrics != nil {
		l.metrics.IncCounter(metricLimiterAccepted, nil)
	}
}

// rejected 输出拒绝指标。
func (l *Limiter) rejected() {
	if l.metrics != nil {
		l.metrics.IncCounter(metricLimiterRejected, nil)
	}
}

// refill 按时间差惰性补充令牌(调用方须持有锁)。
func (l *Limiter) refill(now time.Time) {
	elapsed := now.Sub(l.last).Seconds()
	if elapsed <= 0 {
		return
	}
	l.last = now
	l.tokens += elapsed * l.rate
	if l.tokens > l.burst {
		l.tokens = l.burst
	}
}

// reserveN 预定 n 个令牌并返回等待时长(调用方须持有锁)。
// 预定后 tokens 可为负(欠债),由后续补充先偿还。
func (l *Limiter) reserveN(n int, now time.Time) time.Duration {
	l.refill(now)
	if float64(n) <= l.tokens {
		l.tokens -= float64(n)
		return 0
	}
	wait := time.Duration((float64(n) - l.tokens) / l.rate * float64(time.Second))
	l.tokens -= float64(n)
	return wait
}
