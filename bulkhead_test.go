package resiliencex

import (
	"context"
	testx "github.com/lcylpzls/testx"
	"sync"
	"testing"
	"time"

	"github.com/lcylpzls/errx"
)

func TestNewBulkheadInvalid(t *testing.T) {
	for _, n := range []int{0, -1} {
		if _, err := NewBulkhead(n); err == nil {
			t.Errorf("maxConcurrent=%d 应非法", n)
		} else if code, _ := errx.CodeOf(err); code != CodeInvalidConfig {
			t.Errorf("错误码 = %s", code)
		}
	}
}

func TestTryAcquireAndRelease(t *testing.T) {
	b, err := NewBulkhead(1)
	testx.RequireNoError(t, err)

	release, ok := b.TryAcquire()
	testx.RequireTrue(t, ok)

	if _, ok := b.TryAcquire(); ok {
		t.Fatal("已满应拒绝")
	}
	if b.Available() != 0 {
		t.Errorf("Available = %d,want 0", b.Available())
	}
	release()
	release() // 幂等
	if _, ok := b.TryAcquire(); !ok {
		t.Fatal("释放后应可获取")
	}
}

func TestAcquireBlocksAndCancel(t *testing.T) {
	b, err := NewBulkhead(1)
	testx.RequireNoError(t, err)

	release, err := b.Acquire(context.Background())
	testx.RequireNoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = b.Acquire(ctx)
	testx.RequireError(t, err)

	if code, _ := errx.CodeOf(err); code != CodeWaitCanceled {
		t.Errorf("错误码 = %s,want %s", code, CodeWaitCanceled)
	}
	if kind := errx.KindOf(err); kind != errx.KindCancelled {
		t.Errorf("分类 = %s,want cancelled", kind)
	}
	release()
	// 释放后可获取
	if _, err := b.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBulkheadMetrics(t *testing.T) {
	m := newFakeMetrics()
	logger := &fakeLogger{}
	b, err := NewBulkhead(1, WithMetrics(m), WithLogger(logger))
	testx.RequireNoError(t, err)

	r1, _ := b.TryAcquire()
	if _, ok := b.TryAcquire(); ok {
		t.Fatal("已满应拒绝")
	}
	r1()
	if m.counter(metricBulkheadAccepted) != 1 {
		t.Errorf("accepted = %d,want 1", m.counter(metricBulkheadAccepted))
	}
	if m.counter(metricBulkheadRejected) != 1 {
		t.Errorf("rejected = %d,want 1", m.counter(metricBulkheadRejected))
	}
}

func TestBulkheadConcurrent(t *testing.T) {
	b, err := NewBulkhead(4)
	testx.RequireNoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := b.Acquire(context.Background())
			if err != nil {
				return
			}
			time.Sleep(time.Millisecond)
			release()
		}()
	}
	wg.Wait()
	if b.Available() != 4 {
		t.Errorf("全部释放后 Available = %d,want 4", b.Available())
	}
}

func TestAcquireNilContext(t *testing.T) {
	b, err := NewBulkhead(1)
	testx.RequireNoError(t, err)

	//lint:ignore SA1012 有意覆盖 nil context 防护逻辑
	release, err := b.Acquire(nil)
	testx.RequireNoError(t, err)

	release()
}

// FuzzBulkhead 保证任意并发操作不 panic。
func FuzzBulkhead(f *testing.F) {
	f.Add(1)
	f.Add(4)
	f.Fuzz(func(t *testing.T, n int) {
		if n < 1 {
			n = 1
		}
		b, err := NewBulkhead(n)
		testx.RequireNoError(t, err)

		var releases []func()
		for i := 0; i < n; i++ {
			if r, ok := b.TryAcquire(); ok {
				releases = append(releases, r)
			}
		}
		for _, r := range releases {
			r()
		}
		_ = b.Available()
	})
}
