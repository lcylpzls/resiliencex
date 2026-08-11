package core

import "github.com/lcylpzls/logx"

// configApplier 是组件配置的通用应用接口,
// 使 WithMetrics / WithLogger 可跨组件复用。
type configApplier interface {
	setMetrics(Metrics)
	setLogger(logx.Logger)
}

// Option 修改组件配置,在组件 New 时按顺序应用。
// 组件专属选项对不匹配的组件无效(文档说明)。
type Option func(configApplier)

// WithMetrics 注入指标钩子,空表示关闭指标(默认)。
func WithMetrics(m Metrics) Option {
	return func(c configApplier) { c.setMetrics(m) }
}

// WithLogger 注入结构化日志实现,空表示关闭日志(默认)。
func WithLogger(l logx.Logger) Option {
	return func(c configApplier) { c.setLogger(l) }
}

// limiterConfig 是限流器配置。
type limiterConfig struct {
	metrics Metrics
	logger  logx.Logger
}

func (c *limiterConfig) setMetrics(m Metrics) { c.metrics = m }
func (c *limiterConfig) setLogger(l logx.Logger) {
	c.logger = l
}
