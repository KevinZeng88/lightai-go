# Batch 5 — Security, Tenant, and Runtime Option Governance

## 项目定位

LightAI Go 当前定位是用户 AIDC 内部中小型 GPU 服务器管理平台，面向数台到若干台 GPU 服务器的内部运维、模型部署和模型运行管理场景，不是公网多租户云平台。

本批安全目标是：避免明显误操作、越权、敏感信息泄露；保证 tenant/RBAC 基本边界；保证 Agent 和 Server 通信不被简单串用；保证 Docker 参数可审计、可解释、可测试。不要引入过度复杂的公网云平台安全设计，也不要为了理论安全阻断 NVIDIA、沐曦 / MetaX、华为等厂商模板必需能力。

覆盖：

- R-007
- R-008
- R-009
- R-010
- Q-005
- Agent security review
- tenant/RBAC negative matrix

## Batch 5A — Agent node-bound credentials

当前风险：所有 Agent API 共用全局 bearer token。

目标方案：

- 每个 node/agent 有独立 credential。
- token 与 node_id/agent_id 绑定。
- Agent 请求中的 node_id 与 token subject 不匹配时 401/403。
- task result submission 必须校验 node/task ownership。
- 可保留 bootstrap/global registration token，但注册后必须下发 node-bound token。
- bootstrap token 只用于 registration；heartbeat、task claim/result、docker inspect、file browse、model scan 必须使用 node-bound token。
- 文档说明 bootstrap token 与 node token 生命周期。

建议测试：

```text
internal/server/api/agent_token_binding_test.go
```

## Batch 5B — Docker runtime option governance for AIDC environments

Docker 运行参数采用“模板驱动 + 显式配置 + 基本校验 + 审计记录”的治理策略，而不是一刀切默认拒绝。

原因：真实 GPU 厂商运行环境可能确实需要：

- `devices`
- device mounts
- vendor runtime env
- vendor library mounts
- `/dev/dri`
- `/dev/mxcd`
- `CUDA_VISIBLE_DEVICES`
- 特定 volume
- 特定 security/runtime option
- 特定 network / ipc / shm / ulimit 参数

治理原则：

1. 如果厂商相关 BackendRuntime / NodeBackendRuntime / catalog template / verified runtime template 明确需要这些 Docker 参数，不能被 policy 直接阻止。
2. Docker 参数默认可以在普通自定义场景中关闭或不暴露，但目标运行配置中已经显式打开的参数，RunPlan、preview、dry-run、start 必须能够保留和执行。
3. 策略目标不是把所有高危参数禁掉，而是避免用户无意识随便填任意 host path、arbitrary device、secret env、privileged 等。
4. 对内置厂商模板，应按模板声明的 runtime requirements 放行。
5. 对用户自定义运行配置，应做基本校验、提示和审计记录。
6. 不设计复杂的云平台级 policy engine。
7. 当前只需满足 AIDC 内部运维场景：可用、可解释、可测试、可追踪。
8. 如果某些参数对沐曦 / MetaX / NVIDIA / 华为真实运行必需，测试必须证明它们不会被错误拦截。
9. policy 验收不能只测“危险参数被拒绝”，还必须测“厂商模板需要的 devices / volumes / env 被允许并进入最终 RunPlan / AgentRunSpec / Docker spec”。

目标：

- 保留厂商模板所需 Docker 参数。
- 对普通自定义配置增加基本校验、提示和审计。
- 防止明显错误或无意义的 host path、device、secret env。
- 保证 RunPlan / AgentRunSpec / Docker command preview / real start 一致。
- 不因过度安全策略导致沐曦、NVIDIA、华为等后端不可运行。

新增治理文档：

```text
docs/security/docker-runtime-option-governance.md
```

建议测试：

```text
internal/server/api/docker_runtime_option_governance_test.go
internal/server/runplan/docker_option_preservation_test.go
internal/agent/runtime/docker_spec_governance_test.go
```

验收至少包含：

1. 内置 NVIDIA 模板所需 GPU 参数不被拦截。
2. 内置 MetaX / 沐曦模板所需 `/dev/mxcd`、`/dev/dri`、vendor env、volume 不被拦截。
3. vLLM / SGLang / llama.cpp runtime-specific parameters are preserved。
4. llama.cpp / vLLM / SGLang 真实 smoke 不因 Docker policy 失败。
5. RunPlan preview、dry-run、start 对 Docker 参数判断一致。
6. 非法 host path 或明显错误 device 有清晰错误。
7. 敏感 env 有脱敏和审计，不应在日志/UI 中明文泄露。
8. 如果用户显式打开某项 runtime option，最终 RunPlan 中要么保留，要么给出明确可解释错误，不能静默丢弃。

## Batch 5C — Tenant schema and RBAC negative matrix

修复：

- `gpu_devices tenant_id DEFAULT 'default'`
- handler 中重复 `CREATE TABLE IF NOT EXISTS`
- tenant UUID 初始化不一致

要求：

- fresh DB schema 无 `DEFAULT 'default'`。
- tenant_id 必须显式或引用 default tenant UUID。
- handler 不应偷偷建表；schema 统一在 migration/init。
- fresh DB / rebuild DB 是允许的；如果 schema 改动导致旧 DB 不兼容，应文档说明重建策略，不写复杂迁移兼容逻辑。

新增测试矩阵：

- tenant A 不能 read/write tenant B node/NBR/model/deployment/GPU。
- tenant A 不能通过 aggregate endpoint 看到 tenant B NBR。
- viewer 不能 start/stop/change runtime。
- operator 权限边界明确。
- platform admin 跨 tenant 行为明确。
- Docker runtime option governance 对 tenant/RBAC 的提示、校验、审计一致。

建议文件：

```text
internal/server/api/tenant_negative_matrix_test.go
internal/server/authz/route_policy_test.go
internal/server/db/schema_cleanliness_test.go
```

## 验证命令

```bash
go test ./internal/server/api ./internal/server/auth ./internal/server/authz ./internal/server/db ./internal/server/runplan ./internal/agent/runtime
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
```

涉及 runtime option governance 时，必须运行 llama.cpp / vLLM / SGLang 真实 smoke，除非有命令级外部依赖 blocker 证据。

## 验收

- R-007 CLOSED：node-bound token 可验证，跨 node 复用失败。
- R-008 CLOSED：Docker runtime option governance 完成，厂商模板必需参数不被误拦截，错误自定义配置有清晰错误或提示，敏感 env 脱敏并审计。
- R-009 CLOSED：fresh schema 无非法 tenant default。
- R-010 CLOSED：关键 auth/authz/db/rbac 负向测试存在。
- 安全文档更新，且明确本项目是 AIDC 内部平台，不做云厂商级强隔离。
