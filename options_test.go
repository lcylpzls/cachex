package cachex

import (
	"testing"
	"time"

	"github.com/lcylpzls/errx"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.lruRefresh != 1 {
		t.Errorf("默认 LRU 刷新 = %d,want 1", cfg.lruRefresh)
	}
	if cfg.shards != defaultShards {
		t.Errorf("默认分片 = %d,want %d", cfg.shards, defaultShards)
	}
	if cfg.capacity != defaultCapacity {
		t.Errorf("默认容量 = %d,want %d", cfg.capacity, defaultCapacity)
	}
	if cfg.cleanupInterval != defaultCleanupInterval {
		t.Errorf("默认清理间隔 = %v,want %v", cfg.cleanupInterval, defaultCleanupInterval)
	}
	if cfg.defaultTTL != 0 || cfg.metrics != nil || cfg.onEvicted != nil {
		t.Error("默认 TTL/观测/回调应为关闭")
	}
}

func TestOptionsApply(t *testing.T) {
	metrics := newFakeMetrics()
	var evicted int
	cfg := defaultConfig()
	opts := []Option{
		WithShards(32),
		WithCapacity(5000),
		WithDefaultTTL(30 * time.Second),
		WithCleanupInterval(10 * time.Second),
		WithMetrics(metrics),
		WithOnEvicted(func(string, any, EvictReason) { evicted++ }),
		WithLRURefresh(4),
		nil,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.shards != 32 || cfg.capacity != 5000 ||
		cfg.defaultTTL != 30*time.Second || cfg.cleanupInterval != 10*time.Second {
		t.Error("基础选项应用失败")
	}
	if cfg.metrics != metrics || cfg.onEvicted == nil {
		t.Error("观测/回调选项应用失败")
	}
	if cfg.lruRefresh != 4 {
		t.Errorf("LRU 刷新选项应用失败:%d", cfg.lruRefresh)
	}
	if evicted != 0 {
		t.Error("回调不应立即调用")
	}
}

func TestValidateConfig(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*config)
		wantErr bool
	}{
		{"默认合法", func(*config) {}, false},
		{"分片为 0", func(c *config) { c.shards = 0 }, true},
		{"分片负数", func(c *config) { c.shards = -1 }, true},
		{"容量负数", func(c *config) { c.capacity = -1 }, true},
		{"默认 TTL 负数", func(c *config) { c.defaultTTL = -1 }, true},
		{"清理间隔负数", func(c *config) { c.cleanupInterval = -1 }, true},
		{"LRU 刷新为 0", func(c *config) { c.lruRefresh = 0 }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := defaultConfig()
			tc.mutate(&cfg)
			err := validateConfig(cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatal("应返回错误")
				}
				if code, _ := errx.CodeOf(err); code != CodeInvalidConfig {
					t.Errorf("错误码 = %s,want %s", code, CodeInvalidConfig)
				}
				return
			}
			if err != nil {
				t.Fatalf("不应返回错误:%v", err)
			}
		})
	}
}

func TestEvictReasonString(t *testing.T) {
	cases := []struct {
		r    EvictReason
		want string
	}{
		{EvictCapacity, "capacity"},
		{EvictExpired, "expired"},
		{EvictReason(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.r.String(); got != tc.want {
			t.Errorf("%v:String = %q,want %q", tc.r, got, tc.want)
		}
	}
}

func TestNewNilOptions(t *testing.T) {
	cache, err := New(nil)
	if err != nil {
		t.Fatalf("nil 选项应被忽略:%v", err)
	}
	if cache.cfg.shards != defaultShards {
		t.Error("nil 选项不应改变默认配置")
	}
	cache.Close()
}

func TestNewInvalidConfig(t *testing.T) {
	_, err := New(WithShards(0))
	if err == nil {
		t.Fatal("非法分片应返回错误")
	}
	if code, _ := errx.CodeOf(err); code != CodeInvalidConfig {
		t.Errorf("错误码 = %s,want %s", code, CodeInvalidConfig)
	}
}
