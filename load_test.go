package cachex

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetOrSetHit(t *testing.T) {
	cache, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.Set("k", "cached")
	var loads atomic.Int32
	v, err := GetOrSet(context.Background(), cache, "k", time.Minute, func(context.Context) (any, error) {
		loads.Add(1)
		return "loaded", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if v != "cached" {
		t.Errorf("命中应返回缓存值:%v", v)
	}
	if loads.Load() != 0 {
		t.Errorf("命中不应调用 loader:%d", loads.Load())
	}
}

func TestGetOrSetMissLoads(t *testing.T) {
	cache, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	v, err := GetOrSet(context.Background(), cache, "k", time.Minute, func(context.Context) (any, error) {
		return "loaded", nil
	})
	if err != nil || v != "loaded" {
		t.Fatalf("回源结果 = %v,%v", v, err)
	}
	if got, ok := cache.Get("k"); !ok || got != "loaded" {
		t.Error("回源成功后应写入缓存")
	}
	if cache.Stats().Sets != 1 {
		t.Errorf("Sets = %d,want 1", cache.Stats().Sets)
	}
}

func TestGetOrSetConcurrentSingleLoad(t *testing.T) {
	cache, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	start := make(chan struct{})
	var loads atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			v, err := GetOrSet(context.Background(), cache, "hot", time.Minute, func(context.Context) (any, error) {
				loads.Add(1)
				time.Sleep(50 * time.Millisecond)
				return "value", nil
			})
			if err != nil || v != "value" {
				t.Errorf("回源结果 = %v,%v", v, err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if loads.Load() != 1 {
		t.Errorf("并发未命中应只回源一次:%d", loads.Load())
	}
}

func TestGetOrSetLoaderError(t *testing.T) {
	cache, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	loaderErr := errors.New("回源失败")
	var loads atomic.Int32
	var wg sync.WaitGroup
	results := make([]error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, results[idx] = GetOrSet(context.Background(), cache, "bad", time.Minute, func(context.Context) (any, error) {
				loads.Add(1)
				time.Sleep(20 * time.Millisecond) // 模拟慢回源,确保并发窗口重叠
				return nil, loaderErr
			})
		}(i)
	}
	wg.Wait()
	for i, err := range results {
		if !errors.Is(err, loaderErr) {
			t.Errorf("调用者 %d 错误不符:%v", i, err)
		}
	}
	if loads.Load() != 1 {
		t.Errorf("失败也应合并回源:%d", loads.Load())
	}
	if _, ok := cache.Get("bad"); ok {
		t.Error("loader 失败不应缓存")
	}
	// 失败后再次调用应重新回源
	_, err = GetOrSet(context.Background(), cache, "bad", time.Minute, func(context.Context) (any, error) {
		loads.Add(1)
		return "ok", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if loads.Load() != 2 {
		t.Errorf("失败后应可重试:%d", loads.Load())
	}
}

func TestGetOrSetWaiterCancel(t *testing.T) {
	cache, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = GetOrSet(context.Background(), cache, "slow", time.Minute, func(context.Context) (any, error) {
			<-release
			return "value", nil
		})
	}()
	// 等执行者进入 loader
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cache.loads.mu.Lock()
		_, loading := cache.loads.calls["slow"]
		cache.loads.mu.Unlock()
		if loading {
			break
		}
		time.Sleep(time.Millisecond)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = GetOrSet(ctx, cache, "slow", time.Minute, func(context.Context) (any, error) {
		return nil, errors.New("不应执行")
	})
	if err == nil {
		t.Fatal("等待者取消应返回错误")
	}
	close(release)
	<-done
	if v, ok := cache.Get("slow"); !ok || v != "value" {
		t.Error("执行者应正常完成并写入缓存")
	}
}

func TestGetOrSetNilContext(t *testing.T) {
	cache, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	//lint:ignore SA1012 有意覆盖 nil context 防护逻辑
	v, err := GetOrSet(nil, cache, "k", time.Minute, func(context.Context) (any, error) {
		return "v", nil
	})
	if err != nil || v != "v" {
		t.Fatalf("nil ctx 应视为 Background:%v,%v", v, err)
	}
}

func TestGetOrSetLoaderDurationMetric(t *testing.T) {
	m := newFakeMetrics()
	cache, err := New(WithMetrics(m))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	if _, err := GetOrSet(context.Background(), cache, "k", time.Minute, func(context.Context) (any, error) {
		return "v", nil
	}); err != nil {
		t.Fatal(err)
	}
	if m.durationCount(metricLoaderDur) != 1 {
		t.Errorf("loader_duration 未记录:%d", m.durationCount(metricLoaderDur))
	}
}

// FuzzLoader 保证任意 loader 结果下 GetOrSet 不泄漏、不 panic。
func FuzzLoader(f *testing.F) {
	f.Add("key", int64(1))
	f.Add("", int64(-1))
	f.Fuzz(func(t *testing.T, key string, seed int64) {
		cache, err := New(WithShards(4), WithCapacity(64))
		if err != nil {
			t.Fatal(err)
		}
		defer cache.Close()
		var wg sync.WaitGroup
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := GetOrSet(context.Background(), cache, key, 0,
					func(context.Context) (any, error) {
						if seed%3 == 0 {
							return nil, errors.New("失败")
						}
						return seed, nil
					})
				if err == nil && v == nil {
					t.Error("成功结果不应为 nil")
				}
			}()
		}
		wg.Wait()
	})
}
