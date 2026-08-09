package resiliencex

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/lcylpzls/errx"
	"github.com/lcylpzls/logx"
)

// 熔断器默认参数与窗口默认值。
const (
	defaultFailureThreshold = 0.5
	defaultMinRequests      = 5
	defaultOpenTimeout      = 10 * time.Second
	defaultHalfOpenMax      = 1
	defaultSlotDuration     = time.Second
	defaultWindowSize       = 10
)

// State 是熔断器状态。
type State uint8

const (
	// StateClosed 关闭:请求放行,统计失败率。
	StateClosed State = iota
	// StateOpen 打开:拒绝全部请求,等待探测超时。
	StateOpen
	// StateHalfOpen 半开:允许少量探测请求。
	StateHalfOpen
)

// String 返回状态的稳定名称。
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// Counts 是熔断器窗口统计快照。
type Counts struct {
	// Requests 窗口内请求总数。
	Requests uint64
	// Successes 窗口内成功数。
	Successes uint64
	// Failures 窗口内失败数。
	Failures uint64
}

// windowCount 是单个时间片的计数。
type windowCount struct {
	requests  uint64
	successes uint64
	failures  uint64
}

// slidingWindow 是固定时间片滑动窗口计数。
type slidingWindow struct {
	slot    time.Duration
	size    int
	now     func() time.Time
	start   time.Time
	buckets []windowCount
	total   windowCount
}

func newSlidingWindow(slot time.Duration, size int, now func() time.Time) *slidingWindow {
	return &slidingWindow{
		slot:    slot,
		size:    size,
		now:     now,
		start:   now(),
		buckets: make([]windowCount, size),
	}
}

// add 记录一次结果(成功或失败)。
func (w *slidingWindow) add(success bool) {
	now := w.now()
	w.advance(now)
	idx := int(now.Sub(w.start) / w.slot)
	b := &w.buckets[idx]
	b.requests++
	w.total.requests++
	if success {
		b.successes++
		w.total.successes++
	} else {
		b.failures++
		w.total.failures++
	}
}

// advance 移除过期时间片(调用方须持有窗口锁)。
func (w *slidingWindow) advance(now time.Time) {
	span := time.Duration(w.size) * w.slot
	for now.Sub(w.start) >= span {
		old := w.buckets[0]
		w.total.requests -= old.requests
		w.total.successes -= old.successes
		w.total.failures -= old.failures
		copy(w.buckets, w.buckets[1:])
		w.buckets[w.size-1] = windowCount{}
		w.start = w.start.Add(w.slot)
	}
}

// circuitConfig 是熔断器配置。
type circuitConfig struct {
	failureThreshold float64
	minRequests      int
	openTimeout      time.Duration
	halfOpenMax      int
	slotDuration     time.Duration
	windowSize       int
	onStateChange    func(State, State)
	metrics          Metrics
	logger           logx.Logger
	traceHook        TraceHook
	now              func() time.Time
}

func defaultCircuitConfig() *circuitConfig {
	return &circuitConfig{
		failureThreshold: defaultFailureThreshold,
		minRequests:      defaultMinRequests,
		openTimeout:      defaultOpenTimeout,
		halfOpenMax:      defaultHalfOpenMax,
		slotDuration:     defaultSlotDuration,
		windowSize:       defaultWindowSize,
		now:              time.Now,
	}
}

func (c *circuitConfig) setMetrics(m Metrics) { c.metrics = m }
func (c *circuitConfig) setLogger(l logx.Logger) {
	c.logger = l
}

// WithFailureThreshold 设置失败率阈值(0,1],默认 0.5。
func WithFailureThreshold(ratio float64) Option {
	return func(c configApplier) {
		if cc, ok := c.(*circuitConfig); ok {
			cc.failureThreshold = ratio
		}
	}
}

// WithMinRequests 设置触发熔断的最小请求数,默认 5。
func WithMinRequests(n int) Option {
	return func(c configApplier) {
		if cc, ok := c.(*circuitConfig); ok {
			cc.minRequests = n
		}
	}
}

// WithOpenTimeout 设置熔断打开后的探测等待时长,默认 10s。
func WithOpenTimeout(d time.Duration) Option {
	return func(c configApplier) {
		if cc, ok := c.(*circuitConfig); ok {
			cc.openTimeout = d
		}
	}
}

// WithHalfOpenMax 设置半开状态允许的探测请求数,默认 1。
func WithHalfOpenMax(n int) Option {
	return func(c configApplier) {
		if cc, ok := c.(*circuitConfig); ok {
			cc.halfOpenMax = n
		}
	}
}

// WithWindow 设置滑动窗口时间片与数量(默认 1s × 10)。
func WithWindow(slot time.Duration, size int) Option {
	return func(c configApplier) {
		if cc, ok := c.(*circuitConfig); ok {
			cc.slotDuration = slot
			cc.windowSize = size
		}
	}
}

// WithOnStateChange 设置状态切换回调(锁外调用)。
func WithOnStateChange(fn func(from, to State)) Option {
	return func(c configApplier) {
		if cc, ok := c.(*circuitConfig); ok {
			cc.onStateChange = fn
		}
	}
}

// WithTraceHook 设置受保护调用链路追踪钩子。
func WithTraceHook(h TraceHook) Option {
	return func(c configApplier) {
		if cc, ok := c.(*circuitConfig); ok {
			cc.traceHook = h
		}
	}
}

// CircuitBreaker 是三态熔断器:Closed / Open / HalfOpen。
type CircuitBreaker struct {
	mu              sync.Mutex
	cfg             *circuitConfig
	state           State
	window          *slidingWindow
	openUntil       time.Time
	halfOpenAllowed int
	halfOpenSuccess int
}

// NewCircuitBreaker 创建熔断器。配置非法返回 RESX_INVALID_CONFIG。
func NewCircuitBreaker(opts ...Option) (*CircuitBreaker, error) {
	cfg := defaultCircuitConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	if err := validateCircuitConfig(cfg); err != nil {
		return nil, err
	}
	return &CircuitBreaker{
		cfg:    cfg,
		state:  StateClosed,
		window: newSlidingWindow(cfg.slotDuration, cfg.windowSize, cfg.now),
	}, nil
}

// validateCircuitConfig 校验熔断器配置。
func validateCircuitConfig(cfg *circuitConfig) error {
	if math.IsNaN(cfg.failureThreshold) || math.IsInf(cfg.failureThreshold, 0) ||
		cfg.failureThreshold <= 0 || cfg.failureThreshold > 1 {
		return errx.NewCode(CodeInvalidConfig, "失败率阈值必须在 (0,1]")
	}
	if cfg.minRequests < 1 {
		return errx.NewCode(CodeInvalidConfig, "最小请求数必须大于等于 1")
	}
	if cfg.openTimeout <= 0 {
		return errx.NewCode(CodeInvalidConfig, "打开超时必须为正数")
	}
	if cfg.halfOpenMax < 1 {
		return errx.NewCode(CodeInvalidConfig, "半开探测数必须大于等于 1")
	}
	if cfg.slotDuration <= 0 || cfg.windowSize < 1 {
		return errx.NewCode(CodeInvalidConfig, "窗口参数非法")
	}
	return nil
}

// Allow 检查请求是否可放行。
// Closed 放行;Open 拒绝并在超时后转入 HalfOpen;HalfOpen 限量放行探测。
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	now := cb.cfg.now()
	switch cb.state {
	case StateClosed:
		cb.window.advance(now)
		cb.mu.Unlock()
		cb.emitAccepted()
		return nil
	case StateOpen:
		if now.Before(cb.openUntil) {
			cb.mu.Unlock()
			cb.emitRejected()
			return errx.NewCode(CodeCircuitOpen, "熔断器已打开")
		}
		from := cb.switchStateLocked(StateHalfOpen)
		cb.halfOpenAllowed = 0
		cb.halfOpenSuccess = 0
		cb.halfOpenAllowed++ // 放行首个探测请求
		cb.mu.Unlock()
		cb.notifyStateChange(from, StateHalfOpen)
		cb.emitAccepted()
		return nil
	case StateHalfOpen:
		if cb.halfOpenAllowed >= cb.cfg.halfOpenMax {
			cb.mu.Unlock()
			cb.emitRejected()
			return errx.NewCode(CodeCircuitOpen, "熔断器探测中")
		}
		cb.halfOpenAllowed++
		cb.mu.Unlock()
		cb.emitAccepted()
		return nil
	default:
		cb.mu.Unlock()
		cb.emitRejected()
		return errx.NewCodef(CodeCircuitOpen, "未知熔断状态 %v", cb.state)
	}
}

// Success 上报一次成功。
// Closed 计入窗口;HalfOpen 累计探测成功,达到阈值后回到 Closed。
func (cb *CircuitBreaker) Success() {
	cb.mu.Lock()
	var from, to State
	switched := false
	switch cb.state {
	case StateClosed:
		cb.window.add(true)
	case StateHalfOpen:
		cb.halfOpenSuccess++
		if cb.halfOpenSuccess >= cb.cfg.halfOpenMax {
			from = cb.switchStateLocked(StateClosed)
			to = StateClosed
			switched = true
			cb.window = newSlidingWindow(cb.cfg.slotDuration, cb.cfg.windowSize, cb.cfg.now)
		}
	}
	cb.mu.Unlock()
	if switched {
		cb.notifyStateChange(from, to)
	}
}

// Failure 上报一次失败。
// Closed 计入窗口,失败率达标后转入 Open;HalfOpen 任一失败立即 Open。
func (cb *CircuitBreaker) Failure() {
	cb.mu.Lock()
	var from, to State
	switched := false
	switch cb.state {
	case StateClosed:
		cb.window.add(false)
		if cb.shouldOpenLocked() {
			from = cb.switchStateLocked(StateOpen)
			to = StateOpen
			switched = true
			cb.openUntil = cb.cfg.now().Add(cb.cfg.openTimeout)
		}
	case StateHalfOpen:
		from = cb.switchStateLocked(StateOpen)
		to = StateOpen
		switched = true
		cb.openUntil = cb.cfg.now().Add(cb.cfg.openTimeout)
	}
	cb.mu.Unlock()
	if switched {
		cb.notifyStateChange(from, to)
	}
}

// Execute 执行并自动上报成功/失败。
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if fn == nil {
		return errx.NewCode(CodeInvalidConfig, "执行函数不能为空")
	}
	return cb.ExecuteContext(context.Background(), func(context.Context) error { return fn() })
}

// ExecuteContext 以受保护方式执行 fn，自动熔断并记录链路 span。
func (cb *CircuitBreaker) ExecuteContext(ctx context.Context, fn func(context.Context) error) error {
	if fn == nil {
		return errx.NewCode(CodeInvalidConfig, "执行函数不能为空")
	}
	traceCtx, end := cb.startTrace(ctx)
	if err := cb.Allow(); err != nil {
		end(err)
		return err
	}
	if err := fn(traceCtx); err != nil {
		cb.Failure()
		end(err)
		return err
	}
	cb.Success()
	end(nil)
	return nil
}

// startTrace 开始受保护调用链路（无钩子时 no-op）。
func (cb *CircuitBreaker) startTrace(ctx context.Context) (context.Context, func(error)) {
	if cb.cfg.traceHook == nil {
		return ctx, func(error) {}
	}
	return cb.cfg.traceHook.Start(ctx, "resiliencex.circuit_breaker",
		TraceAttr{Key: "resiliencex.type", Value: "circuit_breaker"},
		TraceAttr{Key: "resiliencex.state", Value: cb.State().String()},
	)
}

// State 返回当前状态。
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Counts 返回窗口统计快照。
func (cb *CircuitBreaker) Counts() Counts {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.window.advance(cb.cfg.now())
	return Counts{
		Requests:  cb.window.total.requests,
		Successes: cb.window.total.successes,
		Failures:  cb.window.total.failures,
	}
}

// shouldOpenLocked 判断失败率是否达到熔断条件(调用方须持有锁)。
func (cb *CircuitBreaker) shouldOpenLocked() bool {
	total := cb.window.total.requests
	if total < uint64(cb.cfg.minRequests) || total == 0 {
		return false
	}
	return float64(cb.window.total.failures)/float64(total) >= cb.cfg.failureThreshold
}

// switchStateLocked 切换状态并返回原状态(调用方须持有锁)。
func (cb *CircuitBreaker) switchStateLocked(to State) State {
	from := cb.state
	cb.state = to
	return from
}

// notifyStateChange 通知状态切换(事件、日志与指标)。
func (cb *CircuitBreaker) notifyStateChange(from, to State) {
	if cb.cfg.onStateChange != nil {
		cb.cfg.onStateChange(from, to)
	}
	if cb.cfg.logger != nil {
		cb.cfg.logger.Warn("熔断器状态切换",
			logx.Fields(logx.String("from", from.String()), logx.String("to", to.String())))
	}
	if cb.cfg.metrics != nil {
		cb.cfg.metrics.IncCounter(metricStateChange, from.String(), to.String())
	}
}

// emitAccepted 输出放行指标。
func (cb *CircuitBreaker) emitAccepted() {
	if cb.cfg.metrics != nil {
		cb.cfg.metrics.IncCounter(metricCBAccepted)
	}
}

// emitRejected 输出拒绝指标。
func (cb *CircuitBreaker) emitRejected() {
	if cb.cfg.metrics != nil {
		cb.cfg.metrics.IncCounter(metricCBRejected)
	}
}
