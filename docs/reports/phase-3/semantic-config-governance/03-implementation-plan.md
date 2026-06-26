# 03. Implementation Plan

## 1. 实施策略

这是一项基础能力重构，必须分批执行。每批完成后都要测试、提交、更新 closeout。

建议分为 6 个批次：

1. Inventory & Design Freeze
2. Semantic Registry & Canonical Key Normalize
3. Config Snapshot Builder
4. Config Projector / Warning Engine / Validator
5. Resolver / RunPlan Adapter Mapping
6. Web Entrypoint Migration & UX Cleanup

## 2. Batch 0：Inventory & Design Freeze

目标：只审查，不改功能代码。

任务：

- 搜索所有参数相关 key、页面、API、resolver。
- 输出重复建模清单。
- 输出 canonical semantic key 表。
- 输出 owner 表。
- 输出 copy-to-next-object 规则。
- 输出 warning/hard-validation 分类。

产物：

```text
docs/reports/phase-3/semantic-config-governance/00-inventory-and-design-review.md
```

验收：

- 所有参数输入入口被列出。
- `backend.common.host` / `launcher.listen_host` 等重复项被明确归类。
- 明确哪些字段要删除、normalize 或保留为诊断。

## 3. Batch 1：Semantic Registry

目标：建立统一参数注册表。

建议模块：

```text
internal/server/semanticconfig/registry.go
internal/server/semanticconfig/types.go
internal/server/semanticconfig/catalog_loader.go
internal/server/semanticconfig/normalize.go
```

核心类型建议：

```go
type SemanticParamDefinition struct {
    Key              string
    Owner            string
    Category         string
    ValueType        string
    LabelZhCN        string
    LabelEnUS        string
    DescriptionZhCN  string
    DescriptionEnUS  string
    DisplayTier      string
    DefaultValue     any
    RecommendedValue any
    HardRules        []ValidationRule
    WarningRules     []WarningRule
    ResolverMappings map[string]ResolverMapping
    CopyPolicy       CopyPolicy
}
```

必须实现：

- canonical key 注册。
- legacy key -> canonical key normalize。
- 冲突值 detection。
- deprecated key 拒绝或诊断。

第一批 canonical keys：

```text
runtime.image_ref
service.listen_host
service.container_port
deployment.host_port
deployment.served_model_name
runtime.command
runtime.entrypoint
runtime.env
runtime.health.path
runtime.health.timeout_seconds
runtime.health.interval_seconds
runtime.model_mount.container_path
docker.shm_size
docker.ipc_mode
docker.privileged
docker.network_mode
docker.security_options
docker.ulimits
docker.devices
docker.optional_devices
docker.group_add
model_runtime.max_model_len
model_runtime.context_length
model_runtime.dtype
model_runtime.quantization
model_runtime.gpu_memory_utilization
model_runtime.max_num_seqs
model_runtime.max_num_batched_tokens
```

需要清理或映射的 legacy keys：

```text
backend.common.host -> service.listen_host
launcher.listen_host -> service.listen_host
backend.common.port -> service.container_port
launcher.container_port -> service.container_port
backend.arg.max_model_len -> model_runtime.max_model_len
backend.arg.context_length -> model_runtime.context_length
backend.arg.gpu_memory_utilization -> model_runtime.gpu_memory_utilization
```

验收：

- normalize 后 ConfigSet 不含第一批 legacy duplicate keys。
- 值冲突产生 diagnostic warning。
- 不做历史兼容迁移；允许 DB rebuild / catalog reload。

## 4. Batch 2：Config Snapshot Builder

目标：建立 copy-on-create 快照生成器。

建议模块：

```text
internal/server/semanticconfig/snapshot.go
internal/server/semanticconfig/copy_policy.go
```

核心结构：

```go
type ConfigSnapshotItem struct {
    Key            string
    Owner          string
    Value          any
    Enabled        bool
    SourceSnapshot SourceSnapshot
    Dirty          bool
    Warnings       []ConfigWarning
}
```

创建逻辑：

- BackendRuntime 从 BackendVersion / catalog default 复制运行环境参数。
- NodeBackendRuntime 从 BackendRuntime 复制运行环境参数，并写入节点镜像选择、设备、env 等。
- Deployment 从 ModelArtifact、NodeBackendRuntime、BackendVersion mapping 复制部署需要的参数。

重要原则：

- 复制后下游可改。
- 上游后续修改不影响已复制对象。
- source_snapshot 只作为来源记录，不作为 live link。

验收：

- NBR 创建后修改 BackendRuntime 不影响 NBR。
- Deployment 创建后修改 NBR 不影响 Deployment。
- Snapshot 中保留 copied_from 信息。

## 5. Batch 3：Projector / Warning / Validation

目标：统一显示字段、分级、warning 和硬校验。

替换/改造：

```text
internal/server/configedit/project.go
internal/server/configedit/taxonomy.go
internal/server/configedit/validate.go
```

建议职责：

- Projector 只从当前对象 snapshot 投影字段。
- 不做旧 alias UI 合并；旧 key 应在 snapshot normalize 前清理。
- 字段携带 owner、tier、source、warnings。
- ConfigEditView 按 tier 分区。

EditField 建议增加：

```go
Owner        string `json:"owner"`
Tier         string `json:"tier"`
OriginalLabel string `json:"original_label,omitempty"`
DisplayLabel  string `json:"display_label,omitempty"`
Warnings     []ConfigWarning `json:"warnings,omitempty"`
Dirty        bool `json:"dirty,omitempty"`
```

硬校验：

- required missing。
- type invalid。
- enum invalid。
- path / port / number format invalid。
- unknown canonical key。
- deprecated legacy key direct patch。

warning：

- 超过推荐范围。
- 可能显存不足。
- 后端不完全支持。
- 修改高级参数可能导致健康检查失败。
- 值来自上游但被修改。

验收：

- 保存合法但 risky 的参数不阻断，返回 warnings。
- 硬错误阻断。
- 参数名前可显示 `!` 或 warning tag。

## 6. Batch 4：Resolver / RunPlan Adapter Mapping

目标：RunPlan 不再直接依赖 legacy technical keys。

建议模块：

```text
internal/server/runplan/**
internal/server/semanticconfig/resolver.go
internal/server/semanticconfig/adapters/vllm.go
internal/server/semanticconfig/adapters/sglang.go
internal/server/semanticconfig/adapters/llamacpp.go
```

实现：

- semantic key -> backend CLI/env/docker mapping。
- vLLM / SGLang / llama.cpp adapter。
- Docker args 从 docker.* semantic keys 生成。
- Health check 从 runtime.health.* + service.container_port 解析。
- Model path 从 ModelLocation + runtime.model_mount 解析。

验收：

- vLLM RunPlan 仍生成正确 `--host` / `--port` / `--max-model-len`。
- SGLang / llama.cpp 使用对应参数。
- RunPlan preview 显示 semantic source 和最终 args。
- 没有从 `backend.common.host` / `launcher.listen_host` 读取主配置。

## 7. Batch 5：Web Entrypoint Migration

目标：所有输入入口复用统一 ConfigEditView 和 semantic metadata。

改造：

- ConfigField 支持中文（英文）label。
- ConfigSection 支持 required/common/recommended/advanced/diagnostic 分区。
- Warning 显示 `!`。
- Advanced 默认折叠。
- Diagnostic 只读。

页面改造：

- BackendRuntimesPage。
- NodeRuntimeConfigWizard。
- RunnerConfigsPage。
- DeploymentWizard。
- DeploymentOverrideEditor。
- BackendsPage。
- Model pages。

验收：

- 页面不再有各自硬编码字段判断。
- 页面只传 object_kind / object_id / context。
- UI 字段由统一 projector 输出。

## 8. Batch 6：Cleanup / Closeout

目标：删除旧逻辑和旧字段。

任务：

- 删除 RuntimeParameterEditor 普通入口。
- 删除或隔离 legacy alias 逻辑。
- 删除 legacy catalog keys。
- 更新文档。
- DB rebuild 说明。

提交：

```bash
git status --short
git add .
git commit -m "runtime: introduce semantic config governance"
git push
```

如果分批提交，commit message 建议：

```text
runtime: audit semantic config ownership
runtime: add semantic config registry
runtime: add config snapshot builder
runtime: project semantic config edit views
runtime: map semantic config into run plans
web: reuse semantic config editing across entrypoints
runtime: clean legacy config aliases
```
