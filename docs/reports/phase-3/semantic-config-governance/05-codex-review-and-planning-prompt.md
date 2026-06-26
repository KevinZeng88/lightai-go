# 05. Codex Review and Planning Prompt

```text
REVIEW_AND_PLAN_ONLY

仓库：

/home/kzeng/projects/ai-platform-study/lightai-go

当前分支继续，不新建分支。

请先阅读：

- docs/reports/phase-3/semantic-config-governance/00-index.md
- docs/reports/phase-3/semantic-config-governance/01-semantic-config-design.md
- docs/reports/phase-3/semantic-config-governance/02-cross-entrypoint-audit-scope.md
- docs/reports/phase-3/semantic-config-governance/03-implementation-plan.md
- docs/reports/phase-3/semantic-config-governance/04-validation-and-test-plan.md

任务：

只做审查和计划，不修改功能代码。

背景：

当前 LightAI Go 的 Runtime Template / ConfigEditView 已多轮修复，但仍暴露出根本问题：同一业务语义被多个字段重复建模，各页面各自判断参数显示和保存，导致运行模板、节点运行配置、部署参数、RunPlan 之间边界不清。

用户已明确要求：

1. 这不是单点问题，是基础能力问题。
2. 不允许每个页面单独写一套参数判断。
3. 必须建立通用程序，而不是在 BackendRuntimesPage / RunnerConfigsPage / DeploymentWizard 分别修。
4. 同一个业务语义只能有一个 semantic key 和一个 owner。
5. 下游对象 copy-on-create 后持有自己的参数快照；下游可改，和上游解除 live 关系。
6. 不使用 override 作为主要概念，使用 snapshot / copied_from / dirty / warnings 表达。
7. 参数限制尽量 warning，不要强阻断；只有类型/必填/格式/不可解析等硬错误才阻断。
8. Backend CLI flag 不应直接作为用户配置 key；应由 resolver/adapter 从 semantic key 生成。
9. 不需要历史兼容；必要时允许 DB rebuild / catalog reload。

请完成以下审查：

A. 全入口审查

找出所有参数输入、参数显示、参数保存、RunPlan 解析入口，包括但不限于：

- BackendRuntimesPage
- NodeRuntimeConfigWizard
- RunnerConfigsPage
- DeploymentWizard
- DeploymentOverrideEditor
- DeploymentServiceEditor
- BackendsPage
- Model pages
- ConfigEditView / ConfigSection / ConfigField
- RuntimeParameterEditor
- JsonViewer config_set/source_metadata 展示
- internal/server/configedit
- internal/server/runplan
- internal/server/api runtime/deployment/model handlers
- catalog yaml / seed code

B. 关键字搜索

必须搜索并汇总：

backend.common.host
backend.common.port
launcher.listen_host
launcher.container_port
service.listen_host
service.container_port
host_port
container_port
backend.arg.
max_model_len
max-model-len
context_length
served_model_name
gpu_memory_utilization
runtime.env
launcher.env
docker_options
devices
optional_devices
group_add
health
model_mount
RuntimeParameterEditor
ConfigEditView
config_set_json
source_metadata_json

C. 输出重复建模清单

至少覆盖：

- host/listen_host/container_port/host_port
- model_runtime 参数
- health check
- model mount
- runtime env
- docker options / device binding
- backend capabilities / runtime requirements

D. 输出 canonical semantic key 表

每个参数列出：

- semantic key
- owner
- value type
- copied_to 哪些对象
- default/recommended 来源
- hard validation
- warning rules
- resolver mapping
- display tier
- legacy keys to remove/normalize

E. 输出程序架构计划

必须包含这些通用模块，而不是页面私有逻辑：

- SemanticConfigRegistry
- SemanticConfigNormalizer
- ConfigSnapshotBuilder
- ConfigProjector
- ConfigWarningEngine
- ConfigValidator
- RunPlanResolver / BackendAdapterMapping
- ConfigEditView renderer changes

F. 输出分批实施计划

按 batch 输出，每个 batch 包括：

- 目标
- 修改文件
- 具体步骤
- 测试
- 验收标准
- 风险

G. 输出 open questions

只列真正需要用户确认的设计问题，不要列执行层面的琐碎问题。

输出文件：

请写入：

docs/reports/phase-3/semantic-config-governance/10-codex-review-and-execution-plan.md

最终输出：

- REVIEW_PLAN_READY / FAIL
- 发现的入口数量
- 发现的重复 semantic groups
- 建议新增模块
- 建议 batch 列表
- open questions
- git status
```
