package core

import "github.com/lcylpzls/metricsx"

// Metrics 是最小指标协议（家族统一契约，定义见 metricsx.Sink）。
// 调用方按 Sink 签名传入标签切片；无标签时传 nil。
type Metrics = metricsx.Sink

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
