package cachex

import (
	"context"
	"github.com/lcylpzls/testx"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestEventHook(t *testing.T) {
	hook := &fakeEventHook{}
	c, err := New(
		WithCapacity(2),
		WithCleanupInterval(0),
		WithEventHook(hook),
	)
	testx.RequireNoError(t, err)
	defer c.Close()

	c.Set("a", 1)
	_, _ = c.Get("a")
	_, _ = c.Get("miss")
	c.Set("b", 2)
	c.Set("c", 3) // 容量 2，逐出 a
	c.Delete("b")
	c.Clear()

	events := hook.snapshot()
	var actions []string
	for _, e := range events {
		actions = append(actions, e.Action)
	}
	got := strings.Join(actions, ",")
	for _, want := range []string{"set", "get_hit", "get_miss", "evict", "delete", "clear"} {
		if !strings.Contains(got, want) {
			t.Fatalf("事件缺少 %s：%s", want, got)
		}
	}
	if len(events) < 6 {
		t.Fatalf("事件数量不足：%d", len(events))
	}
}

func TestEventHookExpired(t *testing.T) {
	hook := &fakeEventHook{}
	c, err := New(
		WithCleanupInterval(10*time.Millisecond),
		WithEventHook(hook),
	)
	testx.RequireNoError(t, err)
	defer c.Close()
	c.SetTTL("k", 1, time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	_, _ = c.Get("k") // 触发过期删除
	time.Sleep(30 * time.Millisecond)
	var hasEvict bool
	for _, e := range hook.snapshot() {
		if e.Action == "evict" {
			hasEvict = true
		}
	}
	if !hasEvict {
		t.Fatal("过期条目应产生 evict 事件")
	}
}

func TestNoEventHook(t *testing.T) {
	c, err := New()
	testx.RequireNoError(t, err)
	defer c.Close()
	c.Set("k", 1)
	_, _ = c.Get("k")
}

type fakeEventHook struct {
	mu   sync.Mutex
	list []CacheEvent
}

func (h *fakeEventHook) OnCacheEvent(_ context.Context, e CacheEvent) {
	h.mu.Lock()
	h.list = append(h.list, e)
	h.mu.Unlock()
}

func (h *fakeEventHook) snapshot() []CacheEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]CacheEvent(nil), h.list...)
}
