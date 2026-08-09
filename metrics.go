package resiliencex

// Metrics 是最小指标接口,默认 no-op,便于外部接 Prometheus 等适配器。
type Metrics interface {
	// IncCounter 增加一个计数指标。
	IncCounter(name string, labels ...string)
	// ObserveDuration 记录一次耗时观测。
	ObserveDuration(name string, seconds float64, labels ...string)
}

// 指标名称:统一为 resx.* 前缀。
const (
	metricLimiterAccepted  = "resx.limiter.accepted"
	metricLimiterRejected  = "resx.limiter.rejected"
	metricCBAccepted       = "resx.circuitbreaker.accepted"
	metricCBRejected       = "resx.circuitbreaker.rejected"
	metricStateChange      = "resx.circuitbreaker.state_changes"
	metricBulkheadAccepted = "resx.bulkhead.accepted"
	metricBulkheadRejected = "resx.bulkhead.rejected"
)
