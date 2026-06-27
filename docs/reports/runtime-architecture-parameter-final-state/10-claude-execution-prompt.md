# Claude Execution Prompt

请在 LightAI Go 仓库中执行 Runtime 架构与参数体系最终收敛任务。

## 1. 项目路径

默认项目路径：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
```

执行前确认：

```bash
pwd
git status --short
git branch --show-current
git log --oneline -15
```

## 2. 本阶段文档目录

本阶段所有新增文档、证据和 closeout 统一写入：

```text
docs/reports/runtime-architecture-parameter-final-state/
```

证据写入：

```text
docs/reports/runtime-architecture-parameter-final-state/evidence/
```

## 3. 本阶段主目标

完成 Runtime 架构、模型元数据、运行能力定义、运行要求定义、参数体系、RunPlan、Preflight、UI/API 行为的最终收敛。

自动化运行是验收要求：用户预设模型、运行配置、节点运行配置、部署参数后，系统通过 API 和状态机自动完成检查、预检、RunPlan 生成、启动、健康检查、日志采集、状态判断和失败归因。

## 4. 必须遵守的架构原则

1. Backend / BackendVersion 硬件无关。
2. GPU vendor、设备文件、Docker runtime、节点差异放在 BackendRuntime、NodeBackendRuntime、Node、Accelerator、DeviceBinding、RunPlan。
3. NodeBackendRuntime 是唯一部署入口。
4. Deployment 只接受 `node_backend_runtime_id`。
5. Deployment 拒绝 `backend_runtime_id`。
6. 不自动创建 NodeBackendRuntime。
7. NodeBackendRuntime 必须显式 enable。
8. check-request 必须通过 Server 代理 Agent 获取真实 evidence。
9. ready 和 ready_with_warnings 可部署。
10. needs_check、missing_image、failed、disabled 不可部署。
11. RunPlan preview 必须与实际 Docker create spec 一致。
12. Preflight 与 RunPlan 使用同一套 RuntimeRequirements 和 BackendCapabilityProfile。
13. 不把具体模型路径写入通用 metadata/catalog。
14. 不把 env、capabilities_json、metadata_json 混入错误字段。
15. 不保留历史兼容逻辑。
16. 不新建分支。

## 5. 先读文档

按顺序读取：

```text
docs/reports/runtime-architecture-parameter-final-state/00-index.md
docs/reports/runtime-architecture-parameter-final-state/01-execution-policy-and-scope.md
docs/reports/runtime-architecture-parameter-final-state/02-current-context-and-known-issues.md
docs/reports/runtime-architecture-parameter-final-state/03-final-runtime-domain-contract.md
docs/reports/runtime-architecture-parameter-final-state/04-final-parameter-contract.md
docs/reports/runtime-architecture-parameter-final-state/05-runtime-requirements-and-capability-profile-design.md
docs/reports/runtime-architecture-parameter-final-state/06-runplan-and-preflight-contract.md
docs/reports/runtime-architecture-parameter-final-state/07-ui-and-api-contract.md
docs/reports/runtime-architecture-parameter-final-state/08-api-first-e2e-and-automation-requirements.md
docs/reports/runtime-architecture-parameter-final-state/09-implementation-plan.md
```

如存在历史报告，读取并核对：

```text
docs/reports/phase-3/runtime-architecture-and-parameter-current-gap-review.md
docs/reports/phase-3/runtime-architecture-and-parameter-repair-plan.md
```

历史报告只作为输入材料。本阶段新输出进入专题目录。

## 6. 执行批次

按以下批次连续执行：

1. Batch 0 — Baseline and Reconciliation；
2. Batch 1 — Domain Contract Alignment；
3. Batch 2 — RuntimeRequirements and CapabilityProfile；
4. Batch 3 — Parameter System；
5. Batch 4 — UI/API Wiring；
6. Batch 5 — RunPlan and Preflight；
7. Batch 6 — API-first E2E；
8. Batch 7 — Cleanup and Closeout。

每个批次完成后输出：

```text
Batch:
Changed files:
Design decisions:
Fixes:
Validation commands:
Validation results:
Evidence path:
Commit id if committed:
Remaining issues:
```

## 7. 必须重点检查的问题

### 7.1 discovered_metadata_json

检查：

```bash
grep -R "discovered_metadata_json" -n internal cmd web docs || true
grep -R "/home/kzeng/models" -n internal cmd web docs || true
```

要求：

1. 模型路径归 ModelLocation；
2. 模型类别 metadata 归 ModelArtifact；
3. 运行能力归 BackendCapabilityProfile；
4. 运行要求归 RuntimeRequirements；
5. 运行结果归 ResolvedRunPlan。

### 7.2 RuntimeRequirements / BackendCapabilityProfile

检查：

```bash
grep -R "RuntimeRequirements" -n internal cmd web docs || true
grep -R "BackendCapabilityProfile" -n internal cmd web docs || true
```

要求：

1. CapabilityProfile 表达后端能力；
2. RuntimeRequirements 表达运行条件；
3. Preflight 使用二者；
4. RunPlan 使用二者；
5. UI 使用二者渲染参数和提示。

### 7.3 参数体系

检查：

```bash
grep -R "parameter_schema_json" -n internal cmd web docs || true
grep -R "parameter_values_json" -n internal cmd web docs || true
grep -R "parameters_json" -n internal cmd web docs || true
```

要求：

1. schema/value 完整保存；
2. enabled/value 分离；
3. disabled input 显示 value；
4. clone 保留 enabled + value；
5. refresh 不丢参数；
6. deployment override 生效；
7. optional 未 enabled 不进入 args；
8. required/default 规则清楚。

### 7.4 UI

重点检查：

1. RunnerConfigsPage；
2. RuntimeParameterEditor；
3. BackendRuntime 页面；
4. NodeBackendRuntime 页面；
5. Deployment 页面；
6. RunPlan preview；
7. Instance 页面；
8. Logs 页面。

### 7.5 RunPlan / Preflight

要求：

1. preview 与 Docker create spec 一致；
2. args 不重复；
3. env 不污染；
4. ports 与 health check 一致；
5. device binding 一致；
6. errors/warnings 可断言；
7. evidence 可复核。

## 8. 验收命令

基础验收：

```bash
go test ./internal/server/...
go test ./internal/agent/...
go build ./cmd/server/...
go build ./cmd/agent/...

cd web
npm run build
npm test
cd ..
```

专题验收：

```bash
go test ./internal/server/... -run 'Runtime|Parameter|RunPlan|Preflight|Deployment'
go test ./internal/agent/... -run 'Docker|Runtime|Device|Health'
```

E2E 验收：

```bash
bash scripts/e2e/e2e-runtime-architecture-parameter-full-chain.sh
```

如果脚本拆分：

```bash
bash scripts/e2e/e2e-runtime-parameter-vllm.sh
bash scripts/e2e/e2e-runtime-parameter-llamacpp.sh
bash scripts/e2e/e2e-runtime-parameter-sglang.sh
```

## 9. 最终 closeout

最终生成：

```text
docs/reports/runtime-architecture-parameter-final-state/07-final-closeout.md
```

closeout 必须包含：

1. Final status；
2. Completed batches；
3. Runtime domain contract result；
4. Parameter contract result；
5. RuntimeRequirements result；
6. BackendCapabilityProfile result；
7. RunPlan / Preflight result；
8. UI/API result；
9. API-first E2E evidence；
10. Test results；
11. Commit list；
12. Push result；
13. git status；
14. Open issues。

## 10. 最终提交

最终执行：

```bash
git status --short
git add .
git commit -m "runtime: align architecture and parameter final state"
git push
git status --short
```

如分多次 commit，在 closeout 中列出全部 commit。
