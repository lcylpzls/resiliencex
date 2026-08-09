# resiliencex 与治理库对比

> 更新日期:2026-08-09 · resiliencex v0.5.0

## 1. 性能(同机实测)

| 场景 | resiliencex | 竞品 | 倍率 |
| --- | --- | --- | --- |
| Limiter Allow | 10.2 ns | x/time/rate 29.9 ns | 2.9x 快 |
| Limiter Wait | 10.5 ns | x/time/rate 46.9 ns | 4.4x 快 |
| 熔断 Execute | 23.3 ns | gobreaker 23.6 ns | 相当 |

## 2. 功能点

| 功能 | resiliencex | x/time/rate | gobreaker | sentinel | resilience4j |
| --- | --- | --- | --- | --- | --- |
| 令牌桶限流 | ✅ Allow/Wait | ✅ 含 Reserve | ❌ | ✅ | ✅ |
| 窗口限流 | ✅ 固定/滑动 | ❌ | ❌ | ✅ | ✅ |
| 熔断三态 | ✅ | ❌ | ✅ | ✅ | ✅ |
| 滑动窗口统计 | ✅ 时间片 | ❌ | ✅ | ✅ | ✅ |
| HalfOpen 探测 | ✅ 可配 | ❌ | ✅ | ✅ | ✅ |
| 舱壁 | ✅ 信号量 | ❌ | ❌ | ✅ 并发数 | ✅ |
| 事件回调 | ✅ 状态切换 | ❌ | ✅ | ✅ | ✅ |
| Metrics | ✅ 可插拔 | ❌ | ❌ | ✅ 内置 | ✅ 可插拔 |
| 错误语义 | ✅ errx Kind | ❌ | ❌ | ❌ | ❌ |
| 依赖 | 零第三方 | x/sys 等 | 零 | 多 | 多 |

## 3. 取舍

- **resiliencex 胜在**:薄(四组件)、快、errx 生态统一、
  零第三方、事件与指标可插拔;
- **x/time/rate 胜在**:官方维护、Reserve 语义(极少用);
- **gobreaker 胜在**:社区成熟;
- **sentinel / resilience4j 胜在**:规则中心、动态配置、多语言
  (对薄库定位过重)。

## 4. 选型建议

- 需要官方维护且接受 Reserve:x/time/rate;
- 需要规则中心 / dashboard:sentinel 或 resilience4j;
- 自用基建栈(confx/logx/errx/webx/httpx):resiliencex。
