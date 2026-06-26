# Batch 0 — Baseline and Inventory

## 目标

建立可执行基线，避免 Codex 被旧文档、旧 evidence、旧脚本误导。

本机测试环境可用，Claude 后续执行必须自行测试。正式修复前必须先盘点现有测试、启动、环境准备、E2E、smoke 和历史证据能力，并优先复用现有资产。

## 任务

### 0.1 创建执行目录

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
mkdir -p docs/reports/codex-project-wide-execution-plan-20260625
```

将本计划文档复制到该目录。

### 0.2 记录当前 Git 状态

Claude 开始 Batch 0 前必须生成：

```bash
docs/reports/codex-project-wide-execution-plan-20260625/workspace-baseline.md
```

记录：

```bash
git status --short
git diff --stat
git diff -- web/package.json web/package-lock.json
git branch --show-current
git log --oneline -30
git remote -v
```

要求：

- 不自动清理用户已有未跟踪文件。
- 对已有 `web/package*.json` 修改、`.mimocode/`、E2E evidence 目录做来源说明。
- 如果后续需要修改这些文件，必须在 closeout 中说明是既有变更还是本批变更。
- 每批必须使用 pathspec-limited `git add <explicit files>`；不允许 `git add .`。
- 不允许把 baseline unrelated files 混入批次提交。
- `.mimocode/` 默认不得提交。
- 旧 E2E evidence 目录不得自动提交，除非某批明确将其归档、标记 historical 或纳入 closeout。
- 如果某批必须修改 baseline 已修改文件，例如 `web/package.json` 或 `web/package-lock.json`，closeout 必须说明为什么它变成 in-scope，并展示 before/after diff。
- commit 前如果出现 unexplained path，必须停止该批并记录，不能强行提交。

### 0.3 建立脚本库存

生成：

```bash
docs/reports/codex-project-wide-execution-plan-20260625/inventory-active-scripts.md
docs/reports/codex-project-wide-execution-plan-20260625/inventory-stale-scripts.md
```

检查：

```bash
find scripts -maxdepth 3 -type f | sort
rg -n "backend_runtime_id|parameters_json|image_present|docker_available|/backend-runtimes/check|/check-request|node_backend_runtime_id|parameter_values_json" scripts
```

分类：

- active-current：当前 contract，可运行。
- active-needs-repair：应修复。
- archive-stale：旧 contract，不应再作为当前证据。
- hardware-only：需要 NVIDIA/Docker/模型。
- reference-only：历史参考，不可用于验收。

### 0.4 建立 API route 库存

```bash
rg -n "Handle[A-Za-z0-9_]+|/api/v1|PathValue|backend-runtimes|deployments|preflight|dry-run|start|node-run-plans" cmd internal/server
```

输出：

```bash
docs/reports/codex-project-wide-execution-plan-20260625/inventory-current-api-routes.md
```

### 0.5 建立测试库存

```bash
find internal -name '*_test.go' | sort
find web -maxdepth 4 -type f \( -name '*test*' -o -name '*.spec.*' \) | sort
go test ./... -cover
```

输出：

```bash
docs/reports/codex-project-wide-execution-plan-20260625/inventory-test-coverage.md
```

### 0.6 建立文档库存

```bash
find docs -maxdepth 5 -type f | sort
rg -n "backend_runtime_id|parameters_json|runtime-environments|run-templates|model-deployments|phase-3|NodeBackendRuntime|RunPlan|check-request|OpenAPI" docs
```

输出：

```bash
docs/reports/codex-project-wide-execution-plan-20260625/inventory-documentation-drift.md
```

### 0.7 建立 runtime/test/script 能力库存

正式修复前必须先盘点现有测试、启动、环境准备、E2E、smoke 和历史证据能力。

建议盘点命令：

```bash
find scripts -maxdepth 4 -type f | sort
find . -maxdepth 4 -type f \( -name '*e2e*' -o -name '*smoke*' -o -name '*start*' -o -name '*prepare*' -o -name '*env*' -o -name '*bootstrap*' -o -name '*setup*' \) | sort
find docs/reports -maxdepth 5 -type f \( -name '*.md' -o -name '*.log' -o -name '*.json' -o -name '*.txt' \) | sort
find internal -name '*_test.go' | sort
find web -maxdepth 4 -type f \( -name '*test*' -o -name '*spec*' \) | sort
rg -n "start-all|server|agent|LIGHTAI|/tmp/lightai|docker|nvidia|CUDA|llama|vllm|sglang|check-request|dry-run|preflight|RunPlan|node_backend_runtime_id|parameter_values_json|backend_runtime_id|parameters_json|image_present" scripts docs internal web cmd
```

输出：

```bash
docs/reports/codex-project-wide-execution-plan-20260625/runtime-test-and-script-inventory.md
```

表格至少包含：

| Category | Existing Asset | Current / Stale / Unknown | Purpose | Reuse Plan | Required Fix |
| -------- | -------------- | ------------------------- | ------- | ---------- | ------------ |

分类至少包括：

- 环境准备脚本
- server/agent 启动脚本
- Docker/NVIDIA 检查脚本
- API-first E2E
- 真实 Docker smoke
- llama.cpp 测试
- vLLM 测试
- SGLang 测试
- RunPlan / dry-run 测试
- OpenAPI / contract 测试
- tenant/RBAC 测试
- frontend static/unit 测试
- Playwright/browser smoke
- 历史 evidence 目录

如果必须新增统一 smoke 入口，只能在确认没有合适现有入口可修复时新增。建议命名：

```bash
scripts/e2e-current-runtime-smoke.sh
```

该脚本必须可重复运行、参数清晰、失败退出码明确、日志输出路径明确、不依赖人工交互、不依赖临时 shell 状态，并纳入 validation matrix。

## 边界

本批原则上只生成 inventory 文档，不修改业务代码。  
只有发现明显会阻碍后续批次的本地环境问题时，才允许补充脚本说明或新增验证说明。

## 验收

- 5 份 inventory 文档已生成，包括 `runtime-test-and-script-inventory.md`。
- `workspace-baseline.md` 已生成，并记录 `git status --short`、`git diff --stat`、`git diff -- web/package.json web/package-lock.json`、`git log --oneline -30`。
- 明确列出 active scripts 与 stale scripts。
- 明确列出当前 route 与 OpenAPI 差异。
- 明确列出低覆盖区域。
- 明确列出现有环境准备、server/agent 启动、Docker/NVIDIA 检查、API-first E2E、真实 smoke、Playwright/browser smoke 和历史 evidence 的复用计划。
- `git status --short` 已记录。
