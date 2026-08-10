package core

import "sync/atomic"

// Stats 是缓存运行统计快照,并发安全。
type Stats struct {
	// Hits 命中次数。
	Hits uint64
	// Misses 未命中次数。
	Misses uint64
	// Sets 写入次数(含更新)。
	Sets uint64
	// Deletes 显式删除次数。
	Deletes uint64
	// Evictions 容量逐出次数。
	Evictions uint64
	// Expired 过期清理次数(惰性 + 后台)。
	Expired uint64
	// Len 当前条目数。
	Len int
	// Capacity 条目上限(0 表示不限)。
	Capacity int
}

// cacheStats 是原子计数器,保证并发安全。
type cacheStats struct {
	hits      atomic.Uint64
	misses    atomic.Uint64
	sets      atomic.Uint64
	deletes   atomic.Uint64
	evictions atomic.Uint64
	expired   atomic.Uint64
}

// Stats 返回运行统计快照。
func (c *Cache) Stats() Stats {
	return Stats{
		Hits:      c.stats.hits.Load(),
		Misses:    c.stats.misses.Load(),
		Sets:      c.stats.sets.Load(),
		Deletes:   c.stats.deletes.Load(),
		Evictions: c.stats.evictions.Load(),
		Expired:   c.stats.expired.Load(),
		Len:       c.Len(),
		Capacity:  c.cfg.capacity,
	}
}
