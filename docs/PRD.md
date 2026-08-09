# cachex 产品需求(PRD)

> 版本:v0.1.0(规划稿) · 状态:评审中

## 1. 背景与动机

业务代码里到处是「map + 锁 + TTL」的临时缓存:

- 淘汰策略、容量上限、过期清理各写各的,参数随手拍;
- 缓存击穿(热点 key 失效瞬间打爆下游)没有统一防护;
- 命中率、逐出量不可观测,出了问题只能猜;
- 与 errx / logx 生态不打通。

结论:**做一个薄的高性能内存缓存库**,
把并发分片、LRU 淘汰、TTL 过期、击穿防护与观测沉淀为库能力,
业务代码只描述「存什么、多久过期、多大容量」。

## 2. 目标

1. 并发分片 + 精确 LRU,默认配置贴合生产实践;
2. 每条目独立 TTL,惰性过期 + 后台清理双通道;
3. `GetOrSet` 回源加载,并发未命中合并为一次(Singleflight);
4. 命中率与逐出可观测,指标外部注入(同 dbx / httpx 形态);
5. 与 errx 打通(配置校验错误码),零第三方依赖;
6. 命中路径无额外分配,基准与手写分片 map 同量级。

## 3. 非目标(明确不做)

- 不做分布式缓存 / 一致性哈希 / 节点同步(groupcache 范畴);
- 不做持久化(磁盘缓存是另一个产品);
- 不做 LFU / TinyLFU / 成本模型(0.1.0 保持 LRU + 条目容量);
- 不做字节存储([]byte 特化,保持 any 类型便利);
- 不提供全局单例,显式 `New`;
- 不做本地多级缓存(与进程内二级缓存解耦)。

## 4. 能力需求

### 4.1 核心

- `New(opts ...Option) (*Cache, error)`;
- `Get(key) (any, bool)`、`Set(key, value)`、`SetTTL(key, value, ttl)`;
- `Delete(key)`、`Clear()`、`Len()`;
- 容量超限按 LRU 逐出,逐出回调可感知原因(容量 / 过期)。

### 4.2 过期

- 每条目独立 TTL,惰性过期(读时判断);
- `Cleanup()` 手动清理过期条目;
- `WithCleanupInterval(d)` 开启后台清理(默认 1 分钟),
  `Close()` 停止清理并释放资源。

### 4.3 击穿防护

- `GetOrSet(ctx, key, ttl, loader)`:
  命中直接返回;未命中合并并发回源;
  loader 失败不缓存,等待者共享错误;
  等待者 ctx 取消立即返回,执行者继续。

### 4.4 可观测

- `Stats()`:命中 / 未命中 / 写入 / 删除 / 逐出 / 过期计数;
- `WithMetrics` 外部注入,指标名 `cachex.*`,默认 no-op;
- `GetOrSet` 记录回源耗时(`cachex.loader_duration`);
- 逐出回调 `WithOnEvicted`。

### 4.5 配置

- `WithShards(n)`(默认 16);
- `WithCapacity(n)`(总条目上限,默认 10000);
- `WithDefaultTTL(d)`(默认不过期);
- `WithCleanupInterval(d)`(默认 1 分钟,0 关闭);
- `WithMetrics` / `WithOnEvicted`。

## 5. 非功能需求

- **性能**:Get 命中与手写分片 map 同量级(目标见迭代计划);
- **资源**:分片锁收敛竞争,后台清理低频率低开销;
- **质量**:语句覆盖率 100%、race、staticcheck、vet、fuzz、三平台 CI;
- **依赖**:标准库 + errx + logx(可选观测),零第三方。

## 6. 验收标准

v0.1.0 发布时:

1. LRU 淘汰、TTL 惰性过期、Cleanup、逐出回调全路径测试;
2. Singleflight 并发合并、loader 失败、ctx 取消全路径测试;
3. Stats / Metrics / 配置校验全路径测试;
4. 100% 语句覆盖率,`go test -race ./...`、staticcheck、vet 全绿;
5. 基准与手写分片 map 对比基线写入文档。
