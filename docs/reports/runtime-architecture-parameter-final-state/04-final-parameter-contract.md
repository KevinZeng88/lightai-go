# Final Parameter Contract

## 1. 目的

本文定义 ConfigSetBundle 模型下的参数契约。第二轮修订后，主线改为：

```text
ConfigSetBundle + ConfigSet + ConfigItem(schema/value/state/provenance/snapshot/presentation)
```

ParameterDefinition / ParameterValue / ParameterSourceMap 可以作为代码命名或内部类型存在，但领域语义必须服从 ConfigItem 字段分级。

## 2. ConfigItem 字段分级

### schema 字段

schema 字段定义参数“是什么”。copy-on-create 后可以被复制到下一层快照，但继承项 schema 只读，owner 不变。

典型字段：

```text
key
owner
owner_layer
config_set_key
category
label
description
type
target
arg_name
env_name
mount_target
port_target
constraints
choices
required
advanced
display_order
read_only
help_text
```

规则：schema 字段可以完整复制到下一层 snapshot；下一层不能修改继承项 schema；新增参数必须属于当前层 own ConfigSet；不再维护复杂 `overridable_at`，是否可编辑由 `schema.read_only` 和 `state.editable` 表达。

### value 字段

value 字段定义参数“当前值是多少”。

```json
{
  "default_value": 0.9,
  "inherited_value": 0.8,
  "local_value": 0.82,
  "effective_value": 0.82
}
```

规则：下一层默认可以修改 value.local_value；effective_value = local_value if set else inherited_value/default_value/system value；default_value 不等于 enabled；required 不等于 checked；inherited_value 不等于 checked；optional 默认不 checked。

### state 字段

state 字段定义参数在当前层的 UI 与生效状态。

```json
{
  "enabled": true,
  "checked": true,
  "editable": true,
  "visible": true,
  "valid": true,
  "validation_error": ""
}
```

规则：checked/enabled 表示当前层显式启用或覆盖该值；不表示默认值、required 或 inherited；disabled input 仍显示 effective_value；advanced 默认折叠；特殊只读项使用 `schema.read_only=true` 或 `state.editable=false`。

### provenance 字段

provenance 字段定义“值从哪里来、最后在哪一层改”。

```json
{
  "value_source": "deployment_local_edit",
  "last_value_layer": "DeploymentConfigBundle",
  "last_value_owner_id": "dep-123",
  "source_chain": [
    {"layer": "BackendVersionConfigBundle", "value": 0.9, "reason": "schema default"},
    {"layer": "NodeBackendRuntimeConfigBundle", "value": 0.8, "reason": "node local edit"},
    {"layer": "DeploymentConfigBundle", "value": 0.82, "reason": "deployment local edit"}
  ]
}
```

每次当前层修改 value/state，应更新 provenance。source_chain 用于 RunPlan preview、排错和审计，不能只是 UI 装饰。

### snapshot 字段

snapshot 字段定义这份 ConfigItem 从哪里 copy 来。

```json
{
  "snapshot_from_layer": "NodeBackendRuntimeConfigBundle",
  "snapshot_from_id": "nbr-456",
  "snapshot_version": 3,
  "snapshot_at": "2026-06-27T22:00:00Z"
}
```

copy-on-create 必须记录 snapshot 来源。snapshot 字段只读。后续上层变化不修改已创建下层 snapshot。

### presentation 字段

presentation 字段定义单个 ConfigItem 如何展示。

```text
section
group
priority
display_mode
placeholder
summary_priority
hide_when_empty
default_expanded
sensitive
```

展示字段不参与最终 RunPlan 语义。

## 3. ConfigSet 参数职责

ConfigSet 可以定义自己的 ConfigItem，包含 child ConfigSet，生成自己的 effective view，生成 summary/edit/preview view，定义 child ConfigSet 展示 slot，输出参与 RunPlan 的字段。

ConfigSet 不可以修改 child 继承项 schema、改变 child ConfigItem owner、重新定义 child ConfigItem、把旧混合结构作为兼容层保留、把 UI checked 直接等同于 default/required。

## 4. Docker options 作为 ConfigItem

Docker 子字段必须纳入统一 ConfigItem 模型，不作为特殊旧字段绕过参数体系。

```text
docker.shm_size
docker.group_add
docker.devices
docker.extra_hosts
docker.ipc_mode
```

规则：每个 Docker 子字段是 ConfigItem，或属于 DockerOptionsConfigSet 中的结构化 ConfigItem；unchecked optional Docker item 不进入 final Docker spec；有 value 但 state.enabled=false 的 optional Docker item 不进入 final Docker spec；system-generated Docker item 可以进入 final Docker spec，但 source 必须标记为 system_generated；旧 `enabled_fields` 应迁移/清理为 ConfigItem.state，不作为长期兼容字段。

## 5. RunPlan 映射

ConfigItem.target 决定如何进入 RunPlan：

```text
args
env
mounts
ports
devices
docker_options
health_check
resource_controls
metadata_only
display_only
```

`metadata_only` 和 `display_only` 不进入执行 spec。RunPlan builder 必须根据 target、effective_value、state、required/read_only/system_generated 规则生成最终 spec，并记录 source map。
