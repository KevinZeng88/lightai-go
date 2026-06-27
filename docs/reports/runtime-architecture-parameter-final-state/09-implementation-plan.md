# Implementation Plan

## 1. 执行策略

先文档与代码现实复核，再按批次修复。每个批次必须有验证命令、证据和结论。

执行前必须阅读本专题全部文档。Codex 先审核文档并生成 `13-codex-review.md`；用户和 ChatGPT 接受或修订后，Claude 再执行代码修复。

## 2. Batch 0 — Baseline and Reconciliation

目标：建立当前状态基线，确认本专题文档与当前代码现实的差距。

命令：

```bash
pwd
git status --short
git branch --show-current
git log --oneline -15
find docs/reports/runtime-architecture-parameter-final-state -maxdepth 3 -type f | sort
find internal cmd web scripts docs -type f | sort
```

输出：

```text
docs/reports/runtime-architecture-parameter-final-state/evidence/batch-0-baseline-summary.md
```

要求：

1. 读取历史相关文档；
2. 对比本专题要求；
3. 列出必须修复问题；
4. 不直接跳过设计边界问题。

## 3. Batch 1 — Runtime Domain Contract Alignment

目标：落实领域边界。

重点：

1. Backend / BackendVersion 硬件无关；
2. Model metadata 与 ModelLocation 分离；
3. RuntimeRequirements / BackendCapabilityProfile 边界；
4. NodeBackendRuntime 唯一部署入口；
5. Deployment snapshot 边界；
6. Instance 运行事实边界。

验收：

```bash
grep -R "/home/kzeng/models" -n internal cmd web docs || true
grep -R "backend_runtime_id" -n internal cmd web | head -100
go test ./internal/server/...
go test ./internal/agent/...
```

## 4. Batch 2 — RuntimeRequirements and BackendCapabilityProfile

目标：让运行要求和后端能力能驱动 Preflight、RunPlan、UI、E2E。

重点：

1. vLLM capability/requirements；
2. SGLang capability/requirements；
3. llama.cpp capability/requirements；
4. resource controls；
5. health check；
6. model format；
7. device binding abstraction；
8. warning/blocking error。

验收：

```bash
go test ./internal/server/... -run 'RuntimeRequirements|Capability|Preflight|RunPlan'
go test ./internal/agent/... -run 'Docker|Runtime|Device|Health'
```

## 5. Batch 2A — Parameter Ownership and Layered Presentation

目标：落实参数单一属主、单一定义、分层展示、copy-on-create、最终 RunPlan 合成。

重点：

1. ParameterDefinition 只有一个 owner；
2. ParameterOverride 引用 definition；
3. 其他层级不复制 schema；
4. copy-on-create 层级快照链；
5. 每一层只叠加自己这一层数据；
6. default/required/optional/enabled/checked 语义；
7. category 分组；
8. source map；
9. clone 不扩大 checked；
10. Deployment override 不重定义 schema。

验收：

```bash
go test ./internal/server/... -run 'Parameter|Ownership|Override|Snapshot|RunPlan'
cd web && npm test && npm run build
```

必须新增测试断言：

1. default 不导致 enabled；
2. required 不显示成用户 checked；
3. optional 默认不 checked；
4. Deployment override 不复制 schema；
5. copy-on-create 后上下层互不污染；
6. RunPlan source map 存在。

## 6. Batch 3 — Parameter Persistence and API

目标：修复 schema/value/enabled/source 保存、刷新、clone、API 返回。

重点：

1. BackendRuntime 参数保存；
2. NodeBackendRuntime 参数保存；
3. Deployment 参数覆盖；
4. clone 参数复制；
5. refresh 后不丢；
6. API 支持 UI 分层展示；
7. API 支持 source map。

验收：

```bash
go test ./internal/server/... -run 'Parameter|Runtime|Deployment|Snapshot|Clone'
```

## 7. Batch 4 — UI Layered Presentation

目标：修复 UI 分层展示和参数编辑体验。

重点：

1. RunnerConfigsPage；
2. RuntimeParameterEditor；
3. Model 页面；
4. BackendRuntime 页面；
5. NodeBackendRuntime 页面；
6. Deployment 页面；
7. Instance 页面；
8. category 分组；
9. advanced 折叠；
10. disabled input 显示；
11. no OOM；
12. no all checked。

验收：

```bash
cd web
npm test
npm run build
cd ..
```

## 8. Batch 5 — RunPlan and Preflight

目标：RunPlan 成为最终执行权威，Preflight 与 RunPlan 共享规则。

重点：

1. preview 与 Docker spec 一致；
2. parameter_source_map；
3. unchecked optional 不进入 args；
4. resource controls；
5. health check；
6. DeviceBinding；
7. errors/warnings；
8. check-request evidence。

验收：

```bash
go test ./internal/server/... -run 'RunPlan|Preflight|Deployment|ParameterSource'
go test ./internal/agent/... -run 'Docker|Runtime|Device|Health'
```

## 9. Batch 6 — API-first E2E

目标：通过 API-first 自动化证明全链路正确。

至少覆盖：

1. vLLM；
2. SGLang；
3. llama.cpp；
4. parameter ownership；
5. copy-on-create；
6. source map；
7. RunPlan preview/spec consistency；
8. missing image；
9. missing model path；
10. ready_with_warnings。

证据目录：

```text
docs/reports/runtime-architecture-parameter-final-state/evidence/
```

## 10. Batch 7 — Cleanup and Closeout

目标：清理旧逻辑并生成最终 closeout。

检查：

```bash
grep -R "parameters_json" -n internal cmd web docs || true
grep -R "/home/kzeng/models" -n internal cmd web docs || true
grep -R "legacy" -n internal cmd web docs || true
grep -R "TODO" -n internal cmd web docs || true
grep -R "FIXME" -n internal cmd web docs || true
```

最终测试：

```bash
gofmt -w cmd internal
go test ./internal/server/...
go test ./internal/agent/...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm run build
cd web && npm test
```

输出 closeout。

## 11. Commit 策略

1. 文档修订单独提交；
2. Codex review 单独提交；
3. Claude 功能修复按逻辑分批提交；
4. 最终 closeout 单独提交；
5. 所有提交必须 push；
6. closeout 记录 commit list 和 git status。
