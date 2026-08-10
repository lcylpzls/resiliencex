package resiliencex

import (
	"context"
	"errors"
	"github.com/lcylpzls/testx"
	"sync"
	"testing"
	"time"
)

// traceCall 记录一次追踪调用。
type traceCall struct {
	name  string
	attrs map[string]string
	err   error
	ended bool
}

// fakeTraceHook 内存追踪钩子。
type fakeTraceHook struct {
	mu    sync.Mutex
	calls []traceCall
}

func (h *fakeTraceHook) Start(ctx context.Context, name string, attrs ...TraceAttr) (context.Context, func(error)) {
	h.mu.Lock()
	h.calls = append(h.calls, traceCall{name: name, attrs: map[string]string{}})
	for _, a := range attrs {
		h.calls[len(h.calls)-1].attrs[a.Key] = a.Value
	}
	h.mu.Unlock()
	return ctx, func(err error) {
		h.mu.Lock()
		h.calls[len(h.calls)-1].err = err
		h.calls[len(h.calls)-1].ended = true
		h.mu.Unlock()
	}
}

func (h *fakeTraceHook) snapshot() []traceCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]traceCall, len(h.calls))
	copy(out, h.calls)
	return out
}

// TestCircuitBreakerTraceHook 覆盖成功/失败/拒绝三路埋点。
func TestCircuitBreakerTraceHook(t *testing.T) {
	hook := &fakeTraceHook{}
	cb := newTestCB(t,
		WithTraceHook(hook),
		WithFailureThreshold(0.5),
		WithMinRequests(1),
		WithOpenTimeout(time.Hour),
	)
	testx.RequireNoError(t, cb.ExecuteContext(context.Background(), func(context.Context) error { return nil }))
	err := cb.ExecuteContext(context.Background(), func(context.Context) error {
		return errors.New("业务失败")
	})
	testx.RequireError(t, err)
	err = cb.ExecuteContext(context.Background(), func(context.Context) error { return nil })
	testx.RequireError(t, err)

	calls := hook.snapshot()
	testx.RequireLen(t, calls, 3)
	for _, c := range calls {
		testx.RequireEqual(t, c.name, "resiliencex.circuit_breaker")
		testx.RequireEqual(t, c.attrs["resiliencex.type"], "circuit_breaker")
		testx.RequireNotEmpty(t, c.attrs["resiliencex.state"])
		testx.RequireTrue(t, c.ended)
	}
	testx.RequireNil(t, calls[0].err)
	testx.RequireNotNil(t, calls[1].err)
	testx.RequireNotNil(t, calls[2].err)

	// Execute 委托（无 ctx）与 nil 校验。
	hook2 := &fakeTraceHook{}
	cb2 := newTestCB(t, WithTraceHook(hook2))
	testx.RequireNoError(t, cb2.Execute(func() error { return nil }))
	testx.RequireError(t, cb2.Execute(nil))
	testx.RequireError(t, cb2.ExecuteContext(context.Background(), nil))
	testx.RequireLen(t, hook2.snapshot(), 1)
}
