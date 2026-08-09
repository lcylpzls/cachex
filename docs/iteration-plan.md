# cachex 迭代计划与质量门槛

## 1. 迭代阶段

### P0 项目骨架

- go.mod(module github.com/lcylpzls/cachex,go 1.26)、目录、CI
  (三平台 + staticcheck + govulncheck + tidy + apidiff)、
  错误码注册与空包测试。

### P1 分片与 LRU 核心

- `Cache` / `New` / `Get` / `Set` / `SetTTL` / `Delete` / `Clear` / `Len`;
- FNV-1a 分片、LRU 链表、容量逐出、更新语义。

**验收**:逐出顺序、更新不增条目、并发写入无竞争错误。

### P2 过期与清理

- 惰性过期、`Cleanup()`、后台清理任务、`Close()` 幂等。

**验收**:过期即失效、清理可停可关、Close 幂等。

### P3 击穿防护

- `GetOrSet` 包级函数:合并回源、loader 失败透传、ctx 取消。

**验收**:并发未命中仅回源一次;失败/取消全路径测试;
`FuzzLoader` 通过(loader 任意返回不泄漏)。

### P4 观测

- Stats 原子计数、Metrics 注入、OnEvicted 逐出回调。

### P5 示例、基准与 v0.1.0

- examples(基础用法、击穿防护);README/docs 收尾;发布 v0.1.0。

### P6 正式版 v1.0.0(需用户批准)

- 0.1.0 线上验证后,API 冻结审查;发布 v1.0.0。

### P7 性能优化批次(v0.2.0)

- WithLRURefresh 可选采样刷新(默认精确);
- 逐出候选原子化(只锁目标分片);
- 自研侵入式链表与热路径显式解锁;
- 发布 v0.2.0(优化后 Get / 逐出基准写入文档)。

**验收**:精确与采样模式行为对比测试;100% 覆盖率;
逐出基准显著下降(对比 v0.1.0 基线)。

## 2. 质量门槛(每阶段强制)

- 语句覆盖率 100%;`go vet` / `staticcheck` 零告警;
- `go test -race` 全绿;fuzz 至少 3 个目标:`FuzzTTL`(过期边界)、
  `FuzzLoader`(回源)、`FuzzShard`(键分桶);
- 三平台 CI × Go 1.26;govulncheck 零告警;
- go.mod tidy 漂移检查;apidiff 对比上一 tag;
- 所有日志、注释、文档使用简体中文。

## 3. 性能基准(目标,实现后建立基线)

| 场景 | 目标 |
| --- | --- |
| `Get` 命中(16 shard) | 与手写分片 map 同量级 |
| `Set` 写入 | 不高于手写 map +30% |
| `GetOrSet` 命中 | 与 `Get` 同量级(仅一次 map 查找) |
| 容量逐出 | 每逐出 1 次链表操作,无额外分配 |

## 4. 风险与对策

| 风险 | 对策 |
| --- | --- |
| LRU 命中路径写锁热点 | 分片收敛;基准对比分片数敏感性 |
| 后台清理与业务竞争 | 清理仅持单 shard 锁,低频间隔 |
| Singleflight 泄漏(loader 永不返回) | ctx 取消感知 + 文档强调 loader 必须响应 ctx |
