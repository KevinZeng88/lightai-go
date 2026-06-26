# Execution Policy and Scope

## 项目定位

LightAI Go 当前定位是用户 AIDC 内部中小型 GPU 服务器管理平台，面向数台到若干台 GPU 服务器的内部运维、模型部署和模型运行管理场景，不是公网多租户云平台。

执行计划必须遵守：

1. 优先保证真实 GPU 后端可运行。
2. 避免明显误操作、越权、敏感信息泄露。
3. 保证 tenant/RBAC 基本边界。
4. 保证 Agent 和 Server 通信不被简单串用。
5. 保证 Docker 参数可审计、可解释、可测试。
6. 不引入过度复杂的公网云平台安全设计。
7. 不为了理论安全而阻断 NVIDIA、沐曦 / MetaX、华为等厂商模板必需能力。
8. 安全策略服务于内部 AIDC 可用性、可维护性和可追踪性，不做云厂商级强隔离。

## 执行模式

本计划面向 Claude 后续无人值守 AUTORUN 执行。Codex 只负责计划审核、计划修订和轻量复审，不作为后续 AUTORUN 执行主体。

当前权威执行入口：

```bash
docs/reports/codex-project-wide-execution-plan-20260625/13-autonomous-claude-execution-prompt.md
```

除非遇到外部依赖缺失、硬件不可用、凭据不可用、破坏性数据操作无法自动确认，Claude 不应等待人工确认。

默认策略：

- 不创建新分支。
- 在当前分支执行。
- 可以修改业务代码、前端代码、测试、脚本、文档、OpenAPI、配置样例。
- 可以新增测试、脚本、文档和证据目录。
- 可以删除或归档明确过时的脚本、旧契约文档和 legacy 入口。
- 每个批次通过验收后提交并 push。
- 每次提交前必须执行本批次验证命令。
- 每次 push 后必须记录 commit id、push result、git status。
- 不保留旧版本兼容逻辑；旧 DB、旧 API、旧 payload、旧脚本、旧运行模板、旧快照应删除、修成最新主线或显式归档为 historical。
- 表结构变化允许按当前主线干净实现；fresh DB / rebuild DB 是允许的。如果 schema 改动导致旧 DB 不兼容，应文档说明重建策略，不写复杂迁移兼容逻辑。
- 干净设计优先于兼容旧路径。
- 本机测试环境可用，执行端必须自行测试，不得要求用户手工启动环境、确认模型路径、确认镜像或确认端口。
- 优先复用现有测试、E2E、smoke、启动脚本、环境准备脚本；不得在盘点现有能力前直接新写 E2E/smoke/start/env 脚本。

## Pre-AUTORUN workspace gate

Claude 开始 Batch 0 前必须生成：

```bash
docs/reports/codex-project-wide-execution-plan-20260625/workspace-baseline.md
```

该文件必须记录以下命令和输出摘要：

```bash
git status --short
git diff --stat
git diff -- web/package.json web/package-lock.json
git log --oneline -30
```

提交规则：

- 每批必须使用 pathspec-limited `git add <explicit files>`。
- 不允许 `git add .`。
- 不允许把 baseline unrelated files 混入批次提交。
- `.mimocode/` 默认不得提交。
- 旧 E2E evidence 目录不得自动提交，除非某批明确将其归档、标记 historical 或纳入 closeout。
- 如果某批必须修改 baseline 已修改文件，例如 `web/package.json` 或 `web/package-lock.json`，closeout 必须说明为什么它变成 in-scope，并展示 before/after diff。
- commit 前如果出现 unexplained path，必须停止该批并记录，不能强行提交。
- 失败批次不得 push partial implementation code。
- 如果只是 GitHub credentials/network 导致 push 失败，应保留本地 commit，记录 `git push` stderr，最终状态为 `BLOCKED_BY_EXTERNAL_DEPENDENCY`。

## 不能做的事

- 不要把本次文档写入 `docs/reports/phase-3`。
- 不要把问题只写成 future/follow-up 却没有 owner、验收和关闭条件。
- 不要只修测试绕过真实问题。
- 不要以 mock/fake agent 结果替代真实 Docker/NVIDIA smoke。
- 不要让 OpenAPI、脚本、文档继续描述旧 contract。
- 不要让 UI 展示后端实际不会保存的字段。
- 不要让 client request body 作为 readiness/security evidence。
- 不要在未记录证据的情况下宣称完成。
- 不要保留旧 `backend_runtime_id` deployment payload 兼容。
- 不要保留旧 `parameters_json` 兼容。
- 不要保留旧 `/runtime-environments`、`/run-templates`、`/model-deployments` 主契约。
- 不要写只在当前会话临时使用的一次性脚本；新增脚本必须沉淀到项目合适目录，并写明用途、参数、前置条件、运行命令、验收输出和失败处理。

## 问题范围

本计划覆盖审计中所有正式问题：

- R-001 到 R-015。
- Q-001 到 Q-008。
- API contract drift。
- RunPlan/preflight drift。
- stale E2E scripts。
- stale OpenAPI。
- frontend misleading workflow。
- Agent security。
- Docker runtime option governance for AIDC environments。
- tenant/RBAC/schema cleanliness。
- reliability/observability/performance/scalability gaps。
- product maturity claim 与真实能力边界。

## 批次完成定义

每个批次必须满足：

1. 代码/文档/测试完成。
2. 对应风险 register 更新。
3. 新增或修复的测试通过。
4. 至少执行：
   ```bash
   go test ./...
   go build ./cmd/server/...
   go build ./cmd/agent/...
   cd web && npm test
   cd web && npm run build
   ```
   如果批次只改文档，可说明为何不跑全量，但最终批次必须全量通过。
5. 如果涉及 runtime，必须执行 API-first dry-run 或真实 Docker smoke。
6. 生成批次 closeout。
7. 使用 pathspec-limited `git add <explicit files>` commit 并 push；禁止 `git add .`。
8. `git status --short` 中不得出现未解释的工作区变更。

本机已知测试环境：

- KZ-LAPTOP / WSL2 Ubuntu。
- NVIDIA RTX 5090 Laptop GPU。
- Docker GPU runtime 可用。
- 已有 llama.cpp / vLLM / SGLang 相关镜像和模型。
- 已有多批 E2E / smoke / runtime evidence。
- 已有自动化环境准备脚本和启动脚本。

真实 runtime smoke 不能默认写成 optional；至少覆盖 llama.cpp / vLLM / SGLang。只有命令级证据证明外部资源不可用时，才允许 `BLOCKED_BY_EXTERNAL_DEPENDENCY`，且必须写明命令、输出、原因、影响、恢复条件。

## 脚本复用原则

- 必须先盘点现有测试、E2E、smoke、启动脚本、环境准备脚本和历史 evidence。
- 如果现有脚本过时，应修复或归档，而不是绕过它新写一份。
- 如果现有脚本使用旧契约，例如 `backend_runtime_id`、`parameters_json`、client-trusted `image_present=true`，应修成当前契约或移入 archive，并从 active E2E 清单中移除。
- 如果已有环境准备脚本可以启动 server、agent、Prometheus、Grafana、Docker/NVIDIA 检查、模型路径检查，应优先复用。
- 如果环境准备脚本不完整，应增强该脚本并沉淀，而不是另写临时命令串。
- 最终 closeout 不接受“手工执行过但没有脚本/命令记录”的证据。

## 失败处理

如果命令失败：

- 先定位根因。
- 能修就立即修。
- 修完重跑。
- 如果失败来自外部条件，例如 Docker daemon 不可用、GPU 不可见、镜像不存在、模型路径不存在、网络无法访问，应写入 `blocked-validation.md`，并提供替代的 dry-run/contract evidence。
- 不允许把可修复失败留给以后。
