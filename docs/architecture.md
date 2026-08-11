# resiliencex 架构设计

> 状态:已实现(v1.4.1),本文描述当前架构;公开 API 以 `go doc` 与 README 为准。

## 1. 总体分层

```text
业务代码
├── limiter(令牌桶:惰性补充 + Allow/Wait)
├── circuitbreaker(三态状态机 + 滑动窗口统计)
├── bulkhead(信号量隔离)
├── window(固定/滑动窗口计数限流)
└── 观测层(Metrics 注入 + 事件回调 + logx 日志)
```

## 2. 核心模块职责

| 模块 | 职责 |
| --- | --- |
| `limiter.go` | 令牌桶:状态、补充、Allow / Wait |
| `circuitbreaker.go` | 三态状态机、统计、探测、事件 |
| `bulkhead.go` | 信号量隔离 |
| `window.go` | 固定 / 滑动窗口计数 |
| `metrics.go` | Metrics 接口与注入 |
| `errors.go` | RESX_* 错误码 |
| `options.go` | 各组件配置与校验 |

## 3. 限流器(令牌桶)

- 桶容量 = burst,初始满桶;
- 每次操作按时间差惰性补充 `rate * elapsed`,上限 burst;
- AllowN:桶余量不足返回 false(不阻塞);
- WaitN:不足则计算等待时长,select ctx.Done / timer;
- 无后台 goroutine,零资源占用。

## 4. 熔断器

- **Closed**:请求放行,滑动窗口记录成功/失败;
  失败率 ≥ 阈值且请求数 ≥ 最小请求数 → Open;
- **Open**:拒绝全部请求;OpenTimeout 后进入 HalfOpen;
- **HalfOpen**:允许有限探测请求;全部成功 → Closed,
  任一失败 → Open(重置计时);
- 统计:固定大小环形窗口,按时间桶聚合,过期桶清除;
- 并发安全:状态与计数互斥保护。

## 5. 舱壁

- 计数信号量(非阻塞 TryAcquire + 阻塞 Acquire(ctx));
- Acquire 返回 release 函数(幂等);
- ctx 取消返回 KindCancelled 包装错误。

## 6. 观测

- Metrics:组件名 + 结果标签(accepted / rejected / state);
- 事件:OnStateChange(熔断);
- 日志:状态切换 Warn(注入 logx 时);
- 默认 no-op,热路径仅原子计数。

## 7. 错误模型

| 场景 | Kind | 错误码 |
| --- | --- | --- |
| 限流拒绝 | KindRateLimited | RESX_RATE_LIMITED |
| 熔断拒绝 | KindUnavailable | RESX_CIRCUIT_OPEN |
| 舱壁拒绝 | KindQuotaExceeded | RESX_BULKHEAD_FULL |
| 配置非法 | KindInvalid | RESX_INVALID_CONFIG |

## 8. 依赖策略

- 仅标准库 + errx + logx(可选观测),零第三方。
