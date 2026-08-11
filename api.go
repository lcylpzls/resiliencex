package resiliencex

import (
	"time"

	"github.com/lcylpzls/logx"
	"github.com/lcylpzls/resiliencex/internal/core"
)

const Version = core.Version

const (
	CodeInvalidConfig = core.CodeInvalidConfig
	CodeRateLimited   = core.CodeRateLimited
	CodeCircuitOpen   = core.CodeCircuitOpen
	CodeBulkheadFull  = core.CodeBulkheadFull
	CodeWaitCanceled  = core.CodeWaitCanceled
)

const (
	StateClosed   = core.StateClosed
	StateOpen     = core.StateOpen
	StateHalfOpen = core.StateHalfOpen
)

type (
	Window            = core.Window
	TraceAttr         = core.TraceAttr
	TraceHook         = core.TraceHook
	Option            = core.Option
	Metrics           = core.Metrics
	Limiter           = core.Limiter
	State             = core.State
	Counts            = core.Counts
	CircuitBreaker    = core.CircuitBreaker
	Bulkhead          = core.Bulkhead
	KeyedWindow       = core.KeyedWindow
	KeyedWindowOption = core.KeyedWindowOption
)

func NewFixedWindow(limit int, window time.Duration, opts ...Option) (*Window, error) {
	return core.NewFixedWindow(limit, window, opts...)
}
func NewSlidingWindow(limit int, window time.Duration, opts ...Option) (*Window, error) {
	return core.NewSlidingWindow(limit, window, opts...)
}
func NewKeyedFixedWindow(limit int, window time.Duration, opts ...KeyedWindowOption) (*KeyedWindow, error) {
	return core.NewKeyedFixedWindow(limit, window, opts...)
}
func NewTokenBucket(rate float64, burst int, opts ...Option) (*Limiter, error) {
	return core.NewTokenBucket(rate, burst, opts...)
}
func NewBulkhead(maxConcurrent int, opts ...Option) (*Bulkhead, error) {
	return core.NewBulkhead(maxConcurrent, opts...)
}
func NewCircuitBreaker(opts ...Option) (*CircuitBreaker, error) {
	return core.NewCircuitBreaker(opts...)
}
func ErrRateLimited() error                          { return core.ErrRateLimited() }
func ErrCircuitOpen() error                          { return core.ErrCircuitOpen() }
func WithMetrics(m Metrics) Option                   { return core.WithMetrics(m) }
func WithLogger(l logx.Logger) Option                { return core.WithLogger(l) }
func WithKeyedWindowMaxKeys(n int) KeyedWindowOption { return core.WithKeyedWindowMaxKeys(n) }
func WithKeyedWindowTTL(d time.Duration) KeyedWindowOption {
	return core.WithKeyedWindowTTL(d)
}
func WithKeyedWindowClock(now func() time.Time) KeyedWindowOption {
	return core.WithKeyedWindowClock(now)
}
func WithKeyedWindowMetrics(m Metrics) KeyedWindowOption {
	return core.WithKeyedWindowMetrics(m)
}
func WithKeyedWindowLogger(l logx.Logger) KeyedWindowOption {
	return core.WithKeyedWindowLogger(l)
}
func WithFailureThreshold(ratio float64) Option        { return core.WithFailureThreshold(ratio) }
func WithMinRequests(n int) Option                     { return core.WithMinRequests(n) }
func WithOpenTimeout(d time.Duration) Option           { return core.WithOpenTimeout(d) }
func WithHalfOpenMax(n int) Option                     { return core.WithHalfOpenMax(n) }
func WithWindow(slot time.Duration, size int) Option   { return core.WithWindow(slot, size) }
func WithOnStateChange(fn func(from, to State)) Option { return core.WithOnStateChange(fn) }
func WithTraceHook(h TraceHook) Option                 { return core.WithTraceHook(h) }
