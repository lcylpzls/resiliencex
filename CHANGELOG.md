# 更新日志

本项目遵循[语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 规划

- 完成调研、PRD、架构、API 草案、ADR 与迭代计划。

## [v0.1.0] - 2026-08-09

### 新增

- 令牌桶限流器:NewTokenBucket(rate, burst),惰性补充零后台任务;
- Allow / AllowN 非阻塞,Wait / WaitN 阻塞(ctx 取消感知,预定归还);
- 单次请求超过桶容量明确报错;
- Metrics 注入(accepted / rejected);通用 Option 机制;
- RESX_* 错误码(errx),限流拒绝 KindRateLimited;
- 零第三方依赖;覆盖率 100%,race / vet / staticcheck / fuzz / vuln 全绿;
- Allow 命中基准 10.6ns,0 分配。

## [v0.2.0] - 2026-08-09

### 新增

- 熔断器:Closed / Open / HalfOpen 三态状态机;
- 滑动窗口统计(默认 1s × 10,可配),失败率 + 最小请求数触发;
- HalfOpen 探测(可配探测数),全成功恢复、任一失败重开;
- Allow / Success / Failure / Execute / State / Counts;
- OnStateChange 事件(锁外调用)、logx 状态 Warn、Metrics 全接入;
- 熔断拒绝 KindUnavailable + RESX_CIRCUIT_OPEN;
- 覆盖率 100%,race / vet / staticcheck / fuzz / vuln 全绿。

## [v0.3.0] - 2026-08-09

### 新增

- 舱壁:Bulkhead 信号量隔离,TryAcquire / Acquire(ctx) /
  release(每个许可独立幂等)/ Available;
- 窗口限流:NewFixedWindow / NewSlidingWindow(环形时间戳,
  任意连续窗口精确计数);
- 窗口 Wait:轮询等待(ctx 取消感知,间隔窗口 1/10);
- 舱壁等待取消 RESX_WAIT_CANCELED;
- Metrics 全接入(bulkhead accepted/rejected);
- 覆盖率 100%,race / vet / staticcheck / fuzz / vuln 全绿。

## [v0.4.0] - 2026-08-09

### 新增

- 观测完善:限流 Wait 耗时观测(resx.limiter.wait_duration),
  窗口拒绝计数(resx.window.rejected),窗口支持 Metrics/Logger 注入;
- ErrRateLimited 辅助错误(统一 KindRateLimited + RESX_RATE_LIMITED);
- 组合示例 examples/gateway:限流 + 熔断 + 舱壁串联调用下游;
- CI 新增 examples job;
- 修复:固定窗口起始时间统一使用注入时钟(CI 时序暴露);
- 覆盖率 100%,race / vet / staticcheck / fuzz / vuln 全绿。

## [v0.5.0] - 2026-08-09

### 性能

- 全组件基准:限流 Allow 10ns、熔断 Allow 8ns、窗口 Allow 8ns,
  热路径 0 分配;
- 与 x/time/rate 对比:Allow 2.9x 快、Wait 4.4x 快;
- 与 gobreaker 对比:Execute 相当(23.3 vs 23.6ns);
- docs/performance.md 基准与对比方法;
- 覆盖率 100%,race / vet / staticcheck / fuzz / vuln 全绿。

## [v0.6.0] - 2026-08-09

### 治理与文档

- SECURITY.md、CODEOWNERS、CONTRIBUTING、issue/PR 模板;
- operations(组件速查/参数表)、quality(质量门槛)、release(发布流程)、
  comparison(与治理库对比)文档;
- ErrCircuitOpen 辅助错误;Version 常量;
- CI fuzz 扩展至 4 个目标(limiter/circuitbreaker/bulkhead/window);
- 覆盖率 100%,race / vet / staticcheck / fuzz / vuln 全绿。

## [v0.7.0] - 2026-08-09

### 新增

- Limiter.SetRate:运行期动态调整限流速率
  (支持基于下游健康度调整),非法速率不改原值;
- 覆盖率 100%,race / vet / staticcheck / fuzz / vuln 全绿。
