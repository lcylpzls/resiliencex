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
