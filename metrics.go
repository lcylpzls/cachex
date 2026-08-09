package cachex

// Metrics 是最小指标接口,默认 no-op,便于外部接 Prometheus 等适配器。
type Metrics interface {
	// IncCounter 增加一个计数指标。
	IncCounter(name string, labels ...string)
	// ObserveDuration 记录一次耗时观测。
	ObserveDuration(name string, seconds float64, labels ...string)
}

// 指标名称:统一为 cachex.* 前缀。
const (
	metricHits      = "cachex.hits"
	metricMisses    = "cachex.misses"
	metricSets      = "cachex.sets"
	metricDeletes   = "cachex.deletes"
	metricEvictions = "cachex.evictions"
	metricExpired   = "cachex.expired"
	metricLoaderDur = "cachex.loader_duration"
)
