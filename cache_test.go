package cachex

import (
	"sync"
	"testing"
	"time"
)

func TestSetGet(t *testing.T) {
	cache, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.Set("k", "v")
	v, ok := cache.Get("k")
	if !ok || v != "v" {
		t.Fatalf("Get = %v,%v,want v,true", v, ok)
	}
	stats := cache.Stats()
	if stats.Hits != 1 || stats.Misses != 0 || stats.Sets != 1 {
		t.Errorf("统计不符:%+v", stats)
	}
}

func TestGetMiss(t *testing.T) {
	cache, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	if v, ok := cache.Get("nope"); ok || v != nil {
		t.Errorf("未命中应返回 nil,false:got %v,%v", v, ok)
	}
	if cache.Stats().Misses != 1 {
		t.Errorf("misses = %d,want 1", cache.Stats().Misses)
	}
}

func TestSetTTLExpired(t *testing.T) {
	var reasons []EvictReason
	cache, err := New(WithOnEvicted(func(key string, value any, reason EvictReason) {
		if key == "k" {
			reasons = append(reasons, reason)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.SetTTL("k", "v", 10*time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	if v, ok := cache.Get("k"); ok || v != nil {
		t.Errorf("过期条目应未命中:got %v,%v", v, ok)
	}
	stats := cache.Stats()
	if stats.Expired != 1 || stats.Misses != 1 {
		t.Errorf("过期统计不符:%+v", stats)
	}
	if len(reasons) != 1 || reasons[0] != EvictExpired {
		t.Errorf("过期回调不符:%v", reasons)
	}
}

func TestSetTTLNonPositive(t *testing.T) {
	cache, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	for _, ttl := range []time.Duration{0, -time.Second} {
		cache.SetTTL("k", "v", ttl)
		if _, ok := cache.Get("k"); !ok {
			t.Errorf("ttl=%v 应永不过期", ttl)
		}
	}
}

func TestSetUpdate(t *testing.T) {
	cache, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.Set("k", 1)
	cache.Set("k", 2)
	if cache.Len() != 1 {
		t.Errorf("更新不应增加条目:Len = %d", cache.Len())
	}
	if v, _ := cache.Get("k"); v != 2 {
		t.Errorf("更新后值 = %v,want 2", v)
	}
	if cache.Stats().Sets != 2 {
		t.Errorf("Sets = %d,want 2", cache.Stats().Sets)
	}
}

func TestLRUEviction(t *testing.T) {
	var evicted []string
	cache, err := New(WithCapacity(3), WithOnEvicted(func(key string, _ any, reason EvictReason) {
		if reason == EvictCapacity {
			evicted = append(evicted, key)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.Set("a", 1)
	cache.Set("b", 2)
	cache.Set("c", 3)
	cache.Set("d", 4)
	if _, ok := cache.Get("a"); ok {
		t.Error("最久未用的 a 应被逐出")
	}
	if _, ok := cache.Get("b"); !ok {
		t.Error("b 应仍在缓存")
	}
	if len(evicted) != 1 || evicted[0] != "a" {
		t.Errorf("逐出顺序不符:%v", evicted)
	}
	if cache.Stats().Evictions != 1 {
		t.Errorf("Evictions = %d,want 1", cache.Stats().Evictions)
	}
}

func TestLRUOrderRefreshedByGet(t *testing.T) {
	cache, err := New(WithCapacity(2))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.Set("a", 1)
	cache.Set("b", 2)
	if _, ok := cache.Get("a"); !ok {
		t.Fatal("a 应命中")
	}
	cache.Set("c", 3)
	if _, ok := cache.Get("b"); ok {
		t.Error("b 未刷新应被逐出")
	}
	if _, ok := cache.Get("a"); !ok {
		t.Error("a 被 Get 刷新后应保留")
	}
}

func TestDelete(t *testing.T) {
	cache, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.Set("k", "v")
	if !cache.Delete("k") {
		t.Fatal("删除应命中")
	}
	if cache.Delete("k") {
		t.Error("重复删除应返回 false")
	}
	if cache.Len() != 0 {
		t.Errorf("Len = %d,want 0", cache.Len())
	}
	if cache.Stats().Deletes != 1 {
		t.Errorf("Deletes = %d,want 1", cache.Stats().Deletes)
	}
}

func TestClear(t *testing.T) {
	var evicted int
	cache, err := New(WithOnEvicted(func(string, any, EvictReason) { evicted++ }))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.Set("a", 1)
	cache.Set("b", 2)
	cache.Clear()
	if cache.Len() != 0 {
		t.Errorf("Clear 后 Len = %d,want 0", cache.Len())
	}
	if _, ok := cache.Get("a"); ok {
		t.Error("Clear 后 a 不应存在")
	}
	if evicted != 0 {
		t.Errorf("Clear 不应触发回调:%d", evicted)
	}
}

func TestLenAcrossShards(t *testing.T) {
	cache, err := New(WithShards(4))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	for i := 0; i < 100; i++ {
		cache.Set(string(rune('a'+i%26))+string(rune('0'+i)), i)
	}
	if cache.Len() != 100 {
		t.Errorf("Len = %d,want 100", cache.Len())
	}
}

func TestCleanup(t *testing.T) {
	var expiredKeys []string
	cache, err := New(WithOnEvicted(func(key string, _ any, reason EvictReason) {
		if reason == EvictExpired {
			expiredKeys = append(expiredKeys, key)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.Set("keep", 1)
	cache.SetTTL("gone1", 2, 5*time.Millisecond)
	cache.SetTTL("gone2", 3, 5*time.Millisecond)
	time.Sleep(15 * time.Millisecond)
	cache.Cleanup()
	if cache.Len() != 1 {
		t.Errorf("Cleanup 后 Len = %d,want 1", cache.Len())
	}
	if _, ok := cache.Get("keep"); !ok {
		t.Error("未过期条目应保留")
	}
	if cache.Stats().Expired != 2 {
		t.Errorf("Expired = %d,want 2", cache.Stats().Expired)
	}
	if len(expiredKeys) != 2 {
		t.Errorf("过期回调数量 = %d,want 2", len(expiredKeys))
	}
}

func TestCleanupNoExpired(t *testing.T) {
	cache, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.Set("a", 1)
	cache.Cleanup()
	if cache.Len() != 1 || cache.Stats().Expired != 0 {
		t.Errorf("无过期条目不应清理:Len=%d Expired=%d", cache.Len(), cache.Stats().Expired)
	}
}

func TestBackgroundCleanup(t *testing.T) {
	cache, err := New(WithCleanupInterval(15 * time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.SetTTL("k", "v", 5*time.Millisecond)
	deadline := time.Now().Add(2 * time.Second)
	for cache.Len() > 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if cache.Len() != 0 {
		t.Error("后台清理应移除过期条目")
	}
}

func TestCloseStopsCleanup(t *testing.T) {
	cache, err := New(WithCleanupInterval(10 * time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	cache.SetTTL("k", "v", 5*time.Millisecond)
	cache.Close()
	cache.Close() // 幂等
	time.Sleep(40 * time.Millisecond)
	if cache.Len() != 1 {
		t.Error("Close 后后台清理应停止")
	}
	// Close 不冻结读写
	cache.Set("x", 1)
	if v, ok := cache.Get("x"); !ok || v != 1 {
		t.Error("Close 后读写应仍可用")
	}
	cache.Cleanup()
	if cache.Len() != 1 {
		t.Error("手动 Cleanup 仍应可用")
	}
}

func TestConcurrentOperations(t *testing.T) {
	cache, err := New(WithCapacity(1000), WithCleanupInterval(10*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				key := string(rune('a'+id)) + string(rune('0'+j%10))
				cache.SetTTL(key, j, time.Millisecond)
				_, _ = cache.Get(key)
				if j%7 == 0 {
					cache.Delete(key)
				}
				if j%50 == 0 {
					cache.Cleanup()
				}
			}
		}(i)
	}
	wg.Wait()
	_ = cache.Stats()
}

func TestHashKeyDeterministic(t *testing.T) {
	cache, err := New(WithShards(16))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	first := cache.shardFor("user:1")
	for i := 0; i < 100; i++ {
		if cache.shardFor("user:1") != first {
			t.Fatal("同 key 应落入同一分片")
		}
	}
}

func TestEvictOldestEmpty(t *testing.T) {
	cache, err := New(WithCapacity(10))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	if _, ok := cache.evictOldest(); ok {
		t.Error("空缓存不应逐出条目")
	}
}

func TestSetTTLStopsWhenNoVictim(t *testing.T) {
	cache, err := New(WithCapacity(2))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	// 模拟全局计数异常:缓存为空但计数超限,逐出应安全停止。
	cache.globalLen.Store(3)
	cache.Set("x", 1)
	if cache.Stats().Evictions != 1 {
		t.Error("超限条目应被逐出")
	}
	total := 0
	for _, s := range cache.shards {
		total += s.lru.Len()
	}
	if total != 0 {
		t.Error("逐出后分片实际条目应为空")
	}
}

func TestEvictOldestStaleVictim(t *testing.T) {
	cache, err := New(WithShards(1), WithCapacity(10))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	// 模拟并发竞争:tailSeq 显示有候选,但锁内链表已空。
	cache.shards[0].tailSeq.Store(42)
	if _, ok := cache.evictOldest(); ok {
		t.Error("空链表不应被逐出")
	}
}

func TestLRURefreshExact(t *testing.T) {
	cache, err := New(WithCapacity(2), WithLRURefresh(1))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.Set("a", 1)
	cache.Set("b", 2)
	if _, ok := cache.Get("a"); !ok {
		t.Fatal("a 应命中")
	}
	cache.Set("c", 3)
	if _, ok := cache.Get("b"); ok {
		t.Error("精确模式下 a 已刷新,b 应被逐出")
	}
	if _, ok := cache.Get("a"); !ok {
		t.Error("a 应保留")
	}
}

func TestLRURefreshSampled(t *testing.T) {
	cache, err := New(WithCapacity(2), WithLRURefresh(8))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.Set("a", 1)
	cache.Set("b", 2)
	if _, ok := cache.Get("a"); !ok {
		t.Fatal("a 应命中")
	}
	// 仅 1 次命中,未到采样间隔 8,a 未刷新 → 逐出 a。
	cache.Set("c", 3)
	if _, ok := cache.Get("a"); ok {
		t.Error("采样模式下 a 未刷新应被逐出")
	}
	if _, ok := cache.Get("b"); !ok {
		t.Error("b 应保留")
	}
}

func TestLRURefreshSampledAfterInterval(t *testing.T) {
	cache, err := New(WithCapacity(2), WithLRURefresh(4))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	cache.Set("a", 1)
	cache.Set("b", 2)
	for i := 0; i < 4; i++ {
		if _, ok := cache.Get("a"); !ok {
			t.Fatal("a 应命中")
		}
	}
	// 第 4 次命中触发刷新 → 逐出 b。
	cache.Set("c", 3)
	if _, ok := cache.Get("b"); ok {
		t.Error("采样到间隔后 a 已刷新,b 应被逐出")
	}
}

func TestLRURefreshInvalid(t *testing.T) {
	if _, err := New(WithLRURefresh(0)); err == nil {
		t.Error("LRURefresh=0 应非法")
	}
	if _, err := New(WithLRURefresh(-1)); err == nil {
		t.Error("LRURefresh 负数应非法")
	}
}
