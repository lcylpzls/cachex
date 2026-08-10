package core

import (
	"sync"
	"sync/atomic"
	"time"
)

// entry 是缓存中的单个条目,同时作为侵入式链表节点。
type entry struct {
	key     string
	value   any
	expire  time.Time // zero 表示永不过期
	lastSeq uint64    // 最近访问序号,用于跨分片 LRU 比较(严格单调)
	hits    uint32    // 命中计数,采样刷新时使用
	prev    *entry
	next    *entry
}

// evictItem 是待锁外回调的逐出结果。
type evictItem struct {
	key    string
	value  any
	reason EvictReason
}

// lruList 是侵入式环形双向链表,head 为哨兵节点。
// head.next 是最近使用,head.prev 是最久未用。
type lruList struct {
	head entry
	size int
}

// init 初始化空链表。
func (l *lruList) init() {
	l.head.next = &l.head
	l.head.prev = &l.head
	l.size = 0
}

// pushFront 将条目插入链表头部。
func (l *lruList) pushFront(e *entry) {
	first := l.head.next
	e.prev = &l.head
	e.next = first
	first.prev = e
	l.head.next = e
	l.size++
}

// moveToFront 将条目移到链表头部;已在头部时为空操作。
func (l *lruList) moveToFront(e *entry) {
	if e.prev == &l.head {
		return
	}
	e.prev.next = e.next
	e.next.prev = e.prev
	first := l.head.next
	e.prev = &l.head
	e.next = first
	first.prev = e
	l.head.next = e
}

// remove 将条目从链表移除。
func (l *lruList) remove(e *entry) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.prev = nil
	e.next = nil
	l.size--
}

// back 返回链尾条目(最久未用),空链表返回 nil。
func (l *lruList) back() *entry {
	if l.size == 0 {
		return nil
	}
	return l.head.prev
}

// Len 返回链表长度。
func (l *lruList) Len() int {
	return l.size
}

// shard 是缓存的一个分片:map + 侵入式 LRU 链表 + 独立锁。
// tailSeq 缓存链尾条目的访问序号,供逐出候选零锁读取。
type shard struct {
	mu      sync.Mutex
	items   map[string]*entry
	lru     lruList
	tailSeq atomic.Uint64 // 0 表示空分片
}

// newShard 创建分片。
func newShard() *shard {
	s := &shard{items: make(map[string]*entry)}
	s.lru.init()
	return s
}

// hashKey 计算 FNV-1a 64 位哈希(零分配)。
func hashKey(key string) uint64 {
	const (
		offset = uint64(14695981039346656037)
		prime  = uint64(1099511628211)
	)
	h := offset
	for i := 0; i < len(key); i++ {
		h ^= uint64(key[i])
		h *= prime
	}
	return h
}

// get 读取条目:命中更新访问序号并按刷新策略移动链表;
// 过期则删除并返回逐出结果(供锁外回调与计数)。
// refresh <= 1 表示精确刷新,>1 表示每 refresh 次命中移动一次。
func (s *shard) get(key string, now time.Time, seq uint64, refresh int) (any, bool, *evictItem) {
	s.mu.Lock()
	e, ok := s.items[key]
	if !ok {
		s.mu.Unlock()
		return nil, false, nil
	}
	if !e.expire.IsZero() && now.After(e.expire) {
		s.lru.remove(e)
		delete(s.items, e.key)
		s.updateTailSeqLocked()
		s.mu.Unlock()
		return nil, false, &evictItem{key: e.key, value: e.value, reason: EvictExpired}
	}
	e.lastSeq = seq
	e.hits++
	if refresh <= 1 || e.hits%uint32(refresh) == 0 {
		s.lru.moveToFront(e)
		s.updateTailSeqLocked()
	}
	value := e.value
	s.mu.Unlock()
	return value, true, nil
}

// set 写入条目:已存在则更新并刷新位置,返回 false;
// 新条目插入头部,返回 true。容量逐出由 Cache 层全局处理。
func (s *shard) set(key string, value any, expire, now time.Time, seq uint64, refresh int) bool {
	s.mu.Lock()
	if e, ok := s.items[key]; ok {
		e.value = value
		e.expire = expire
		e.lastSeq = seq
		e.hits++
		if refresh <= 1 || e.hits%uint32(refresh) == 0 {
			s.lru.moveToFront(e)
			s.updateTailSeqLocked()
		}
		s.mu.Unlock()
		return false
	}
	e := &entry{key: key, value: value, expire: expire, lastSeq: seq}
	s.lru.pushFront(e)
	s.items[key] = e
	if s.lru.Len() == 1 {
		s.tailSeq.Store(seq)
	}
	s.mu.Unlock()
	return true
}

// delete 删除条目,返回是否命中。
func (s *shard) delete(key string) bool {
	s.mu.Lock()
	e, ok := s.items[key]
	if !ok {
		s.mu.Unlock()
		return false
	}
	s.lru.remove(e)
	delete(s.items, key)
	s.updateTailSeqLocked()
	s.mu.Unlock()
	return true
}

// clear 清空分片(不触发逐出回调)。
func (s *shard) clear() {
	s.mu.Lock()
	s.items = make(map[string]*entry)
	s.lru.init()
	s.tailSeq.Store(0)
	s.mu.Unlock()
}

// cleanup 清理过期条目,返回待锁外回调的逐出结果。
func (s *shard) cleanup(now time.Time) []evictItem {
	s.mu.Lock()
	var expired []*entry
	// 从链尾(最久未用)向链头遍历,遇到哨兵 head 停止。
	for e := s.lru.head.prev; e != &s.lru.head; {
		prev := e.prev // 保存前一个(更新鲜)条目,删除当前后继续遍历
		if !e.expire.IsZero() && now.After(e.expire) {
			expired = append(expired, e)
		}
		e = prev
	}
	if len(expired) == 0 {
		s.mu.Unlock()
		return nil
	}
	items := make([]evictItem, 0, len(expired))
	for _, e := range expired {
		s.lru.remove(e)
		delete(s.items, e.key)
		items = append(items, evictItem{key: e.key, value: e.value, reason: EvictExpired})
	}
	s.updateTailSeqLocked()
	s.mu.Unlock()
	return items
}

// updateTailSeqLocked 同步链尾条目的访问序号(调用方须持有锁)。
func (s *shard) updateTailSeqLocked() {
	if back := s.lru.back(); back != nil {
		s.tailSeq.Store(back.lastSeq)
	} else {
		s.tailSeq.Store(0)
	}
}
