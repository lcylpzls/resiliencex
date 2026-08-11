package core

import "github.com/lcylpzls/metricsx"

// Metrics 是最小指标协议（家族统一契约，定义见 metricsx.Sink）。
// 调用方按 Sink 签名传入标签切片；无标签时传 nil。
type Metrics = metricsx.Sink

// 指标名称:统一为 resx.* 前缀。
const (
	metricLimiterAccepted  = "resx.limiter.accepted"
	metricLimiterRejected  = "resx.limiter.rejected"
	metricCBAccepted       = "resx.circuitbreaker.accepted"
	metricCBRejected       = "resx.circuitbreaker.rejected"
	metricStateChange      = "resx.circuitbreaker.state_changes"
	metricBulkheadAccepted = "resx.bulkhead.accepted"
	metricBulkheadRejected = "resx.bulkhead.rejected"
	metricWindowRejected   = "resx.window.rejected"
	metricLimiterWaitDur   = "resx.limiter.wait_duration"
)
