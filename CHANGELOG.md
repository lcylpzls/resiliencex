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
