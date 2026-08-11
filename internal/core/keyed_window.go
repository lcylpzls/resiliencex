package core

import (
	"context"
	"sync"
	"time"

	"github.com/lcylpzls/errx"
	"github.com/lcylpzls/logx"
)

// 默认按 key 窗口表容量与空闲 TTL。
const (
	defaultKeyedMaxKeys = 10000
	keyedTTLFactor      = 10
)

// keyedWindowConfig 是按 key 窗口限流配置。
type keyedWindowConfig struct {
	limit   int
	window  time.Duration
	maxKeys int
	ttl     time.Duration
	now     func() time.Time
	metrics Metrics
	logger  logx.Logger
}

// KeyedWindowOption 修改按 key 窗口限流器配置。
type KeyedWindowOption func(*keyedWindowConfig)

// WithKeyedWindowMaxKeys 设置 key 容量上限；达到上限后按最近使用淘汰。
func WithKeyedWindowMaxKeys(n int) KeyedWindowOption {
	return func(c *keyedWindowConfig) {
		if n > 0 {
			c.maxKeys = n
		}
	}
}

// WithKeyedWindowTTL 设置 key 空闲过期时长；0 表示仅按容量淘汰。
func WithKeyedWindowTTL(d time.Duration) KeyedWindowOption {
	return func(c *keyedWindowConfig) {
		if d > 0 {
			c.ttl = d
		} else {
			c.ttl = 0
		}
	}
}

// WithKeyedWindowClock 注入时间源（测试用）。
func WithKeyedWindowClock(now func() time.Time) KeyedWindowOption {
	return func(c *keyedWindowConfig) {
		if now != nil {
			c.now = now
		}
	}
}

// WithKeyedWindowMetrics 注入指标钩子。
func WithKeyedWindowMetrics(m Metrics) KeyedWindowOption {
	return func(c *keyedWindowConfig) { c.metrics = m }
}

// WithKeyedWindowLogger 注入结构化日志实现。
func WithKeyedWindowLogger(l logx.Logger) KeyedWindowOption {
	return func(c *keyedWindowConfig) { c.logger = l }
}

// keyedWindowEntry 是单个 key 的窗口与最近使用时间。
type keyedWindowEntry struct {
	w    *Window
	last time.Time
}

// KeyedWindow 是按任意 key 维护固定窗口的限流器：
// 每个 key 独立计数，带容量上限与空闲 TTL 清理，防止 key 膨胀。
type KeyedWindow struct {
	mu      sync.Mutex
	cfg     *keyedWindowConfig
	entries map[string]*keyedWindowEntry
}

// NewKeyedFixedWindow 创建按 key 固定窗口限流器。
// 每个 key 在 window 内最多 limit 次；空闲超过 TTL 或容量超限时淘汰。
func NewKeyedFixedWindow(limit int, window time.Duration, opts ...KeyedWindowOption) (*KeyedWindow, error) {
	if limit < 1 {
		return nil, errx.NewCode(CodeInvalidConfig, "limit 必须大于等于 1")
	}
	if window <= 0 {
		return nil, errx.NewCode(CodeInvalidConfig, "窗口时长必须为正数")
	}
	cfg := &keyedWindowConfig{
		limit:   limit,
		window:  window,
		maxKeys: defaultKeyedMaxKeys,
		ttl:     window * keyedTTLFactor,
		now:     time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	return &KeyedWindow{
		cfg:     cfg,
		entries: make(map[string]*keyedWindowEntry),
	}, nil
}

// Allow 尝试通过指定 key 的窗口；超限返回 false。
func (k *KeyedWindow) Allow(key string) bool {
	if key == "" {
		return false
	}
	k.mu.Lock()
	now := k.cfg.now()
	e, ok := k.entries[key]
	if ok && k.cfg.ttl > 0 && now.Sub(e.last) > k.cfg.ttl {
		delete(k.entries, key)
		ok = false
	}
	if !ok {
		if len(k.entries) >= k.cfg.maxKeys {
			k.evictExpiredLocked(now)
			if len(k.entries) >= k.cfg.maxKeys {
				k.evictOldestLocked()
			}
		}
		// 构造参数已在 NewKeyedFixedWindow 校验，单窗口构造不会失败。
		w, _ := NewFixedWindow(k.cfg.limit, k.cfg.window,
			WithMetrics(k.cfg.metrics), WithLogger(k.cfg.logger))
		e = &keyedWindowEntry{w: w, last: now}
		k.entries[key] = e
	} else {
		e.last = now
	}
	k.mu.Unlock()
	return e.w.Allow()
}

// Wait 阻塞等待指定 key 的许可，ctx 取消立即返回；ctx 必须非 nil。
func (k *KeyedWindow) Wait(ctx context.Context, key string) error {
	poll := k.cfg.window / 10
	if poll < windowPollBase {
		poll = windowPollBase
	}
	if poll > windowPollMax {
		poll = windowPollMax
	}
	for {
		if k.Allow(key) {
			return nil
		}
		timer := time.NewTimer(poll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errx.WrapCode(ctx.Err(), CodeWaitCanceled, "等待窗口许可被取消")
		case <-timer.C:
		}
	}
}

// Len 返回当前维护的 key 数量。
func (k *KeyedWindow) Len() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.entries)
}

// Reset 清空全部 key 窗口。
func (k *KeyedWindow) Reset() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.entries = make(map[string]*keyedWindowEntry)
}

// evictExpiredLocked 清理空闲超时的条目（调用方持锁）。
func (k *KeyedWindow) evictExpiredLocked(now time.Time) {
	if k.cfg.ttl <= 0 {
		return
	}
	for key, e := range k.entries {
		if now.Sub(e.last) > k.cfg.ttl {
			delete(k.entries, key)
		}
	}
}

// evictOldestLocked 淘汰最近使用最早的条目（调用方持锁）。
func (k *KeyedWindow) evictOldestLocked() {
	var oldestKey string
	var oldest time.Time
	for key, e := range k.entries {
		if oldestKey == "" || e.last.Before(oldest) {
			oldestKey = key
			oldest = e.last
		}
	}
	if oldestKey != "" {
		delete(k.entries, oldestKey)
	}
}
