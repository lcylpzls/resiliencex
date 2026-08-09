# 服务治理设计调研手册

> 调研日期:2026-08-09 · 范围:Go 治理库 / Java 治理标杆 / Rust 中间件
> 目的:只取设计思想,代码全部自研。

## 1. Go 生态

### 1.1 golang.org/x/time/rate(令牌桶事实标准)

**设计思想**

- 令牌桶算法:以 rate 匀速补充令牌,burst 允许瞬时突发;
- `Allow`(非阻塞)/ `Wait`(阻塞,ctx 取消感知)/ `Reserve`(预定);
- 惰性计算:无后台 goroutine,基于时间戳差值补充令牌;
- 并发安全:互斥锁保护桶状态,原子读 Limit。

**取其精华**:令牌桶的惰性补充模型(零后台任务)、ctx 感知 Wait、
Allow/Wait 双 API。

**去其糟粕**:Reserve 的取消/过期语义复杂,业务极少用。

### 1.2 go.uber.org/ratelimit

**设计思想**

- 漏桶/滑动窗口:恒定速率,限制突发;
- `Take()` 同步阻塞,简单直接。

**取其精华**:恒定速率场景的简单语义。

**去其糟粕**:无 ctx 感知;无非阻塞 API;缺少观测。

### 1.3 sony/gobreaker(熔断器主流)

**设计思想**

- 三态状态机:Closed / Open / HalfOpen;
- 滑动窗口统计成功/失败(按时间窗口切片);
- 触发条件:连续失败数或失败率阈值 + 最小请求数;
- HalfOpen 探测:允许少量请求验证下游恢复;
- 事件回调:OnStateChange。

**取其精华**:三态状态机、滑动窗口计数、最小请求数防抖、
HalfOpen 探测、状态事件。

**去其糟粕**:默认连续失败计数(对波动流量不敏感);
无观测接口;错误分类粗糙。

### 1.4 alibaba/sentinel-golang

**设计思想**

- 大而全:流控(并发/QPS)、熔断、系统自适应、热点参数;
- 规则可动态配置,统计指标丰富。

**取其精华**:失败率滑动窗口统计、最小请求数与探测语义。

**去其糟粕**:规则中心/dashboard/动态更新体系过重,
对「薄而克制」的库定位不可取。

## 2. Java 生态

### 2.1 Resilience4j(治理标杆)

**设计思想**

- 六组件:RateLimiter / CircuitBreaker / Bulkhead / Retry /
  TimeLimiter / Cache;
- 可插拔指标:micrometer 对接,事件监听器
  (onSuccess / onFailure / onStateChange);
- 舱壁:信号量隔离与固定线程池隔离两种。

**取其精华**:组件边界清晰、事件监听、指标可插拔;
熔断器的失败率 + 滑动窗口 + 探测是工业标准。

**去其糟粕**:线程池隔离在 Go 中无对应物(goroutine 廉价),
信号量隔离足够;组件间无编排(组合由调用方做)。

### 2.2 Hystrix(已停更)

**设计思想**:线程池隔离、熔断、降级、指标 dashboard 一体化。

**取其精华**:历史参照(熔断参数模型)。

**去其糟粕**:线程池隔离复杂、dashboard 依赖重、已停更。

## 3. Rust 生态

### 3.1 tower

**设计思想**:中间件可组合(重试/超时/限流/熔断各自独立 Service)。

**取其精华**:组件独立可组合的思想(Go 侧用函数/方法组合)。

**去其糟粕**:trait/Service 抽象重,Go 生态不适用。

## 4. 设计思想汇总

- 限流:令牌桶为主(平滑 + 突发),惰性补充零后台任务,
  Allow / Wait(ctx) 双 API;
- 熔断:三态状态机 + 滑动窗口失败率 + 最小请求数 +
  HalfOpen 探测 + 状态事件;
- 舱壁:信号量隔离,ctx 取消感知;
- 错误语义:限流 → KindRateLimited,熔断 → KindUnavailable,
  舱壁 → KindQuotaExceeded,统一 errx;
- 观测:Metrics 外部注入 + 事件回调 + logx 状态日志,默认 no-op;
- 组件独立,组合由调用方完成(不引入编排框架)。

## 5. 明确不采纳

- sentinel 的规则中心与动态配置;
- Hystrix 的线程池隔离与 dashboard;
- tower 的 Service 抽象;
- x/time/rate 的 Reserve 复杂语义;
- gobreaker 的默认连续失败计数(改用滑动窗口失败率)。

> 本手册为 resiliencex 设计输入,不构成对外承诺;实现时按需取舍。
