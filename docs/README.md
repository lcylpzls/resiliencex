# resiliencex 文档

## 阅读顺序

1. [PRD.md](PRD.md) — 要什么、不要什么;
2. [architecture.md](architecture.md) — 限流 / 熔断 / 舱壁分层;
3. [api-design.md](api-design.md) — API 草案与迭代范围;
4. [decisions.md](decisions.md) — 架构决策记录(ADR);
5. [iteration-plan.md](iteration-plan.md) — 迭代顺序与质量门槛。

设计输入:[resilience-research.md](resilience-research.md) — 治理领域调研手册
(x/time/rate、gobreaker、sentinel、Resilience4j、Hystrix 等)。

运行与治理:

6. [operations.md](operations.md) — 组件速查与参数表;
7. [performance.md](performance.md) — 性能基准与竞品对比;
8. [comparison.md](comparison.md) — 与治理库全维度对比;
9. [quality.md](quality.md) — 质量门槛与测试策略;
10. [release.md](release.md) — 版本与发布流程。

## 决策状态

ADR 按全自动迭代授权执行,不再逐项确认;API 随版本冻结演进。
