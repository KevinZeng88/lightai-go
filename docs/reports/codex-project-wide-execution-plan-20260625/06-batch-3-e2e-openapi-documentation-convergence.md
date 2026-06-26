# Batch 3 — E2E, OpenAPI, and Documentation Convergence

## 目标

让脚本、OpenAPI、当前文档、测试和真实 API contract 一致，避免旧证据继续误导开发。

本项目当前不需要兼容旧 DB、旧 API、旧 payload、旧脚本、旧运行模板、旧快照。stale scripts 应修成最新合约或归档；stale docs / evidence 应标注 historical，不能作为当前契约。

Claude 后续执行必须优先复用现有测试、E2E、smoke、启动脚本、环境准备脚本。不得在盘点现有脚本前直接新写 E2E/smoke/start/env 脚本。

覆盖：

- R-002
- R-006
- Q-006
- Q-007
- code/documentation gap
- test/evidence trust gap

## 任务

### 3.1 修复或归档 stale scripts

先基于 Batch 0 inventory 处理所有脚本。

必须修复：

- 仍发送 `backend_runtime_id` 的 active E2E。
- 仍发送 `parameters_json` 的 active E2E。
- 仍调用 client-trusted `/backend-runtimes/check` 的 active E2E。
- 仍用 `image_present/docker_available` 伪造 readiness 的 active E2E。

修复后 active scripts 应只使用：

- `node_backend_runtime_id`
- `/nodes/{node_id}/backend-runtimes/{nbr_id}/check-request`
- `parameter_values_json`
- final `/deployments/preflight`
- `/deployments/{id}/dry-run`
- `/deployments/{id}/start`

无法立刻修复的脚本：

- 移到 `scripts/archive/legacy-contract/`，或文件头标注 `LEGACY_CONTRACT_DO_NOT_USE_FOR_CURRENT_VALIDATION`。
- 当前 CI/验证命令不得调用它们。

必须生成当前 active 清单：

```bash
docs/testing/active-e2e-scripts.md
```

该清单必须列出每个 active script 的用途、前置条件、运行命令、是否需要 Docker/GPU/模型、输出 evidence 路径和失败处理。

### 3.2 新增当前 E2E contract 文档

生成：

```bash
docs/testing/current-e2e-contract.md
```

内容包括：

- 当前 route。
- 当前 payload。
- forbidden legacy fields。
- required evidence source。
- hardware skip 标准。
- fake agent vs real Docker smoke 的证据边界。
- 如何复现实机 NVIDIA smoke。
- active E2E 清单与 historical/archive 规则。

### 3.3 更新 OpenAPI

更新或重写：

```bash
docs/api/openapi.yaml
```

必须覆盖：

- `/api/v1/backend-runtimes`
- `/api/v1/nodes/{id}/backend-runtimes`
- `/api/v1/nodes/{id}/backend-runtimes/enable`
- 安全版 `/check` 或 `/check-request`
- `/api/v1/deployments`
- `/api/v1/deployments/preflight`
- `/api/v1/deployments/{id}/dry-run`
- `/api/v1/deployments/{id}/start`
- `/api/v1/node-run-plans/{id}`
- `/api/v1/nodes/{id}/model-roots`
- `/api/v1/nodes/{id}/files`
- `/api/v1/nodes/{id}/model-paths/scan`

必须删除或标记 archived：

- `/runtime-environments`
- `/run-templates`
- `/model-deployments`

OpenAPI 不得继续描述旧 `backend_runtime_id` deployment payload 或旧 `parameters_json` runtime parameter payload。

### 3.4 标记历史 evidence

对以下历史 evidence 目录建立 README 或 contract marker：

```bash
docs/reports/model-runtime-node-wizard/e2e-*
```

要求：

- 说明生成时间。
- 使用 contract version。
- 是否仍可作为当前证据。
- 如果包含 `backend_runtime_id` 或 `parameters_json`，必须标记为 historical，不得作为当前通过依据。

### 3.5 修复或新增 API-first E2E

优先修复现有可复用脚本。只有确认没有合适现有入口可修复时，才新增脚本。至少修复或新增一个非 GPU dry-run E2E：

```bash
scripts/e2e-current-contract-api-dryrun.sh
```

至少修复或新增一个 NVIDIA real smoke：

```bash
scripts/e2e-current-contract-nvidia-llamacpp-smoke.sh
```

要求：

- 登录/CSRF 正确。
- 创建/选择 NBR。
- 通过 `/check-request` 验证。
- create deployment 使用 `node_backend_runtime_id`。
- preflight/dry-run/start contract 一致。
- stop/logs/cleanup。
- hardware 缺失时明确 SKIP，不误报 PASS。

新增脚本必须沉淀到项目合适目录，并写明用途、参数、前置条件、运行命令、验收输出和失败处理；不得写只在当前会话临时使用的一次性脚本。

## 验证命令

```bash
bash -n scripts/e2e-current-contract-api-dryrun.sh
bash -n scripts/e2e-current-contract-nvidia-llamacpp-smoke.sh
scripts/e2e-current-contract-api-dryrun.sh
# NVIDIA smoke 在本机可运行时执行
scripts/e2e-current-contract-nvidia-llamacpp-smoke.sh
rg -n "backend_runtime_id|parameters_json|image_present|docker_available|/backend-runtimes/check" scripts docs/testing
rg -n "/runtime-environments|/run-templates|/model-deployments" docs/api/openapi.yaml && exit 1 || true
rg -n "backend_runtime_id|parameters_json" docs/api/openapi.yaml && exit 1 || true

go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
```

## 验收

- R-002 CLOSED。
- R-006 CLOSED。
- `docs/testing/current-e2e-contract.md` 存在。
- OpenAPI 不再描述旧主 contract。
- active scripts 不再发送 deprecated payload。
- `docs/testing/active-e2e-scripts.md` 存在，且 active scripts stale gate 通过。
- 至少一个 API-first dry-run 当前 contract E2E 可运行。
- NVIDIA smoke 能运行或给出真实 hardware skip。
