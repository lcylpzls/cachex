package cachex

import (
	"sync"
	"sync/atomic"
	"time"
)

// Cache 是自研内存缓存入口,持有分片、统计与生命周期。
// 所有方法并发安全,可在多个 goroutine 间共享。
type Cache struct {
	cfg       config
	shards    []*shard
	stats     cacheStats
	globalLen atomic.Int64
	accessSeq atomic.Uint64
	loads     *loadGroup
	stopCh    chan struct{}
	stopOnce  sync.Once
}

// New 创建缓存。配置非法时返回 CACHEX_INVALID_CONFIG。
func New(opts ...Option) (*Cache, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	c := &Cache{
		cfg:    cfg,
		shards: make([]*shard, cfg.shards),
		loads:  newLoadGroup(),
		stopCh: make(chan struct{}),
	}
	for i := range c.shards {
		c.shards[i] = newShard()
	}
	if cfg.cleanupInterval > 0 {
		go c.cleanupLoop(cfg.cleanupInterval)
	}
	return c, nil
}

// Get 读取条目:命中返回并刷新 LRU 顺序;
// 不存在或已过期返回 false(过期条目即时删除)。
func (c *Cache) Get(key string) (any, bool) {
	now := time.Now()
	s := c.shardFor(key)
	value, ok, expired := s.get(key, now, c.accessSeq.Add(1), c.cfg.lruRefresh)
	if expired != nil {
		c.stats.expired.Add(1)
		c.globalLen.Add(-1)
		c.emitMetric(metricExpired)
		c.notifyEvicted(*expired)
	}
	if ok {
		c.stats.hits.Add(1)
		c.emitMetric(metricHits)
	} else {
		c.stats.misses.Add(1)
		c.emitMetric(metricMisses)
	}
	return value, ok
}

// Set 写入条目,使用默认 TTL;已存在时更新值并刷新 LRU 位置。
func (c *Cache) Set(key string, value any) {
	c.SetTTL(key, value, 0)
}

// SetTTL 写入条目并指定 TTL;ttl <= 0 表示永不过期。
// 容量超限时按 LRU 逐出最久未用条目。
func (c *Cache) SetTTL(key string, value any, ttl time.Duration) {
	now := time.Now()
	seq := c.accessSeq.Add(1)
	expire := time.Time{}
	if ttl > 0 {
		expire = now.Add(ttl)
	}
	inserted := c.shardFor(key).set(key, value, expire, now, seq, c.cfg.lruRefresh)
	if inserted {
		c.globalLen.Add(1)
	}
	c.stats.sets.Add(1)
	c.emitMetric(metricSets)
	// 全局容量逐出:跨分片比较访问时间,逐出最久未用条目。
	var evicted []evictItem
	for c.cfg.capacity > 0 && c.globalLen.Load() > int64(c.cfg.capacity) {
		item, ok := c.evictOldest()
		if !ok {
			break
		}
		c.globalLen.Add(-1)
		evicted = append(evicted, item)
	}
	for _, item := range evicted {
		c.stats.evictions.Add(1)
		c.emitMetric(metricEvictions)
		c.notifyEvicted(item)
	}
}

// Delete 删除条目,返回是否命中。
func (c *Cache) Delete(key string) bool {
	if c.shardFor(key).delete(key) {
		c.stats.deletes.Add(1)
		c.globalLen.Add(-1)
		c.emitMetric(metricDeletes)
		return true
	}
	return false
}

// Clear 清空全部条目,不触发逐出回调。
func (c *Cache) Clear() {
	for _, s := range c.shards {
		s.clear()
	}
	c.globalLen.Store(0)
}

// Len 返回当前条目数(全局原子计数)。
func (c *Cache) Len() int {
	return int(c.globalLen.Load())
}

// Cleanup 手动清理全部过期条目,幂等且并发安全。
func (c *Cache) Cleanup() {
	now := time.Now()
	var evicted []evictItem
	for _, s := range c.shards {
		items := s.cleanup(now)
		if len(items) > 0 {
			evicted = append(evicted, items...)
			c.globalLen.Add(-int64(len(items)))
		}
	}
	for _, item := range evicted {
		c.stats.expired.Add(1)
		c.emitMetric(metricExpired)
		c.notifyEvicted(item)
	}
}

// Close 停止后台清理任务,幂等可重复调用。
// 不冻结缓存:后续读写仍可用,只是不再自动清理。
func (c *Cache) Close() {
	c.stopOnce.Do(func() { close(c.stopCh) })
}

// cleanupLoop 按间隔定期清理过期条目,直到 Close。
func (c *Cache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.Cleanup()
		}
	}
}

// shardFor 返回 key 对应的分片(FNV-1a 哈希取模)。
func (c *Cache) shardFor(key string) *shard {
	return c.shards[hashKey(key)%uint64(len(c.shards))]
}

// notifyEvicted 在锁外调用逐出回调。
func (c *Cache) notifyEvicted(item evictItem) {
	if c.cfg.onEvicted != nil {
		c.cfg.onEvicted(item.key, item.value, item.reason)
	}
}

// evictOldest 跨分片逐出全局最久未用条目:
// 只读各分片尾部序号原子值选候选,仅锁目标分片。
// 并发竞争导致目标分片已空时返回 false,由调用方重试或停止。
func (c *Cache) evictOldest() (evictItem, bool) {
	bestIdx := -1
	var bestSeq uint64
	for i, s := range c.shards {
		seq := s.tailSeq.Load()
		if seq == 0 {
			continue
		}
		if bestIdx == -1 || seq < bestSeq {
			bestIdx = i
			bestSeq = seq
		}
	}
	if bestIdx == -1 {
		return evictItem{}, false
	}
	s := c.shards[bestIdx]
	s.mu.Lock()
	back := s.lru.back()
	if back == nil {
		s.mu.Unlock()
		return evictItem{}, false
	}
	s.lru.remove(back)
	delete(s.items, back.key)
	s.updateTailSeqLocked()
	s.mu.Unlock()
	return evictItem{key: back.key, value: back.value, reason: EvictCapacity}, true
}

// emitMetric 输出计数指标(未注入时为空操作)。
func (c *Cache) emitMetric(name string) {
	if c.cfg.metrics != nil {
		c.cfg.metrics.IncCounter(name)
	}
}
