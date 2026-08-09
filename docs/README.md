# cachex 文档

## 阅读顺序

1. [PRD.md](PRD.md) — 要什么、不要什么;
2. [architecture.md](architecture.md) — 分片 / LRU / 过期 / 击穿防护怎么切;
3. [api-design.md](api-design.md) — API 草案与待决策点;
4. [decisions.md](decisions.md) — 架构决策记录(ADR);
5. [iteration-plan.md](iteration-plan.md) — 迭代顺序与质量门槛。

设计输入:[cache-research.md](cache-research.md) — 缓存领域调研手册
(groupcache、bigcache、freecache、ristretto、go-cache、Redis 等)。

## 决策状态

D1–D7 已全部确认(全部采纳推荐项),v0.1.0 API 已冻结,进入实现阶段。
