# 01. Semantic Config Design

## 1. 背景问题

当前 LightAI Go 的参数编辑经历了多轮修复，已经具备 ConfigEditView、ConfigSet、RunPlan、copy-on-create 等基础。但近期手工验证暴露出更深层问题：

- 同一语义被多个字段重复建模，例如：
  - `backend.common.host`
  - `launcher.listen_host`
  - `service.listen_host`
- 参数名从 backend / launcher / Docker 技术层直接泄露到用户界面。
- BackendRuntime、NodeBackendRuntime、Deployment 等不同页面各自理解参数，导致显示、保存、预览、运行链路不一致。
- UI 里字段过多，大量高级/不常用参数默认 checked。
- 保存时出现 `unknown config field "service.listen_host"`，说明显示层 canonical key 和底层 storage key 不一致。
- Health check、port、model mount、model runtime 参数等归属边界不清。

这些不是单点 UI 问题，而是参数治理模型不完整。

## 2. 核心目标

建立一套通用的 Semantic Config 基础能力，所有参数编辑入口复用它。

目标不是把重复字段在 UI 层简单合并，而是从源头上规定：

> 一个业务语义只能有一个 semantic key，一个参数只能有一个 owner。其他对象如需使用该参数，必须通过 copy-on-create 快照获得自己的副本，而不是重新定义同义字段。

## 3. 基本概念

### 3.1 Semantic Parameter Definition

每个参数先作为语义参数定义，而不是页面字段或 backend flag。

建议结构：

```yaml
key: model_runtime.max_model_len
owner: model_runtime
value_type: integer
label:
  zh-CN: 最大上下文长度
  en-US: Max Model Length
description: 模型运行时最大上下文长度
category: model_runtime
display_tier: advanced
hard_validation:
  - integer
  - value > 0
warning_rules:
  - value > model_context_length: warn
  - estimated_vram_exceeds_available: warn
resolver_mapping:
  vllm:
    cli_flag: --max-model-len
  sglang:
    cli_flag: --context-length
copy_policy:
  to_deployment: true
```

### 3.2 Semantic Key

参数 key 必须表达业务语义，而不是某个后端或 launcher 的实现细节。

正确示例：

```text
service.listen_host
service.container_port
deployment.host_port
model_runtime.max_model_len
model_runtime.gpu_memory_utilization
runtime.image_ref
docker.shm_size
docker.ipc_mode
runtime.health.path
runtime.model_mount.container_path
```

不应长期存在的示例：

```text
backend.common.host
backend.common.port
launcher.listen_host
launcher.container_port
backend.arg.max_model_len
backend.arg.gpu_memory_utilization
```

这些可以作为 catalog normalize 的输入兼容项，不能作为长期配置模型。

### 3.3 Owner

owner 表示“这个参数语义属于谁”。owner 不等于只能在哪里修改。

| Owner | 含义 | 示例 |
|---|---|---|
| backend_capability | 后端能力/参数映射定义 | 支持 OpenAI 协议、支持哪些 semantic keys |
| model_runtime | 模型运行参数 | max_model_len、dtype、quantization |
| runtime_environment | 运行环境参数 | image_ref、docker.shm_size、devices |
| runtime_service | 容器内模型服务 | listen_host、container_port、health path |
| deployment_exposure | 外部访问暴露 | host_port、route、served endpoint |
| scheduler_resource | 调度资源 | GPU 数量、GPU device binding、memory limit |
| diagnostic | 诊断/来源 | source_metadata、capabilities raw |

### 3.4 Config Snapshot

BackendRuntime、NodeBackendRuntime、Deployment 等对象保存的是自己的参数快照。

快照字段建议包含：

```yaml
key: service.container_port
owner: runtime_service
value: 8000
enabled: true
source_snapshot:
  object_kind: backend_runtime
  object_id: xxx
  key: service.container_port
  value: 8000
  copied_at: 2026-06-27T00:00:00Z
dirty: false
warnings: []
```

下游创建时复制上游参数，复制后就是下游自己的配置。上游后续变化不自动影响下游。

### 3.5 Warning 优先

不要把参数限制理解成“哪些层允许改”。下游复制参数后原则上可以改。

规则：

- 类型不合法、必填为空、格式非法、路径不可解析等硬错误阻断保存。
- 超出建议值、可能显存不足、后端可能不支持、性能风险等显示 warning，不阻断保存。
- UI 可在参数名前加 `!`、黄色图标或 warning tag。

## 4. Copy-on-create 语义

创建链路建议：

```text
BackendVersion / ModelArtifact semantic definitions
        ↓ copy selected defaults
BackendRuntime template snapshot
        ↓ copy selected runtime env defaults
NodeBackendRuntime node snapshot
        ↓ copy selected runtime + model params
Deployment snapshot
        ↓ resolve
RunPlan
```

每一步只复制需要出现在下一对象中的 semantic config item。

### 4.1 不使用 override 作为核心模型

旧说法“Deployment override max_model_len”容易误导，好像 Deployment 拥有这个参数。

新的说法：

```text
Deployment 持有 model_runtime.max_model_len 的一份配置副本。该副本复制自模型/后端建议值，用户可改，改动后生成 warning 或 dirty 标记。
```

## 5. 参数归属示例

### 5.1 端口

| 语义 | key | owner | 说明 |
|---|---|---|---|
| 容器内监听地址 | service.listen_host | runtime_service | 后端进程绑定地址，通常 0.0.0.0 |
| 容器内监听端口 | service.container_port | runtime_service | vLLM/SGLang/llama.cpp 内部服务端口 |
| 外部访问端口 | deployment.host_port | deployment_exposure | Docker host port / gateway port |
| 健康检查端口 | 默认引用 service.container_port | runtime_service | 特殊情况可用 health.port_override |

禁止重复建模：

```text
backend.common.host
launcher.listen_host
backend.common.port
launcher.container_port
```

### 5.2 模型上下文长度

| 语义 | key | owner | 说明 |
|---|---|---|---|
| 最大上下文长度 | model_runtime.max_model_len | model_runtime | 模型运行参数，Deployment 可以持有副本并修改 |

BackendVersion 只定义它如何映射到后端：

```yaml
vllm: --max-model-len
sglang: --context-length
```

### 5.3 镜像

| 语义 | key | owner | 说明 |
|---|---|---|---|
| 运行模板默认镜像 | runtime.image_ref | runtime_environment | 模板默认 Docker image |
| 节点实际镜像 | runtime.image_ref | runtime_environment | NBR 复制后可改，值来自节点镜像列表或手工输入 |

同一个 key，复制后成为不同对象自己的快照值。

### 5.4 Docker 参数

| 语义 | key | owner | 说明 |
|---|---|---|---|
| 共享内存 | docker.shm_size | runtime_environment | 常用，运行模板/NBR 可见 |
| IPC 模式 | docker.ipc_mode | runtime_environment | 常用或高级 |
| 设备映射 | docker.devices | runtime_environment | 高级，默认折叠 |
| 附加用户组 | docker.group_add | runtime_environment | 高级，默认关闭 |

## 6. Backend flag 与 semantic key 的关系

Backend CLI flag 不是配置 key。

示例：

| Semantic key | vLLM | SGLang | llama.cpp |
|---|---|---|---|
| service.listen_host | --host | --host | --host |
| service.container_port | --port | --port | --port |
| model_runtime.max_model_len | --max-model-len | --context-length | --ctx-size |
| deployment.served_model_name | --served-model-name | --served-model-name | 可能不支持 |

BackendVersion 应保存 adapter mapping，而不是生成 `backend.arg.*` 用户字段。

## 7. UI 展示规则

### 7.1 字段名

中文界面建议：

```text
中文名（English Name）
```

例如：

```text
容器监听端口（Container Port）
最大上下文长度（Max Model Length）
共享内存大小（Shared Memory）
```

### 7.2 显示分级

| Tier | 展示方式 | 示例 |
|---|---|---|
| required | 普通区，必须，不能关闭 | 镜像、必要端口 |
| common | 普通区，可改 | 容器端口、shm_size |
| recommended | 普通区或建议区，有建议值 | health.path、ipc_mode |
| advanced | 高级折叠，默认不启用 | devices、ulimits、security_options |
| diagnostic | 开发诊断，只读折叠 | raw ConfigSet、source metadata |

### 7.3 Warning 展示

参数有 warning 时：

```text
! 最大上下文长度（Max Model Length）
```

鼠标悬停或下方提示：

```text
当前值超过模型建议上下文长度，可能导致显存不足。
```

## 8. 非目标

本次设计不是：

- 为每个页面单独写字段隐藏逻辑。
- 继续补 alias 合并显示。
- 为历史 DB 做复杂兼容迁移。
- 把所有后端 CLI flag 暴露为用户配置项。

必要时允许重建 DB / 重新加载 catalog，以保持 schema 和配置模型干净。
