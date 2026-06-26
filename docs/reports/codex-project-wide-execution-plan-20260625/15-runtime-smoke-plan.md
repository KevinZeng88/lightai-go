# Runtime Smoke Plan

## 项目定位

LightAI Go 当前定位是用户 AIDC 内部中小型 GPU 服务器管理平台，面向数台到若干台 GPU 服务器的内部运维、模型部署和模型运行管理场景，不是公网多租户云平台。

真实 runtime smoke 的目标是证明 AIDC 内部 GPU 后端可用、Docker 参数可解释可追踪、Agent/Server 链路可运行，而不是验证公网云平台级强隔离。

## 目标

用本机 NVIDIA + Docker + 已有镜像/模型验证当前主链路，不再只依赖 fake agent 或 dry-run。

本机测试环境可用，Claude 后续执行必须自行测试。不能把本机 runtime smoke 默认写成 optional，不能要求用户手工启动环境，不能要求用户手工确认模型路径、镜像、端口。必须先盘点并复用已有脚本和测试能力；如果环境准备脚本不完整，应增强并沉淀。

## 本机假定资源

根据项目上下文，本机通常具备：

- KZ-LAPTOP / WSL2 Ubuntu。
- NVIDIA GPU。
- NVIDIA RTX 5090 Laptop GPU。
- Docker + NVIDIA runtime。
- llama.cpp CUDA image：
  ```text
  ghcr.io/ggml-org/llama.cpp:server-cuda13
  ```
- vLLM image：
  ```text
  vllm/vllm-openai:latest
  ```
- SGLang image：
  ```text
  lmsysorg/sglang:latest
  ```
- Qwen GGUF 模型：
  ```text
  /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
  ```
- Qwen HF 模型：
  ```text
  /home/kzeng/models/Qwen3-0.6B-Instruct-2512
  ```

执行端必须先检测，不要假设一定存在。

只有命令级证据证明外部资源不可用时，才允许 `BLOCKED_BY_EXTERNAL_DEPENDENCY`。必须写明命令、输出、原因、影响、恢复条件。

## 脚本复用要求

- 优先复用现有测试、E2E、smoke、启动脚本、环境准备脚本。
- 不得在盘点现有脚本前直接新写 E2E/smoke/start/env 脚本。
- 如果现有脚本过时，应修复或归档，而不是绕过它新写一份。
- 如果必须新增统一 smoke 入口，优先命名为 `scripts/e2e-current-runtime-smoke.sh`，并写明用途、参数、前置条件、运行命令、验收输出和失败处理。
- 新增脚本必须可重复运行、参数清晰、失败退出码明确、日志输出路径明确、不依赖人工交互、不依赖临时 shell 状态。

## Docker runtime option governance smoke focus

Smoke 不仅验证危险参数被拦截，也必须验证厂商模板所需 Docker 参数不会被误拦截：

- NVIDIA GPU 参数不被拦截。
- MetaX / 沐曦模板所需 `/dev/mxcd`、`/dev/dri`、vendor env、volume 不被拦截。
- vLLM / SGLang / llama.cpp runtime-specific parameters are preserved。
- RunPlan preview、dry-run、start、AgentRunSpec、Docker spec 对 Docker 参数判断一致。

## Smoke 0 — 环境检查

```bash
nvidia-smi
docker version
docker info | rg -i "runtimes|nvidia"
docker image inspect ghcr.io/ggml-org/llama.cpp:server-cuda13
test -f /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
test -d /home/kzeng/models/Qwen3-0.6B-Instruct-2512
```

## Smoke 1 — llama.cpp 当前 contract

目标：最小真实 Docker 启动链路。

步骤：

1. 使用已有环境准备/启动脚本启动 server/agent，或增强这些脚本；不得要求用户手工启动。
2. 登录获取 session/CSRF。
3. 确认 node online。
4. 确认 BackendRuntime/NBR 存在或创建。
5. 调用 `/check-request`，不得使用 client-trusted `/check` evidence。
6. 创建 deployment，payload 使用 `node_backend_runtime_id`。
7. 调用 final `/deployments/preflight`。
8. 调用 `/deployments/{id}/dry-run`。
9. 调用 `/deployments/{id}/start`。
10. 等待 instance running/healthy。
11. 调用 `/v1/models` 或 backend health endpoint。
12. 发送一次最小 inference。
13. 读取 logs。
14. stop。
15. cleanup。
16. 确认 Docker 无遗留相关 container。
17. 保存 evidence。

证据目录建议：

```text
docs/reports/codex-project-wide-execution-plan-20260625/evidence/runtime-smoke-llamacpp-YYYYMMDDHHMMSS/
```

## Smoke 2 — vLLM 当前 contract

目标：验证 HF 模型、OpenAI-compatible path、resource parameter。

重点：

- `--gpu-memory-utilization`
- `--max-model-len`
- `/v1/models`
- chat/completions 或 fallback completions。
- failed diagnostics if Qwen3/chat path differs。

## Smoke 3 — SGLang 当前 contract

目标：验证 SGLang runtime 参数和 health/logs。

重点：

- `--mem-fraction-static`
- context length。
- port/health。
- logs diagnostics。

## Smoke PASS 标准

每个 smoke 至少记录：

- environment check。
- request payloads。
- responses。
- resolved RunPlan。
- equivalent Docker command preview。
- container ID。
- API health/inference response。
- logs tail。
- stop/cleanup evidence。
- final `docker ps`.
- final `git status --short`.

真实 runtime smoke 至少覆盖 llama.cpp / vLLM / SGLang。

## Smoke SKIP 标准

允许 SKIP 的情况仅限命令级证据证明外部资源不可用：

- Docker daemon unavailable。
- NVIDIA runtime unavailable。
- image not present and network pull not allowed。
- model path missing。
- server/agent cannot bind port due unrelated local process。

SKIP 必须记录检测命令和输出，不得写成 PASS。
