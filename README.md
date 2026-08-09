# cachex

自研高性能内存缓存库:并发分片、精确 LRU 淘汰、独立 TTL 过期、
Singleflight 击穿防护,与 errx / logx 生态打通。

> 当前状态:**v0.2.0 实现完成,待 CI 验证与发布**。

## 定位

cachex 不是分布式缓存,不解决「多节点共享」的问题;它解决单进程内
每个业务都要重复的部分:

- 分片锁 + LRU 淘汰,容量超限自动逐出;
- 每条目独立 TTL,惰性过期 + 后台清理;
- `GetOrSet` 回源加载,并发未命中只回源一次;
- 命中率、逐出量可观测,Metrics 外部注入。
- LRU 刷新可选:`WithLRURefresh(1)` 精确(默认),
  `WithLRURefresh(n)` 采样刷新降低写锁竞争。

所有方法并发安全,可在多个 goroutine 间共享。

## 快速上手

```go
cache, err := cachex.New(
	cachex.WithCapacity(10000),
	cachex.WithDefaultTTL(5*time.Minute),
)
if err != nil {
	panic(err)
}
defer cache.Close()

cache.SetTTL("user:1", user, 10*time.Minute)
if v, ok := cache.Get("user:1"); ok {
	u := v.(User)
}

// 击穿防护:并发未命中只回源一次
v, err := cachex.GetOrSet(ctx, cache, "user:1", 10*time.Minute,
	func(ctx context.Context) (any, error) {
		return loadUser(ctx, 1)
	})
```

## 质量门槛

- 语句覆盖率 100%,race、vet、staticcheck、fuzz、govulncheck 全绿;
- 三平台 CI(ubuntu / windows / macos);
- 性能基准与手写分片 map 同量级(见 docs/iteration-plan.md)。

## 文档

- [docs/README.md](docs/README.md) — 文档索引
- [docs/cache-research.md](docs/cache-research.md) — 缓存领域调研手册
- [examples/basic](examples/basic) — 缓存读写与 TTL
- [examples/load](examples/load) — 击穿防护 GetOrSet

## License

MIT © [lcylpzls](https://github.com/lcylpzls)
