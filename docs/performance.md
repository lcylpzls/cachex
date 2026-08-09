# 性能基准

## 方法

```powershell
go test -run '^$' -bench . -benchmem -benchtime=1s .
```

CI 的 bench job 记录每次 main 推送的基准日志(artifact),不设硬性门禁。

## 参考数据(v0.2.0,Windows / AMD Ryzen 5 7600)

| Benchmark | ns/op | B/op | allocs/op |
| --- | --- | --- | --- |
| GetHit(精确 LRU) | 15.5 | 0 | 0 |
| GetHitSampled(interval=8) | 15.2 | 0 | 0 |
| Set | 26.3 | 8 | 0 |
| GetOrSetHit | 18.7 | 0 | 0 |
| Eviction(容量逐出) | 218.7 | 167 | 4 |
| ShardedSet(分片分散写) | 244.9 | 176 | 4 |

## v0.1.0 → v0.2.0 对比

| 场景 | v0.1.0 | v0.2.0 | 变化 |
| --- | --- | --- | --- |
| Get 命中 | 16.5 ns | 15.5 ns | -6% |
| 容量逐出 | 754 ns | 218.7 ns | **-71%** |
| 分片分散写 | 973 ns | 244.9 ns | **-75%** |

主要收益来自逐出候选原子化(只锁目标分片)与自研侵入式链表。

## 与竞品对比(临时工程,未入库)

对比代码位于本地 `.bench-compare/`(已 gitignore,
含 go-cache / bigcache / freecache / ristretto / 手写分片 map 基线):

```powershell
cd .bench-compare
go test -run '^$' -bench . -benchmem -benchtime=1s .
```

v0.1.0 实测(同机):

| 场景 | cachex | 分片 map | go-cache | bigcache | freecache | ristretto |
| --- | --- | --- | --- | --- | --- | --- |
| Get 命中 | 21.3 ns | 12.4 ns | 7.1 ns | 37.1 ns | 42.9 ns | 30.5 ns |
| Set | 28.8 ns | 24.0 ns | 21.5 ns | 87.9 ns | 43.2 ns | 177.1 ns |

解读:go-cache 的 Get 快是因为全局 RWMutex 读锁且无 LRU 刷新;
cachex 的命中要维护 LRU 顺序与统计,Set 反而快于全部字节型竞品。

## 优化原则

- 热路径零分配(Get / GetOrSet 命中 0 allocs);
- 可选能力默认关闭或保持精确语义(采样刷新由调用方决定);
- 逐出路径只锁目标分片,候选选择零锁读取。
