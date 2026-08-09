# 性能基准

## 方法

```powershell
go test -run '^$' -bench . -benchmem -benchtime=1s .
```

CI 的 bench job 记录每次 main 推送的基准日志(artifact),不设硬性门禁。

## 参考数据(v0.5.0,Windows / AMD Ryzen 5 7600)

| Benchmark | ns/op | B/op | allocs/op |
| --- | --- | --- | --- |
| Limiter Allow 命中 | 10.3 | 0 | 0 |
| Limiter Wait 命中 | 10.6 | 0 | 0 |
| Limiter Allow 拒绝 | 10.0 | 0 | 0 |
| 熔断 Allow(Closed) | 8.0 | 0 | 0 |
| 舱壁 TryAcquire+release | 46.3 | 40 | 2 |
| 窗口 Allow(固定) | 8.3 | 0 | 0 |

## 与 x/time/rate、gobreaker 对比(临时工程,未入库)

对比代码位于本地 `.bench-compare/`(已 gitignore):

```powershell
cd .bench-compare
go test -run '^$' -bench . -benchmem -benchtime=1s .
```

实测(同机):

| 场景 | resiliencex | 竞品 | 倍率 |
| --- | --- | --- | --- |
| Limiter Allow | 10.2 ns | x/time/rate 29.9 ns | 2.9x 快 |
| Limiter Wait | 10.5 ns | x/time/rate 46.9 ns | 4.4x 快 |
| 熔断 Execute | 23.3 ns | gobreaker 23.6 ns | 相当 |

解读:

- x/time/rate 的 Allow 含更复杂的限速状态机(Reserve 语义),
  resiliencex 聚焦 Allow/Wait 双 API,状态更轻;
- 熔断 Execute 两者相当,gobreaker 的窗口统计与闭包开销接近;
- 舱壁 release 的幂等闭包带来 2 次分配,换取安全语义,
  非热路径可接受。

## 优化原则

- 热路径(Allow / Wait / CB Allow / Window Allow)零分配;
- 惰性计算:限流无后台任务,熔断窗口按时间片惰性滑动;
- 幂等性优先于极致性能(release 闭包)。
