// gateway 示例:限流 + 熔断 + 舱壁组合调用下游。
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lcylpzls/errx"
	"github.com/lcylpzls/resiliencex"
)

// downstream 模拟下游调用(可注入失败)。
func downstream() error {
	return nil
}

func main() {
	// 限流:每秒 50 次,突发 10
	limiter, err := resiliencex.NewTokenBucket(50, 10)
	if err != nil {
		panic(err)
	}
	// 熔断:失败率 50%、最小 10 请求、探测 5s
	cb, err := resiliencex.NewCircuitBreaker(
		resiliencex.WithFailureThreshold(0.5),
		resiliencex.WithMinRequests(10),
		resiliencex.WithOpenTimeout(5*time.Second),
	)
	if err != nil {
		panic(err)
	}
	// 舱壁:最多 20 个并发
	bulkhead, err := resiliencex.NewBulkhead(20)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	call := func() error {
		if !limiter.Allow() {
			return resiliencex.ErrRateLimited()
		}
		release, err := bulkhead.Acquire(ctx)
		if err != nil {
			return err
		}
		defer release()
		return cb.Execute(downstream)
	}

	for i := 0; i < 100; i++ {
		if err := call(); err != nil {
			if !errx.Is(err, resiliencex.CodeRateLimited) {
				fmt.Printf("第 %d 次调用失败:%v\n", i+1, err)
			}
		}
	}
	fmt.Println("组合调用完成")
}
