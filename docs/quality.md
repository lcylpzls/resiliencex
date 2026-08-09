# 质量保障

## 门槛(每个版本强制)

- 语句覆盖率 100%;
- `go vet` / `staticcheck` 零告警;
- `go test -race` 全绿;
- fuzz 目标 4 个(limiter / circuitbreaker / bulkhead / window);
- 三平台 CI(ubuntu / windows / macos)× Go 1.26;
- govulncheck 零告警;go.mod tidy 无漂移;apidiff 对比上一 tag;
- 示例全部可构建并通过 vet。

## 测试策略

- 限流:补充 / 突发 / 等待 / 取消归还 / 时钟注入;
- 熔断:三态迁移、失败率阈值、最小请求、探测成功/失败、
  事件回调、窗口滑动、未知状态防御;
- 舱壁:获取 / 拒绝 / 等待取消 / release 幂等 / 并发;
- 窗口:固定与滑动边界、环形复用、等待与取消;
- 并发:race 检测 + 时钟注入消除平台时序 flake。

## 性能

见 [performance.md](performance.md)。热路径(Allow / Wait /
CB Allow / Window Allow)要求 0 分配。

## API 兼容性

- <1.0.0 允许有意的破坏性变更,须在 CHANGELOG 说明;
- v1.0.0 起 API 冻结,破坏性变更仅随大版本。
