package cachex

import "sync"

// fakeMetrics 是 Metrics 的内存实现,用于断言指标输出。
type fakeMetrics struct {
	mu        sync.Mutex
	counters  map[string]int
	durations map[string][]float64
}

func newFakeMetrics() *fakeMetrics {
	return &fakeMetrics{
		counters:  make(map[string]int),
		durations: make(map[string][]float64),
	}
}

func (m *fakeMetrics) IncCounter(name string, labels ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := name
	for _, l := range labels {
		key += "|" + l
	}
	m.counters[key]++
}

func (m *fakeMetrics) ObserveDuration(name string, seconds float64, labels ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := name
	for _, l := range labels {
		key += "|" + l
	}
	m.durations[key] = append(m.durations[key], seconds)
}

func (m *fakeMetrics) counter(name string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[name]
}

func (m *fakeMetrics) durationCount(name string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.durations[name])
}
