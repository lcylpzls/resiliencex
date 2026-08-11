# resiliencex

自研服务治理组件库:令牌桶限流、熔断器、舱壁隔离,
拒绝语义统一 errx,观测外部注入,零第三方依赖。

> 当前状态:**v1.4.2**。

## 快速上手

```go
// 每秒 100 次,突发 20
limiter, err := resiliencex.NewTokenBucket(100, 20)
if err != nil {
	panic(err)
}
if !limiter.Allow() {
	// 拒绝,统一 errx KindRateLimited
}
limiter.SetRate(200) // 运行期动态调整速率
// 或阻塞等待(ctx 可取消)
if err := limiter.Wait(ctx); err != nil {
	// 等待被取消
}

// 熔断:失败率 50%、最小 5 请求、探测超时 10s
cb, err := resiliencex.NewCircuitBreaker(
	resiliencex.WithFailureThreshold(0.5),
	resiliencex.WithMinRequests(5),
	resiliencex.WithOpenTimeout(10*time.Second),
)
if err != nil {
	panic(err)
}
err = cb.Execute(func() error { return callDownstream() })

// 舱壁:最多 10 个并发
bulkhead, err := resiliencex.NewBulkhead(10)
if err != nil {
	panic(err)
}
release, err := bulkhead.Acquire(ctx)
if err != nil {
	return err
}
defer release()

// 窗口限流:固定或滑动窗口
window, err := resiliencex.NewSlidingWindow(100, time.Minute)
if err != nil {
	panic(err)
}
if !window.Allow() {
	// 窗口内超限
}
```

## 定位

resiliencex 不是治理编排框架,不解决「组件怎么串」的问题;它提供
每个依赖调用方都要重复的三个独立组件:

- 限流:令牌桶,平滑 + 突发,Allow / Wait(ctx) 双 API;
- 熔断:三态状态机、滑动窗口失败率、HalfOpen 探测;
- 舱壁:信号量隔离,ctx 取消感知。
- 组合示例:限流 + 熔断 + 舱壁串联(见 examples/gateway)。

## 性能

- 限流 Allow 10ns / 熔断 Allow 8ns / 窗口 Allow 8ns,全部 0 分配;
- 限流为 x/time/rate 的 3 倍快,熔断 Execute 与 gobreaker 相当;
- 详见 [docs/performance.md](docs/performance.md)。

## 文档

- [docs/README.md](docs/README.md) — 文档索引
- [docs/operations.md](docs/operations.md) — 运行手册
- [examples/gateway](examples/gateway) — 组合示例

## 贡献与安全

- [CONTRIBUTING.md](CONTRIBUTING.md) — 开发流程与质量门槛
- [SECURITY.md](SECURITY.md) — 安全说明与漏洞报告

## 稳定性承诺

- 本库遵循[语义化版本](https://semver.org/lang/zh-CN/);
- 家族约定:破坏性变更统一走 minor 版本(不强制主版本升级);
- 每个版本发布前执行:100% 覆盖率、race、staticcheck、fuzz ×4、
  govulncheck 与三平台 CI。

## 文档

- [docs/README.md](docs/README.md) — 文档索引

## License

MIT © [lcylpzls](https://github.com/lcylpzls)
