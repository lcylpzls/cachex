package cachex

import (
	"context"
	"strconv"
	"testing"
)

// BenchmarkGetHit 基准:16 分片命中读取。
func BenchmarkGetHit(b *testing.B) {
	cache, err := New()
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()
	cache.Set("key", 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get("key")
	}
}

// BenchmarkSet 基准:写入固定键。
func BenchmarkSet(b *testing.B) {
	cache, err := New()
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("key", i)
	}
}

// BenchmarkGetOrSetHit 基准:GetOrSet 命中路径。
func BenchmarkGetOrSetHit(b *testing.B) {
	cache, err := New()
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()
	cache.Set("key", 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetOrSet(b.Context(), cache, "key", 0, func(context.Context) (any, error) {
			return 1, nil
		})
	}
}

// BenchmarkEviction 基准:容量逐出循环。
func BenchmarkEviction(b *testing.B) {
	cache, err := New(WithCapacity(1000))
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("key"+strconv.Itoa(i%2000), i)
	}
}

// BenchmarkShardedSet 基准:16 分片分散写入。
func BenchmarkShardedSet(b *testing.B) {
	cache, err := New(WithShards(16))
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("user:"+strconv.Itoa(i%100000), i)
	}
}
