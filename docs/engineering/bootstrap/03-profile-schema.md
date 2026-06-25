# Bootstrap Profile Schema

状态：`READY_FOR_IMPLEMENTATION`

> **注意**：`configs/bootstrap/` 目录当前不存在，将在实现阶段创建。本文件描述目标 schema。

## 文件位置

提交两个 profile：

```text
configs/bootstrap/bootstrap-profile.example.yaml
configs/bootstrap/local-kz-laptop.yaml
```

`local-kz-laptop.yaml` 可以提交，因为它只包含开发测试环境路径、镜像、端口、模型信息，不包含真实密码、token、cookie、CSRF。

## Schema 顶层结构

```yaml
profile_name: local-kz-laptop

server: {}
auth: {}
tenant: {}
node: {}
models: {}
runtimes: {}
bootstrap: {}
```

## server

```yaml
server:
  base_url: http://localhost:18080
  agent_url: http://localhost:19091
  runtime_dir: /tmp/lightai
```

字段：

| 字段 | 说明 |
|---|---|
| `base_url` | LightAI Server URL |
| `agent_url` | LightAI Agent URL |
| `runtime_dir` | 运行目录，用于查 DB、logs、initial credentials file |

## auth

```yaml
auth:
  username: admin
  initial_password_env: LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD
  initial_password: admin
  initial_password_file: ""
  initial_password_runtime_files:
    - auto
  final_password_env: LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD
  final_password_file: ""
```

字段：

| 字段 | 说明 |
|---|---|
| `username` | 管理员用户名 |
| `initial_password_env` | 初始密码环境变量名 |
| `initial_password` | profile 中的低优先级初始密码，不用于生产敏感场景 |
| `initial_password_file` | 初始密码文件 |
| `initial_password_runtime_files` | runtime-dir 下初始化凭据文件查找策略，`auto` 表示按代码确认路径自动查找 |
| `final_password_env` | 目标管理员密码环境变量名 |
| `final_password_file` | 目标管理员密码文件 |

安全要求：

- profile 默认不写真实密码；
- export 默认不输出密码；
- profile 可以提交到仓库时，不得包含真实密码、token、cookie、CSRF。

## tenant

```yaml
tenant:
  name: default
```

可选字段：

```yaml
tenant:
  name: default
  id: ""
```

bootstrap 应按 name 查找 tenant，必要时记录 id 到 `bootstrap-state.json`。

## node

```yaml
node:
  name: KZ-LAPTOP
  gpu_vendor: nvidia
  gpu_ids:
    - "0"
```

可选字段：

```yaml
node:
  id: ""
  accelerator_ids:
    - "0"
```

说明：

- `gpu_ids`：用于 node registration / agent probe / 当前节点 GPU 表示（映射到 `POST /api/v1/agent/register` 和相关 agent 接口）。
- `accelerator_ids`：vendor-neutral accelerator IDs，用于 deployment / NBR / RunPlan / 设备绑定语义（映射到 `POST /api/v1/deployments` 的 `accelerator_ids` 字段）。
- profile 中两个字段可同时存在，但实现时不得混用。bootstrap 脚本根据 API 目的选择对应字段。
- 当前本机为 NVIDIA GPU；后续支持 MetaX / Huawei 时应使用 vendor-neutral accelerator_ids。

## models

当前开发机测试模型：

```yaml
models:
  qwen3_small:
    display_name: Qwen3-0.6B-Instruct-2512
    kind: huggingface
    path: /home/kzeng/models/Qwen3-0.6B-Instruct-2512

  qwen35_gguf:
    display_name: Qwen3.5-9B-Q4_K_M
    kind: gguf
    path: /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
```

字段：

| 字段 | 说明 |
|---|---|
| key | profile 内引用名，例如 `qwen3_small` |
| `display_name` | UI 显示名 |
| `kind` | `huggingface` 或 `gguf` |
| `path` | 本机模型路径 |
| `artifact_id` | export 可写入，bootstrap 可忽略并按名称/路径查找 |
| `location_id` | export 可写入，bootstrap 可忽略并按路径查找 |
| `source` | 可选，标记 API/export/manual |

路径检查：

- `huggingface` 应检查目录存在；
- `gguf` 应检查文件存在；
- 路径不存在时，models-only 失败并写入明确错误。

## runtimes

### vLLM

```yaml
runtimes:
  vllm:
    backend: vllm
    image: vllm/vllm-openai:latest
    model: qwen3_small
    container_port: 8000
    host_port: 8004
    parameters:
      gpu_memory_utilization:
        enabled: true
        value: "0.65"
      max_model_len:
        enabled: true
        value: "8192"
```

### SGLang

```yaml
  sglang:
    backend: sglang
    image: lmsysorg/sglang:latest
    model: qwen3_small
    container_port: 30000
    host_port: 30000
    parameters:
      mem_fraction_static:
        enabled: true
        value: "0.65"
      context_length:
        enabled: true
        value: "8192"
```

### llama.cpp

```yaml
  llamacpp:
    backend: llamacpp
    image: ghcr.io/ggml-org/llama.cpp:server-cuda13
    model: qwen35_gguf
    container_port: 8000
    host_port: 8002
    parameters:
      n_gpu_layers:
        enabled: true
        value: "99"
      ctx_size:
        enabled: true
        value: "8192"
```

字段：

| 字段 | 说明 |
|---|---|
| key | runtime 引用名 |
| `backend` | backend catalog ID，必须与 `configs/backend-catalog/backends/{name}.yaml` 中的 backend ID 一致。实现时从 API catalog（`GET /api/v1/backends`）或 `configs/backend-catalog/` 校验。当前实际 catalog ID 为：`vllm`、`sglang`、`llamacpp`（注意是单字母 `s` 全小写，不是 `llama.cpp`） |
| `backend_version` | 可选，指定版本 |
| `image` | Docker image |
| `model` | 引用 `models` key |
| `container_port` | 容器端口 |
| `host_port` | 主机端口 |
| `parameters` | RuntimeParameterEditor 同构参数，保留 enabled + value |
| `docker_json` | export 可写入 |
| `args_override_json` | export 可写入 |
| `default_env_json` | export 可写入，默认脱敏 |
| `parameter_values_json` | export 可写入 |
| `health_check` | export 可写入 |
| `status` | export 可写入，用作参考，不作为 bootstrap 强制状态 |

## bootstrap

```yaml
bootstrap:
  default_mode: dry-run
  output_dir: /tmp/lightai/e2e/bootstrap
  allow_real_container_start: false
  allow_chat_completion: false
```

可选字段：

```yaml
bootstrap:
  keep_containers_after_full: false
  default_export_profile: configs/bootstrap/local-kz-laptop.yaml
  include_runtime_state: false
```

## local-kz-laptop.yaml 完整示例

```yaml
profile_name: local-kz-laptop

server:
  base_url: http://localhost:18080
  agent_url: http://localhost:19091
  runtime_dir: /tmp/lightai

auth:
  username: admin
  initial_password_env: LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD
  initial_password: admin
  initial_password_file: ""
  initial_password_runtime_files:
    - auto
  final_password_env: LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD
  final_password_file: ""

tenant:
  name: default

node:
  name: KZ-LAPTOP
  gpu_vendor: nvidia
  gpu_ids:
    - "0"

models:
  qwen3_small:
    display_name: Qwen3-0.6B-Instruct-2512
    kind: huggingface
    path: /home/kzeng/models/Qwen3-0.6B-Instruct-2512

  qwen35_gguf:
    display_name: Qwen3.5-9B-Q4_K_M
    kind: gguf
    path: /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf

runtimes:
  vllm:
    backend: vllm
    image: vllm/vllm-openai:latest
    model: qwen3_small
    container_port: 8000
    host_port: 8004
    parameters:
      gpu_memory_utilization:
        enabled: true
        value: "0.65"
      max_model_len:
        enabled: true
        value: "8192"

  sglang:
    backend: sglang
    image: lmsysorg/sglang:latest
    model: qwen3_small
    container_port: 30000
    host_port: 30000
    parameters:
      mem_fraction_static:
        enabled: true
        value: "0.65"
      context_length:
        enabled: true
        value: "8192"

  llamacpp:
    backend: llamacpp
    image: ghcr.io/ggml-org/llama.cpp:server-cuda13
    model: qwen35_gguf
    container_port: 8000
    host_port: 8002
    parameters:
      n_gpu_layers:
        enabled: true
        value: "99"
      ctx_size:
        enabled: true
        value: "8192"

bootstrap:
  default_mode: dry-run
  output_dir: /tmp/lightai/e2e/bootstrap
  allow_real_container_start: false
  allow_chat_completion: false
```

## 安装场景 example profile

`configs/bootstrap/bootstrap-profile.example.yaml` 应更通用：

- 不写本机特定模型路径；
- 用注释说明如何填写；
- 保留 vLLM / SGLang / llama.cpp 示例；
- 密码只写 env 名，不写明文；
- runtime_dir 默认 `/tmp/lightai` 或安装文档指定目录。

## Profile 校验

脚本读取 profile 后应校验：

1. YAML 可解析；
2. server 字段完整；
3. auth username 存在；
4. models 路径存在；
5. runtime 引用的 model key 存在；
6. runtime backend 名称可在 catalog 中找到；
7. host_port 不重复；
8. full 模式时 `allow_real_container_start` 必须为 true，且命令行传 `--allow-real-start`。

## Export 写回规则

export 输出 profile 时：

- 字段顺序稳定；
- 不输出密码、token、cookie、CSRF；
- 默认写 `initial_password_env` 和 `final_password_env`；
- `initial_password_runtime_files` 可以记录路径但不记录密码；
- 已存在输出文件先备份。
