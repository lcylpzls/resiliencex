package core

import (
	"context"
	"testing"
	"time"

	"github.com/lcylpzls/errx"
)

// TestKeyedWindowValidate 覆盖构造参数校验。
func TestKeyedWindowValidate(t *testing.T) {
	if _, err := NewKeyedFixedWindow(0, time.Second); !errx.Is(err, CodeInvalidConfig) {
		t.Fatalf("limit<1 应报错：%v", err)
	}
	if _, err := NewKeyedFixedWindow(1, 0); !errx.Is(err, CodeInvalidConfig) {
		t.Fatalf("window<=0 应报错：%v", err)
	}
}

// TestKeyedWindowPerKey 覆盖各 key 独立计数。
func TestKeyedWindowPerKey(t *testing.T) {
	k, err := NewKeyedFixedWindow(1, time.Second)
	if err != nil {
		t.Fatalf("New 失败：%v", err)
	}
	if !k.Allow("a") || !k.Allow("b") {
		t.Fatal("不同 key 应各自允许")
	}
	if k.Allow("a") || k.Allow("b") {
		t.Fatal("同 key 超限应拒绝")
	}
	if k.Allow("") {
		t.Fatal("空 key 应拒绝")
	}
	if got := k.Len(); got != 2 {
		t.Fatalf("Len 应为 2，得到 %d", got)
	}
}

// TestKeyedWindowTTL 覆盖空闲 TTL 清理与时钟注入。
func TestKeyedWindowTTL(t *testing.T) {
	now := time.Unix(100, 0)
	k, err := NewKeyedFixedWindow(1, time.Second,
		WithKeyedWindowTTL(50*time.Millisecond),
		WithKeyedWindowClock(func() time.Time { return now }),
		WithKeyedWindowMaxKeys(0),
		WithKeyedWindowMetrics(nil),
		WithKeyedWindowLogger(nil),
	)
	if err != nil {
		t.Fatalf("New 失败：%v", err)
	}
	if !k.Allow("a") {
		t.Fatal("首次应允许")
	}
	now = now.Add(60 * time.Millisecond)
	if !k.Allow("a") {
		t.Fatal("TTL 过期后应重建窗口并允许")
	}

	// TTL=0 表示关闭时间清理，仅按容量淘汰。
	k2, _ := NewKeyedFixedWindow(1, time.Second,
		WithKeyedWindowTTL(0), WithKeyedWindowMaxKeys(1))
	if !k2.Allow("x") {
		t.Fatal("首次应允许")
	}
	if !k2.Allow("y") {
		t.Fatal("容量淘汰后应允许新 key")
	}

	// 容量已满且存在过期条目：先清过期再建新条目。
	now3 := time.Unix(200, 0)
	k3, _ := NewKeyedFixedWindow(1, time.Second,
		WithKeyedWindowMaxKeys(1),
		WithKeyedWindowTTL(50*time.Millisecond),
		WithKeyedWindowClock(func() time.Time { return now3 }),
	)
	if !k3.Allow("a") {
		t.Fatal("key a 应允许")
	}
	now3 = now3.Add(60 * time.Millisecond)
	if !k3.Allow("b") {
		t.Fatal("过期条目清理后应允许新 key")
	}
	if got := k3.Len(); got != 1 {
		t.Fatalf("清理后 Len 应为 1，得到 %d", got)
	}
}

// TestKeyedWindowCapacity 覆盖容量上限与最旧淘汰。
func TestKeyedWindowCapacity(t *testing.T) {
	now := time.Unix(100, 0)
	k, err := NewKeyedFixedWindow(1, time.Second,
		WithKeyedWindowMaxKeys(1),
		WithKeyedWindowClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("New 失败：%v", err)
	}
	if !k.Allow("a") {
		t.Fatal("key a 应允许")
	}
	now = now.Add(time.Second)
	if !k.Allow("b") {
		t.Fatal("key b 应允许并淘汰 key a")
	}
	if got := k.Len(); got != 1 {
		t.Fatalf("容量上限后 Len 应为 1，得到 %d", got)
	}
	if !k.Allow("a") {
		t.Fatal("key a 被淘汰后应可重新建立窗口")
	}
}

// TestKeyedWindowWait 覆盖等待成功与取消分支。
func TestKeyedWindowWait(t *testing.T) {
	k, err := NewKeyedFixedWindow(1, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("New 失败：%v", err)
	}
	if !k.Allow("a") {
		t.Fatal("首次应允许")
	}

	done := make(chan error, 1)
	go func() {
		done <- k.Wait(context.Background(), "a")
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Wait 应成功：%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait 超时")
	}

	if !k.Allow("b") {
		t.Fatal("key b 应允许")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done2 := make(chan error, 1)
	go func() {
		done2 <- k.Wait(ctx, "b")
	}()
	cancel()
	select {
	case err := <-done2:
		if !errx.Is(err, CodeWaitCanceled) {
			t.Fatalf("取消应返回 CodeWaitCanceled：%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("取消未生效")
	}

	// 长窗口覆盖 poll 上限分支。
	kw, _ := NewKeyedFixedWindow(1, time.Hour)
	if !kw.Allow("z") {
		t.Fatal("首次应允许")
	}
	ctx2, cancel2 := context.WithCancel(context.Background())
	done3 := make(chan error, 1)
	go func() {
		done3 <- kw.Wait(ctx2, "z")
	}()
	cancel2()
	select {
	case err := <-done3:
		if !errx.Is(err, CodeWaitCanceled) {
			t.Fatalf("取消应返回 CodeWaitCanceled：%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("长窗口取消未生效")
	}
}

// TestKeyedWindowReset 覆盖清空全部窗口。
func TestKeyedWindowReset(t *testing.T) {
	k, err := NewKeyedFixedWindow(1, time.Second)
	if err != nil {
		t.Fatalf("New 失败：%v", err)
	}
	_ = k.Allow("a")
	k.Reset()
	if got := k.Len(); got != 0 {
		t.Fatalf("Reset 后 Len 应为 0，得到 %d", got)
	}
	if !k.Allow("a") {
		t.Fatal("Reset 后应可重新计数")
	}
}
