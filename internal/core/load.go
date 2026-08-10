package core

import (
	"context"
	"sync"
	"time"
)

// loadCall 是一次进行中的回源调用。
type loadCall struct {
	done  chan struct{}
	value any
	err   error
}

// loadGroup 按 key 合并并发的回源调用(Singleflight)。
type loadGroup struct {
	mu    sync.Mutex
	calls map[string]*loadCall
}

func newLoadGroup() *loadGroup {
	return &loadGroup{calls: make(map[string]*loadCall)}
}

// GetOrSet 命中返回缓存值;未命中时合并并发回源,只执行一次 loader。
// loader 成功按 ttl 写入缓存;失败不缓存,错误透传给全部等待者;
// 等待者 ctx 取消立即返回,不中断执行者。
func GetOrSet(ctx context.Context, c *Cache, key string, ttl time.Duration,
	loader func(context.Context) (any, error)) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if v, ok := c.Get(key); ok {
		return v, nil
	}

	g := c.loads
	g.mu.Lock()
	if call, ok := g.calls[key]; ok {
		g.mu.Unlock()
		select {
		case <-call.done:
			return call.value, call.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	call := &loadCall{done: make(chan struct{})}
	g.calls[key] = call
	g.mu.Unlock()

	start := time.Now()
	traceCtx, end := c.startTrace(ctx, "cachex.load", key)
	value, err := loader(traceCtx)
	end(err)
	if c.cfg.metrics != nil {
		c.cfg.metrics.ObserveDuration(metricLoaderDur, time.Since(start).Seconds(), nil)
	}
	call.value = value
	call.err = err
	// 先移除分组再唤醒等待者,保证后续调用能重新回源。
	g.mu.Lock()
	delete(g.calls, key)
	g.mu.Unlock()
	close(call.done)

	if err == nil {
		c.SetTTL(key, value, ttl)
	}
	return value, err
}

// startTrace 开始回源加载链路（无钩子时 no-op）。
func (c *Cache) startTrace(ctx context.Context, name, key string) (context.Context, func(error)) {
	if c.cfg.traceHook == nil {
		return ctx, func(error) {}
	}
	return c.cfg.traceHook.Start(ctx, name,
		TraceAttr{Key: "cachex.key", Value: key},
		TraceAttr{Key: "cachex.operation", Value: "load"},
	)
}
