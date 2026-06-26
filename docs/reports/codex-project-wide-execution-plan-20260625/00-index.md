# Codex Project-Wide Execution Plan 2026-06-25

本目录是基于 `docs/reports/codex-project-wide-review-20260625` 审计结果生成的执行计划文档群。

目标：把审计中所有 R-001 到 R-015、Q-001 到 Q-008，以及相关设计/测试/文档差距，转化为可以由 Claude 后续无人值守 AUTORUN 执行的开发计划、批次任务、验收标准和 closeout 要求。

## 执行者口径

- Codex：只负责计划审核、计划修订、轻量复审。
- Claude：负责后续无人值守 AUTORUN 执行。
- 当前权威执行入口：
  ```bash
  docs/reports/codex-project-wide-execution-plan-20260625/13-autonomous-claude-execution-prompt.md
  ```

## 目标仓库

```bash
/home/kzeng/projects/ai-platform-study/lightai-go
```

GitHub 仓库：

```text
https://github.com/KevinZeng88/lightai-go
```

本机 push 后会同步到该仓库。

## 计划文档输出目录

执行端应在仓库内创建并维护：

```bash
docs/reports/codex-project-wide-execution-plan-20260625
```

不要使用 `docs/reports/phase-3`。  
不要把本次执行计划、证据、closeout 散落到旧目录。

## 阅读顺序

| 文档 | 用途 |
| --- | --- |
| `00-index.md` | 本入口文件 |
| `01-execution-policy-and-scope.md` | 自主执行原则、边界、提交/push 策略 |
| `02-risk-to-workstream-map.md` | R-001 到 R-015、Q-001 到 Q-008 与执行批次映射 |
| `03-batch-0-baseline-and-inventory.md` | 基线、脚本/接口/测试库存、避免旧证据误导 |
| `04-batch-1-contract-readiness-hardening.md` | NBR readiness、client-trusted check、legacy payload |
| `05-batch-2-runplan-preflight-convergence.md` | preflight/dry-run/start 统一到最终 RunPlan 边界 |
| `06-batch-3-e2e-openapi-documentation-convergence.md` | E2E、OpenAPI、当前 API contract、历史证据归档 |
| `07-batch-4-ui-workflow-repair.md` | 部署/运行配置 UI 误导入口、聚合 NBR、浏览器 smoke |
| `08-batch-5-security-tenant-hardening.md` | Agent token、Docker policy、tenant/RBAC 负向矩阵 |
| `09-batch-6-reliability-observability-scale.md` | GPU lease、task timeout、node offline、logs、metrics |
| `10-batch-7-performance-scalability-cleanup.md` | 分页/索引、N+1、日志限制、前端 chunk |
| `11-batch-8-product-scope-and-gateway-boundaries.md` | 多副本/OpenAI gateway/usage/observability claim 收敛 |
| `12-validation-matrix.md` | 每批必须执行的验证命令与证据标准 |
| `13-autonomous-claude-execution-prompt.md` | 可直接交给 Claude 的完整 AUTORUN 指令，当前权威执行提示 |
| `13-autonomous-codex-execution-prompt.md` | 兼容旧索引的指针文件；Codex 不作为后续 AUTORUN 执行主体 |
| `14-closeout-template.md` | 每批和最终 closeout 模板 |
| `15-runtime-smoke-plan.md` | 本机 NVIDIA/Docker/模型真实 smoke 计划 |
| `16-commit-and-push-strategy.md` | commit 粒度、push、失败处理、git status 要求 |

## 总体优先级

1. 先关掉会造成错误 readiness、错误部署、错误验收的链路。
2. 再统一 RunPlan/API/E2E/OpenAPI 契约。
3. 然后修 UI、安全、tenant、可靠性、性能、产品边界。
4. 每批必须有测试、文档、证据和提交。
5. 所有发现的问题都必须进入计划，不允许只处理 P0/P1。
