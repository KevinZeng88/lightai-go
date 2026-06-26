# Batch 1 — Contract and Readiness Hardening

## 目标

关闭最高风险 contract 问题：NBR readiness 不能信任 client request body；旧 deployment payload 不能静默通过；snapshot legacy mutation 收敛。

本项目当前不需要兼容旧 DB、旧 API、旧 payload、旧脚本、旧运行模板、旧快照。Batch 1 的默认优先级是：干净设计 > 兼容旧路径。

覆盖：

- R-001
- R-005 中 legacy branch 的高风险部分
- Q-001
- Q-003
- 与 R-002/R-003 的前置接口改造

## 任务

### 1.1 移除或改造 session `/nodes/{id}/backend-runtimes/check`

当前问题：session 用户可提交 `image_present=true`、`docker_available=true` 影响 NBR ready 状态。

默认方案：

1. 保留路由名但改造成 server-side probe wrapper：
   - 不读取 request body 的 readiness evidence。
   - 只能调用同等于 `/check-request` 的 server-to-agent probe。
   - probe 结果必须来自 Agent `/docker-images`、`/docker-image-inspect` 或更明确的 Agent inspect API。
   - handler 必须忽略 `image_present`、`docker_available` 和任何 request body readiness evidence。
2. 如果 Claude 判断删除旧 `/check` route 更干净，也可以删除，但必须同步 UI、OpenAPI、scripts、tests、docs。

不能保留任何 client-trusted ready 逻辑。

### 1.2 删除 client-trusted 字段路径

检查并删除/拒绝：

- `image_present`
- `docker_available`
- 前端提交的 Docker image readiness evidence
- 任何可以从 request body 直接推导 ready 的字段

服务端写入 `ready`、`ready_with_warnings`、`missing_image`、`needs_check` 等状态时，必须记录 evidence source：

- `agent_image_inspect`
- `agent_docker_unavailable`
- `version_probe_warning`
- `server_validation`
- `manual_disabled`

可写入日志/operation evidence，不一定马上新增 DB 字段；但测试必须能证明 request body 不生效。

### 1.3 明确 deployment payload

修改 `HandleCreateDeployment`、`HandlePatchDeployment`、`HandlePreflightDeployment` 等入口：

- `backend_runtime_id`：在 deployment create/preflight 中继续 400。
- `parameters_json`：改为 400，不允许静默忽略。
- `parameter_values_json`：唯一支持的 structured runtime parameter payload。
- `node_backend_runtime_id`：唯一部署选择字段。

如果 patch/update 允许 partial fields，未知字段策略要一致：危险 legacy 字段必须拒绝。

### 1.4 清理 snapshot legacy branch

定位：

- `runtime_handlers.go` legacy snapshot rebuild branch。
- `db.go` 中会修改 snapshot 的 migration/rebuild path。
- template fallback / backward compatibility comments。

处理原则：

- 能删除就删除。
- fresh DB / rebuild DB 是允许的。
- 如果 schema 改动导致旧 DB 不兼容，应文档说明重建策略，而不是写复杂迁移兼容逻辑。
- 文档写清楚：新主线不支持旧 DB/旧 payload/旧 template 兼容。
- closeout 中不能把“保留兼容路径”当成修复完成。

### 1.5 测试

新增/修复测试：

- session caller POST `/check` with `image_present=true,docker_available=true` cannot set status ready。
- `/check` 或 `/check-request` 在 Agent 缺失镜像时返回 missing_image/needs_check，不 ready。
- `/check-request` 对 `ready_with_warnings` 保持 deployable。
- deployment create with `parameters_json` returns 400。
- deployment create with `backend_runtime_id` returns 400。
- deployment create with `node_backend_runtime_id` works。
- snapshot 不因 legacy fallback 被隐式 rebuild。

建议文件：

```text
internal/server/api/nbr_readiness_contract_test.go
internal/server/api/deployment_payload_contract_test.go
internal/server/api/snapshot_boundary_contract_test.go
```

## 验证命令

```bash
go test ./internal/server/api ./internal/server/runplan
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
```

## 验收

- R-001 CLOSED。
- `/check` 不再信任 request body。
- 旧 payload 明确 400。
- 测试覆盖 false-ready、legacy payload、snapshot boundary。
- UI 如受影响，已改为调用安全路径。
- Batch closeout 记录 commit id、push result、git status。
