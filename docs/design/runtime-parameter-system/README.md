# Runtime Parameter System Design

This directory contains the design, review, and execution plan for the LightAI Go runtime parameter system.

## 文档目录

| 文档 | 说明 |
|------|------|
| `00-review-questions-for-mimo.md` | MiMo 审查问题清单 |
| `01-parameter-layering-design.md` | 参数分层设计 |
| `02-backend-vendor-parameter-catalog.md` | 后端和厂商参数 catalog |
| `03-param-trace-e2e-plan.md` | 参数溯源 E2E 计划 |
| `04-mimo-review-and-comments.md` | MiMo 审查意见和风险评估 |
| `05-development-steps.md` | 分阶段开发步骤（Phase 0-7） |
| `06-acceptance-and-test-plan.md` | 验收和测试计划 |
| `07-open-questions-and-risks.md` | 已决策项和开放问题 |
| `08-execution-governance-and-decisions.md` | **执行治理入口** — 固定原则、自动推进规则、停止条件 |

## 推荐阅读顺序

1. `08-execution-governance-and-decisions.md` — 了解固定原则和执行规则
2. `05-development-steps.md` — 了解分阶段开发计划
3. `06-acceptance-and-test-plan.md` — 了解每个 Phase 的验收标准
4. `07-open-questions-and-risks.md` — 了解已决策项和开放问题
5. `04-mimo-review-and-comments.md` — 了解审查意见
6. `01-parameter-layering-design.md` — 了解参数分层设计
7. `02-backend-vendor-parameter-catalog.md` — 了解参数 catalog
8. `03-param-trace-e2e-plan.md` — 了解 E2E 计划

## 当前状态

- 设计文档已完成审查
- 已决策项已记录在 `07-open-questions-and-risks.md`
- 执行治理规则已记录在 `08-execution-governance-and-decisions.md`
- 准备进入 Phase 0（现状审计）

## 后续执行方式

按照 `08-execution-governance-and-decisions.md` 中的自动推进规则，从 Phase 0 开始逐 Phase 执行。每个 Phase 完成后自动进入下一 Phase，除非触发停止条件。

## Phase 摘要

| Phase | 目标 | 一句话 |
|-------|------|--------|
| 0 | 现状审计 | 跑三后端 E2E，保存基线 evidence |
| 1 | 参数语义正确性 | required locked, optional enabled/value, Layer 3 模板替换 |
| 2 | UI 分组和唯一入口 | 参数分组、唯一编辑入口、Deployment override 可输入 |
| 3 | 冲突检测 | extra_args 冲突、required 缺失、vendor 不匹配 |
| 4 | vendor 隔离 | NVIDIA/MetaX/Huawei 参数隔离 |
| 5 | 参数溯源 E2E | 完整链路验证 |
| 6 | 矩阵扩展 | 三后端/三厂商参数覆盖 |
| 7 | 外置 help | help 文档和 UI ? 弹窗 |
