package resiliencex

import (
	"context"
	"errors"
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
	if err := cb.ExecuteContext(context.Background(), func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := cb.ExecuteContext(context.Background(), func(context.Context) error {
		return errors.New("业务失败")
	}); err == nil {
		t.Fatal("应返回业务失败")
	}
	if err := cb.ExecuteContext(context.Background(), func(context.Context) error { return nil }); err == nil {
		t.Fatal("熔断开启后应拒绝")
	}

	calls := hook.snapshot()
	if len(calls) != 3 {
		t.Fatalf("应调用 3 次追踪钩子，实际：%d", len(calls))
	}
	for i, c := range calls {
		if c.name != "resiliencex.circuit_breaker" ||
			c.attrs["resiliencex.type"] != "circuit_breaker" ||
			c.attrs["resiliencex.state"] == "" || !c.ended {
			t.Fatalf("第 %d 次追踪调用不符：%+v", i, c)
		}
	}
	if calls[0].err != nil || calls[1].err == nil || calls[2].err == nil {
		t.Fatalf("结束回调错误记录不符：%+v", calls)
	}

	// Execute 委托（无 ctx）与 nil 校验。
	hook2 := &fakeTraceHook{}
	cb2 := newTestCB(t, WithTraceHook(hook2))
	if err := cb2.Execute(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := cb2.Execute(nil); err == nil {
		t.Fatal("nil 执行函数应报错")
	}
	if err := cb2.ExecuteContext(context.Background(), nil); err == nil {
		t.Fatal("ExecuteContext nil 执行函数应报错")
	}
	if len(hook2.snapshot()) != 1 {
		t.Fatalf("Execute 应委托埋点：%+v", hook2.snapshot())
	}
}
