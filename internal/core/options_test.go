package core

import (
	testx "github.com/lcylpzls/testx"
	"testing"
	"time"

	"github.com/lcylpzls/errx"
)

func TestVersion(t *testing.T) {
	testx.Equal(t, Version, "v1.4.0")

}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.lruRefresh != 1 {
		t.Errorf("默认 LRU 刷新 = %d,want 1", cfg.lruRefresh)
	}
	testx.Equal(t, cfg.shards, defaultShards)

	testx.Equal(t, cfg.capacity, defaultCapacity)

	testx.Equal(t, cfg.cleanupInterval, defaultCleanupInterval)

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
				testx.RequireError(t, err)

				if code, _ := errx.CodeOf(err); code != CodeInvalidConfig {
					t.Errorf("错误码 = %s,want %s", code, CodeInvalidConfig)
				}
				return
			}
			testx.RequireNoError(t, err)

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
	testx.RequireNoError(t, err)

	testx.Equal(t, cache.cfg.shards, defaultShards)

	cache.Close()
}

func TestNewInvalidConfig(t *testing.T) {
	_, err := New(WithShards(0))
	testx.RequireError(t, err)

	if code, _ := errx.CodeOf(err); code != CodeInvalidConfig {
		t.Errorf("错误码 = %s,want %s", code, CodeInvalidConfig)
	}
}
