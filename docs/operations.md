# 运行手册

## 组件速查

| 组件 | 创建 | 核心 API | 拒绝语义 |
| --- | --- | --- | --- |
| 限流 | NewTokenBucket(rate, burst) | Allow / Wait(ctx) | KindRateLimited |
| 熔断 | NewCircuitBreaker(opts...) | Allow + Success/Failure / Execute | KindUnavailable |
| 舱壁 | NewBulkhead(max) | TryAcquire / Acquire(ctx) | KindCancelled(等待取消) |
| 固定窗口 | NewFixedWindow(limit, window) | Allow / Wait(ctx) | 布尔拒绝 |
| 滑动窗口 | NewSlidingWindow(limit, window) | Allow / Wait(ctx) | 布尔拒绝 |

## 熔断参数

| 参数 | 默认 | 说明 |
| --- | --- | --- |
| FailureThreshold | 0.5 | 失败率阈值 (0,1] |
| MinRequests | 5 | 触发熔断的最小请求数 |
| OpenTimeout | 10s | 打开后探测等待 |
| HalfOpenMax | 1 | 半开探测请求数 |
| Window | 1s × 10 | 滑动统计窗口 |

## 常见场景

### 组合调用下游(限流 + 熔断 + 舱壁)

```go
if !limiter.Allow() {
	return errx.New(errx.KindRateLimited, resiliencex.CodeRateLimited, "限流拒绝")
}
release, err := bulkhead.Acquire(ctx)
if err != nil {
	return err
}
defer release()
return cb.Execute(callDownstream)
```

### 观测

- Metrics:resx.limiter.accepted/rejected、resx.circuitbreaker.accepted/
  rejected/state_changes、resx.bulkhead.accepted/rejected、
  resx.window.rejected、resx.limiter.wait_duration;
- 熔断状态切换:OnStateChange 事件 + logx Warn 日志;
- 指标标签:熔断状态切换含 from/to。

## 注意事项

- 所有组件并发安全,可在多个 goroutine 间共享;
- 组件配置启动期确定,运行期不动态变更;
- 舱壁 release 必须在 defer 中释放;
- 时钟基于单调时间,回拨不影响限流与窗口。
