package resiliencex

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lcylpzls/errx"
)

// fakeWindowClock 是可推进的测试时钟。
type fakeWindowClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeWindowClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}
func (c *fakeWindowClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestNewWindowInvalid(t *testing.T) {
	for _, tc := range []struct {
		limit  int
		window time.Duration
	}{
		{0, time.Second},
		{-1, time.Second},
		{10, 0},
		{10, -time.Second},
	} {
		if _, err := NewFixedWindow(tc.limit, tc.window); err == nil {
			t.Errorf("limit=%d window=%v 应非法", tc.limit, tc.window)
		}
		if _, err := NewSlidingWindow(tc.limit, tc.window); err == nil {
			t.Errorf("滑动 limit=%d window=%v 应非法", tc.limit, tc.window)
		}
	}
}

func TestFixedWindow(t *testing.T) {
	clock := &fakeWindowClock{t: time.Now()}
	w, err := NewFixedWindow(2, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	w.cfg.now = clock.now
	for i := 0; i < 2; i++ {
		if !w.Allow() {
			t.Fatal("窗口内前两次应通过")
		}
	}
	if w.Allow() {
		t.Fatal("窗口内第三次应拒绝")
	}
	clock.advance(time.Second)
	if !w.Allow() {
		t.Fatal("新窗口应重置")
	}
}

func TestSlidingWindow(t *testing.T) {
	clock := &fakeWindowClock{t: time.Now()}
	w, err := NewSlidingWindow(2, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	w.cfg.now = clock.now
	for i := 0; i < 2; i++ {
		if !w.Allow() {
			t.Fatal("窗口内前两次应通过")
		}
	}
	if w.Allow() {
		t.Fatal("第三次应拒绝")
	}
	// 前进 600ms:无记录过期,仍拒绝
	clock.advance(600 * time.Millisecond)
	if w.Allow() {
		t.Fatal("600ms 内无过期应仍拒绝")
	}
	// 再前进 500ms(共 1.1s):t0 记录过期,应可通过
	clock.advance(500 * time.Millisecond)
	if !w.Allow() {
		t.Fatal("最早记录过期后应可再次通过")
	}
	// 再前进 100ms(共 1.2s):窗口内有 1 条,可通过
	clock.advance(100 * time.Millisecond)
	if !w.Allow() {
		t.Fatal("窗口内未满应通过")
	}
	// 再前进 100ms(共 1.3s):size=2 已满且无过期,拒绝
	clock.advance(100 * time.Millisecond)
	if w.Allow() {
		t.Fatal("窗口内仍超限应拒绝")
	}
	// 再前进 900ms(共 2.2s):t0+1.1s 记录过期,应可通过
	clock.advance(900 * time.Millisecond)
	if !w.Allow() {
		t.Fatal("第二批最早记录过期后应可再次通过")
	}
}

func TestWindowWait(t *testing.T) {
	clock := &fakeWindowClock{t: time.Now()}
	w, err := NewFixedWindow(1, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	w.cfg.now = clock.now
	if !w.Allow() {
		t.Fatal("首次应通过")
	}
	go func() {
		time.Sleep(60 * time.Millisecond)
		clock.advance(60 * time.Millisecond)
	}()
	if err := w.Wait(context.Background()); err != nil {
		t.Fatalf("等待窗口应通过:%v", err)
	}
}

func TestWindowWaitCanceled(t *testing.T) {
	clock := &fakeWindowClock{t: time.Now()}
	w, err := NewFixedWindow(1, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	w.cfg.now = clock.now
	if !w.Allow() {
		t.Fatal("首次应通过")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err = w.Wait(ctx)
	if err == nil {
		t.Fatal("等待应被取消")
	}
	if code, _ := errx.CodeOf(err); code != CodeWaitCanceled {
		t.Errorf("错误码 = %s,want %s", code, CodeWaitCanceled)
	}
}

func TestWindowMetrics(t *testing.T) {
	m := newFakeMetrics()
	logger := &fakeLogger{}
	w, err := NewFixedWindow(1, time.Second, WithMetrics(m), WithLogger(logger))
	if err != nil {
		t.Fatal(err)
	}
	if !w.Allow() {
		t.Fatal("首次应通过")
	}
	if w.Allow() {
		t.Fatal("应拒绝")
	}
	if m.counter(metricWindowRejected) != 1 {
		t.Errorf("rejected = %d,want 1", m.counter(metricWindowRejected))
	}
}

// FuzzWindow 保证任意窗口参数与操作不 panic。
func FuzzWindow(f *testing.F) {
	f.Add(1, int64(1000000))
	f.Add(10, int64(-1))
	f.Fuzz(func(t *testing.T, limit int, windowNs int64) {
		w, err := NewSlidingWindow(limit, time.Duration(windowNs))
		if err != nil {
			return
		}
		for i := 0; i < 20; i++ {
			_ = w.Allow()
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = w.Wait(ctx)
	})
}
