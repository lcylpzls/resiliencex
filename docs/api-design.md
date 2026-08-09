# resiliencex API 参考

> 状态:**v1.0.0 API 已冻结**。新增组件与能力以次版本发布,
> 破坏性变更仅随主版本;任何修改须经 apidiff 对比并记录 CHANGELOG。

## 1. 快速上手

```go
// 限流:每秒 100 次,突发 20
limiter, _ := resiliencex.NewTokenBucket(100, 20)
if !limiter.Allow() {
	return errx.New(errx.KindRateLimited, resiliencex.CodeRateLimited, "请求过于频繁")
}

// 熔断:失败率 50%、最小 5 请求、探测超时 10s
cb, _ := resiliencex.NewCircuitBreaker(
	resiliencex.WithFailureThreshold(0.5),
	resiliencex.WithMinRequests(5),
	resiliencex.WithOpenTimeout(10*time.Second),
)
err := cb.Execute(func() error { return callDownstream() })

// 舱壁:最多 10 个并发
bulkhead, _ := resiliencex.NewBulkhead(10)
release, err := bulkhead.Acquire(ctx)
defer release()
```

## 2. 限流器(v0.1.0)

```go
type Limiter struct { /* 未导出 */ }

func NewTokenBucket(rate float64, burst int) (*Limiter, error)
func (l *Limiter) Allow() bool
func (l *Limiter) AllowN(n int) bool
func (l *Limiter) Wait(ctx context.Context) error
func (l *Limiter) WaitN(ctx context.Context, n int) error
func (l *Limiter) Rate() float64
func (l *Limiter) Burst() int
```

参数校验:rate > 0、burst >= 1;非法返回 RESX_INVALID_CONFIG。

## 3. 熔断器(v0.2.0)

```go
type CircuitBreaker struct { /* 未导出 */ }
type State uint8 // Closed / Open / HalfOpen

func NewCircuitBreaker(opts ...Option) (*CircuitBreaker, error)
func (cb *CircuitBreaker) Allow() error
func (cb *CircuitBreaker) Success()
func (cb *CircuitBreaker) Failure()
func (cb *CircuitBreaker) Execute(fn func() error) error
func (cb *CircuitBreaker) State() State
func (cb *CircuitBreaker) Counts() Counts // 窗口统计快照

// 选项
WithFailureThreshold(ratio float64) // (0,1],默认 0.5
WithMinRequests(n int)              // 默认 5
WithOpenTimeout(d time.Duration)    // 默认 10s
WithHalfOpenMax(n int)              // 探测请求数,默认 1
WithOnStateChange(fn func(State, State))
```

## 4. 舱壁与窗口(v0.3.0)

```go
type Bulkhead struct { /* 未导出 */ }
func NewBulkhead(maxConcurrent int) (*Bulkhead, error)
func (b *Bulkhead) TryAcquire() (release func(), ok bool)
func (b *Bulkhead) Acquire(ctx context.Context) (func(), error)
func (b *Bulkhead) Available() int

func NewFixedWindow(limit int, window time.Duration) (*Window, error)
func NewSlidingWindow(limit int, window time.Duration) (*Window, error)
type Window struct { /* 未导出 */ }
func (w *Window) Allow() bool
func (w *Window) Wait(ctx context.Context) error
```

## 5. 观测

```go
// Metrics 与 dbx / httpx / cachex 同形态。
type Metrics interface {
	IncCounter(name string, labels ...string)
	ObserveDuration(name string, seconds float64, labels ...string)
}

WithMetrics(m Metrics)
WithLogger(l logx.Logger)
```

## 6. 错误码

| 错误码 | 含义 |
| --- | --- |
| RESX_INVALID_CONFIG | 配置非法 |
| RESX_RATE_LIMITED | 限流拒绝 |
| RESX_CIRCUIT_OPEN | 熔断拒绝 |
| RESX_BULKHEAD_FULL | 舱壁拒绝 |

## 7. 迭代范围

| 版本 | 内容 |
| --- | --- |
| v0.1.0 | 令牌桶限流器 + 错误码 + Metrics 骨架 |
| v0.2.0 | 熔断器(三态 / 滑动窗口 / 事件) |
| v0.3.0 | 舱壁 + 固定 / 滑动窗口限流 |
| v0.4.0 | 观测完善 + 组合示例 |
| v0.5.0 | 性能优化 + 对比基准 |
| v0.6.0+ | 工业级打磨与自我审查 |

## 8. 组件 API 总览(v0.8.0)

### 限流器

- NewTokenBucket(rate, burst, opts...) / Rate / Burst / SetRate;
- Allow / AllowN / Wait(ctx) / WaitN(ctx, n);
- 拒绝:KindRateLimited + RESX_RATE_LIMITED(ErrRateLimited 辅助)。

### 熔断器

- NewCircuitBreaker(opts...) / Allow / Success / Failure / Execute /
  State / Counts;
- 选项:WithFailureThreshold / WithMinRequests / WithOpenTimeout /
  WithHalfOpenMax / WithWindow / WithOnStateChange / WithMetrics / WithLogger;
- 拒绝:KindUnavailable + RESX_CIRCUIT_OPEN(ErrCircuitOpen 辅助)。

### 舱壁

- NewBulkhead(max, opts...) / TryAcquire / Acquire(ctx) / Available;
- 等待取消:KindCancelled + RESX_WAIT_CANCELED。

### 窗口限流

- NewFixedWindow(limit, window, opts...) / NewSlidingWindow(...);
- Allow / Wait(ctx),布尔拒绝。

### 通用

- WithMetrics / WithLogger(跨组件复用);
- 所有阻塞 API 对 nil context 视为 Background;
- Execute 对 nil 执行函数返回 RESX_INVALID_CONFIG。
