# Autonomous Claude Execution Prompt

下面内容可直接交给 Claude 执行。

---

你现在在 LightAI Go 项目中执行全项目审计后的修复与优化计划。

仓库路径：

```bash
/home/kzeng/projects/ai-platform-study/lightai-go
```

GitHub 仓库：

```text
https://github.com/KevinZeng88/lightai-go
```

本机 push 后会同步到 GitHub。

审计报告目录：

```bash
docs/reports/codex-project-wide-review-20260625
```

本执行计划目录：

```bash
docs/reports/codex-project-wide-execution-plan-20260625
```

请先阅读：

```bash
docs/reports/codex-project-wide-review-20260625/00-index.md
docs/reports/codex-project-wide-review-20260625/10-risk-register.md
docs/reports/codex-project-wide-review-20260625/11-next-development-recommendations.md
docs/reports/codex-project-wide-review-20260625/13-api-contract-review.md
docs/reports/codex-project-wide-review-20260625/14-runtime-and-runplan-review.md
docs/reports/codex-project-wide-review-20260625/15-frontend-review.md
docs/reports/codex-project-wide-review-20260625/16-agent-docker-review.md
docs/reports/codex-project-wide-review-20260625/17-open-questions.md
docs/reports/codex-project-wide-execution-plan-20260625/00-index.md
docs/reports/codex-project-wide-execution-plan-20260625/01-execution-policy-and-scope.md
docs/reports/codex-project-wide-execution-plan-20260625/02-risk-to-workstream-map.md
docs/reports/codex-project-wide-execution-plan-20260625/03-batch-0-baseline-and-inventory.md
docs/reports/codex-project-wide-execution-plan-20260625/12-validation-matrix.md
docs/reports/codex-project-wide-execution-plan-20260625/15-runtime-smoke-plan.md
docs/reports/codex-project-wide-execution-plan-20260625/16-commit-and-push-strategy.md
```

## 项目定位

LightAI Go 当前定位是用户 AIDC 内部中小型 GPU 服务器管理平台，面向数台到若干台 GPU 服务器的内部运维、模型部署和模型运行管理场景，不是公网多租户云平台。

执行时遵守：

1. 优先保证真实 GPU 后端可运行。
2. 避免明显误操作、越权、敏感信息泄露。
3. 保证 tenant/RBAC 基本边界。
4. 保证 Agent 和 Server 通信不被简单串用。
5. 保证 Docker 参数可审计、可解释、可测试。
6. 不引入过度复杂的公网云平台安全设计。
7. 不为了理论安全而阻断 NVIDIA、沐曦 / MetaX、华为等厂商模板必需能力。
8. 安全策略服务于内部 AIDC 可用性、可维护性和可追踪性，不做云厂商级强隔离。

## 执行授权

AUTORUN_ALLOWED。

Codex 只负责计划审核、计划修订和轻量复审。Claude 负责后续无人值守 AUTORUN 执行。本文件是当前权威执行入口。

除非有命令级证据证明外部依赖不可用、凭据不可用、破坏性数据操作无法安全判断，不要等待人工确认。按批次自主修改代码、测试、脚本、OpenAPI 和文档。每批通过验收后 commit 并 push。不要新建分支。不要使用 `docs/reports/phase-3`。

本机测试环境可用，Claude 必须自行测试，不能要求用户手工启动环境、确认模型路径、确认镜像或确认端口。

已知环境：

```text
KZ-LAPTOP / WSL2 Ubuntu
NVIDIA RTX 5090 Laptop GPU
Docker GPU runtime 可用
已有 llama.cpp / vLLM / SGLang 相关镜像和模型
已有多批 E2E / smoke / runtime evidence
已有自动化环境准备脚本和启动脚本
```

## 最新主线硬约束

本项目当前不需要兼容旧 DB、旧 API、旧 payload、旧脚本、旧运行模板、旧快照。

Claude 执行时必须遵守：

- 可以修改设计和程序后只保留最新版本。
- 不需要保留 legacy fallback。
- 不需要为旧 `backend_runtime_id` 部署 payload 做兼容。
- 不需要为旧 `parameters_json` 做兼容。
- 不需要保留旧 `/runtime-environments`、`/run-templates`、`/model-deployments` 合约。
- stale scripts 应修成最新合约或归档。
- stale docs / evidence 应标注 historical，不能作为当前契约。
- fresh DB / rebuild DB 是允许的。
- 如果 schema 改动导致旧 DB 不兼容，应文档说明重建策略，而不是写复杂迁移兼容逻辑。
- closeout 中不能把“保留兼容路径”当成修复完成。

对于 `/nodes/{id}/backend-runtimes/check`：

- 如果保留 route name，只能作为 server-to-Agent probe wrapper。
- handler 必须忽略 request body readiness evidence。
- 不能保留任何 client-trusted ready 逻辑。
- 如果删除旧 `/check` route 更干净，可以删除，但必须同步 UI、OpenAPI、scripts、tests、docs。
- 默认优先级是：干净设计 > 兼容旧路径。

## 脚本和测试复用硬约束

- 优先复用现有测试、E2E、smoke、启动脚本、环境准备脚本。
- 不得在盘点现有脚本前直接新写 E2E/smoke/start/env 脚本。
- 不得写只在当前会话临时使用的一次性脚本。
- 如果确实需要新增脚本，必须沉淀到项目合适目录，并写明用途、参数、前置条件、运行命令、验收输出和失败处理。
- 如果现有脚本过时，应修复或归档，而不是绕过它新写一份。
- 如果现有脚本使用旧契约，例如 `backend_runtime_id`、`parameters_json`、client-trusted `image_present=true`，应修成当前契约或移入 archive，并从 active E2E 清单中移除。
- 如果已有环境准备脚本可以启动 server、agent、Prometheus、Grafana、Docker/NVIDIA 检查、模型路径检查，应优先复用。
- 如果环境准备脚本不完整，应增强该脚本并沉淀，而不是另写临时命令串。
- 所有真实 smoke、API-first E2E、dry-run、RunPlan preview、OpenAPI contract、tenant/RBAC negative tests 都应尽量接入可重复执行的脚本或测试命令。
- 最终 closeout 不接受“手工执行过但没有脚本/命令记录”的证据。

Batch 0 必须生成：

```bash
docs/reports/codex-project-wide-execution-plan-20260625/runtime-test-and-script-inventory.md
```

## Pre-AUTORUN workspace gate

Claude 开始 Batch 0 前必须生成：

```bash
docs/reports/codex-project-wide-execution-plan-20260625/workspace-baseline.md
```

并记录：

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

## Docker runtime option governance

Docker 运行参数采用“模板驱动 + 显式配置 + 基本校验 + 审计记录”的治理策略，而不是一刀切默认拒绝。

真实 GPU 厂商运行环境可能需要 `devices`、device mounts、vendor runtime env、vendor library mounts、`/dev/dri`、`/dev/mxcd`、`CUDA_VISIBLE_DEVICES`、特定 volume、特定 security/runtime option、特定 network / ipc / shm / ulimit 参数。

执行要求：

- 如果厂商相关 BackendRuntime / NodeBackendRuntime / catalog template / verified runtime template 明确需要这些 Docker 参数，不能被 policy 直接阻止。
- Docker 参数默认可以在普通自定义场景中关闭或不暴露，但目标运行配置中已经显式打开的参数，RunPlan、preview、dry-run、start 必须能够保留和执行。
- 策略目标不是把所有高危参数禁掉，而是避免用户无意识随便填任意 host path、arbitrary device、secret env、privileged 等。
- 对内置厂商模板，应按模板声明的 runtime requirements 放行。
- 对用户自定义运行配置，应做基本校验、提示和审计记录。
- 不设计复杂的云平台级 policy engine。
- 当前只需要满足 AIDC 内部运维场景：可用、可解释、可测试、可追踪。
- 如果某些参数对沐曦 / MetaX / NVIDIA / 华为真实运行必需，测试必须证明它们不会被错误拦截。
- policy 验收不能只测“危险参数被拒绝”，还必须测“厂商模板需要的 devices / volumes / env 被允许并进入最终 RunPlan / AgentRunSpec / Docker spec”。

## Deployment edit runtime selector

所谓 Deployment edit runtime selector，是指部署编辑页面里展示了运行配置 / runtime selector，但当前提交逻辑没有真正提交和生效该字段，导致用户误以为已经切换了部署运行配置。

本项目不允许出现这种“看起来可以操作，但实际没有生效”的功能。

默认处理：

- 如果实现完整 NBR change flow 麻烦，则先简单去除 Deployment edit runtime selector。
- 生成部署运行环境后暂时不允许在普通编辑页修改运行配置。
- 部署创建时选择 NBR / 运行配置。
- 部署创建后，普通编辑页不再提供看似可以切换运行配置的 selector。
- 运行配置可以只读展示，例如 source NBR、BackendRuntime、snapshot 信息。
- 如果后续需要改变运行配置，可以通过删除重建部署，或将来单独实现明确的 NBR change flow。

如果完整 NBR change flow 很简单，也可以实现，但必须满足：

1. 有明确入口，而不是混在普通编辑字段里。
2. API 真正支持修改 source NBR 或创建新的 deployment snapshot。
3. UI 显示当前 NBR 与目标 NBR diff。
4. 修改后 NBR 需要重新 check 或明确继承已验证状态。
5. 明确 snapshot semantics。
6. 明确不会 live mutation 已运行实例。
7. preview / dry-run / start 使用新的最终 RunPlan。
8. API / UI / E2E 测试覆盖。
9. 用户不会误以为已经修改运行中实例。

默认优先级：

```text
先去除误导功能 > 后续单独实现完整 NBR change flow > 保留假功能
```

## 执行顺序

1. Batch 0：建立 inventory、runtime-test-and-script inventory 和当前基线。
2. Batch 1：关闭 client-trusted NBR readiness 和 legacy payload。
3. Batch 2：统一 preflight/dry-run/start 到 final RunPlan。
4. Batch 3：修复 E2E、OpenAPI、current contract docs。
5. Batch 4：修 UI workflow 和 NBR 聚合 endpoint。
6. Batch 5A：Agent node-bound credentials。
7. Batch 5B：Docker runtime option governance for AIDC environments。
8. Batch 5C：tenant/RBAC/schema hardening。
9. Batch 6：reliability、lease、task timeout、logs、observability。
10. Batch 7：performance/scalability cleanup。
11. Batch 8：product scope/gateway/replica/MetaX readiness 边界。
12. Final closeout：风险状态、测试、E2E、runtime smoke、commit/push、git status。

## 全局验收命令

每批默认执行：

```bash
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
```

涉及 E2E / smoke 时必须先盘点并复用现有脚本；如需新增，沉淀到项目目录。真实 runtime smoke 至少覆盖 llama.cpp / vLLM / SGLang。

OpenAPI / active script gate 必须执行：

```bash
rg -n "backend_runtime_id|parameters_json|image_present|docker_available|/backend-runtimes/check" scripts docs/testing
rg -n "/runtime-environments|/run-templates|/model-deployments" docs/api/openapi.yaml && exit 1 || true
rg -n "backend_runtime_id|parameters_json" docs/api/openapi.yaml && exit 1 || true
```

如果 `web/package.json` 已有 Playwright 依赖或相关脚本，必须执行；如果 browser binary 缺失，记录 blocker，并增加 component tests 作为补充。UI P1/P2 修改不能只靠 static string test。

## 状态词

R-001 到 R-015 只允许：

- `CLOSED`
- `CLOSED_BY_SCOPE_REDUCTION`
- `BLOCKED_BY_EXTERNAL_DEPENDENCY`

禁止使用：

- `INTENTIONALLY_DEFERRED_WITH_OWNER_AND_ACCEPTANCE`
- `future`
- `follow-up`
- `later`
- `manual verification later`

`BLOCKED_BY_EXTERNAL_DEPENDENCY` 只有命令级证据证明外部资源不可用时才允许，且必须写明命令、输出、原因、影响、恢复条件。

## 最终输出

最后在终端输出：

1. 所有生成/修改文档。
2. 所有代码/测试/脚本/OpenAPI 改动摘要。
3. R-001 到 R-015 状态表。
4. Q-001 到 Q-008 状态表。
5. 每批 commit id。
6. 每批 push result。
7. 全量验证命令结果。
8. llama.cpp / vLLM / SGLang runtime smoke 结果或真实 external dependency blocker。
9. 剩余问题清单，只允许 `BLOCKED_BY_EXTERNAL_DEPENDENCY`。
10. `git status --short`。
