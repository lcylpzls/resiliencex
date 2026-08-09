# resiliencex 架构决策记录(ADR)

> 状态说明:**已接受**(按全自动迭代授权,无需逐项确认)。

## ADR-001:令牌桶为主,惰性补充零后台任务

- **决策**:限流默认令牌桶(rate + burst),按时间差惰性补充;
- **影响**:零 goroutine 开销,平滑限流 + 允许突发。

## ADR-002:熔断三态 + 滑动窗口失败率

- **决策**:Closed / Open / HalfOpen 状态机;滑动窗口统计
  失败率 + 最小请求数 + 探测机制;
- **影响**:对波动流量比连续失败计数更敏感,防抖参数可调。

## ADR-003:舱壁用信号量,不做线程池

- **决策**:并发上限用计数信号量,Acquire 返回 release;
- **影响**:Go goroutine 场景足够,无线程池成本。

## ADR-004:拒绝语义统一 errx Kind

- **决策**:限流 KindRateLimited、熔断 KindUnavailable、
  舱壁 KindQuotaExceeded;
- **影响**:调用方按 Kind 统一处理,与 webx/httpx 生态一致。

## ADR-005:观测外部注入,默认 no-op

- **决策**:Metrics / logx / 事件回调全部可选注入;
- **影响**:热路径零隐式 I/O,与 dbx / httpx / cachex 一致。

## ADR-006:组件独立,不引入编排框架

- **决策**:限流 / 熔断 / 舱壁各自独立,组合由调用方完成;
- **影响**:API 面小,业务按需组合。
