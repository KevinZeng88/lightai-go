# Final Runtime Domain Contract

## 1. 目的

本文定义 Runtime 架构与参数体系的最终领域模型。第二轮修订后，领域模型以 ConfigSetBundle 为核心，不再把 ConfigSet 视为旧实现兼容层，也不把所有参数强行平面化。

## 2. 核心领域对象

### Backend / BackendVersion

Backend / BackendVersion 表达推理后端及版本能力，例如 vLLM、SGLang、llama.cpp。它们不表达节点硬件、Docker runtime、具体 GPU vendor、模型本机路径或部署状态。

BackendVersion 可以拥有 ConfigSetBundle，典型 own ConfigSet：

```text
BackendCapabilityConfigSet
BackendParameterConfigSet
BackendEndpointConfigSet
BackendResourceControlConfigSet
```

### ModelArtifact / ModelLocation

ModelArtifact 表达模型本身的稳定信息，例如格式、模型家族、上下文能力、量化、文件类型。ModelLocation 表达某个节点或存储位置上的模型路径、挂载方式、校验信息。

禁止把本机具体路径写入通用 Backend / BackendVersion catalog。

### BackendRuntime

BackendRuntime 表达运行模板。它 copy-on-create BackendVersionConfigBundle effective snapshot，并增加模板自己的 ConfigSet。

典型 own ConfigSet：

```text
RuntimeTemplateConfigSet
RuntimeImageConfigSet
RuntimeCommandConfigSet
RuntimeDockerConfigSet
RuntimeHealthCheckConfigSet
```

### NodeBackendRuntime

NodeBackendRuntime 是唯一部署入口。它 copy-on-create BackendRuntimeConfigBundle effective snapshot，并增加节点运行环境自己的 ConfigSet。

典型 own ConfigSet：

```text
NodeRuntimeEnvironmentConfigSet
NodeDeviceBindingConfigSet
NodeRuntimeCheckEvidenceConfigSet
NodeRuntimeLocalPathConfigSet
```

NodeBackendRuntime 必须显式 enable + check。禁止自动创建 NBR。Deployment 必须引用 NBR，不接受 BackendRuntime 作为部署入口。

### Deployment

Deployment copy-on-create NodeBackendRuntimeConfigBundle effective snapshot，并包含 ModelArtifact / ModelLocation snapshot。Deployment 增加部署级 own ConfigSet 和 local edits。

典型 own ConfigSet：

```text
DeploymentOverrideConfigSet
DeploymentPortConfigSet
DeploymentVolumeConfigSet
DeploymentHealthCheckConfigSet
DeploymentReplicaIntentConfigSet
```

Deployment 不重新定义继承项 schema。Deployment 可以修改继承项的 value/state，并记录 provenance。

### ConfigSetBundle

每个领域层持有一个 ConfigSetBundle。

```text
ConfigSetBundle =
  inherited_bundle_snapshots[]
  own_sets[]
  local_edits[]
  effective_view
```

下一层创建时：

```text
next_layer_bundle = deep_copy(parent.effective_bundle_snapshot) + own_sets + local_edits
```

ConfigSetBundle 是 copy-on-create snapshot，不是 live reference。上层后续修改不污染已创建下层，下层修改也不反向污染上层。

### ConfigSet

ConfigSet 是自解释、自描述、可组合的配置单元。它拥有自己的 ConfigItem，可以包含 child ConfigSet，可以定义 child ConfigSet 的使用方式、展示位置和展示模式，可以生成 summary_view、edit_view、preview_view、effective view，并可以参与最终 RunPlan resolve。

ConfigSet 不是旧 `config_set_json` 混合容器。现有旧结构如果 schema/value/state/provenance 混杂，必须按最终模型清理。

### ConfigItem

ConfigItem 是 ConfigSet 内的最小配置项。字段必须分级：

```text
schema        定义字段，只读
value         值字段，可按层修改
state         checked/enabled/editable/visible/valid
provenance    值来源、最后修改层、source_chain
snapshot      copy-on-create 来源
presentation  展示元信息
```

继承项 copy 到下一层后，schema/snapshot 只读，owner 不变；下一层默认可以修改 value/state。特殊不可编辑通过 `schema.read_only=true` 或 `state.editable=false` 表达，不维护复杂 `overridable_at`。

### ResolvedRunPlan

ResolvedRunPlan 是唯一最终执行权威。它只读取 DeploymentConfigBundle effective snapshot，不读取上游 live 配置。

ResolvedRunPlan 输出：

```text
image
command
args
env
mounts
ports
devices
docker_options
health_check
resource_controls
parameter_source_map
plan_hash
audit_refs
```

preview、preflight、dry-run、start 必须共用同一个 RunPlan builder。

### Instance

Instance 只记录运行事实，例如 container id、actual Docker spec summary、status、health result、logs、errors、operation_id。Instance 不编辑 ConfigSet，不修改 DeploymentConfigBundle。

## 3. 分层关系

```text
BackendVersionConfigBundle
  own_sets:
    - BackendCapabilityConfigSet
    - BackendParameterConfigSet

BackendRuntimeConfigBundle
  inherited_bundle_snapshots:
    - BackendVersionConfigBundle effective snapshot
  own_sets:
    - RuntimeTemplateConfigSet
    - RuntimeDockerConfigSet
    - RuntimeHealthCheckConfigSet

NodeBackendRuntimeConfigBundle
  inherited_bundle_snapshots:
    - BackendRuntimeConfigBundle effective snapshot
  own_sets:
    - NodeRuntimeEnvironmentConfigSet
    - NodeDeviceBindingConfigSet
    - NodeRuntimeCheckEvidenceConfigSet

DeploymentConfigBundle
  inherited_bundle_snapshots:
    - NodeBackendRuntimeConfigBundle effective snapshot
    - ModelArtifactConfigBundle snapshot
    - ModelLocationConfigBundle snapshot
  own_sets:
    - DeploymentOverrideConfigSet
    - DeploymentPortConfigSet
    - DeploymentVolumeConfigSet
    - DeploymentHealthCheckConfigSet

ResolvedRunPlan
  input:
    - DeploymentConfigBundle effective snapshot
```

## 4. 干净实现原则

1. 不做旧 DB / 旧 API / 旧 snapshot 兼容。
2. schema 变化允许 fresh DB rebuild。
3. 旧字段、旧分支、旧 UI 入口、旧 resolver path 如果与最终模型冲突，应删除。
4. ConfigSet 保留为最终领域概念，但旧 `config_set_json` 混合语义不保留。
5. 所有页面、API、RunPlan、测试必须围绕 ConfigSetBundle / ConfigSet / ConfigItem 字段分级建立同一口径。
