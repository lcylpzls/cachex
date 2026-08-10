package core

import (
	"context"
	"errors"
	testx "github.com/lcylpzls/testx"
	"sync"
	"testing"
	"time"
)

// traceCall 记录一次追踪调用。
type traceCall struct {
	name  string
	attrs map[string]string
	err   error
	ended bool
}

// fakeTraceHook 内存追踪钩子。
type fakeTraceHook struct {
	mu    sync.Mutex
	calls []traceCall
}

func (h *fakeTraceHook) Start(ctx context.Context, name string, attrs ...TraceAttr) (context.Context, func(error)) {
	h.mu.Lock()
	h.calls = append(h.calls, traceCall{name: name, attrs: map[string]string{}})
	for _, a := range attrs {
		h.calls[len(h.calls)-1].attrs[a.Key] = a.Value
	}
	h.mu.Unlock()
	return ctx, func(err error) {
		h.mu.Lock()
		h.calls[len(h.calls)-1].err = err
		h.calls[len(h.calls)-1].ended = true
		h.mu.Unlock()
	}
}

func (h *fakeTraceHook) snapshot() []traceCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]traceCall, len(h.calls))
	copy(out, h.calls)
	return out
}

// TestTraceHook 覆盖回源加载追踪埋点（成功/失败/命中不埋点）。
func TestTraceHook(t *testing.T) {
	hook := &fakeTraceHook{}
	c, err := New(WithTraceHook(hook))
	testx.RequireNoError(t, err)

	ctx := context.Background()
	v, err := GetOrSet(ctx, c, "k1", time.Minute, func(context.Context) (any, error) {
		return "v1", nil
	})
	if err != nil || v != "v1" {
		t.Fatalf("加载失败：%v %v", v, err)
	}
	// 命中缓存不再回源。
	if _, err := GetOrSet(ctx, c, "k1", time.Minute, func(context.Context) (any, error) {
		return "v2", nil
	}); err != nil {
		t.Fatal(err)
	}
	// 回源失败。
	if _, err := GetOrSet(ctx, c, "k2", time.Minute, func(context.Context) (any, error) {
		return nil, errors.New("回源失败")
	}); err == nil {
		t.Fatal("应返回回源错误")
	}

	calls := hook.snapshot()
	if len(calls) != 2 {
		t.Fatalf("应调用 2 次追踪钩子，实际：%d", len(calls))
	}
	for i, call := range calls {
		if call.name != "cachex.load" || call.attrs["cachex.operation"] != "load" ||
			call.attrs["cachex.key"] == "" || !call.ended {
			t.Fatalf("第 %d 次追踪调用不符：%+v", i, call)
		}
	}
	if calls[1].err == nil {
		t.Fatal("失败回源应记录错误")
	}
}
