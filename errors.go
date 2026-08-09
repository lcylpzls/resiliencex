package resiliencex

import "github.com/lcylpzls/errx"

// 错误码定义:resiliencex 各失败场景的错误码。
const (
	// CodeInvalidConfig 配置非法。
	CodeInvalidConfig errx.Code = "RESX_INVALID_CONFIG"
	// CodeRateLimited 限流拒绝。
	CodeRateLimited errx.Code = "RESX_RATE_LIMITED"
	// CodeCircuitOpen 熔断拒绝。
	CodeCircuitOpen errx.Code = "RESX_CIRCUIT_OPEN"
	// CodeBulkheadFull 舱壁拒绝。
	CodeBulkheadFull errx.Code = "RESX_BULKHEAD_FULL"
	// CodeWaitCanceled 等待限流许可被取消。
	CodeWaitCanceled errx.Code = "RESX_WAIT_CANCELED"
)

func init() {
	errx.RegisterCode(CodeInvalidConfig, "配置非法")
	errx.RegisterCode(CodeRateLimited, "限流拒绝")
	errx.RegisterCode(CodeCircuitOpen, "熔断拒绝")
	errx.RegisterCode(CodeBulkheadFull, "舱壁拒绝")
	errx.RegisterCode(CodeWaitCanceled, "等待限流许可被取消")
}
