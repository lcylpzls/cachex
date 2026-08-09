package cachex

import (
	"testing"
	"time"
)

// FuzzTTL 保证任意 TTL 与键序列下缓存操作不 panic、统计自洽。
func FuzzTTL(f *testing.F) {
	f.Add("k", int64(0), int64(1000000))
	f.Add("", int64(-1), int64(-1))
	f.Fuzz(func(t *testing.T, key string, ttlNs int64, seed int64) {
		cache, err := New(WithCapacity(16))
		if err != nil {
			t.Fatal(err)
		}
		defer cache.Close()
		for i := 0; i < 20; i++ {
			ttl := time.Duration(ttlNs + int64(i)*seed)
			cache.SetTTL(key+string(rune('a'+i%26)), seed, ttl)
			_, _ = cache.Get(key)
			if i%3 == 0 {
				cache.Delete(key)
			}
			if i%5 == 0 {
				cache.Cleanup()
			}
		}
		stats := cache.Stats()
		if stats.Len < 0 || stats.Len > 16 {
			t.Fatalf("Len 越界:%d", stats.Len)
		}
	})
}

// FuzzShard 保证任意键哈希分桶稳定且在范围内。
func FuzzShard(f *testing.F) {
	f.Add("")
	f.Add("user:1")
	f.Add(string([]byte{0, 1, 2, 255}))
	f.Fuzz(func(t *testing.T, key string) {
		cache, err := New(WithShards(16))
		if err != nil {
			t.Fatal(err)
		}
		defer cache.Close()
		first := cache.shardFor(key)
		for i := 0; i < 16; i++ {
			if cache.shardFor(key) != first {
				t.Fatal("同键分桶不稳定")
			}
		}
	})
}
