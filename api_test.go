package cachex_test

import (
	"context"
	"testing"
	"time"

	"github.com/lcylpzls/cachex"
)

// TestPublicAPI 黑盒冒烟测试：覆盖根包全部转发函数、类型别名与常量。
func TestPublicAPI(t *testing.T) {
	if cachex.Version != "v1.4.1" {
		t.Fatalf("Version = %s", cachex.Version)
	}

	c, err := cachex.New(
		cachex.WithShards(4),
		cachex.WithCapacity(100),
		cachex.WithDefaultTTL(time.Second),
		cachex.WithCleanupInterval(time.Second),
		cachex.WithMetrics(nil),
		cachex.WithTraceHook(nil),
		cachex.WithEventHook(nil),
		cachex.WithOnEvicted(func(string, any, cachex.EvictReason) {}),
		cachex.WithLRURefresh(1),
	)
	if err != nil || c == nil {
		t.Fatalf("New 失败：%v", err)
	}

	c.Set("k", "v")
	if got, ok := c.Get("k"); !ok || got != "v" {
		t.Fatalf("Get 失败：%v %v", got, ok)
	}
	v, err := cachex.GetOrSet(context.Background(), c, "k2", time.Second,
		func(context.Context) (any, error) { return "loaded", nil })
	if err != nil || v != "loaded" {
		t.Fatalf("GetOrSet 失败：%v %v", v, err)
	}
	c.Delete("k")

	var _ cachex.Stats
	var _ cachex.EvictReason = cachex.EvictCapacity
	_ = cachex.EvictExpired
	var _ cachex.Option
	var _ cachex.CacheEvent
	var _ cachex.EventHook
	var _ cachex.Metrics
	var _ cachex.TraceHook
	var _ cachex.TraceAttr
	_ = cachex.CodeInvalidConfig
}
