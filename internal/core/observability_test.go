package core

import (
	testx "github.com/lcylpzls/testx"
	"testing"
	"time"
)

func TestMetricsCounters(t *testing.T) {
	m := newFakeMetrics()
	cache, err := New(WithMetrics(m), WithCapacity(2))
	testx.RequireNoError(t, err)

	defer cache.Close()

	cache.Set("a", 1)
	cache.Set("b", 2)
	_, _ = cache.Get("a")                    // hit,刷新 a 的 LRU 位置
	_, _ = cache.Get("x")                    // miss
	cache.SetTTL("t", 3, 5*time.Millisecond) // 逐出 b(最久未用)
	time.Sleep(15 * time.Millisecond)
	_, _ = cache.Get("t") // expired
	cache.Delete("a")

	if m.counter(metricSets) != 3 {
		t.Errorf("sets = %d,want 3", m.counter(metricSets))
	}
	if m.counter(metricHits) != 1 {
		t.Errorf("hits = %d,want 1", m.counter(metricHits))
	}
	if m.counter(metricMisses) != 2 {
		t.Errorf("misses = %d,want 2(x 与过期 t)", m.counter(metricMisses))
	}
	if m.counter(metricEvictions) != 1 {
		t.Errorf("evictions = %d,want 1", m.counter(metricEvictions))
	}
	if m.counter(metricExpired) != 1 {
		t.Errorf("expired = %d,want 1", m.counter(metricExpired))
	}
	if m.counter(metricDeletes) != 1 {
		t.Errorf("deletes = %d,want 1", m.counter(metricDeletes))
	}
}

func TestMetricsNoop(t *testing.T) {
	cache, err := New()
	testx.RequireNoError(t, err)

	defer cache.Close()
	cache.Set("k", "v")
	_, _ = cache.Get("k")
	cache.Delete("k")
	cache.Cleanup()
}

func TestStatsSnapshot(t *testing.T) {
	cache, err := New(WithCapacity(100))
	testx.RequireNoError(t, err)

	defer cache.Close()
	cache.Set("a", 1)
	_, _ = cache.Get("a")
	_, _ = cache.Get("a")
	_, _ = cache.Get("miss")
	stats := cache.Stats()
	if stats.Hits != 2 || stats.Misses != 1 || stats.Sets != 1 {
		t.Errorf("统计不符:%+v", stats)
	}
	if stats.Len != 1 || stats.Capacity != 100 {
		t.Errorf("Len/Capacity 不符:%+v", stats)
	}
}
