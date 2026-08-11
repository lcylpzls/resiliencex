package core

import (
	"errors"
	testx "github.com/lcylpzls/testx"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/lcylpzls/errx"
)

func newTestCB(t *testing.T, opts ...Option) *CircuitBreaker {
	t.Helper()
	cb, err := NewCircuitBreaker(opts...)
	testx.RequireNoError(t, err)

	return cb
}

// fakeClock 是可推进的测试时钟。
type fakeClock struct {
	t time.Time
}

func (c *fakeClock) now() time.Time { return c.t }
func (c *fakeClock) advance(d time.Duration) {
	c.t = c.t.Add(d)
}

func TestStateString(t *testing.T) {
	cases := []struct {
		s    State
		want string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half_open"},
		{State(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.want {
			t.Errorf("%v:String = %q,want %q", c.s, got, c.want)
		}
	}
}

func TestNewCircuitBreakerInvalid(t *testing.T) {
	cases := []Option{
		WithFailureThreshold(0),
		WithFailureThreshold(-0.1),
		WithFailureThreshold(1.1),
		WithFailureThreshold(math.NaN()),
		WithFailureThreshold(math.Inf(1)),
		WithMinRequests(0),
		WithOpenTimeout(0),
		WithOpenTimeout(-time.Second),
		WithHalfOpenMax(0),
		WithWindow(0, 10),
		WithWindow(time.Second, 0),
	}
	for _, opt := range cases {
		if _, err := NewCircuitBreaker(opt); err == nil {
			t.Errorf("配置应非法:%v", opt)
		} else if code, _ := errx.CodeOf(err); code != CodeInvalidConfig {
			t.Errorf("错误码 = %s", code)
		}
	}
}

func TestClosedAllowsAndCounts(t *testing.T) {
	cb := newTestCB(t)
	if err := cb.Allow(); err != nil {
		t.Fatalf("Closed 应放行:%v", err)
	}
	cb.Success()
	cb.Failure()
	counts := cb.Counts()
	if counts.Requests != 2 || counts.Successes != 1 || counts.Failures != 1 {
		t.Errorf("统计不符:%+v", counts)
	}
	if cb.State() != StateClosed {
		t.Error("失败率未达标应保持 Closed")
	}
}

func TestOpenByFailureRate(t *testing.T) {
	cb := newTestCB(t, WithFailureThreshold(0.5), WithMinRequests(2))
	cb.Success()
	cb.Failure()
	if cb.State() != StateOpen {
		t.Fatal("失败率 50% 且达到最小请求数应 Open")
	}
}

func TestMinRequestsProtection(t *testing.T) {
	cb := newTestCB(t, WithFailureThreshold(1.0), WithMinRequests(5))
	for i := 0; i < 4; i++ {
		cb.Failure()
	}
	if cb.State() != StateClosed {
		t.Error("未达到最小请求数不应 Open")
	}
	cb.Failure()
	if cb.State() != StateOpen {
		t.Error("达到最小请求数且全部失败应 Open")
	}
}

func TestOpenRejects(t *testing.T) {
	cb := newTestCB(t, WithFailureThreshold(1.0), WithMinRequests(1))
	cb.Failure()
	err := cb.Allow()
	testx.RequireError(t, err)

	if code, _ := errx.CodeOf(err); code != CodeCircuitOpen {
		t.Errorf("错误码 = %s,want %s", code, CodeCircuitOpen)
	}
	if kind := errx.KindOf(err); kind != errx.KindUnavailable {
		t.Errorf("分类 = %s,want unavailable", kind)
	}
}

func TestOpenTransitionsToHalfOpen(t *testing.T) {
	t0 := time.Now()
	clock := &fakeClock{t: t0}
	cb := newTestCB(t, WithFailureThreshold(1.0), WithMinRequests(1), WithOpenTimeout(10*time.Second))
	cb.cfg.now = clock.now
	cb.Failure() // → Open
	if err := cb.Allow(); err == nil {
		t.Fatal("超时前应拒绝")
	}
	// 时间前进后首个 Allow 转入 HalfOpen 并放行探测
	clock.advance(11 * time.Second)
	if err := cb.Allow(); err != nil {
		t.Fatalf("超时后应放行探测:%v", err)
	}
	if cb.State() != StateHalfOpen {
		t.Error("应处于 HalfOpen")
	}
	// halfOpenMax=1:第二个探测拒绝
	if err := cb.Allow(); err == nil {
		t.Error("HalfOpen 探测数超限应拒绝")
	}
}

func TestHalfOpenSuccessCloses(t *testing.T) {
	t0 := time.Now()
	clock := &fakeClock{t: t0}
	cb := newTestCB(t,
		WithFailureThreshold(1.0), WithMinRequests(1),
		WithOpenTimeout(time.Second), WithHalfOpenMax(2),
	)
	cb.cfg.now = clock.now
	cb.Failure() // → Open
	clock.advance(2 * time.Second)
	if err := cb.Allow(); err != nil {
		t.Fatal(err)
	}
	if err := cb.Allow(); err != nil {
		t.Fatal(err)
	}
	cb.Success()
	cb.Success() // 探测全部成功 → Closed
	if cb.State() != StateClosed {
		t.Error("探测成功应回到 Closed")
	}
	// 窗口已重置
	if counts := cb.Counts(); counts.Requests != 0 {
		t.Errorf("重置后统计应清零:%+v", counts)
	}
}

func TestHalfOpenFailureReopens(t *testing.T) {
	t0 := time.Now()
	clock := &fakeClock{t: t0}
	cb := newTestCB(t,
		WithFailureThreshold(1.0), WithMinRequests(1),
		WithOpenTimeout(time.Second),
	)
	cb.cfg.now = clock.now
	cb.Failure() // → Open
	clock.advance(2 * time.Second)
	if err := cb.Allow(); err != nil {
		t.Fatal(err)
	}
	cb.Failure() // 探测失败 → Open
	if cb.State() != StateOpen {
		t.Fatal("探测失败应回到 Open")
	}
	// 重置计时:再次前进 0.5s 仍拒绝
	clock.advance(500 * time.Millisecond)
	if err := cb.Allow(); err == nil {
		t.Error("重新计时后未到期应拒绝")
	}
	// 再前进 1s 后放行
	clock.advance(time.Second)
	if err := cb.Allow(); err != nil {
		t.Errorf("重新计时到期应放行:%v", err)
	}
}

func TestExecute(t *testing.T) {
	cb := newTestCB(t, WithFailureThreshold(0.5), WithMinRequests(1))
	if err := cb.Execute(func() error { return nil }); err != nil {
		t.Fatalf("成功执行应通过:%v", err)
	}
	boom := errors.New("boom")
	if err := cb.Execute(func() error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("失败执行应透传:%v", err)
	}
	if cb.State() != StateOpen {
		t.Error("失败率 50% 应 Open")
	}
	// Open 时 Execute 直接拒绝
	if err := cb.Execute(func() error { return nil }); err == nil {
		t.Error("Open 时 Execute 应拒绝")
	}
}

func TestExecuteNilFn(t *testing.T) {
	cb := newTestCB(t)
	err := cb.Execute(nil)
	testx.RequireError(t, err)

	if code, _ := errx.CodeOf(err); code != CodeInvalidConfig {
		t.Errorf("错误码 = %s,want %s", code, CodeInvalidConfig)
	}
}

func TestOnStateChange(t *testing.T) {
	var events []string
	cb := newTestCB(t,
		WithFailureThreshold(1.0), WithMinRequests(1),
		WithOpenTimeout(time.Second),
		WithOnStateChange(func(from, to State) {
			events = append(events, from.String()+"->"+to.String())
		}),
	)
	cb.Failure()
	time.Sleep(1100 * time.Millisecond)
	if err := cb.Allow(); err != nil {
		t.Fatal(err)
	}
	cb.Success()
	want := []string{"closed->open", "open->half_open", "half_open->closed"}
	if len(events) != len(want) {
		t.Fatalf("事件数 = %d,want %d:%v", len(events), len(want), events)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Errorf("事件 %d = %q,want %q", i, events[i], want[i])
		}
	}
}

func TestStateChangeLoggerAndMetrics(t *testing.T) {
	logger := &fakeLogger{}
	m := newFakeMetrics()
	cb := newTestCB(t,
		WithFailureThreshold(1.0), WithMinRequests(1),
		WithMetrics(m), WithLogger(logger),
	)
	if err := cb.Allow(); err != nil {
		t.Fatal(err)
	}
	if m.counter(metricCBAccepted) != 1 {
		t.Errorf("accepted = %d,want 1", m.counter(metricCBAccepted))
	}
	cb.Failure()
	_ = cb.Allow() // Open 拒绝
	if m.counter(metricCBRejected) != 1 {
		t.Errorf("rejected = %d,want 1", m.counter(metricCBRejected))
	}
	if !logger.hasWarn("熔断器状态切换") {
		t.Error("状态切换应输出 Warn 日志")
	}
	if m.counter(metricStateChange+"|closed|open") != 1 {
		t.Errorf("state_changes = %d,want 1", m.counter(metricStateChange+"|closed|open"))
	}
}

func TestAllowUnknownState(t *testing.T) {
	cb := newTestCB(t)
	cb.mu.Lock()
	cb.state = State(99)
	cb.mu.Unlock()
	if err := cb.Allow(); err == nil {
		t.Fatal("未知状态应拒绝")
	}
}

func TestWindowSlides(t *testing.T) {
	t0 := time.Now()
	clock := &fakeClock{t: t0}
	cb := newTestCB(t)
	cb.cfg.now = clock.now
	cb.Success()
	cb.Success()
	cb.Failure()
	clock.advance(11 * time.Second)
	if counts := cb.Counts(); counts.Requests != 0 {
		t.Errorf("窗口滑动后统计应清零:%+v", counts)
	}
}

func TestCircuitBreakerConcurrent(t *testing.T) {
	cb := newTestCB(t, WithFailureThreshold(0.5), WithMinRequests(10))
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				if err := cb.Allow(); err != nil {
					continue
				}
				if (id+j)%3 == 0 {
					cb.Failure()
				} else {
					cb.Success()
				}
			}
		}(i)
	}
	wg.Wait()
	_ = cb.State()
	_ = cb.Counts()
}

// FuzzCircuitBreaker 保证任意操作序列下熔断器不 panic。
func FuzzCircuitBreaker(f *testing.F) {
	f.Add(uint8(0), uint8(0))
	f.Add(uint8(1), uint8(1))
	f.Fuzz(func(t *testing.T, op uint8, n uint8) {
		cb, err := NewCircuitBreaker(
			WithFailureThreshold(0.5),
			WithMinRequests(2),
			WithOpenTimeout(time.Millisecond),
		)
		testx.RequireNoError(t, err)

		for i := uint8(0); i < n%8; i++ {
			switch op % 3 {
			case 0:
				_ = cb.Allow()
			case 1:
				cb.Success()
			case 2:
				cb.Failure()
			}
		}
		_ = cb.State()
		_ = cb.Counts()
	})
}
