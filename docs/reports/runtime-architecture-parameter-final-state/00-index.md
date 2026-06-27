# Runtime Architecture & Parameter Final-State — 文档索引

## 目录标准

本专题所有新增文档、审查结果、执行证据、closeout 统一放在：

```text
docs/reports/runtime-architecture-parameter-final-state/
```

本目录采用专题生命周期命名，不再绑定历史阶段编号。历史报告可以作为输入材料读取；本专题新增输出、执行证据、Codex review、Claude closeout 全部进入本目录。

## 当前目录结构

```text
docs/reports/runtime-architecture-parameter-final-state/
├── 00-index.md
├── 01-execution-policy-and-scope.md
├── 02-current-context-and-known-issues.md
├── 03-final-runtime-domain-contract.md
├── 04-final-parameter-contract.md
├── 04a-parameter-ownership-and-layered-presentation-contract.md
├── 05-runtime-requirements-and-capability-profile-design.md
├── 05a-configset-bundle-composition-and-presentation-contract.md
├── 06-runplan-and-preflight-contract.md
├── 07-ui-and-api-contract.md
├── 08-api-first-e2e-and-automation-requirements.md
├── 09-implementation-plan.md
├── 10-claude-execution-prompt.md
├── 11-final-closeout-template.md
├── 12-codex-review-instructions.md
├── 13-codex-review.md
├── 14-codex-review-fix-plan.md
├── 15-codex-second-review-instructions.md
├── 16-codex-second-review.md
├── 17-codex-final-implementation-audit.md
├── evidence/
├── templates/
└── manifest.json
```

`13-codex-review.md` 为 Codex 第一轮审核结果。`14-codex-review-fix-plan.md` 是用户与 ChatGPT 对 review 的正式设计决策。`15-codex-second-review-instructions.md` 用于第二轮 Codex 复审。`16-codex-second-review.md` 为第二轮复审结果。
`17-codex-final-implementation-audit.md` 为当前 main HEAD 的最终实现审计结果。

## 第二轮修订定位

第一轮 Codex review 给出 `ACCEPT_WITH_FIXES`。第二轮讨论后，形成以下正式决策：

1. 不做旧代码兼容，不接受修修补补式过渡模型。
2. ConfigSet 不是 seed-only，也不是旧 `config_set_json` 混合容器；ConfigSet 是最终领域概念。
3. 每一层持有 ConfigSetBundle，而不是单个 ConfigSet。
4. ConfigSetBundle 由上一层 copy-on-create 的 effective bundle snapshot、本层 own ConfigSet、本层 local edits 和 effective view 组成。
5. ConfigSet 可以包含 child ConfigSet；父 ConfigSet 负责编排 child ConfigSet 的使用位置、展示模式和拼接顺序。
6. 每个 ConfigSet 是自解释、自描述、可组合的配置单元，可以输出自己的 summary/edit/preview view。
7. 外部页面展示的是 Config / ConfigView / ConfigPanel，不直接暴露内部 ConfigSet 原始结构。
8. 每个 ConfigItem 分为 schema/value/state/provenance/snapshot/presentation 六类字段。
9. copy-on-create 可以完整复制上一层 schema 快照，但继承项 schema/snapshot 只读，owner 不变。
10. 下一层默认可以修改继承项的 value/state；不再维护复杂的 `overridable_at`。
11. 特殊只读由 `schema.read_only` 或 `state.editable=false` 表达。
12. checked/enabled 表示当前层显式修改或启用，不表示 required/default/inherited。
13. RunPlan 只从 DeploymentConfigBundle effective snapshot 生成最终执行 spec。
14. preview/preflight/dry-run/start 必须共用同一个 RunPlan builder。
15. parameter_source_map 必须覆盖 args/env/mounts/ports/devices/docker_options/health_check，并保留 source_chain。

## 阅读顺序

Codex 第二轮审核和 Claude 执行前按以下顺序阅读：

1. `01-execution-policy-and-scope.md`
2. `02-current-context-and-known-issues.md`
3. `13-codex-review.md`
4. `14-codex-review-fix-plan.md`
5. `03-final-runtime-domain-contract.md`
6. `04-final-parameter-contract.md`
7. `04a-parameter-ownership-and-layered-presentation-contract.md`
8. `05-runtime-requirements-and-capability-profile-design.md`
9. `05a-configset-bundle-composition-and-presentation-contract.md`
10. `06-runplan-and-preflight-contract.md`
11. `07-ui-and-api-contract.md`
12. `08-api-first-e2e-and-automation-requirements.md`
13. `09-implementation-plan.md`
14. `10-claude-execution-prompt.md`
15. `11-final-closeout-template.md`
16. `15-codex-second-review-instructions.md`
17. `16-codex-second-review.md`

## 阶段主目标

完成 LightAI Go Runtime 架构、模型元数据、运行能力定义、RuntimeRequirements、BackendCapabilityProfile、ConfigSetBundle 参数体系、RunPlan、Preflight、UI/API 行为的最终收敛。

自动化运行是验收要求：用户预设模型、运行配置、节点运行配置、部署参数后，系统通过 API 和状态机自动完成检查、预检、RunPlan 生成、启动、健康检查、日志采集、状态判断和失败归因。

## 输出原则

1. 新增文档进入本专题目录。
2. 证据进入 `evidence/`。
3. Codex 第二轮 review 输出建议为 `16-codex-second-review.md`。
4. Claude final closeout 按 `11-final-closeout-template.md` 生成。
5. 不新建分支。
6. 不保留历史兼容逻辑。
7. 数据库 schema 变化允许重建数据库。
8. 所有可定位、可修复、可验证的问题在本阶段处理。
9. 无法验证的问题进入 closeout open issues，并说明验证条件。
