package cachex

import (
	"time"

	"github.com/lcylpzls/errx"
)

// 默认配置:生产实践取值。
const (
	defaultShards          = 16
	defaultCapacity        = 10000
	defaultCleanupInterval = time.Minute
	defaultLRURefresh      = 1
)

// EvictReason 是条目被移除的原因。
type EvictReason uint8

const (
	// EvictCapacity 容量超限,LRU 逐出。
	EvictCapacity EvictReason = iota
	// EvictExpired 条目过期被清理。
	EvictExpired
)

// String 返回逐出原因的稳定名称。
func (r EvictReason) String() string {
	switch r {
	case EvictCapacity:
		return "capacity"
	case EvictExpired:
		return "expired"
	default:
		return "unknown"
	}
}

// config 是 Cache 的完整配置,由 Option 按顺序修改。
type config struct {
	shards          int
	capacity        int
	defaultTTL      time.Duration
	cleanupInterval time.Duration
	metrics         Metrics
	onEvicted       func(key string, value any, reason EvictReason)
	lruRefresh      int
}

// defaultConfig 返回默认配置。
func defaultConfig() config {
	return config{
		shards:          defaultShards,
		capacity:        defaultCapacity,
		cleanupInterval: defaultCleanupInterval,
		lruRefresh:      defaultLRURefresh,
	}
}

// Option 修改 Cache 配置,在 New 时按顺序应用。
type Option func(*config)

// WithShards 设置分片数量(必须大于 0,默认 16)。
func WithShards(n int) Option {
	return func(c *config) { c.shards = n }
}

// WithCapacity 设置总条目上限;0 表示不限制(默认 10000)。
func WithCapacity(n int) Option {
	return func(c *config) { c.capacity = n }
}

// WithDefaultTTL 设置默认过期时长;0 表示永不过期(默认)。
// 显式 SetTTL 仍可覆盖单个条目。
func WithDefaultTTL(d time.Duration) Option {
	return func(c *config) { c.defaultTTL = d }
}

// WithCleanupInterval 设置后台清理间隔;0 关闭后台清理(默认 1 分钟)。
func WithCleanupInterval(d time.Duration) Option {
	return func(c *config) { c.cleanupInterval = d }
}

// WithMetrics 注入指标钩子,空表示关闭指标(默认)。
func WithMetrics(m Metrics) Option {
	return func(c *config) { c.metrics = m }
}

// WithOnEvicted 设置逐出回调:容量逐出与过期清理时在锁外调用,
// 回调内可安全反查或写入 cache。Clear 不触发回调。
func WithOnEvicted(fn func(key string, value any, reason EvictReason)) Option {
	return func(c *config) { c.onEvicted = fn }
}

// WithLRURefresh 设置 LRU 刷新策略:1 表示精确 LRU
// (每次命中都移动链表,默认);>1 表示采样刷新
// (每 interval 次命中移动一次,写锁竞争降至 1/interval)。
func WithLRURefresh(interval int) Option {
	return func(c *config) { c.lruRefresh = interval }
}

// validateConfig 校验配置参数。
func validateConfig(cfg config) error {
	if cfg.shards <= 0 {
		return errx.NewCode(CodeInvalidConfig, "Shards 必须大于 0")
	}
	if cfg.capacity < 0 {
		return errx.NewCode(CodeInvalidConfig, "Capacity 不能为负数")
	}
	if cfg.defaultTTL < 0 {
		return errx.NewCode(CodeInvalidConfig, "DefaultTTL 不能为负数")
	}
	if cfg.cleanupInterval < 0 {
		return errx.NewCode(CodeInvalidConfig, "CleanupInterval 不能为负数")
	}
	if cfg.lruRefresh < 1 {
		return errx.NewCode(CodeInvalidConfig, "LRURefresh 必须大于等于 1")
	}
	return nil
}
