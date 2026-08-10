# 更新日志

本项目遵循[语义化版本](https://semver.org/lang/zh-CN/)。

## [v1.3.1] - 2026-08-10

### 变更

- 家族依赖统一对齐到最新基线（errx v1.5.4 / logx v1.3.0 / testx v1.4.1 / validx v1.2.2 / webx v1.5.2 / confx v1.0.0 / cryptox v1.0.0 等），无 API 变更。

### 质量

- 全部库包语句覆盖率保持 100%；race / vet / staticcheck / fuzz / govulncheck 全绿。

## [v1.3.0] - 2026-08-10

### 变更

- 配置校验统一迁移至家族 `validx`：注册 `cachex_config` 全局规则，调用点走 `validx.ValidateField`；
- errx 错误码保持 cachex 语义，行为不变。

### 质量

- 全部库包语句覆盖率保持 100%；race / vet / staticcheck / fuzz / govulncheck 全绿。

## [v1.2.0] - 2026-08-10

### 新增

- `EventHook` 事件钩子（零依赖可选接口，默认 no-op）：
  get_hit / get_miss / set / delete / clear / evict 操作结束时触发
  `CacheEvent`，由 eventx 等外部适配器接入；
- `WithEventHook` 选项注入。

### 质量

- 根包语句覆盖率 100%；race / vet / staticcheck / fuzz / govulncheck 全绿。

## [v1.1.3] - 2026-08-10

### 变更

- go 指令与 CI/Release 工作流统一为 Go 1.26.5；
- README Go 版本徽章同步更新。

## [v1.1.2] - 2026-08-10

### 变更

- 家族统一 Go 1.21：全部 go.mod 与 CI/Release 工作流版本号对齐 1.21；
- testx 依赖升级 v1.2.1。

## [v1.1.1] - 2026-08-10

### 修复

- examples/basic、examples/load 示例模块 go.mod 与最新依赖对齐
  （go mod tidy），修复 main CI 示例构建失败。

## [v1.1.0] - 2026-08-10

### 变更

- 家族测试底座接入：全部测试改用语义等价的 testx 断言
  （含 Require* 致命断言）；
- 测试依赖新增 `testx v1.2.0`，errx 同步升级 v1.4.0。

### 质量

- 根包语句覆盖率 100%；race / vet / staticcheck 全绿。

## [v1.0.3] - 2026-08-10

### 新增

- `TraceHook` 回源加载链路追踪钩子（零依赖接口 + `WithTraceHook`）：
  GetOrSet 实际回源时自动埋点（cachex.key / cachex.operation 属性），
  命中缓存不产生 span，由 tracex 等外部适配器接入；
- 成功/失败/命中路径埋点测试，覆盖率保持 100%。

## [Unreleased]

### 规划

- 完成调研、PRD、架构、API 草案、迭代计划与决策记录;
- D1–D7 决策点已全部确认并冻结 v0.1.0 API(见 docs/api-design.md)。

## [v0.1.0] - 2026-08-09

### 新增

- 并发分片(默认 16)+ 精确 LRU 淘汰,全局容量严格上限;
- 每条目独立 TTL,惰性过期 + 后台清理(默认 1 分钟)+ Cleanup;
- GetOrSet Singleflight 击穿防护,loader 失败透传、等待者 ctx 取消感知;
- Stats 原子统计 + Metrics 注入 + OnEvicted 逐出回调(锁外调用);
- 配置校验统一 CACHEX_INVALID_CONFIG(errx);
- 零第三方依赖;覆盖率 100%,race / vet / staticcheck / fuzz / vuln 全绿。

## [v0.2.0] - 2026-08-09

### 新增

- WithLRURefresh:LRU 刷新策略可选,1=精确(默认),>1=采样刷新;
- 逐出候选原子化:只读分片尾序号选候选,仅锁目标分片;
- 自研侵入式双向链表,去掉 container/list 接口装箱;
- 热路径显式解锁,Get / Set 省去 defer 开销。

### 性能

- 逐出路径预期大幅下降;Get 采样模式接近纯分片 map;
- 对比基准见 docs/iteration-plan.md(v0.1.0 基线 vs v0.2.0)。

## [v1.0.0] - 2026-08-09

### 正式版

- 公开 API 冻结,遵循语义化版本;
- Version 常量更新为 v1.0.0;
- README 增加稳定性承诺(兼容性策略与发布门禁);
- docs/comparison.md 竞品全维度对比入库;
- 全量回归:100% 覆盖率、race、staticcheck、fuzz、govulncheck、
  apidiff 对比 v0.2.0、三平台 CI。

### 版本历程

- v0.1.0:并发分片 / 精确 LRU / TTL / Singleflight 击穿防护;
- v0.2.0:性能优化(采样 LRU、逐出原子候选、侵入式链表,
  逐出 -71%、分片写 -75%);
- v1.0.0:正式版,API 冻结。
