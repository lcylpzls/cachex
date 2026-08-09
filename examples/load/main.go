// load 示例:GetOrSet 击穿防护,并发未命中只回源一次。
package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lcylpzls/cachex"
)

func main() {
	cache, err := cachex.New(cachex.WithCapacity(1000))
	if err != nil {
		panic(err)
	}
	defer cache.Close()

	var loads atomic.Int32
	loader := func(context.Context) (any, error) {
		loads.Add(1)
		time.Sleep(50 * time.Millisecond) // 模拟慢回源
		return "热点数据", nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := cachex.GetOrSet(context.Background(), cache, "hot", time.Minute, loader)
			if err != nil {
				panic(err)
			}
			_ = v
		}()
	}
	wg.Wait()
	fmt.Printf("10 个并发调用,实际回源 %d 次\n", loads.Load())
}
