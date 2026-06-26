# Risk to Workstream Map

本文件把审计风险和 open questions 映射为执行批次，确保所有问题都有闭环路径。

## R 风险映射

| Risk | Severity | 修复批次 | 关闭目标 |
| --- | --- | --- | --- |
| R-001 client-trusted NBR `/check` | P0 | Batch 1 | Session 用户无法通过请求体置 ready；只有 Agent/server probe evidence 可改变 readiness |
| R-002 stale E2E scripts | P1 | Batch 3 | active scripts 全部使用 `node_backend_runtime_id`、`/check-request`、`parameter_values_json` |
| R-003 preflight 与 final RunPlan 不一致 | P1 | Batch 2 | preflight/dry-run/start 共享 deployability/helper/resolver/error schema |
| R-004 部署编辑 UI runtime selector 误导 | P1 | Batch 4 | 移除误导字段或实现显式 NBR change flow |
| R-005 snapshot migration/legacy mutation | P1 | Batch 1 + Batch 2 | 清理 legacy branch；snapshot 只由显式用户动作或命名 migration 改变 |
| R-006 stale OpenAPI | P1 | Batch 3 | OpenAPI 覆盖当前 routes/payload/errors/examples |
| R-007 global Agent bearer token | P1 | Batch 5 | token 与 node/agent 绑定；跨 node 复用失败 |
| R-008 Docker runtime option governance | P1 | Batch 5B | 厂商模板必需 Docker 参数被保留；自定义配置有基本校验、提示、审计和一致 RunPlan 行为 |
| R-009 `tenant_id DEFAULT 'default'` | P2 | Batch 5 | fresh DB schema 不再出现非法 default tenant |
| R-010 weak auth/authz/db/rbac coverage | P2 | Batch 5 | 负向权限矩阵和关键 schema/migration 测试覆盖 |
| R-011 frontend per-node NBR fan-out | P2 | Batch 4 + Batch 7 | 聚合 NBR endpoint；页面不再 N+1 请求 |
| R-012 replica fields but single-instance path | P2 | Batch 8 | 未实现前 API/UI 明确拒绝或隐藏 replicas > 1 |
| R-013 Prom/Grafana supervision not in Go | P2 | Batch 8 | 文档与实现一致；不夸大 server-managed observability |
| R-014 OpenAI gateway/API key/usage billing missing | P2 | Batch 8 | 当前 maturity claim 收敛；形成 gateway/usage 设计或明确未支持 |
| R-015 frontend main chunk warning | P3 | Batch 7 | code split 或记录可接受阈值并建立监控 |

## Q open questions 映射

| Question | 决策方式 | 批次 | 默认决策 |
| --- | --- | --- | --- |
| Q-001 `/check` 是否保留 | 干净设计优先 | Batch 1 | 若保留 route name，只能作为 server-to-Agent probe wrapper；handler 忽略 request body readiness evidence；也可删除 route 并同步 UI/OpenAPI/scripts/tests/docs |
| Q-002 preflight 是否成为 full RunPlan preflight | 契约一致优先 | Batch 2 | preflight 走 final resolver；candidate check 另命名 |
| Q-003 supported parameter payload | 当前 contract 优先 | Batch 1/3 | 只支持 `parameter_values_json`；`parameters_json` 400 |
| Q-004 NBR reapply/change 是否当前实现 | 避免 UI 误导 | Batch 4 | 若不实现，隐藏入口；若实现，必须有 diff/check/snapshot warning |
| Q-005 Docker runtime option governance | AIDC 可用性优先 | Batch 5B | 模板驱动 + 显式配置 + 基本校验 + 审计记录；不一刀切默认拒绝厂商模板所需 devices/volumes/env/runtime options |
| Q-006 API contract source | 当前 router/tests 优先 | Batch 3 | OpenAPI + contract tests 双轨维护 |
| Q-007 current E2E scripts | 库存审查 | Batch 3 | stale scripts 归档，active scripts 必须可运行 |
| Q-008 MetaX readiness bar | 证据优先 | Batch 8/15 | 没有真实硬件 evidence 前只能标记 `BLOCKED_BY_EXTERNAL_DEPENDENCY`，不得宣称生产 ready |

## 执行原则

本项目当前不需要兼容旧 DB、旧 API、旧 payload、旧脚本、旧运行模板、旧快照。Claude 后续执行可以修改设计和程序后只保留最新主线，不需要为旧 `backend_runtime_id`、旧 `parameters_json` 或旧 `/runtime-environments`、`/run-templates`、`/model-deployments` 合约保留兼容路径。

R-001 到 R-015 最终 closeout 只允许以下状态：

- CLOSED
- CLOSED_BY_SCOPE_REDUCTION
- BLOCKED_BY_EXTERNAL_DEPENDENCY

状态定义：

- `CLOSED`：代码/测试/文档/脚本全部闭环，验证通过。
- `CLOSED_BY_SCOPE_REDUCTION`：该能力不做，但 UI/API/docs 不再声称支持，相关入口已隐藏、禁用或拒绝。
- `BLOCKED_BY_EXTERNAL_DEPENDENCY`：只有外部资源确实不可用，且有命令级证据。

禁止使用模糊状态或词语：`INTENTIONALLY_DEFERRED_WITH_OWNER_AND_ACCEPTANCE`、`future`、`follow-up`、`later`、`manual verification later`、`todo only`。

如果某设计 item 不实现，只能通过 scope reduction 关闭，不能模糊延期。Closeout 中不能把“保留兼容路径”当成修复完成。
