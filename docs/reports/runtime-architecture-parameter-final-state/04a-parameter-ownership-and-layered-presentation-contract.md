# Parameter Ownership and Layered Presentation Contract

## 1. 目的

本文是参数体系的硬约束文档。它专门定义：

1. 参数单一属主；
2. 参数单一定义；
3. copy-on-create 层级快照链；
4. 每层只叠加自己的内容；
5. 分层展示；
6. 最终 ResolvedRunPlan 合成。

执行端修复代码时，应把本文作为最高优先级参数契约。

## 2. 核心原则

一句话规则：

```text
一个参数只有一个属主、一个 schema 定义位置；每一层创建时拷贝上一层当时有效视图，再叠加自己这一层的数据或 override；页面只展示本层拥有或允许覆盖的内容；只有 ResolvedRunPlan 合成全部参数。
```

## 3. 参数单一属主

一个参数只能属于一个 owner。owner 决定 schema 定义位置。

owner 类型：

| Owner | 允许拥有的参数 |
|---|---|
| Model / ModelArtifact | 模型 metadata、格式、上下文、量化、模型文件能力 |
| Backend / BackendVersion | 后端能力、后端参数 schema、协议、endpoint、resource control schema |
| BackendRuntime | 运行模板 image/command/默认 args/env/mounts/ports/health check、模板 override |
| NodeBackendRuntime | 节点运行环境、设备绑定、节点 env、节点路径、check evidence、节点 override |
| Deployment | 部署级 override、资源覆盖、端口覆盖、卷覆盖、健康检查覆盖 |
| ResolvedRunPlan | 最终合成参数、最终 Docker spec、source map |
| Instance | 运行事实、状态、日志、实际 Docker spec 摘要 |

禁止：

1. 同一参数在 BackendRuntime 和 Deployment 各定义一份 schema；
2. UI 为了展示复制 schema；
3. Deployment 覆盖时重新定义 schema；
4. clone 时改变 schema owner；
5. 把最终 RunPlan 参数反写为上层 schema。

## 4. 参数单一定义

ParameterDefinition 只能定义一次。

Definition identity：

```text
parameter_definition_id
或
owner_type + owner_key + parameter_key
```

其他层级只能引用 definition 并保存 override：

```json
{
  "definition_ref": "backend_version:vllm.openai.latest:gpu_memory_utilization",
  "override_owner_type": "deployment",
  "override_owner_id": "...",
  "enabled": true,
  "value": 0.82
}
```

Override 不能包含：

1. label；
2. category；
3. type；
4. arg_name；
5. env_name；
6. target；
7. constraints；
8. choices；
9. schema-level help。

这些字段只能从 ParameterDefinition 读取。

## 5. copy-on-create 层级快照链

层级链：

```text
BackendVersion / ModelArtifact
        ↓ copy-on-create
BackendRuntime
        ↓ copy-on-create
NodeBackendRuntime
        ↓ copy-on-create
Deployment
        ↓ resolve
ResolvedRunPlan
        ↓ execute
Instance
```

### 5.1 BackendRuntime 创建

输入：BackendVersion 当前有效视图。

输出：BackendRuntime snapshot。

保存：

1. BackendVersion 能力快照；
2. BackendVersion 参数 definition 引用；
3. BackendRuntime 自己拥有的配置；
4. BackendRuntime override。

不做：

1. 不修改 BackendVersion；
2. 不把 BackendVersion schema 拷贝成 BackendRuntime schema；
3. 不写入节点状态。

### 5.2 NodeBackendRuntime 创建

输入：BackendRuntime 当前有效视图。

输出：NodeBackendRuntime snapshot。

保存：

1. BackendRuntime 有效视图快照；
2. 参数 definition 引用；
3. 节点运行环境配置；
4. 节点 override；
5. check-request evidence。

不做：

1. 不自动创建；
2. 不修改 BackendRuntime；
3. 不重定义上层参数 schema。

### 5.3 Deployment 创建

输入：NodeBackendRuntime 当前有效视图 + ModelArtifact / ModelLocation。

输出：Deployment snapshot。

保存：

1. selected NBR 快照；
2. selected model/location 快照；
3. 部署 override；
4. desired state；
5. 资源、端口、卷、健康检查覆盖。

不做：

1. 不自动创建 NBR；
2. 不修改 NBR；
3. 不重定义参数 schema；
4. 不把所有参数 checked。

### 5.4 ResolvedRunPlan 生成

输入：Deployment snapshot。

输出：最终运行 spec。

合成：

1. image；
2. command；
3. args；
4. env；
5. mounts；
6. ports；
7. devices；
8. health check；
9. resource controls；
10. parameter_source_map。

RunPlan preview 和 Agent Docker create 必须使用同一份 ResolvedRunPlan。

### 5.5 Instance 创建

输入：ResolvedRunPlan 执行结果。

保存：

1. container id；
2. actual Docker spec summary；
3. status；
4. health result；
5. logs；
6. errors；
7. operation_id。

Instance 不编辑参数。

## 6. 每一层只叠加自己的内容

层级叠加规则：

```text
current_effective_view = parent_snapshot + own_definitions + own_overrides + own_evidence
```

限制：

1. Model 只叠加模型内容；
2. BackendVersion 只叠加后端版本能力；
3. BackendRuntime 只叠加模板内容；
4. NodeBackendRuntime 只叠加节点运行环境内容；
5. Deployment 只叠加部署覆盖；
6. RunPlan 只做最终合成；
7. Instance 只记录运行事实。

## 7. 分层展示规则

### 7.1 Model 页面

展示：模型自己的 metadata、格式、能力、上下文、量化、文件信息、location。

隐藏：Docker image、Docker args、env、GPU runtime、端口、设备绑定。

### 7.2 Backend / BackendVersion 页面

展示：后端能力、版本能力、支持的 endpoint、参数能力定义。

隐藏：节点 image evidence、本机模型路径、部署覆盖。

### 7.3 BackendRuntime 页面

展示：运行模板 image、command、模板默认参数、模板 override、模板健康检查。

隐藏：模型实例路径、节点 check evidence、部署级端口覆盖。

### 7.4 NodeBackendRuntime 页面

展示：节点运行环境、image check、Docker runtime、设备绑定、节点 env、节点 override、check evidence。

隐藏：模型 metadata 编辑项、BackendVersion schema 编辑项。

### 7.5 Deployment 页面

展示：模型选择、NBR 选择、部署 override、端口/卷/健康检查覆盖、最终有效参数预览、RunPlan preview。

限制：Deployment 只能保存 override，不能重新定义 schema。

### 7.6 Instance 页面

展示：运行状态、健康检查、日志、实际 Docker spec 摘要、错误。

限制：不编辑运行参数。

## 8. checked / enabled 展示规则

UI 必须区分：

| 状态 | 含义 | 是否 checked |
|---|---|---|
| default | schema 默认值 | 否 |
| required | 必填参数 | 否，除非用户覆盖 |
| inherited | 从上层快照继承 | 否 |
| override | 当前层显式覆盖 | 是 |
| system_generated | 系统生成 | 否 |
| runtime_detected | 运行时检测 | 否 |

规则：

1. default value 不导致 checked；
2. required 不导致 checked；
3. optional 默认不 checked；
4. advanced 默认不 checked；
5. disabled input 仍显示值；
6. checked 只表示当前层用户显式覆盖；
7. unchecked optional 不进入当前层 override。

## 9. RunPlan source map

ResolvedRunPlan 必须提供 parameter_source_map。

示例：

```json
{
  "args": [
    {
      "key": "gpu_memory_utilization",
      "arg": "--gpu-memory-utilization",
      "value": 0.82,
      "source": "deployment_override",
      "definition_ref": "backend_version:vllm.openai.latest:gpu_memory_utilization",
      "override_ref": "deployment:dep-123:gpu_memory_utilization"
    },
    {
      "key": "host",
      "arg": "--host",
      "value": "0.0.0.0",
      "source": "backend_runtime_default",
      "definition_ref": "backend_version:vllm.openai.latest:host"
    }
  ]
}
```

source 至少支持：

1. default；
2. model；
3. backend_version；
4. backend_runtime；
5. node_backend_runtime；
6. deployment_override；
7. system_generated；
8. runtime_detected。

## 10. 验收要求

必须通过测试或 E2E 证明：

1. 一个参数只有一个 schema 定义；
2. Deployment override 不复制 schema；
3. NodeBackendRuntime override 不复制 schema；
4. copy-on-create 后上层修改不污染下层；
5. copy-on-create 后下层修改不污染上层；
6. Model 页面不展示 Docker 参数；
7. Deployment 页面可以覆盖允许覆盖的运行参数；
8. 所有参数没有默认 checked；
9. default value 不导致 enabled；
10. required 不显示为用户 checked；
11. optional 默认不进入 override；
12. RunPlan preview 显示 source；
13. RunPlan preview 与 Docker spec 一致；
14. clone 不扩大 checked 范围。
