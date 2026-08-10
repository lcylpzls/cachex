package cachex

import (
	"context"
	"time"

	"github.com/lcylpzls/cachex/internal/core"
	"github.com/lcylpzls/metricsx"
	"github.com/lcylpzls/tracex"
)

const Version = core.Version

const (
	EvictCapacity = core.EvictCapacity
	EvictExpired  = core.EvictExpired
)

const CodeInvalidConfig = core.CodeInvalidConfig

type (
	Stats       = core.Stats
	EvictReason = core.EvictReason
	Option      = core.Option
	Cache       = core.Cache
	CacheEvent  = core.CacheEvent
	EventHook   = core.EventHook
	Metrics     = metricsx.Sink
	TraceHook   = tracex.TraceHook
	TraceAttr   = tracex.TraceAttr
)

func New(opts ...Option) (*Cache, error) { return core.New(opts...) }
func GetOrSet(ctx context.Context, c *Cache, key string, ttl time.Duration,
	loader func(context.Context) (any, error)) (any, error) {
	return core.GetOrSet(ctx, c, key, ttl, loader)
}
func WithShards(n int) Option               { return core.WithShards(n) }
func WithCapacity(n int) Option             { return core.WithCapacity(n) }
func WithDefaultTTL(d time.Duration) Option { return core.WithDefaultTTL(d) }
func WithCleanupInterval(d time.Duration) Option {
	return core.WithCleanupInterval(d)
}
func WithMetrics(m Metrics) Option     { return core.WithMetrics(m) }
func WithTraceHook(h TraceHook) Option { return core.WithTraceHook(h) }
func WithEventHook(h EventHook) Option { return core.WithEventHook(h) }
func WithOnEvicted(fn func(key string, value any, reason EvictReason)) Option {
	return core.WithOnEvicted(fn)
}
func WithLRURefresh(interval int) Option { return core.WithLRURefresh(interval) }
