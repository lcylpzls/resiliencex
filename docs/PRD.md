# resiliencex 产品需求(PRD)

> 版本:v1.0.0(正式版) · 状态:已发布

## 1. 背景与动机

业务调用外部依赖时,限流、熔断、舱壁各写各的:

- 令牌桶 / 熔断状态机 / 信号量重复实现,参数随手拍;
- 拒绝语义不统一(限流 429、熔断 503、舱壁 429 混用);
- 状态变化与拒绝量不可观测,故障时只能猜。

结论:**做一个薄的服务治理组件库**,
限流 / 熔断 / 舱壁三个独立组件,错误统一 errx,观测外部注入。

## 2. 目标

1. 令牌桶限流:平滑 + 突发,Allow / Wait(ctx) 双 API;
2. 熔断器:三态状态机、滑动窗口失败率、HalfOpen 探测;
3. 舱壁:信号量隔离,ctx 取消感知;
4. 错误语义:限流 / 熔断 / 舱壁拒绝分别映射 errx Kind;
5. 观测:Metrics 注入 + 事件回调 + logx 状态日志,默认 no-op;
6. 零第三方依赖。

## 3. 非目标(明确不做)

- 不做规则中心 / 动态配置(sentinel 范畴);
- 不做组件编排框架(组合由调用方完成);
- 不做线程池隔离(Go 无需);
- 不做超时 / 重试组件(httpx 已内置重试,超时由 context 承担);
- 不做全局单例。

## 4. 能力需求

### 4.1 限流器(v0.1.0)

- `NewTokenBucket(rate float64, burst int) (*Limiter, error)`;
- `Allow()` / `AllowN(n)` 非阻塞;
- `Wait(ctx)` / `WaitN(ctx, n)` 阻塞,ctx 取消立即返回;
- 惰性补充,零后台任务;并发安全。

### 4.2 熔断器(v0.2.0)

- `NewCircuitBreaker(opts...)`;
- 三态:Closed / Open / HalfOpen;
- 滑动窗口失败率 + 最小请求数 + 探测超时 + 探测请求数;
- `Allow() error` + `Success()` / `Failure()` 手动上报;
- `Execute(fn)` 自动执行与上报;
- OnStateChange 事件回调。

### 4.3 舱壁与窗口限流(v0.3.0)

- `NewBulkhead(maxConcurrent)` + `Acquire(ctx) (release, error)`;
- `NewFixedWindow(limit, window)` / `NewSlidingWindow(limit, window)`。

### 4.4 观测(v0.4.0)

- Metrics:拒绝 / 放行 / 状态切换 / 耗时,默认 no-op;
- logx 注入:状态切换 Warn 日志;
- 组合示例:限流 + 熔断 + 舱壁串联。

## 5. 非功能需求

- **性能**:Allow / Acquire 命中路径 < 100ns,无额外分配;
- **质量**:语句覆盖率 100%、race、staticcheck、vet、fuzz、三平台 CI;
- **依赖**:标准库 + errx + logx(可选),零第三方。

## 6. 验收标准

v0.1.0 发布时:

1. 令牌桶补充 / 突发 / 等待 / 取消全路径测试;
2. 配置校验与错误码断言;
3. 100% 语句覆盖率,race / staticcheck / vet 全绿;
4. 基准与 x/time/rate 对比基线写入文档。
