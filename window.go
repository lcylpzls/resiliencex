package resiliencex

import (
	"context"
	"sync"
	"time"

	"github.com/lcylpzls/errx"
	"github.com/lcylpzls/logx"
)

// windowPollInterval 是窗口限流 Wait 的轮询间隔(取窗口的 1/10,有上限)。
const (
	windowPollBase = 10 * time.Millisecond
	windowPollMax  = 100 * time.Millisecond
)

// windowConfig 是窗口限流配置。
type windowConfig struct {
	limit   int
	window  time.Duration
	now     func() time.Time
	metrics Metrics
	logger  logx.Logger
}

func (c *windowConfig) setMetrics(m Metrics) { c.metrics = m }
func (c *windowConfig) setLogger(l logx.Logger) {
	c.logger = l
}

// Window 是固定或滑动窗口限流器。
type Window struct {
	mu      sync.Mutex
	cfg     *windowConfig
	metrics Metrics
	start   time.Time   // 固定窗口起点
	count   int         // 固定窗口计数
	times   []time.Time // 滑动窗口环形时间戳
	head    int
	size    int
}

// NewFixedWindow 创建固定窗口限流:每 window 内最多 limit 次。
func NewFixedWindow(limit int, window time.Duration, opts ...Option) (*Window, error) {
	return newWindow(limit, window, false, opts)
}

// NewSlidingWindow 创建滑动窗口限流:任意连续 window 内最多 limit 次。
func NewSlidingWindow(limit int, window time.Duration, opts ...Option) (*Window, error) {
	return newWindow(limit, window, true, opts)
}

func newWindow(limit int, window time.Duration, sliding bool, opts []Option) (*Window, error) {
	if limit < 1 {
		return nil, errx.New(errx.KindInvalid, CodeInvalidConfig, "limit 必须大于等于 1")
	}
	if window <= 0 {
		return nil, errx.New(errx.KindInvalid, CodeInvalidConfig, "窗口时长必须为正数")
	}
	cfg := &windowConfig{limit: limit, window: window, now: time.Now}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	w := &Window{
		cfg:     cfg,
		metrics: cfg.metrics,
		start:   cfg.now(),
	}
	if sliding {
		w.times = make([]time.Time, limit)
	}
	return w, nil
}

// Allow 尝试通过;窗口内超限返回 false。
func (w *Window) Allow() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := w.cfg.now()
	if w.times != nil {
		// 滑动窗口:清理过期时间戳。
		for w.size > 0 && now.Sub(w.times[w.head]) >= w.cfg.window {
			w.head = (w.head + 1) % len(w.times)
			w.size--
		}
		if w.size >= w.cfg.limit {
			w.emitRejected()
			return false
		}
		idx := (w.head + w.size) % len(w.times)
		w.times[idx] = now
		w.size++
		return true
	}
	// 固定窗口:窗口过期重置。
	if now.Sub(w.start) >= w.cfg.window {
		w.start = now
		w.count = 0
	}
	if w.count >= w.cfg.limit {
		w.emitRejected()
		return false
	}
	w.count++
	return true
}

// emitRejected 输出拒绝指标。
func (w *Window) emitRejected() {
	if w.metrics != nil {
		w.metrics.IncCounter(metricWindowRejected)
	}
}

// Wait 阻塞等待许可,ctx 取消立即返回。
// 采用轮询实现(间隔为窗口的 1/10,10-100ms)。
func (w *Window) Wait(ctx context.Context) error {
	poll := w.cfg.window / 10
	if poll < windowPollBase {
		poll = windowPollBase
	}
	if poll > windowPollMax {
		poll = windowPollMax
	}
	for {
		if w.Allow() {
			return nil
		}
		timer := time.NewTimer(poll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errx.Wrap(ctx.Err(), errx.KindCancelled, CodeWaitCanceled, "等待窗口许可被取消")
		case <-timer.C:
		}
	}
}
