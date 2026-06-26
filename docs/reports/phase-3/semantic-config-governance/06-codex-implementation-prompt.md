# 06. Codex Implementation Prompt

```text
AUTORUN_AFTER_PLAN_APPROVAL

仓库：

/home/kzeng/projects/ai-platform-study/lightai-go

当前分支继续，不新建分支。

前置条件：

已经生成并确认：

docs/reports/phase-3/semantic-config-governance/10-codex-review-and-execution-plan.md

请严格按该计划分批执行。每批必须测试、提交、更新 closeout。不要跳过测试，不要把可修复问题留作 future。

总目标：

建立 LightAI Go 的 Semantic Config 基础能力：

- 单一 semantic key / 单一 owner。
- copy-on-create config snapshot。
- warning 优先，硬错误才阻断。
- 通用 ConfigProjector / Validator / Resolver。
- 所有参数编辑入口复用统一程序。
- 删除或 normalize 重复建模字段。

核心约束：

1. 不允许在页面里各写一套字段判断。
2. 不允许继续把 backend.common.host / launcher.listen_host / service.listen_host 作为三个长期配置字段。
3. 不允许长期使用 backend.arg.* 作为普通用户配置 key。
4. Backend CLI flags 只能作为 resolver mapping。
5. 下游对象复制参数快照后可改，但不与上游 live 绑定。
6. 风险用 warnings / ! 标识；硬错误才阻断保存。
7. 不做历史兼容迁移；必要时说明 DB rebuild / catalog reload。

执行批次建议：

Batch 1: Semantic registry and canonical normalize
- 新增 semanticconfig 模块。
- 定义 canonical keys。
- normalize legacy keys。
- 去除 host/port 第一批重复建模。

Batch 2: Config snapshot builder
- 实现 copy-on-create 快照。
- 记录 source_snapshot / dirty / warnings。
- 保证上游修改不影响下游。

Batch 3: Projector / Warning / Validator
- ConfigEditView 使用 semantic metadata。
- 增加 tier/owner/original_label/warnings。
- 普通/高级/诊断分区。
- warning 不阻断，hard validation 阻断。

Batch 4: Resolver / RunPlan adapter mapping
- 从 semantic keys 生成 CLI/env/docker args。
- vLLM/SGLang/llama.cpp 至少覆盖当前内置后端。
- RunPlan preview 显示最终参数和 warnings。

Batch 5: Web entrypoint migration
- BackendRuntimesPage
- NodeRuntimeConfigWizard
- RunnerConfigsPage
- DeploymentWizard
- DeploymentOverrideEditor
- BackendsPage
- Model pages
全部复用统一 ConfigEditView / semantic projector。

Batch 6: Cleanup and closeout
- 删除普通入口 RuntimeParameterEditor。
- 删除 legacy duplicate keys。
- 更新 docs。
- 最终测试、commit、push。

每批必须运行相关测试；最终必须运行：

go build ./cmd/server/...
go build ./cmd/agent/...
go test ./internal/server/...
go test ./internal/agent/...
cd web && npm run build
cd web && npm test

必须新增测试：

1. normalize 后不存在 backend.common.host / launcher.listen_host。
2. 只存在 service.listen_host。
3. normalize 后不存在 backend.common.port / launcher.container_port。
4. 只存在 service.container_port。
5. patch legacy key 被拒绝或 normalize。
6. ConfigEditView 普通区不显示 legacy key。
7. BackendRuntime 不显示 model_runtime.max_model_len。
8. NBR 不显示 model_runtime.max_model_len。
9. Deployment 显示 model_runtime.max_model_len，但默认高级。
10. risk value 生成 warning，不阻断保存。
11. hard invalid value 阻断保存。
12. RunPlan 由 semantic key 生成 vLLM --host / --port / --max-model-len。
13. health check 默认端口引用 service.container_port。
14. 下游 copy-on-create 后上游修改不影响下游。
15. UI label 为中文（English）。

文档更新：

更新或新增：

docs/reports/phase-3/semantic-config-governance/20-implementation-closeout.md

docs/reports/phase-3/runtime-template-catalog-redesign/final-closeout.md

必须说明：

- 新增模块
- canonical key 表
- removed/normalized legacy keys
- copy-on-create snapshot 行为
- warning/hard validation 行为
- 页面入口迁移结果
- RunPlan resolver 映射结果
- 测试结果
- commit id / push result / git status

提交要求：

每批可单独 commit。最终 git status 必须 clean。

最终输出：

- PASS / FAIL
- commits
- push result
- test summary
- closeout paths
- remaining blocked items
- git status
```
