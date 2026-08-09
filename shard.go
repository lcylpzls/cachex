package cachex

import (
	"container/list"
	"sync"
	"time"
)

// entry 是缓存中的单个条目。
type entry struct {
	key     string
	value   any
	expire  time.Time // zero 表示永不过期
	lastSeq uint64    // 最近访问序号,用于跨分片 LRU 比较(严格单调)
	element *list.Element
}

// evictItem 是待锁外回调的逐出结果。
type evictItem struct {
	key    string
	value  any
	reason EvictReason
}

// shard 是缓存的一个分片:map + LRU 双向链表 + 独立锁。
type shard struct {
	mu    sync.Mutex
	items map[string]*entry
	lru   *list.List
}

// newShard 创建分片。
func newShard() *shard {
	return &shard{
		items: make(map[string]*entry),
		lru:   list.New(),
	}
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

// get 读取条目:命中刷新 LRU 顺序;过期则删除并返回逐出结果。
func (s *shard) get(key string, now time.Time, seq uint64) (any, bool, *evictItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.items[key]
	if !ok {
		return nil, false, nil
	}
	if !e.expire.IsZero() && now.After(e.expire) {
		s.removeLocked(e)
		return nil, false, &evictItem{key: e.key, value: e.value, reason: EvictExpired}
	}
	e.lastSeq = seq
	s.lru.MoveToFront(e.element)
	return e.value, true, nil
}

// set 写入条目:已存在则更新并刷新位置,返回 false;
// 新条目插入头部,返回 true。容量逐出由 Cache 层全局处理。
func (s *shard) set(key string, value any, expire, now time.Time, seq uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.items[key]; ok {
		e.value = value
		e.expire = expire
		e.lastSeq = seq
		s.lru.MoveToFront(e.element)
		return false
	}
	e := &entry{key: key, value: value, expire: expire, lastSeq: seq}
	e.element = s.lru.PushFront(e)
	s.items[key] = e
	return true
}

// delete 删除条目,返回是否命中。
func (s *shard) delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.items[key]
	if !ok {
		return false
	}
	s.removeLocked(e)
	return true
}

// tail 返回链尾条目副本(最久未用,含访问序号),空链表返回 nil。
func (s *shard) tail() *entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	back := s.lru.Back()
	if back == nil {
		return nil
	}
	e := back.Value.(*entry)
	return &entry{key: e.key, value: e.value, lastSeq: e.lastSeq, element: e.element}
}

// removeEntry 删除指定条目;条目已被其他 goroutine 删除或替换时返回 false。
func (s *shard) removeEntry(e *entry) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.items[e.key]
	if !ok || cur.element != e.element {
		return false
	}
	s.removeLocked(cur)
	return true
}

// clear 清空分片(不触发逐出回调)。
func (s *shard) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[string]*entry)
	s.lru.Init()
}

// cleanup 清理过期条目,返回待锁外回调的逐出结果。
func (s *shard) cleanup(now time.Time) []evictItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	var expired []*entry
	for e := s.lru.Front(); e != nil; e = e.Next() {
		entry := e.Value.(*entry)
		if !entry.expire.IsZero() && now.After(entry.expire) {
			expired = append(expired, entry)
		}
	}
	if len(expired) == 0 {
		return nil
	}
	items := make([]evictItem, 0, len(expired))
	for _, entry := range expired {
		s.removeLocked(entry)
		items = append(items, evictItem{key: entry.key, value: entry.value, reason: EvictExpired})
	}
	return items
}

// removeLocked 从 map 与链表中移除条目(调用方须持有锁)。
func (s *shard) removeLocked(e *entry) {
	delete(s.items, e.key)
	s.lru.Remove(e.element)
}
