# Validation Matrix

## 项目定位与测试责任

LightAI Go 当前定位是用户 AIDC 内部中小型 GPU 服务器管理平台，面向数台到若干台 GPU 服务器的内部运维、模型部署和模型运行管理场景，不是公网多租户云平台。

本机测试环境可用，Claude 后续执行必须自行测试。已知环境包括 KZ-LAPTOP / WSL2 Ubuntu、NVIDIA RTX 5090 Laptop GPU、Docker GPU runtime、llama.cpp / vLLM / SGLang 相关镜像和模型、多批 E2E / smoke / runtime evidence、自动化环境准备脚本和启动脚本。

验证策略必须优先保证真实 GPU 后端可运行，避免明显误操作、越权、敏感信息泄露，保证 tenant/RBAC 基本边界和 Agent/Server 通信不被简单串用。不要为了理论安全而阻断 NVIDIA、沐曦 / MetaX、华为等厂商模板必需能力。

## 全局验证

每个代码批次默认执行：

```bash
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
```

如果改动只涉及文档，可跳过代码构建，但最终总验收必须全量执行。

## 分批验证矩阵

| 批次 | 必跑命令 | 额外验证 | 证据 |
| --- | --- | --- | --- |
| Batch 0 | `git status`, inventory `rg/find`, `go test ./... -cover` | runtime-test-and-script inventory | inventory docs |
| Batch 1 | `go test ./internal/server/api ./internal/server/runplan` | false-ready, legacy payload 400 | test output + curl samples |
| Batch 2 | `go test ./internal/server/api ./internal/server/runplan` | preflight/dry-run/start matrix | matrix output |
| Batch 3 | `bash -n scripts/e2e-*`, current API dry-run E2E | OpenAPI sample validation | script logs |
| Batch 4 | `cd web && npm test && npm run build` | Playwright/browser smoke if available | screenshot/log |
| Batch 5 | `go test ./internal/server/api ./internal/server/auth ./internal/server/authz ./internal/server/db ./internal/server/runplan ./internal/agent/runtime` | tenant negative + Docker runtime option governance | test output + real smoke evidence |
| Batch 6 | `go test ./internal/server/api ./internal/agent/runtime` | concurrent start, node offline | test output |
| Batch 7 | `npm run build`, API list tests | pagination/index audit | build log |
| Batch 8 | full global validation | product scope tests | docs + tests |

## Real NVIDIA smoke

本机测试环境可用，真实 runtime smoke 不能默认写成 optional。Claude 不能要求用户手工启动环境、确认模型路径、镜像或端口；必须先盘点并复用已有脚本和测试能力。

本机已知可用条件：

- NVIDIA GPU。
- Docker。
- llama.cpp CUDA image。
- Qwen GGUF 模型。
- vLLM/SGLang images/models 可用于额外 smoke。

推荐 smoke 顺序：

1. llama.cpp GGUF：最小路径，启动快，验证 Docker/device/model mount/logs/stop。
2. vLLM Qwen：验证 OpenAI compatible `/v1/models`、chat/completions fallback。
3. SGLang Qwen：验证不同 backend runtime 参数。

Smoke 必须记录：

- GPU info。
- Docker image inspect。
- model path exists。
- server/agent pid。
- API create/check/preflight/dry-run/start。
- container id。
- `/v1/models`。
- one inference request。
- logs tail。
- stop/cleanup。
- final `docker ps` relevant containers。
- git status。

真实 runtime smoke 至少覆盖：

- llama.cpp
- vLLM
- SGLang

只有命令级证据证明外部资源不可用时，才允许 `BLOCKED_BY_EXTERNAL_DEPENDENCY`。

## Docker runtime option governance 测试

不要只验证 "dangerous options denied"。对 AIDC 内部平台，很多 Docker 参数是运行必需参数，必须正确表达。

### A. 必需运行参数放行测试

至少覆盖：

- NVIDIA template required GPU options are preserved.
- MetaX / 沐曦 template required devices, volumes, env are preserved.
- vLLM / SGLang / llama.cpp runtime-specific parameters are preserved.
- RunPlan preview, dry-run, start produce consistent Docker spec.
- 厂商模板需要的 devices / volumes / env 被允许并进入最终 RunPlan / AgentRunSpec / Docker spec。

### B. 明显错误配置拦截测试

至少覆盖：

- 不存在的 host path 有明确错误。
- 不存在的 device 有明确错误。
- 敏感 env 不明文显示。
- 普通用户随意添加 host root mount 时有 warning 或阻断，具体按当前产品角色设计。
- 同一配置在 save / preview / dry-run / start 的结果一致。
- 如果用户显式打开某项 runtime option，最终 RunPlan 中要么保留，要么给出明确可解释错误，不能静默丢弃。

## Active script stale gate

必须生成：

```bash
docs/testing/active-e2e-scripts.md
```

active scripts 中不得出现旧契约。执行 gate：

```bash
rg -n "backend_runtime_id|parameters_json|image_present|docker_available|/backend-runtimes/check" scripts docs/testing
```

如果 historical/archive 中出现旧契约，必须标注为 historical，不能列为 current validation。

## OpenAPI validation

Batch 3 必须执行：

```bash
rg -n "/runtime-environments|/run-templates|/model-deployments" docs/api/openapi.yaml && exit 1 || true
rg -n "backend_runtime_id|parameters_json" docs/api/openapi.yaml && exit 1 || true
python3 - <<'PY'
import yaml
with open('docs/api/openapi.yaml') as f:
    doc = yaml.safe_load(f)
paths = doc.get('paths', {})
assert '/api/v1/deployments' in paths
assert '/api/v1/deployments/preflight' in paths
PY
```

如果 PyYAML 不可用，Claude 应新增 repo-local validator 或记录明确 external dependency blocker。

## Browser / Playwright

- 如果 `web/package.json` 已有 Playwright 依赖或相关脚本，必须执行。
- 如果 browser binary 缺失，记录 blocker，并增加 component tests 作为补充。
- UI P1/P2 修改不能只靠 static string test。
- Deployment edit runtime selector 相关验证必须覆盖 UI 表单字段与 API payload 一致性。
- 部署编辑页不再出现不会生效的 runtime selector；或者 selector 的变更真实提交、生效、可验证。

## 失败判定

- 测试失败：批次不得 close。
- 构建失败：批次不得 close。
- E2E 脚本由于 contract 错误失败：必须修。
- Docker/GPU 不可用：可以 hardware SKIP，但必须有检测日志。
- 模型路径缺失：可以 SKIP，但应列出预期路径。
- 真实容器启动失败：如果根因在代码/参数，必须修；如果镜像本身不支持当前硬件，记录为 external image compatibility。
- 不能把本机 runtime smoke 默认跳过。只有命令级证据证明 Docker/GPU/image/model/server/agent 等外部资源不可用时才可 `BLOCKED_BY_EXTERNAL_DEPENDENCY`。
