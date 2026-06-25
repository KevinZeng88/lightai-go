# LightAI Go ConfigSet / ConfigItem 正式设计

目标路径建议：

```text
docs/design/catalog-configset-and-runtime-snapshot.md
```

## 1. 设计定位

LightAI Go 不维护 vLLM / SGLang / llama.cpp 的官方全量参数百科。平台维护的是：

```text
常用、可展示、可渲染、可验证或可文档化的结构化参数集
+
extra 透传机制
```

目标是：

- 常用参数尽量结构化覆盖；
- 未覆盖参数允许透传；
- 不用 Go 代码写死 backend 参数；
- 尽力跑通，不用过强校验限制用户；
- 保留参数来源与最终修改层级，便于审计和排障。

## 2. 核心对象

### 2.1 ConfigItem

一个 ConfigItem 表示一个可配置项。它可以是启动载体参数、运行环境参数、模型/后端参数。

最小 schema：

```json
{
  "code": "backend.vllm.gpu_memory_utilization",
  "category": "model_runtime",
  "kind": "cli_arg",
  "type": "number",
  "value": 0.85,
  "default_value": 0.85,
  "enabled": true,
  "render": {
    "target": "cli",
    "flag": "--gpu-memory-utilization",
    "style": "flag_space_value"
  },
  "order": 320,
  "constraints": {
    "min": 0.1,
    "max": 1.0,
    "severity": "warning"
  },
  "support_level": "verified",
  "source": {
    "layer": "BackendVersion",
    "ref": "vllm-compatible",
    "reason": "catalog_default"
  },
  "last_modified": {
    "layer": "Deployment",
    "ref": "dep-xxx",
    "operation": "override"
  }
}
```

字段说明：

| 字段 | 说明 |
|---|---|
| `code` | 稳定内部编码，不使用 CLI flag 作为唯一身份 |
| `category` | `launcher` / `runtime_env` / `model_runtime` |
| `kind` | `cli_arg` / `cli_args` / `env` / `env_lines` / `port` / `volume` / `device` / `health_check` / `launcher_option` |
| `type` | `string` / `integer` / `number` / `boolean` / `array` / `object` / `lines` |
| `value` | 当前有效值 |
| `default_value` | 默认值 |
| `enabled` | 是否启用 |
| `render` | 渲染目标与格式 |
| `order` | 渲染顺序 |
| `constraints` | 轻量约束，默认 warning |
| `support_level` | `verified` / `documented` / `experimental` |
| `source` | 初始来源 |
| `last_modified` | 最后在哪一层修改 |

不要求定义 `editable_at`。系统只记录谁改了，不预设谁能改。

### 2.2 ConfigSet

每一级保存一个完整 materialized ConfigSet：

```json
{
  "schema_version": 1,
  "context": {
    "backend": "vllm",
    "backend_version": "vllm-compatible",
    "launcher_kind": "docker"
  },
  "items": {
    "launcher.image": {},
    "runtime.visible_devices": {},
    "runtime.model_mount": {},
    "backend.vllm.gpu_memory_utilization": {},
    "backend.extra_args": {}
  },
  "source_metadata": {
    "source_ref": "configs/backend-catalog/backends/vllm.yaml#runtimes/vllm-docker-default",
    "source_hash": "sha256:...",
    "parent_ref": "backend_version:vllm-compatible",
    "materialized_at": "catalog-import"
  }
}
```

`context` 当前只保留：

- `backend`
- `backend_version`
- `launcher_kind`

未来确有需要时再扩展 `accelerator_profile`，不在每个 ConfigItem 上引入复杂适用范围。

## 3. 三类配置项

### 3.1 launcher

启动载体参数，控制服务如何被启动。它不是 Docker 专属。

典型项：

```text
launcher.kind
launcher.image
launcher.executable
launcher.entrypoint
launcher.command
launcher.working_dir
launcher.user
launcher.network
launcher.ports
launcher.volumes
launcher.devices
launcher.privileged
launcher.shm_size
launcher.restart_policy
launcher.log_policy
launcher.extra_options
```

当前必须实现 Docker renderer。Process/systemd/k8s 只保留设计，不需要本轮完整实现。

### 3.2 runtime_env

运行环境参数，控制模型服务进程所在环境。

典型项：

```text
runtime.env
runtime.visible_devices
runtime.model_path
runtime.model_mount
runtime.cache_dir
runtime.hf_home
runtime.library_path
runtime.health.endpoint
runtime.health.timeout
runtime.extra_env
```

### 3.3 model_runtime

模型/后端进程参数，通常渲染为 vLLM / SGLang / llama.cpp CLI 参数。

典型项：

```text
backend.common.host
backend.common.port
backend.common.served_model_name
backend.vllm.gpu_memory_utilization
backend.sglang.mem_fraction_static
backend.llamacpp.ctx_size
backend.extra_args
```

## 4. BackendVersion 的职责

BackendVersion 是兼容性记录，不是运行模板。

它记录：

- 支持的模型格式；
- 支持的任务/API；
- LightAI Go 已建模的常用参数；
- 参数别名/重命名；
- 已知限制；
- 版本/镜像兼容风险说明。

它不直接决定 image、device、volume、runtime env。真正运行模板由 BackendRuntime 表达。

BackendVersion 示例：

```json
{
  "id": "sglang-compatible",
  "backend_id": "sglang",
  "version": "compatible",
  "capabilities": {
    "supported_formats": ["huggingface", "safetensors"],
    "supported_tasks": ["chat", "completion"],
    "api": ["openai_compatible"]
  },
  "supported_config_items": [
    "backend.common.host",
    "backend.common.port",
    "backend.sglang.mem_fraction_static",
    "backend.sglang.context_length",
    "backend.sglang.trust_remote_code",
    "backend.extra_args"
  ],
  "warnings": [
    "Compatible profile; exact image version may differ from declared compatibility."
  ]
}
```

`BackendVersion.supported_config_items` 不表示官方参数全集，只表示平台已建模/已文档化/已验证的参数。

## 5. 常用参数与 extra 机制

### 5.1 常用参数结构化

初始结构化参数覆盖：

#### 通用

```text
backend.common.host
backend.common.port
backend.common.served_model_name
runtime.model.path
runtime.model.mount_path
backend.extra_args
runtime.extra_env
launcher.extra_options
```

#### vLLM

```text
backend.vllm.gpu_memory_utilization
backend.vllm.max_model_len
backend.vllm.dtype
backend.vllm.tensor_parallel_size
backend.vllm.trust_remote_code
backend.vllm.enforce_eager
backend.vllm.max_num_seqs
backend.vllm.max_num_batched_tokens
backend.vllm.swap_space
backend.vllm.disable_log_requests
```

#### SGLang

```text
backend.sglang.mem_fraction_static
backend.sglang.context_length
backend.sglang.tp
backend.sglang.trust_remote_code
backend.sglang.dtype
backend.sglang.max_running_requests
backend.sglang.disable_cuda_graph
backend.sglang.enable_metrics
```

#### llama.cpp

```text
backend.llamacpp.model_file
backend.llamacpp.ctx_size
backend.llamacpp.n_gpu_layers
backend.llamacpp.threads
backend.llamacpp.batch_size
backend.llamacpp.ubatch_size
backend.llamacpp.parallel
backend.llamacpp.cont_batching
backend.llamacpp.flash_attn
backend.llamacpp.jinja
```

### 5.2 backend.extra_args

用于模型/后端 CLI 透传。UI 每行一个参数。

允许格式：

```text
--flag
--flag value
--flag=value
```

内部示例：

```json
{
  "code": "backend.extra_args",
  "category": "model_runtime",
  "kind": "cli_args",
  "type": "lines",
  "value": [
    "--enable-prefix-caching",
    "--max-num-seqs 16"
  ],
  "render": {
    "target": "cli",
    "style": "raw_lines"
  },
  "order": 900,
  "support_level": "experimental"
}
```

### 5.3 runtime.extra_env

每行一个 `KEY=VALUE`：

```text
HF_HOME=/data/hf-cache
VLLM_LOGGING_LEVEL=DEBUG
```

### 5.4 launcher.extra_options

每行一个启动载体 option。当前可保留设计，后续高级模式开放。

## 6. extra 排除规则

extra 不能重复结构化参数。

| 场景 | 处理 |
|---|---|
| `backend.extra_args` 重复结构化 CLI flag | error |
| `runtime.extra_env` 重复结构化 env name | error |
| `launcher.extra_options` 重复结构化 launcher option | error |
| unknown extra flag/env/option | warning，允许 |
| extra line 无法解析 | warning 优先 |
| 结构化参数类型完全错误 | error |
| 结构化参数超出建议范围 | warning |
| 模型路径不存在 | error |
| image 不存在 | error |

## 7. Materialization 与 Copy-on-create

每一级只做：

```text
copy parent ConfigSet
apply current layer overrides
update source / last_modified
save full materialized ConfigSet
```

链路：

```text
Backend
→ BackendVersion
→ BackendRuntime
→ NodeBackendRuntime
→ Deployment
→ ResolvedRunPlan
```

规则：

```text
BackendVersion.config_set =
  copy(Backend.config_set) + apply(version compatibility overrides)

BackendRuntime.config_set =
  copy(BackendVersion.config_set) + apply(runtime launcher/template overrides)

NodeBackendRuntime.config_set =
  copy(BackendRuntime.config_set) + apply(node/runtime overrides)

Deployment.config_set =
  copy(NodeBackendRuntime.config_set) + apply(deployment overrides)

RunPlan =
  render(Deployment.config_set + model/location facts)
```

RunPlan 不回读 parent Backend / BackendVersion / BackendRuntime / NBR live data。

## 8. Renderer

Renderer 只根据 ConfigItem 渲染，不根据 backend name 写死。

### 8.1 CLI renderer

支持：

```text
flag_space_value
flag_equals_value
flag_if_true
repeat_flag
positional
raw_lines
```

### 8.2 Launcher renderer

根据 `ConfigSet.context.launcher_kind` 选择：

```text
docker
process
systemd
k8s
```

本轮必须实现 Docker，其他只保留接口和设计。

### 8.3 Docker renderer

处理：

```text
launcher.image
launcher.ports
launcher.volumes
launcher.devices
launcher.privileged
launcher.shm_size
runtime.env
runtime.visible_devices
runtime.model_mount
model_runtime cli args
```

### 8.4 Health renderer

处理：

```text
runtime.health.endpoint
runtime.health.timeout
runtime.health.expected_status
```

## 9. 数据库设计

不做旧 DB 兼容。可以重建 schema。

目标表字段：

```text
backends.config_set_json
backend_versions.config_set_json
backend_runtimes.config_set_json
node_backend_runtimes.config_set_json
model_deployments.config_set_json
```

并增加：

```text
source_metadata_json
```

删除旧权威字段：

```text
capabilities_json
parameter_schema_json
parameter_values_json
env_json
ports_json
volumes_json
devices_json
health_check_json
resource_controls_json
parameters_json
default_args_json
parameter_defs_json
default_backend_params_json
default_images_json
image_candidates_json
docker_options_json
model_mount_json
```

Implementation rule:

- Fresh DB clean schema is the only accepted DB baseline.
- Do not preserve the V1->V28 historical compatibility migration chain as the final database initialization path.
- Do not implement additive migration that keeps old authority columns.
- Do not add V29-style additive migration.
- Do not keep ALTER TABLE ADD COLUMN compatibility migrations for old authority fields.
- Do not keep old columns to protect legacy API read paths.
- Do not dual-read or dual-write old fields and ConfigSet.
- Do not preserve legacy fallback.
- Do not keep old-data backfill, repair, normalizeLegacy, seed repair, or dual-read/dual-write paths.
- Useful current table definitions from old migrateVx functions must be collapsed into the clean schema initializer.
- Legacy upgrade logic, old authority columns, old catalog seed literals, seed-only backend versions, and compatibility repair functions must be deleted.
- schema_version may remain only as a clean-schema baseline marker. It must not imply support for upgrading historical DB versions.
- If old response shapes are needed for display, they must be derived from ConfigSet after the API contract is updated, not stored as DB authority and not accepted as create/update payload.
- Each committed checkpoint must preserve this clean-state invariant.

如果某些 response 短期需要旧形态，只能从 ConfigSet 投影，不得继续作为 DB 权威字段保存。

## 10. API 设计

旧字段删除。新 API 使用：

```json
{
  "config_set": {},
  "config_overrides": {},
  "source_metadata": {}
}
```

所有 create/update/clone/enable/deploy 接口都围绕 ConfigSet / override 工作。

## 11. UI 设计

UI 参数编辑器按 category 展示：

```text
launcher
runtime_env
model_runtime
extra_args
extra_env
```

显示：

```text
当前值
默认值
enabled
support_level
source
last_modified
warning/error
```

不需要 editable_at。谁修改就记录谁。

## 12. 文档归档

过时文档归档到：

```text
docs/archive/<date>-pre-configset-catalog-model/
```

过时文档包括：

- 仍以 `capabilities_json` 为权威的文档；
- 仍以 `parameter_schema_json` / `parameter_values_json` 为权威的文档；
- 仍描述 `db.go seed` 为 catalog 来源的文档；
- 仍描述旧 `/check` route 的文档；
- 仍描述 `parameters_json` 的文档；
- 仍把 preflight PASS / task claimed / image present 当 runtime smoke PASS 的文档。

归档文档顶部加：

```text
Archived. Superseded by docs/design/catalog-configset-and-runtime-snapshot.md.
```

## 13. 验收标准

最终必须满足：

```text
db.go 不再含 vLLM/SGLang/llama.cpp catalog literal
YAML + registry 是唯一 catalog 权威来源
旧字段/旧 API/旧兼容逻辑已删除
过时文档已归档
Backend/BackendVersion/BackendRuntime/NBR/Deployment 全部以 ConfigSet copy-on-create
RunPlan/DockerSpec 只从最终 ConfigSet 渲染
extra_args / extra_env 可用且排除重复结构化参数
fresh DB 无例外通过
vLLM / SGLang / llama.cpp 三 runtime platform-chain smoke PASS
go test ./... PASS
server/agent build PASS
web test/build PASS
git status 原文可解释
```
