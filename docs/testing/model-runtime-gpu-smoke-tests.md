> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

# LightAI Go Model Runtime GPU Smoke Tests

> 日期：2026-06-17
> 本机验证环境：KZ-LAPTOP, RTX 5090, Ubuntu 24.04 WSL2

## 1. 测试目的

验证以下能力在本地 GPU 环境中可用：

- Docker GPU runtime (nvidia container toolkit)
- NVIDIA GPU 能被容器识别
- vLLM 后端加载本地 HuggingFace 模型并通过 OpenAI-compatible API 推理
- SGLang 后端加载本地 HuggingFace 模型并通过 OpenAI-compatible API 推理
- llama.cpp 后端加载本地 GGUF 模型并通过 OpenAI-compatible API 推理
- `/v1/models` 和 `/v1/chat/completions` 端点可用
- LightAI RunPlan 能生成与真实 Docker 命令一致的结构化执行计划

## 2. 测试环境

| 项目 | 值 |
|------|-----|
| Host | KZ-LAPTOP |
| GPU | NVIDIA GeForce RTX 5090 Laptop GPU, 24,463 MiB |
| Docker | 29.5.3, nvidia runtime available |
| NVIDIA-SMI | 610.43.02, CUDA UMD 13.3 |

### 模型和镜像

| Backend | Image | Model | Host Port | Container Port |
|---------|-------|-------|-----------|----------------|
| vLLM | `vllm/vllm-openai:latest` | `/home/kzeng/models/Qwen3-0.6B-Instruct-2512` | 8004 | 8000 |
| SGLang | `lmsysorg/sglang:latest` | `/home/kzeng/models/Qwen3-0.6B-Instruct-2512` | 30000 | 30000 |
| llama.cpp | `ghcr.io/ggml-org/llama.cpp:server-cuda13` | `/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf` | 8002 | 8080 |

## 3. 前置检查

### 3.1 Docker + NVIDIA 环境

```bash
docker version
docker info | sed -n '1,100p'
nvidia-smi
docker run --rm --gpus all --entrypoint nvidia-smi vllm/vllm-openai:latest
```

### 3.2 模型文件

```bash
ls -la /home/kzeng/models/Qwen3-0.6B-Instruct-2512/
# 必须包含 model.safetensors, config.json, tokenizer.json
ls -l /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
```

### 3.3 镜像

```bash
docker image inspect vllm/vllm-openai:latest >/dev/null 2>&1 || docker pull vllm/vllm-openai:latest
docker image inspect lmsysorg/sglang:latest >/dev/null 2>&1 || docker pull lmsysorg/sglang:latest
docker image inspect ghcr.io/ggml-org/llama.cpp:server-cuda13 >/dev/null 2>&1 || docker pull ghcr.io/ggml-org/llama.cpp:server-cuda13
```

### 3.4 端口

```bash
ss -tlnp | grep -E "8004|30000|8002" && echo "WARNING: port in use" || echo "ports free"
```

## 4. vLLM Smoke Test

### 4.1 启动

```bash
docker rm -f qwen3-06b-vllm 2>/dev/null || true

docker run -d \
  --name qwen3-06b-vllm \
  --gpus all \
  -p 8004:8000 \
  -v /home/kzeng/models/Qwen3-0.6B-Instruct-2512:/models/Qwen3-0.6B-Instruct-2512:ro \
  vllm/vllm-openai:latest \
  --model /models/Qwen3-0.6B-Instruct-2512 \
  --served-model-name Qwen3-0.6B-Instruct-2512 \
  --host 0.0.0.0 \
  --port 8000 \
  --max-model-len 4096 \
  --gpu-memory-utilization 0.6
```

### 4.2 等待就绪

```bash
# vLLM 加载模型需要 60-120s
for i in $(seq 1 60); do
  sleep 2
  CODE=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8004/v1/models)
  [ "$CODE" = "200" ] && echo "ready after $((i*2))s" && break
done
```

### 4.3 API 验证

```bash
# 模型列表
curl -s http://127.0.0.1:8004/v1/models | python3 -m json.tool

# 推理
curl -s http://127.0.0.1:8004/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Qwen3-0.6B-Instruct-2512",
    "messages": [{"role": "user", "content": "你好，请用一句话介绍你自己。"}],
    "temperature": 0.7,
    "max_tokens": 128
  }' | python3 -m json.tool
```

### 4.4 查看日志

```bash
docker logs --tail=50 qwen3-06b-vllm
```

### 4.5 清理

```bash
docker rm -f qwen3-06b-vllm
```

### 4.6 常见问题

| 问题 | 原因 | 解决 |
|------|------|------|
| `--gpus all` 不可用 | nvidia-container-toolkit 未安装 | `sudo apt install nvidia-container-toolkit` |
| OOM | `--gpu-memory-utilization` 太高 | 降低到 0.5-0.6 |
| entrypoint 不匹配 | 新版 vLLM 用 `vllm serve` 不是 `--model` | 检查镜像版本 |

## 5. SGLang Smoke Test

### 5.1 启动

```bash
docker rm -f qwen3-06b-sglang 2>/dev/null || true

docker run -d \
  --name qwen3-06b-sglang \
  --gpus all \
  --shm-size 32g \
  --ipc=host \
  -p 30000:30000 \
  -v /home/kzeng/models/Qwen3-0.6B-Instruct-2512:/models/Qwen3-0.6B-Instruct-2512:ro \
  lmsysorg/sglang:latest \
  python3 -m sglang.launch_server \
    --model-path /models/Qwen3-0.6B-Instruct-2512 \
    --host 0.0.0.0 \
    --port 30000
```

> SGLang 需要 `--shm-size` 足够大（≥16g 建议 32g）和 `--ipc=host`。

### 5.2 等待就绪

```bash
# SGLang 加载需要 30-90s，完成后 /health 返回 200
for i in $(seq 1 60); do
  sleep 2
  CODE=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:30000/health)
  [ "$CODE" = "200" ] && echo "ready after $((i*2))s" && break
done
```

### 5.3 API 验证

```bash
# 获取 SGLang 实际 model id（SGLang 返回的是模型路径，不是 served name）
MODEL_ID=$(curl -s http://127.0.0.1:30000/v1/models | python3 -c "import sys,json; print(json.load(sys.stdin)['data'][0]['id'])")
echo "Model ID: $MODEL_ID"

# 推理（使用实际 model id）
curl -s http://127.0.0.1:30000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d "{
    \"model\": \"$MODEL_ID\",
    \"messages\": [{\"role\": \"user\", \"content\": \"你好，请用一句话介绍你自己。\"}],
    \"temperature\": 0.7,
    \"max_tokens\": 128
  }" | python3 -m json.tool
```

### 5.4 清理

```bash
docker rm -f qwen3-06b-sglang
```

### 5.5 常见问题

| 问题 | 原因 | 解决 |
|------|------|------|
| model id 不对 | SGLang 返回路径而非 served name | 用 `/v1/models` 返回的实际 id |
| `/health` 不返回 200 | SGLang 老版本无此端点 | 等待更久或改用 `/v1/models` 探测 |
| `--shm-size` 太小 | 默认 64MB 不够 | 使用 `--shm-size 32g` |
| CUDA OOM | 模型太大 | 降低 max_tokens 或换更小模型 |

## 6. llama.cpp Smoke Test

参考：`docs/RUNBOOK-LLAMA-CPP-GGUF-NVIDIA-5090.md`

### 6.1 启动

```bash
docker rm -f qwen35-9b-q4-llama 2>/dev/null || true

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

### 6.2 等待就绪

```bash
# llama.cpp 加载较快，10-30s
for i in $(seq 1 30); do
  sleep 2
  CODE=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8002/v1/models)
  [ "$CODE" = "200" ] && echo "ready after $((i*2))s" && break
done
```

### 6.3 API 验证

```bash
curl -s http://127.0.0.1:8002/v1/models | python3 -m json.tool

curl -s http://127.0.0.1:8002/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Qwen3.5-9B-Q4_K_M.gguf",
    "messages": [{"role": "user", "content": "你好，请用一句话介绍你自己。"}],
    "temperature": 0.7,
    "max_tokens": 128
  }' | python3 -m json.tool
```

### 6.4 清理

```bash
docker rm -f qwen35-9b-q4-llama
```

### 6.5 常见问题

| 问题 | 原因 | 解决 |
|------|------|------|
| `content` 为空 | Qwen reasoning 模型返回 `reasoning_content` | 检查 `choices[0].message.reasoning_content` |
| model 名不对 | llama.cpp 使用文件名作为 model id | 用 `/v1/models` 返回的 id |

## 7. 通用 API 验证命令

```bash
# 模型列表
curl -s http://127.0.0.1:<PORT>/v1/models

# 推理
curl -s http://127.0.0.1:<PORT>/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "<MODEL_ID>",
    "messages": [
      {"role": "user", "content": "你好，请用一句话介绍你自己。"}
    ],
    "temperature": 0.7,
    "max_tokens": 128
  }'
```

说明：
- vLLM model 使用 `Qwen3-0.6B-Instruct-2512`
- SGLang model 以 `/v1/models` 实际返回为准（通常是路径）
- llama.cpp model 使用 `Qwen3.5-9B-Q4_K_M.gguf`
- Qwen reasoning 模型可能返回 `reasoning_content`
- `content` 为空不一定失败，检查 `choices`、`usage`、`finish_reason`、`reasoning_content`

## 8. LightAI RunPlan Triple Backend Verification

验证 LightAI RunPlan Resolver 能为三个后端生成结构化执行计划。

### 8.1 运行

```bash
go test ./internal/server/runplan/... -v -run 'TestLlamaCpp|TestResolveBasic' -count=1
```

当前测试 (18 tests total, 0 failures)：
- `TestLlamaCppNvidiaRunPlan` — llama.cpp NVIDIA RunPlan 验证
- `TestLlamaCppRunPlanNoGPU` — CPU-only 模式
- `TestResolveBasic` — vLLM 基本解析
- `TestResolveImagePriority` — 三级 image 解析
- `TestResolveArgs` — args 合并
- `TestResolveEnv` — 6 层 env 合并
- `TestEquivalentCommandPreview` — Docker 命令预览

### 8.2 每个 Backend 验证清单

| 验证项 | vLLM | SGLang | llama.cpp |
|--------|------|--------|-----------|
| image 正确 | ✓ | ✓ | ✓ |
| model host path | ✓ | ✓ | ✓ |
| container model path | ✓ | ✓ | ✓ |
| ports (host:container) | ✓ | ✓ | ✓ |
| mounts (readonly) | ✓ | ✓ | ✓ |
| args 生成 | ✓ | ✓ | ✓ |
| GPU spec | ✓ | ✓ | ✓ |
| health check | ✓ | ✓ | ✓ |
| EquivalentCommandPreview | ✓ | ✓ | ✓ |
| ConvertRunplanToAgentSpec | ✓ | ✓ | ✓ |

> **EquivalentCommandPreview 只用于展示和排错，不能作为 Agent 执行输入。**
> Agent 执行必须使用结构化 ResolvedRunPlan / AgentRunSpec。

### 8.3 当前状态

| 能力 | 状态 |
|------|------|
| RunPlan generation | Completed (18 tests) |
| AgentRunSpec conversion | Completed (2 tests) |
| llama.cpp RunPlan vs real Docker cmd | Verified |
| API → start → Agent → Docker lifecycle | **Pending** |

## 9. 验收标准

每个 smoke test 通过标准：

- [ ] 容器能成功启动 (`docker ps` 可见)
- [ ] `docker logs` 无 fatal error / CUDA error / model load failure
- [ ] `/v1/models` 返回有效 JSON (HTTP 200)
- [ ] `/v1/chat/completions` 返回有效 JSON (HTTP 200)
- [ ] GPU 在 `nvidia-smi` 中可见且有进程
- [ ] 测试结束后容器已清理 (`docker ps -a | grep` 无残留)

## 10. 辅助脚本

```bash
# 环境检查
bash scripts/smoke-model-backends.sh env

# 单独测试
bash scripts/smoke-model-backends.sh vllm
bash scripts/smoke-model-backends.sh sglang
bash scripts/smoke-model-backends.sh llamacpp

# LightAI RunPlan 验证
bash scripts/smoke-model-backends.sh runplan

# 全部测试
bash scripts/smoke-model-backends.sh all

# 清理残留容器
bash scripts/smoke-model-backends.sh cleanup
```
