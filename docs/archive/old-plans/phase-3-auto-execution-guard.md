> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# LightAI Go Phase 3 自动连续执行守则

> 日期: 2026-06-16
> 适用于: Phase 2-6 自动连续执行

## 1. 适用范围

本守则适用于 Phase 2-6 自动连续执行场景。

Phase 1 + Phase 1.1 已完成并通过验收：
- 旧链路删除 ✓
- 新表创建 ✓
- 内置 Backend/Version seed ✓
- 权限 seed + 角色映射 ✓
- 干净环境 migration 验证 ✓

后续执行目标是自动完成：
- Phase 2: RunPlan Resolver
- Phase 3: Docker Executor
- Phase 4: API
- Phase 5: Web
- Phase 6: E2E 总体验收

## 2. 执行原则

1. **不需要每个 Phase 等待人工确认**。自动连续推进。
2. **每个 Phase 必须先完成实现、测试、修复、报告**，再进入下一 Phase。
3. **验证失败必须自行修复并重新验证**。
4. **不允许跳过失败测试**。
5. **不允许把失败项写成"已知问题"后继续**。
6. **只有符合第 3 节定义的重大问题才允许停止**。

## 3. 重大问题定义

只有以下情况可以停止：

1. 设计文档之间存在无法自行判断的重大矛盾。
2. 需要重构 auth/session/RBAC 核心机制，而不是只追加权限点或接入权限校验。
3. 需要改变 Agent 注册、心跳、GPU 采集主链路。
4. migration 会导致不可恢复的数据破坏，且文档未覆盖处理方式。
5. Docker/GPU 环境完全不可用，且该项属于必须实机验证项。
6. 需要用户提供真实模型文件、GPU 服务器、账号密码等外部条件才能继续。

停止时必须输出：

```text
# 重大问题停止报告

## 重大问题描述
## 已尝试的修复
## 卡住的具体文件 / 命令 / 错误
## 可选方案
## 推荐方案
```

## 4. 非重大问题处理

以下问题**不得停止**，必须自行修复：

1. 编译失败
2. 单元测试失败
3. Web build 失败
4. API 401/403 测试失败
5. tenant 过滤失败
6. roundtrip 测试失败
7. E2E 脚本普通错误
8. 字段名不一致
9. JSON 序列化错误
10. PATCH 丢字段
11. RunPlan hash 不稳定
12. docker_preview 为空
13. 脱敏失败
14. 权限 seed 缺失
15. 旧链路残留

修复策略：定位根因 → 修改代码/测试 → 重新验证 → 记录修复过程。

## 5. Phase 2-6 自动推进规则

### Phase 2 完成条件

- RunPlan Resolver 完成。
- `go test ./internal/server/runplan/... -v -cover` 通过。
- `go test ./...` 通过。
- `go build ./cmd/server/` 通过。
- `go build ./cmd/agent/` 通过。
- `npm --prefix web run build` 通过。
- `git diff --check` 通过。
- 写入 `docs/reports/phase-3/phase-2-report.md`。
- 通过后自动进入 Phase 3。

### Phase 3 完成条件

- DockerExecutor 只消费 ResolvedRunPlan。
- Docker 参数映射测试通过。
- Docker/GPU-only 项如无环境可 SKIP，但必须说明原因和补测命令。
- `go test ./internal/agent/runtime/... -v` 通过。
- `go test ./...` 通过。
- `go build ./cmd/agent/` 通过。
- `go build ./cmd/server/` 通过。
- `git diff --check` 通过。
- 写入 `docs/reports/phase-3/phase-3-report.md`。
- 通过后自动进入 Phase 4。

### Phase 4 完成条件

- 所有 API 完成。
- Auth/RBAC/tenant 过滤完成。
- RunPlan/env/docker_preview 脱敏完成。
- Start 事务顺序完成。
- API roundtrip 脚本完成。
- 401/403/跨租户/脱敏测试完成。
- `go test ./...` 通过。
- `bash test/e2e/model-runtime-api-roundtrip.sh` 通过。
- `git diff --check` 通过。
- 写入 `docs/reports/phase-3/phase-4-report.md`。
- 通过后自动进入 Phase 5。

### Phase 5 完成条件

- Web 新页面完成。
- Web 新 API client 完成。
- Web 权限显示完成。
- RunPlan/docker_preview 默认脱敏。
- Web 保存 roundtrip 完成。
- `npm --prefix web run build` 通过。
- `node web/tests/apiSaveRoundtrip.test.mjs` 通过。
- `go test ./...` 通过。
- `git diff --check` 通过。
- 写入 `docs/reports/phase-3/phase-5-report.md`。
- 通过后自动进入 Phase 6。

### Phase 6 完成条件

- E2E API roundtrip 通过。
- E2E model runtime 通过，Docker/GPU-only 项可按规则 SKIP。
- RBAC 401/403 通过。
- tenant isolation 通过。
- RunPlan preview 通过。
- Web save roundtrip 通过。
- 脱敏测试通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- server build 通过。
- agent build 通过。
- web build 通过。
- `git diff --check` 通过。
- 写入 `docs/reports/phase-3/final-report.md`。

## 6. 最终硬验收

最终必须满足以下 36 项：

| # | 验收项 | 类别 |
|---|--------|------|
| 1 | 旧 runtime_env + run_template 主链路删除 | 删除 |
| 2 | Backend 不含 vendor | 设计 |
| 3 | BackendVersion 含 default_version / is_default / default_backend_params / env / parameter_defs | 设计 |
| 4 | BackendRuntime 含 vendor + image + docker spec | 设计 |
| 5 | NodeRuntimeOverride 可覆盖 image/env/docker/modelRoot | 设计 |
| 6 | Deployment 只引用 backend_runtime_id | 设计 |
| 7 | ResolvedRunPlan 独立落库且不可变 | 设计 |
| 8 | 每次 start/restart 生成新 RunPlan | 行为 |
| 9 | DockerExecutor 只消费 RunPlan | 实现 |
| 10 | 只支持 {{var}} | 实现 |
| 11 | 未知变量 error | 实现 |
| 12 | 不支持 ${VAR} | 实现 |
| 13 | 内置 backends = vllm / sglang / llamacpp | 数据 |
| 14 | 内置 backend versions = 5 个 | 数据 |
| 15 | 内置 runtime templates = 5 个 | 数据 |
| 16 | vllm-metax-docker 表达完整 Docker 参数 | 数据 |
| 17 | llama.cpp NVIDIA 生成 GGUF RunPlan | 行为 |
| 18 | API roundtrip 全通过 | 测试 |
| 19 | Web 保存后刷新不丢失 | 测试 |
| 20 | Web PATCH 后 GET 字段持久化 | 测试 |
| 21 | RunPlan preview docker_preview 非空 | 测试 |
| 22 | Start 事务失败能 rollback | 测试 |
| 23 | GPU lease 创建和释放正确 | 测试 |
| 24 | Auth / RBAC 接入全部新 API | 实现 |
| 25 | 401 / 403 / 跨租户测试通过 | 测试 |
| 26 | RunPlan env 脱敏通过 | 测试 |
| 27 | docker_preview 脱敏通过 | 测试 |
| 28 | go test ./... 通过 | 构建 |
| 29 | go vet ./... 通过 | 构建 |
| 30 | server build 通过 | 构建 |
| 31 | agent build 通过 | 构建 |
| 32 | web build 通过 | 构建 |
| 33 | web tests 通过 | 构建 |
| 34 | E2E 通过或 Docker/GPU-only 项明确 SKIPPED | 测试 |
| 35 | git diff --check 通过 | 质量 |
| 36 | docs/reports/phase-3/final-report.md 存在 | 文档 |

## 7. SKIP 规则

1. Docker 容器启动/停止测试：如无 Docker 环境，SKIP 并注明"需要 Docker 环境，补测命令: docker run ..."。
2. GPU 相关测试：如无 NVIDIA/MetaX GPU，SKIP 并注明硬件要求。
3. 所有 SKIP 项必须在对应 phase 报告中列出原因和补测命令。
4. 非 Docker/GPU 项（API、Web、Resolver、RBAC、脱敏）不得 SKIP。

## 8. 最终输出格式

全部 Phase 完成后只输出：

```text
# Phase 2-6 自动完成最终报告

## 1. 阶段报告路径
## 2. 最终验证命令结果
## 3. E2E 结果
## 4. SKIPPED 项及原因
## 5. 旧链路残留检查
## 6. 权限 / 租户 / 脱敏测试结果
## 7. 当前 git status --short
## 8. 是否建议提交
```
