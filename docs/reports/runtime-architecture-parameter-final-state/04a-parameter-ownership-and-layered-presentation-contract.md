# Parameter Ownership and Layered Presentation Contract

## 1. 目的

本文修正第一版“父 ConfigSet 不复制子 schema”的表述。第二轮设计明确：copy-on-create 可以完整复制上一层 effective ConfigSetBundle snapshot，包括 schema/value/state/provenance/snapshot/presentation；继承项 schema/snapshot 只读，owner 不变；当前层默认可以修改 value/state/provenance。

## 2. 核心规则

```text
每一层完整复制上一层 effective ConfigSetBundle 快照；本层新增自己的 ConfigSet；继承项 schema/snapshot 只读、owner 不变；本层只修改 value/state/provenance；最终 RunPlan 从 DeploymentConfigBundle effective snapshot 展开。
```

## 3. owner 语义

owner 表示 ConfigItem 的定义来源，不因 copy-on-create 改变。Deployment 可以修改 value.local_value、value.effective_value、state.enabled、state.checked、provenance.last_value_layer，但不能修改 schema.owner、schema.key、schema.type、schema.target、schema.arg_name、schema.constraints、snapshot.snapshot_from_layer。

## 4. copy-on-create

```text
BackendVersionConfigBundle
        ↓ copy-on-create
BackendRuntimeConfigBundle
        ↓ copy-on-create
NodeBackendRuntimeConfigBundle
        ↓ copy-on-create
DeploymentConfigBundle
        ↓ resolve
ResolvedRunPlan
        ↓ execute
Instance
```

每次创建下一层：deep copy parent effective bundle snapshot；保留 inherited ConfigSet 和 ConfigItem；记录 snapshot 来源；添加当前层 own ConfigSet；应用当前层 local_edits；生成当前层 effective_view。

## 5. ConfigSetBundle 组合

每一层不是单个 ConfigSet，而是一组 ConfigSet 的组合：

```text
ConfigSetBundle = inherited_bundle_snapshots + own_sets + local_edits + effective_view
```

父层可以包含 child ConfigSet，但展示和解析应保留 child ConfigSet 的边界。

## 6. 字段边界

schema 是只读定义字段；value 是当前层可修改字段；state 是当前层可修改 checked/enabled/editable/visible/valid；provenance 在当前层修改 value/state 时更新；snapshot 是 copy-on-create 来源，只读。

## 7. 不再使用 overridable_at 作为核心规则

不维护复杂 `overridable_at`。默认情况下，下一层拿到快照后可以修改继承项 value/state。特殊不可改项通过 `schema.read_only=true` 或 `state.editable=false` 表达。

## 8. checked / enabled 规则

| 状态 | 含义 | 是否 checked |
|---|---|---|
| default | schema 默认值 | 否 |
| required | 必填参数 | 否，除非当前层显式覆盖 |
| inherited | 从上层快照继承 | 否 |
| local_edit | 当前层显式修改 | 是 |
| system_generated | 系统生成 | 否 |
| runtime_detected | 运行时检测 | 否 |

规则：default value 不导致 checked；required 不导致 checked；inherited 不导致 checked；optional 默认不 checked；advanced 默认折叠；disabled input 仍显示 effective_value；checked 只表示当前层 local edit；unchecked optional 不进入当前层 local_edits；unchecked optional 不进入最终 RunPlan，除非它是 required/system_generated。

## 9. RunPlan source chain

ResolvedRunPlan 必须输出 parameter_source_map，并保留 source_chain。

```json
{
  "args": [
    {
      "key": "gpu_memory_utilization",
      "arg": "--gpu-memory-utilization",
      "value": 0.82,
      "effective_source": "deployment_local_edit",
      "source_chain": [
        {"layer": "BackendVersionConfigBundle", "value": 0.9, "reason": "schema default"},
        {"layer": "NodeBackendRuntimeConfigBundle", "value": 0.8, "reason": "node local edit"},
        {"layer": "DeploymentConfigBundle", "value": 0.82, "reason": "deployment local edit"}
      ]
    }
  ]
}
```

## 10. 验收要求

必须证明：copy-on-create 后上层修改不污染下层；下层修改不污染上层；schema/snapshot 只读；value/state 可在当前层修改；owner 不因 copy 改变；default/required/inherited 不显示为用户 checked；optional 默认不进入 RunPlan；Docker 子字段 obey ConfigItem.state；RunPlan preview 显示 source_chain；RunPlan preview 与实际 Docker spec 一致；preview/preflight/dry-run/start 共用同一个 builder。
