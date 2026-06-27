# ConfigSet Bundle Composition and Presentation Contract

## 1. 目的

本文是第二轮设计新增文档，专门定义 ConfigSet / ConfigSetBundle 的组合关系与展示契约。

核心决策：ConfigSet 是内部组合与定义单元；外部页面展示的是 Config / ConfigView / ConfigPanel；每个 ConfigSet 自解释、自描述、可组合；父 ConfigSet 负责编排 child ConfigSet 的使用方式；子 ConfigSet 负责解释和展示自己的内部配置。

## 2. 领域概念

### ConfigSetBundle

每个领域层持有 ConfigSetBundle：

```text
ConfigSetBundle
  inherited_bundle_snapshots[]
  own_sets[]
  local_edits[]
  effective_view
```

### ConfigSet

ConfigSet 是自解释、可组合对象。它可以拥有自己的 ConfigItem，包含 child ConfigSet，定义 child ConfigSet 的展示 slot，输出 summary_view、edit_view、preview_view、effective_view 和 RunPlan mapping hints。

### ConfigView / ConfigPanel

外部页面不直接展示内部 ConfigSet 原始结构，而是展示 ConfigSet 生成的 ConfigView / ConfigPanel。

```text
ConfigView
  title
  subtitle
  summary
  sections[]
  child_panels[]
  local_edits_summary
  effective_preview
```

## 3. 组合模型

例如 ConfigSet A 由自己参数 + ConfigSet B + ConfigSet C 组成：

```text
ConfigSetA
  own_items
  child_sets:
    - ConfigSetB
    - ConfigSetC
```

展示时：

```text
ConfigSetA.render()
  render own sections
  render child slot B by calling ConfigSetB.render()
  render child slot C by calling ConfigSetC.render()
```

父 ConfigSet 不平铺、重画、改写 child ConfigSet 的内部 ConfigItem。父 ConfigSet 只决定 child 放在哪里、用 summary/edit/preview 哪种 view、默认展开还是折叠、外部标题/说明/排序、是否按当前页面上下文隐藏。

## 4. presentation contract

每个 ConfigSet 必须提供统一展示契约：

```json
{
  "config_set_key": "deployment",
  "title": "部署配置",
  "summary_view": {},
  "edit_view": {},
  "preview_view": {},
  "own_sections": [],
  "child_slots": []
}
```

### own_sections

own_sections 定义 ConfigSet 自己的参数如何分组。推荐基础分组：required、common、advanced、readonly、local_edits、diagnostics。

```json
{
  "own_sections": [
    {"key": "required", "title": "必填配置", "match": {"required": true}, "default_expanded": true, "priority": 10},
    {"key": "common", "title": "常用配置", "match": {"group": "common"}, "default_expanded": true, "priority": 20},
    {"key": "advanced", "title": "高级配置", "match": {"advanced": true}, "default_expanded": false, "priority": 90}
  ]
}
```

### child_slots

child_slots 定义 child ConfigSet 如何插入父 ConfigSet view。

```json
{
  "child_slots": [
    {"slot": "runtime", "child_config_set_key": "node_backend_runtime", "title": "继承的节点运行配置", "view": "summary_then_edit", "display_mode": "panel", "default_expanded": true, "order": 30},
    {"slot": "model", "child_config_set_key": "model_location", "title": "模型位置", "view": "summary", "display_mode": "card", "default_expanded": true, "order": 40}
  ]
}
```

## 5. 自解释展示规则

每个 ConfigSet 应能自己决定哪些字段必填、常用、高级、只读、本层 local edit、摘要展示、默认折叠、需要 custom renderer、需要展示哪些 child ConfigSet、如何参与 RunPlan preview。父 ConfigSet 不应理解 child 内部参数含义。

## 6. renderer 规则

默认使用 GenericConfigSetRenderer：render ConfigSet.summary_view、own_sections、child_slots、local_edits_summary、preview_view。

少数复杂 ConfigSet 可以注册 CustomRendererRegistry，例如 device_binding、health_check、mounts、ports、docker_options。custom renderer 必须消费同一份 ConfigView schema，不能绕过 ConfigItem value/state/provenance 规则，不能重新定义 schema，输出必须可被测试断言。

## 7. 页面展示规则

Model 页面展示 ModelArtifactConfigSet / ModelLocationConfigSet，不展示 Docker/runtime/GPU/deployment override 参数。

Backend / BackendVersion 页面展示 BackendCapabilityConfigSet / BackendParameterConfigSet / BackendEndpointConfigSet，不展示节点检测结果、本机模型路径、部署覆盖。

BackendRuntime 页面展示 BackendRuntimeConfigBundle view，包括 inherited BackendVersion summary、RuntimeTemplateConfigSet、RuntimeDockerConfigSet、RuntimeHealthCheckConfigSet、local edits。

NodeBackendRuntime 页面展示 inherited BackendRuntime view、NodeRuntimeEnvironmentConfigSet、NodeDeviceBindingConfigSet、NodeRuntimeCheckEvidenceConfigSet、local edits。

Deployment 页面展示 deployment required/common/advanced、ModelArtifact / ModelLocation child panels、NodeBackendRuntime child panel、DeploymentPortConfigSet、DeploymentVolumeConfigSet、DeploymentHealthCheckConfigSet、local edits summary、ResolvedRunPlan preview。Deployment 页面不应把所有参数无来源平铺成一个大表。

Instance 页面展示 ResolvedRunPlan summary、actual Docker spec summary、status、health result、logs、errors。Instance 不展示 editable ConfigSet。

## 8. 展示与运行解耦

UI 展示可以是树状、面板、卡片、折叠区，但 RunPlan 不依赖 UI 展示结构。RunPlan 只读取 DeploymentConfigBundle effective snapshot。

## 9. 验收要求

必须测试：ConfigSet 可以输出 own_sections；ConfigSet 可以输出 child_slots；父 ConfigSet 调用 child ConfigSet view，而不是平铺 child ConfigItem；required/common 显眼展示；advanced 默认折叠；local edits 单独可见；inherited value 与 current local edit 可区分；custom renderer 不绕过 ConfigItem 规则；Deployment 页面有 RunPlan preview；Instance 页面不能编辑 ConfigSet。
