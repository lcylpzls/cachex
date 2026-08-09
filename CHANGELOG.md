# 更新日志

本项目遵循[语义化版本](https://semver.org/lang/zh-CN/)。

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
