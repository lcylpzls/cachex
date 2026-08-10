package core

import "context"

// CacheEvent 描述一次缓存操作事件。
type CacheEvent struct {
	// Action 操作类型：get_hit / get_miss / set / delete / clear / evict。
	Action string
	// Key 键名；Clear 等全局操作为空。
	Key string
	// Err 操作结果错误；nil 表示成功。
	Err error
}

// EventHook 是可选事件钩子（默认 no-op），由 eventx 等外部适配器接入。
type EventHook interface {
	// OnCacheEvent 在缓存操作结束时调用。
	OnCacheEvent(ctx context.Context, e CacheEvent)
}
