# cachex API 参考

> 状态:**v1.0.0 API 已冻结**。新增能力以次版本发布,
> 破坏性变更仅随主版本;任何修改须经 apidiff 对比并记录 CHANGELOG。

## 1. 快速上手

```go
cache, err := cachex.New(
	cachex.WithCapacity(10000),
	cachex.WithDefaultTTL(5*time.Minute),
)

cache.SetTTL("user:1", user, 10*time.Minute)
if v, ok := cache.Get("user:1"); ok {
	u := v.(User)
}

// 击穿防护:并发未命中只回源一次
v, err := cachex.GetOrSet(ctx, cache, "user:1", 10*time.Minute,
	func(ctx context.Context) (any, error) {
		return loadUser(ctx, 1)
	})
u := v.(User)
```

## 2. 核心类型

```go
type Cache struct { /* 未导出 */ }

func New(opts ...Option) (*Cache, error)

func (c *Cache) Get(key string) (any, bool)
func (c *Cache) Set(key string, value any)
func (c *Cache) SetTTL(key string, value any, ttl time.Duration)
func (c *Cache) Delete(key string)
func (c *Cache) Clear()
func (c *Cache) Len() int
func (c *Cache) Cleanup()
func (c *Cache) Close()
func (c *Cache) Stats() Stats
```

`Get` 命中时更新 LRU 位置;过期条目视为未命中并删除。
`SetTTL` 的 ttl <= 0 表示永不过期(与默认一致);
`Close()` 仅停止后台清理,不冻结后续读写;
`Clear()` 直接清空全部条目,不触发 OnEvicted(避免批量回调阻塞)。

所有方法并发安全,可在多个 goroutine 间共享同一 Cache。

## 3. 配置选项

```go
func WithShards(n int) Option           // 默认 16,必须 > 0
func WithCapacity(n int) Option         // 总条目上限,默认 10000,0 表示不限制
func WithDefaultTTL(d time.Duration) Option // 默认不过期,负数非法
func WithCleanupInterval(d time.Duration) Option // 默认 1 分钟,0 关闭后台清理
func WithMetrics(m Metrics) Option      // 默认 no-op
func WithOnEvicted(fn func(key string, value any, reason EvictReason)) Option
func WithLRURefresh(interval int) Option // v0.2.0:1=精确(默认),>1=采样刷新
```

## 4. 过期与清理

```go
// Cleanup 手动清理全部过期条目(幂等,并发安全)。
func (c *Cache) Cleanup()

// Close 停止后台清理任务(幂等,可重复调用)。
func (c *Cache) Close()
```

## 5. 击穿防护

```go
// GetOrSet 命中返回缓存;未命中合并并发回源。
// loader 失败不缓存,错误透传;等待者 ctx 取消立即返回。
func GetOrSet(ctx context.Context, c *Cache,
	key string, ttl time.Duration,
	loader func(context.Context) (any, error)) (any, error)
```

`GetOrSet` 设计为包级函数,便于将来扩展为 `GetOrSetBatch`
或按 key 分组,不侵入 `Cache` 方法集。

## 6. 可观测

```go
type Stats struct {
	Hits      uint64 // 命中次数
	Misses    uint64 // 未命中次数
	Sets      uint64 // 写入次数
	Deletes   uint64 // 删除次数
	Evictions uint64 // 逐出次数(容量)
	Expired   uint64 // 过期次数(惰性 + 清理)
	Len       int    // 当前条目数
	Capacity  int    // 条目上限(0 表示不限)
}

type EvictReason uint8

const (
	EvictCapacity EvictReason = iota // 容量超限,LRU 逐出
	EvictExpired                     // 过期清理
)

type Metrics interface {
	IncCounter(name string, labels ...string)
	ObserveDuration(name string, seconds float64, labels ...string)
}
```

指标名:`cachex.hits` / `cachex.misses` / `cachex.sets` /
`cachex.deletes` / `cachex.evictions` / `cachex.expired`;
`GetOrSet` 另记录 `cachex.loader_duration`(回源耗时观测)。

## 7.5 LRU 刷新策略(v0.2.0)

- `WithLRURefresh(1)`(默认):每次命中都刷新链表位置,精确 LRU;
- `WithLRURefresh(n)`(n > 1):每 n 次命中才移动链表一次,
  写锁竞争降至 1/n,LRU 为近似语义(误差随 n 增大);
- 逐出与跨分片比较始终基于单调访问序号,与刷新粒度无关。

## 7. 已确认决策

| 编号 | 决策 | 结论 |
| --- | --- | --- |
| D1 | GetOrSet 形态 | 包级函数 `GetOrSet(ctx, c, key, ttl, loader)` |
| D2 | 容量模型 | 总条目上限 `WithCapacity`(默认 10000,0 不限) |
| D3 | 后台清理默认值 | 默认开启,间隔 1 分钟,可关闭 |
| D4 | 默认 TTL | 默认不过期,显式 SetTTL |
| D5 | 值类型 | any,零拷贝 |
| D6 | 逐出回调 | WithOnEvicted + reason,锁外调用 |
| D7 | Metrics 形态 | IncCounter + ObserveDuration(同 dbx) |

## 8. 错误码

| 错误码 | 含义 |
| --- | --- |
| `CACHEX_INVALID_CONFIG` | 配置非法 |

## 9. 泛型与兼容

- 核心方法使用 `any`,类型断言由调用方负责;
- 不提供泛型方法(Go 不支持方法级泛型),避免包级泛型 API 混淆;
- `GetOrSet` 返回值 `any`,调用方自行断言。
