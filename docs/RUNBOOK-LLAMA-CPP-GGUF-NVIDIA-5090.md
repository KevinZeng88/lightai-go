# LightAI Go — NVIDIA 5090 + llama.cpp + GGUF 测试环境文档

> 创建日期：2026-06-15
> 用途：Phase 1 / Phase 2 / Phase 4 本机 NVIDIA 验证参考
> 状态：已验证可用

## 目录

1. [环境信息](#1-环境信息)
2. [Docker GPU runtime 验证](#2-docker-gpu-runtime-验证)
3. [llama.cpp CUDA Server 启动命令](#3-llamacpp-cuda-server-启动命令)
4. [`/v1/models` 测试](#4-v1models-测试)
5. [`/v1/chat/completions` 测试](#5-v1chatcompletions-测试)
6. [对 LightAI 对象的映射](#6-对-lightai-对象的映射)
7. [对 Phase 1 的测试意义](#7-对-phase-1-的测试意义)
8. [对 Phase 2 的测试意义](#8-对-phase-2-的测试意义)
9. [对 Phase 4 Gateway 的测试意义](#9-对-phase-4-gateway-的测试意义)
10. [故障排查](#10-故障排查)

---

## 1. 环境信息

### 硬件

```text
Host: KZ-LAPTOP
GPU: NVIDIA GeForce RTX 5090 Laptop GPU
VRAM: 24,463 MiB (~24 GB)
```

### 驱动

```text
NVIDIA-SMI:  610.43.02
KMD Version: 610.47
CUDA UMD Version: 13.3
```

### Docker GPU runtime

已通过 `docker run --gpus all` 验证容器内可访问 GPU。

### 模型文件

```text
路径: /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
格式: GGUF
量化: Q4_K_M
大小: ~5.6 GB
参数量: 8,953,803,264 (~9B)
```

模型目录结构：

```text
/home/kzeng/models/Qwen3.5-9B-Q4/
  Qwen3.5-9B-Q4_K_M.gguf
  README.md
  configuration.json
```

### Docker 镜像

```text
ghcr.io/ggml-org/llama.cpp:server-cuda13
```

CUDA 13 兼容，支持 RTX 5090。

### 端口

```text
host_port:      8002
container_port: 8080
```

---

## 2. Docker GPU runtime 验证

### 验证命令

```bash
docker run --rm --gpus all \
  --entrypoint nvidia-smi \
  vllm/vllm-openai:latest
```

### 已验证输出摘要

```text
+-----------------------------------------------------------------------------------------+
| NVIDIA-SMI 610.43.02              Driver Version: 610.43.02     CUDA UMD Version: 13.3  |
|-----------------------------------------+------------------------+----------------------+
| GPU  Name                 Persistence-M | Bus-Id          Disp.A | Volatile Uncorr. ECC |
| Fan  Temp   Perf          Pwr:Usage/Cap |           Memory-Usage | GPU-Util  Compute M. |
|                                         |                        |               MIG M. |
|=========================================+========================+======================|
|   0  NVIDIA GeForce RTX 5090 ...    Off |   00000000:01:00.0 Off |                  N/A |
| N/A   45C    P8              7W /  120W |       0MiB /  24463MiB |      0%      Default |
|                                         |                        |                  N/A |
+-----------------------------------------+------------------------+----------------------+
```

**结论**：容器内可看到 NVIDIA GeForce RTX 5090 Laptop GPU，Docker GPU runtime 正常。

---

## 3. llama.cpp CUDA Server 启动命令

### 启动

```bash
docker rm -f qwen35-9b-q4-llama 2>/dev/null || true

docker run -d \
  --name qwen35-9b-q4-llama \
  --gpus all \
  -p 8002:8080 \
  -v "$HOME/models/Qwen3.5-9B-Q4:/models:ro" \
  ghcr.io/ggml-org/llama.cpp:server-cuda13 \
  -m /models/Qwen3.5-9B-Q4_K_M.gguf \
  --host 0.0.0.0 \
  --port 8080 \
  --ctx-size 4096 \
  --n-gpu-layers 999
```

参数说明：

| 参数 | 值 | 说明 |
|------|---|------|
| `--name` | `qwen35-9b-q4-llama` | 容器名 |
| `--gpus` | `all` | 暴露全部 GPU 给容器 |
| `-p` | `8002:8080` | 宿主机 8002 → 容器 8080 |
| `-v` | `$HOME/models/Qwen3.5-9B-Q4:/models:ro` | 模型目录只读挂载 |
| `-m` | `/models/Qwen3.5-9B-Q4_K_M.gguf` | GGUF 模型文件 |
| `--host` | `0.0.0.0` | 监听全部接口 |
| `--port` | `8080` | 容器内端口 |
| `--ctx-size` | `4096` | 上下文长度 |
| `--n-gpu-layers` | `999` | 全部层加载到 GPU |

### 日志

```bash
docker logs -f --tail=200 qwen35-9b-q4-llama
```

### 停止

```bash
docker stop qwen35-9b-q4-llama
```

### 删除容器

```bash
docker rm -f qwen35-9b-q4-llama
```

### 显存观察

```bash
watch -n 1 nvidia-smi
```

---

## 4. `/v1/models` 测试

### 测试命令

```bash
curl http://127.0.0.1:8002/v1/models
```

### 已验证响应特征

```text
- model id: Qwen3.5-9B-Q4_K_M.gguf
- format: gguf
- owned_by: llamacpp
- n_ctx: 4096
- n_ctx_train: 262144
- n_params: 8953803264
- size: 5616076800
```

### 重要说明

llama.cpp server 默认使用 GGUF 文件名作为 OpenAI-compatible model id。

**后续请求建议使用实际返回的 id：`Qwen3.5-9B-Q4_K_M.gguf`**。

---

## 5. `/v1/chat/completions` 测试

### 测试命令

```bash
curl http://127.0.0.1:8002/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Qwen3.5-9B-Q4_K_M.gguf",
    "messages": [
      {"role": "user", "content": "你好，请用一句话介绍你自己。"}
    ],
    "temperature": 0.7,
    "max_tokens": 256
  }'
```

### 已观察到的响应特征

```text
1. response.object = chat.completion
2. response.model = Qwen3.5-9B-Q4_K_M.gguf
3. usage.prompt_tokens 可用
4. usage.completion_tokens 可用
5. usage.total_tokens 可用
6. timings.prompt_per_second 可用
7. timings.predicted_per_second 可用
8. choices[].message 可能包含 reasoning_content
9. choices[].message.content 可能为空（reasoning 模型在思考阶段不输出 content）
10. finish_reason 可能是 length（max_tokens 太小，回复被截断）
```

### 简化响应示例

```json
{
  "object": "chat.completion",
  "model": "Qwen3.5-9B-Q4_K_M.gguf",
  "choices": [
    {
      "finish_reason": "length",
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "",
        "reasoning_content": "Thinking Process: The user is asking me to introduce myself in one sentence..."
      }
    }
  ],
  "usage": {
    "prompt_tokens": 17,
    "completion_tokens": 128,
    "total_tokens": 145
  },
  "timings": {
    "prompt_per_second": 85.9,
    "predicted_per_second": 92.5
  }
}
```

### 注意事项

```text
- Qwen3.5 是 reasoning 模型，可能在 thinking 阶段输出 reasoning_content 而非 content。
- 如果 response.content 为空，检查 reasoning_content。
- 增加 max_tokens 或调整 prompt 可改善回复完整度。
- finish_reason=length 时说明 max_tokens 不够。
```

---

## 6. 对 LightAI 对象的映射

以下对象可直接用于 Phase 1 API 测试。

### ModelArtifact

```json
{
    "name": "qwen35-9b-q4",
    "display_name": "Qwen3.5 9B Q4_K_M (GGUF)",
    "source_type": "local_path",
    "path": "/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf",
    "format": "gguf",
    "task_type": "chat",
    "architecture": "qwen",
    "size_label": "9B",
    "quantization": "Q4_K_M",
    "default_context_length": 4096,
    "estimated_vram_bytes": 8000000000,
    "required_gpu_count": 1
}
```

### RuntimeEnvironment

```json
{
    "name": "llama-cpp-server-cuda13",
    "display_name": "llama.cpp CUDA 13 Server",
    "runtime_type": "docker",
    "backend_type": "llama_cpp",
    "vendor": "nvidia",
    "openai_compatible": true,
    "default_port": 8080,
    "health_check_path": "/health",
    "docker": {
        "image": "ghcr.io/ggml-org/llama.cpp:server-cuda13",
        "image_pull_policy": "never",
        "privileged": { "enabled": false },
        "ipc_mode": { "enabled": false },
        "shm_size": { "enabled": false },
        "gpu_visible_env_key": "CUDA_VISIBLE_DEVICES"
    }
}
```

### RunTemplate

```json
{
    "name": "llama-cpp-gguf-standard",
    "display_name": "llama.cpp GGUF Standard Template",
    "runtime_type": "docker",
    "vendor": "nvidia",
    "backend_type": "llama_cpp",
    "required_variables": ["MODEL_PATH", "GPU_IDS"],
    "optional_variables": ["CTX_SIZE", "N_GPU_LAYERS"],
    "args_template": [
        "-m", "${MODEL_PATH}",
        "--host", "0.0.0.0",
        "--port", "8080",
        "--ctx-size", "${CTX_SIZE}",
        "--n-gpu-layers", "${N_GPU_LAYERS}"
    ],
    "volume_mappings": {
        "enabled": true,
        "value": [
            {
                "host_path": "${MODEL_PATH}",
                "container_path": "${MODEL_PATH}",
                "readonly": true
            }
        ]
    },
    "port_mappings": {
        "enabled": true,
        "value": [
            {
                "host_port": "${HOST_PORT}",
                "container_port": 8080,
                "protocol": "tcp"
            }
        ]
    }
}
```

### ModelDeployment 参数

```json
{
    "host_port": 8002,
    "container_port": 8080,
    "served_model_name": "Qwen3.5-9B-Q4_K_M.gguf",
    "max_model_len": 4096,
    "gpu_memory_utilization": 0.9,
    "schedule_mode": "manual",
    "replicas": 1,
    "env_overrides": {
        "CTX_SIZE": "4096",
        "N_GPU_LAYERS": "999"
    }
}
```

---

## 7. 对 Phase 1 的测试意义

Phase 1 不启动容器，不实现 Agent DockerRuntimeDriver，不实现 Gateway。

但 Phase 1 的 `render-preview` 和 `dry-run` 可以使用本样例，验证：

1. `ModelArtifact` CRUD 可登记 Qwen3.5-9B GGUF 模型；
2. `RuntimeEnvironment` CRUD 可登记 llama.cpp CUDA 13 Docker 环境；
3. `RunTemplate` CRUD 可登记 llama.cpp 标准启动模板；
4. `render-preview` 能正确生成 `ResolvedRunSpec`；
5. `equivalent_command_preview` 能正确表达以下 Docker 命令结构：

```bash
docker run -d \
  --name qwen35-9b-q4-llama \
  --gpus all \
  -p 8002:8080 \
  -v /home/kzeng/models/Qwen3.5-9B-Q4:/models:ro \
  ghcr.io/ggml-org/llama.cpp:server-cuda13 \
  -m /models/Qwen3.5-9B-Q4_K_M.gguf \
  --host 0.0.0.0 \
  --port 8080 \
  --ctx-size 4096 \
  --n-gpu-layers 999
```

6. `dry-run` 能校验：节点在线、GPU 健康、端口可用、模型路径非空、vendor 匹配。

**重要提醒**：

```text
equivalent_command_preview 仅用于展示和排错。
Agent 后续执行必须使用结构化 ResolvedRunSpec。
不得把 command preview 当作执行输入。
```

---

## 8. 对 Phase 2 的测试意义

Phase 2 实现 Agent DockerRuntimeDriver 后，应使用本样例作为本机 NVIDIA 验证用例。

### 验收点

| # | 验收项 | 验证方式 |
|---|--------|---------|
| 1 | Agent 能根据 ResolvedRunSpec 启动 llama.cpp Docker 容器 | `docker ps` 可见容器 |
| 2 | 容器能成功加载 GGUF 模型 | 日志出现 `llama_model_load` 成功 |
| 3 | `/v1/models` 可访问 | `curl http://127.0.0.1:8002/v1/models` 返回模型列表 |
| 4 | `/v1/chat/completions` 可访问 | `curl` 返回有效 JSON |
| 5 | `ModelInstance.actual_state` 变为 `running` | API 查询状态 |
| 6 | `GpuLease` 从 `reserved` 变为 `active` | API 查询 lease 状态 |
| 7 | Stop 后容器停止 | `docker ps` 无该容器 |
| 8 | `GpuLease` 能释放为 `released` | API 查询 lease 状态 |
| 9 | Logs API 能返回 llama.cpp 启动日志 | 日志含 `llama.cpp` 版本信息 |

### Phase 2 测试流程

```text
1. API 登记 ModelArtifact（qwen35-9b-q4）
2. API 登记 RuntimeEnvironment（llama-cpp-server-cuda13）
3. API 登记 RunTemplate（llama-cpp-gguf-standard）
4. API 创建 ModelDeployment（绑定 node/gpu/port）
5. API dry-run → valid=true
6. API start → Agent 启动 Docker 容器
7. 等待 → actual_state=running
8. curl /v1/chat/completions → 验证推理可用
9. API stop → 容器停止 → lease 释放
```

---

## 9. 对 Phase 4 Gateway 的测试意义

后续 Gateway 阶段使用本样例验证时，应注意以下 llama.cpp 特性：

### 响应兼容性

```text
1. llama.cpp 返回的 model id 可能是 GGUF 文件名（如 Qwen3.5-9B-Q4_K_M.gguf），
   与部署配置中的 served_model_name 可能不同；
2. response 中可能包含 reasoning_content 字段；
3. choices[].message.content 可能为空（reasoning 模型特性）；
4. finish_reason 可能是 length（非 stop），意味着 max_tokens 不够；
5. response 中包含 usage（prompt_tokens / completion_tokens / total_tokens）；
6. response 中包含 timings（prompt_per_second / predicted_per_second）；
7. Gateway 应尽量原样透传后端响应，不要丢弃 reasoning_content / timings 等扩展字段；
8. Usage 记录可优先从 usage.prompt_tokens / usage.completion_tokens / usage.total_tokens 提取。
```

### Gateway 测试命令

```bash
# Phase 4 验证：通过 Gateway 代理调用（非直接访问 8002）
curl http://127.0.0.1:18081/v1/chat/completions \
  -H "Authorization: Bearer <api-key>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen35-9b-q4",
    "messages": [{"role": "user", "content": "你好"}],
    "max_tokens": 128
  }'
```

---

## 10. 故障排查

### 常用排查命令

```bash
# 容器状态
docker ps -a | grep qwen35-9b-q4-llama

# 容器日志
docker logs --tail=200 qwen35-9b-q4-llama

# 容器详情
docker inspect qwen35-9b-q4-llama

# GPU 显存
nvidia-smi

# API 测试
curl http://127.0.0.1:8002/v1/models
curl http://127.0.0.1:8002/health
```

### 常见问题

| 问题 | 原因 | 解决 |
|------|------|------|
| `docker --gpus all` 不可用 | NVIDIA Container Toolkit 未安装 | `sudo apt install nvidia-container-toolkit` |
| 镜像拉取失败 | 网络不可达 | 离线环境先 `docker pull` + `docker save` / `docker load` |
| 模型路径不存在 | host volume 路径错误 | 确认 `$HOME/models/Qwen3.5-9B-Q4/` 存在并包含 `.gguf` 文件 |
| `/v1/models` 无响应 | 容器未启动或端口映射错误 | `docker logs` 查看启动日志 |
| `chat completion content` 为空 | Qwen3.5 thinking 阶段只输出 reasoning_content | 检查 `reasoning_content` 字段；增加 `max_tokens` |
| `finish_reason: length` | `max_tokens` 太小 | 增大 `max_tokens`（建议 >= 512） |
| GPU 显存不足 | 模型大于可用显存 | 减少 `--n-gpu-layers`，部分层放到 CPU |
