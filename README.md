# resiliencex

自研服务治理组件库:令牌桶限流、熔断器、舱壁隔离,
拒绝语义统一 errx,观测外部注入,零第三方依赖。

> 当前状态:**v0.2.0 实现完成,待 CI 验证与发布**。

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
```

## 定位

resiliencex 不是治理编排框架,不解决「组件怎么串」的问题;它提供
每个依赖调用方都要重复的三个独立组件:

- 限流:令牌桶,平滑 + 突发,Allow / Wait(ctx) 双 API;
- 熔断:三态状态机、滑动窗口失败率、HalfOpen 探测;
- 舱壁:信号量隔离,ctx 取消感知。

## 文档

- [docs/README.md](docs/README.md) — 文档索引
- [docs/resilience-research.md](docs/resilience-research.md) — 治理领域调研手册

## License

MIT © [lcylpzls](https://github.com/lcylpzls)
