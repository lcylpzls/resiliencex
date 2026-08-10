package resiliencex

import (
	"context"
	"sync"

	"github.com/lcylpzls/errx"
	"github.com/lcylpzls/logx"
)

// bulkheadConfig 是舱壁配置。
type bulkheadConfig struct {
	maxConcurrent int
	metrics       Metrics
	logger        logx.Logger
}

func (c *bulkheadConfig) setMetrics(m Metrics) { c.metrics = m }
func (c *bulkheadConfig) setLogger(l logx.Logger) {
	c.logger = l
}

// Bulkhead 是信号量舱壁:限制并发在途请求数。
type Bulkhead struct {
	cfg *bulkheadConfig
	sem chan struct{}
}

// NewBulkhead 创建舱壁,限制 maxConcurrent 个并发。
func NewBulkhead(maxConcurrent int, opts ...Option) (*Bulkhead, error) {
	if maxConcurrent < 1 {
		return nil, errx.NewCode(CodeInvalidConfig, "maxConcurrent 必须大于等于 1")
	}
	cfg := &bulkheadConfig{maxConcurrent: maxConcurrent}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	return &Bulkhead{cfg: cfg, sem: make(chan struct{}, maxConcurrent)}, nil
}

// TryAcquire 非阻塞获取许可;成功返回 release 函数(幂等),
// 失败返回 ok=false。
func (b *Bulkhead) TryAcquire() (release func(), ok bool) {
	select {
	case b.sem <- struct{}{}:
		b.emitAccepted()
		return b.releaseOnce(), true
	default:
		b.emitRejected()
		return nil, false
	}
}

// Acquire 阻塞获取许可,ctx 取消立即返回。
func (b *Bulkhead) Acquire(ctx context.Context) (func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case b.sem <- struct{}{}:
		b.emitAccepted()
		return b.releaseOnce(), nil
	case <-ctx.Done():
		b.emitRejected()
		return nil, errx.WrapCode(ctx.Err(), CodeWaitCanceled, "等待舱壁许可被取消")
	}
}

// Available 返回当前可用许可数。
func (b *Bulkhead) Available() int {
	return cap(b.sem) - len(b.sem)
}

// releaseOnce 返回幂等的许可归还函数(每个许可独立计数)。
func (b *Bulkhead) releaseOnce() func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			<-b.sem
		})
	}
}

// emitAccepted 输出放行指标。
func (b *Bulkhead) emitAccepted() {
	if b.cfg.metrics != nil {
		b.cfg.metrics.IncCounter(metricBulkheadAccepted, nil)
	}
}

// emitRejected 输出拒绝指标。
func (b *Bulkhead) emitRejected() {
	if b.cfg.metrics != nil {
		b.cfg.metrics.IncCounter(metricBulkheadRejected, nil)
	}
}
