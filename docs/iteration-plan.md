# resiliencex 迭代计划与质量门槛

## 1. 迭代阶段

### P0 项目骨架

- go.mod(module github.com/lcylpzls/resiliencex,go 1.26)、目录、CI
  (三平台 + staticcheck + govulncheck + tidy + apidiff + bench)。

### P1 v0.1.0 令牌桶限流器

- NewTokenBucket、Allow / AllowN / Wait / WaitN、Rate / Burst;
- ctx 取消、惰性补充、配置校验、Metrics 骨架。

### P2 v0.2.0 熔断器

- 三态状态机、滑动窗口统计、探测、Allow / Success / Failure /
  Execute、OnStateChange。

### P3 v0.3.0 舱壁与窗口限流

- Bulkhead 信号量;FixedWindow / SlidingWindow。

### P4 v0.4.0 观测与示例

- Metrics 全接入、logx 状态日志;组合示例(限流 + 熔断 + 舱壁)。

### P5 v0.5.0 性能

- 原子计数、无锁快路径;与 x/time/rate、gobreaker 对比基准。

### P6 v0.6.0+ 工业级打磨与自我审查

- 治理文件、运行/质量/发布文档;
- 持续自审:并发边界、统计正确性、性能、文档完备性;
- 成熟后停下征询用户是否发布 1.0.0。

### P6 执行记录

- v0.6.0:治理文件与运行/质量/发布/对比文档,CI fuzz ×4;
- v0.7.0:动态限流 SetRate;
- v0.8.0:nil context 防护(限流/舱壁/窗口)、Execute nil 函数防护,
  API 参考与稳定性承诺完备。

> 状态:组件能力与质量门槛齐备,等待 1.0.0 决策。

## 2. 质量门槛(每阶段强制)

- 语句覆盖率 100%;`go vet` / `staticcheck` 零告警;
- `go test -race` 全绿;fuzz 至少 2 个目标;
- 三平台 CI × Go 1.26;govulncheck 零告警;
- go.mod tidy 漂移检查;apidiff 对比上一 tag;
- 所有日志、注释、文档使用简体中文。

## 3. 性能目标

| 场景 | 目标 |
| --- | --- |
| Allow 命中 | < 100ns,0 分配 |
| Wait 命中 | 与 Allow 同量级 |
| 熔断 Allow(Closed) | < 200ns |
| 舱壁 TryAcquire | < 100ns |

## 4. 风险与对策

| 风险 | 对策 |
| --- | --- |
| 时钟回拨导致限流异常 | 基于单调时钟(monotonic) |
| 滑动窗口统计竞争 | 时间桶粒度 + 原子计数 |
| HalfOpen 探测风暴 | 探测请求数限制 + 成功阈值 |
