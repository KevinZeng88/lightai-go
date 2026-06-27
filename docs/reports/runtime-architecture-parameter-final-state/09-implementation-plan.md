# Implementation Plan

## Batch 0 — Baseline and Reconciliation

### 目标

建立当前代码、文档、测试基线，生成复核文档。

### 执行命令

```bash
pwd
git status --short
git branch --show-current
git log --oneline -15

go version
node -v
npm -v
docker version || true
docker ps || true

find docs -maxdepth 4 -type f | sort
find internal cmd web -type f | sort
```

### 必须读取

```text
docs/reports/runtime-architecture-parameter-final-state/*.md
```

如果历史报告存在，也读取作为输入。

### 输出

```text
docs/reports/runtime-architecture-parameter-final-state/00-existing-docs-and-code-reconciliation.md
```

### 验收

复核文档必须列出：

1. 已读文档；
2. 已查代码路径；
3. 当前 P0/P1/P2 问题；
4. 已解决项；
5. 待修复项；
6. 无法验证项。

## Batch 1 — Domain Contract Alignment

### 目标

让代码中的领域模型与最终契约一致。

### 修改范围

重点检查：

```text
internal/server
internal/agent
cmd/server
cmd/agent
web/src
```

### 重点

1. Backend / BackendVersion 硬件无关；
2. NodeBackendRuntime 是唯一部署入口；
3. ModelArtifact / ModelLocation 边界清楚；
4. RuntimeRequirements 与 CapabilityProfile 字段位置正确；
5. Deployment snapshot 边界清楚；
6. ResolvedRunPlan 作为最终执行权威。

### 验收

```bash
grep -R "backend_runtime_id" -n internal cmd web | head -100
grep -R "/home/kzeng/models" -n internal cmd web docs || true
go test ./internal/server/...
go test ./internal/agent/...
```

## Batch 2 — RuntimeRequirements and CapabilityProfile

### 目标

实现可被 Preflight、RunPlan、UI、E2E 使用的 RuntimeRequirements 和 BackendCapabilityProfile。

### 重点

1. vLLM；
2. SGLang；
3. llama.cpp；
4. NVIDIA；
5. MetaX structure；
6. Huawei extension placeholder；
7. model format；
8. health check；
9. endpoint；
10. resource controls。

### 验收

```bash
go test ./internal/server/... -run 'RuntimeRequirements|Capability|Preflight|RunPlan'
go test ./internal/agent/... -run 'Docker|Image|Device|Runtime'
```

## Batch 3 — Parameter System

### 目标

修复参数 schema/value/enabled/default/override/copy-on-create 全链路。

### 重点

1. BackendRuntime 参数；
2. NodeBackendRuntime 参数；
3. Deployment 参数；
4. schema round-trip；
5. values round-trip；
6. enabled/value 分离；
7. disabled input；
8. clone；
9. refresh；
10. RunPlan binding。

### 验收

```bash
go test ./internal/server/... -run 'Parameter|Runtime|Deployment|RunPlan'
cd web
npm test
npm run build
cd ..
```

## Batch 4 — UI/API Wiring

### 目标

修复用户可见页面和 API 行为。

### 重点

1. RunnerConfigsPage；
2. RuntimeParameterEditor；
3. BackendRuntime page；
4. NodeBackendRuntime page；
5. Deployment page；
6. RunPlan preview；
7. Instance page；
8. Logs page；
9. i18n；
10. status display。

### 验收

```bash
cd web
npm test
npm run build
cd ..
go test ./internal/server/...
```

## Batch 5 — RunPlan and Preflight

### 目标

保证 Preflight 判断和 RunPlan 执行一致。

### 重点

1. RunPlan preview；
2. Docker create spec；
3. source map；
4. errors；
5. warnings；
6. health check；
7. device binding；
8. resource controls；
9. args/env/mounts/ports 去重；
10. Agent execution evidence。

### 验收

```bash
go test ./internal/server/... -run 'RunPlan|Preflight|Deployment'
go test ./internal/agent/... -run 'Docker|Runtime|Device|Health'
```

## Batch 6 — API-first E2E

### 目标

建立自动化验收闭环。

### 重点

1. vLLM full-chain；
2. SGLang full-chain 或失败 evidence；
3. llama.cpp full-chain；
4. negative cases；
5. ready_with_warnings；
6. RunPlan/Docker diff；
7. evidence。

### 验收

```bash
find scripts/e2e -type f -name '*runtime*' | sort
bash scripts/e2e/e2e-runtime-architecture-parameter-full-chain.sh
```

如脚本按后端拆分：

```bash
bash scripts/e2e/e2e-runtime-parameter-vllm.sh
bash scripts/e2e/e2e-runtime-parameter-llamacpp.sh
bash scripts/e2e/e2e-runtime-parameter-sglang.sh
```

## Batch 7 — Cleanup and Closeout

### 目标

清理旧逻辑，完成最终收口。

### 检查

```bash
grep -R "parameters_json" -n internal cmd web docs || true
grep -R "/home/kzeng/models" -n internal cmd web docs || true
grep -R "legacy" -n internal cmd web docs || true
grep -R "TODO" -n internal cmd web docs || true
grep -R "FIXME" -n internal cmd web docs || true
```

### 最终测试

```bash
gofmt -w cmd internal

go test ./internal/server/...
go test ./internal/agent/...
go build ./cmd/server/...
go build ./cmd/agent/...

cd web
npm run build
npm test
cd ..

git status --short
git log --oneline -15
```

### 输出

```text
docs/reports/runtime-architecture-parameter-final-state/07-final-closeout.md
```

## Commit 策略

建议按批次提交：

1. `docs: add runtime architecture parameter final-state plan`
2. `runtime: align requirements and capability profile`
3. `runtime: fix parameter schema and value flow`
4. `web: fix runtime parameter editing and deployment preview`
5. `runtime: align preflight runplan and docker execution`
6. `test: add runtime architecture parameter e2e`
7. `docs: close runtime architecture parameter final-state`

最终 push 后 closeout 记录：

1. commit list；
2. push result；
3. git status；
4. test results；
5. evidence paths。
