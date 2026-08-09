# cachex 架构设计

> 版本:v0.1.0(规划稿) · 状态:评审中

## 1. 总体分层

```text
业务代码
├── cachex.Cache(Get/Set/Delete/Clear/GetOrSet)
├── 分片层(shard: map + LRU 链表 + mutex)
├── 过期层(条目 TTL + 惰性过期 + Cleanup)
├── 击穿防护层(Singleflight 回源合并)
└── 观测层(Stats 快照 / Metrics 注入 / 逐出回调)
```

## 2. 核心模块职责

| 模块 | 职责 |
| --- | --- |
| `cache.go` | `Cache`、`New`、`Get/Set/SetTTL/Delete/Clear/Len` |
| `shard.go` | 分片结构:hash 分桶、LRU 链表、容量逐出 |
| `ttl.go` | 过期判定、`Cleanup`、后台清理任务 |
| `load.go` | `GetOrSet` Singleflight 合并回源 |
| `stats.go` | 原子计数与 `Stats()` 快照 |
| `metrics.go` | 指标接口与注入 |
| `options.go` | `Option`、默认值、配置校验 |
| `errors.go` | `CACHEX_*` 错误码 |

## 3. 分片与 LRU

- 默认 16 个 shard,键经 FNV-1a 哈希取模分桶;
- 每 shard:`map[string]*entry` + 双向链表(container/list);
- entry 保存 key / value / expire / 链表元素;
- 读命中将条目移到链表头(写锁,分片收敛竞争);
- 写入超容量:逐出链表尾(最久未用),触发 `OnEvicted`;
- 更新已存在键:更新值、刷新链表位置(不新增条目)。
- `Clear()` 直接重建全部 shard,不触发逐出回调;
- **逐出回调在锁外调用**:先收集逐出结果,释放 shard 锁后再回调,
  回调可安全反查或写入 cache。

## 4. 过期模型

- 每条目独立 `expire` 时间戳(0 表示永不过期);
- **惰性过期**:Get 命中时校验过期,过期即删除并计 `expired`;
- **定期清理**:后台 goroutine 按 `WithCleanupInterval`
  扫描各分片清理过期条目;`Cleanup()` 可手动触发;
- `Close()` 停止后台任务,可重复调用(幂等)。
- `Close()` 不冻结缓存:后续 Get / Set 仍可用,
  只是不再有后台清理(可手动 `Cleanup`)。

## 5. 击穿防护(GetOrSet)

- 未命中时按 key 进入 singleflight 分组;
- 首个调用者执行 loader,其余等待共享结果;
- loader 成功:按 ttl 写入缓存后返回;
- loader 失败:不缓存,错误透传给全部等待者;
- 等待者 ctx 取消:立即返回取消错误,不中断执行者;
- 执行者 ctx 取消:loader 收到取消,由 loader 自行决定(透传)。

## 6. 观测

- 原子计数器:hits / misses / sets / deletes / evictions / expired;
- `Stats()` 返回快照(含 Len 与容量);
- Metrics 注入后输出 `cachex.hits` / `cachex.misses` 等计数;
- `GetOrSet` 记录 `cachex.loader_duration` 回源耗时;
- 逐出回调带 reason(`EvictCapacity` / `EvictExpired`)。

## 7. 错误模型

- 配置非法返回 `CACHEX_INVALID_CONFIG`(errx);
- loader 错误原样透传,不包装(由调用方 errx 语义负责);
- `Close` / `Cleanup` 不产生错误(幂等安全)。

## 8. 目标目录结构

```text
cachex/
├── README.md
├── CHANGELOG.md
├── go.mod           # module github.com/lcylpzls/cachex
├── cache.go
├── shard.go
├── ttl.go
├── load.go
├── stats.go
├── metrics.go
├── options.go
├── errors.go
├── docs/
├── examples/
└── bench_test.go
```

## 9. 依赖策略

- 仅标准库 + errx(错误码)+ logx(可选观测);
- 禁止第三方依赖,保持零第三方承诺。
