# Claude Execution Prompt

请在当前仓库继续执行，不新建分支。

工作目录：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
```

先完整阅读：

```text
docs/reports/runtime-architecture-parameter-final-state/
```

阅读顺序：

1. 00-index.md
2. 01-execution-policy-and-scope.md
3. 02-current-context-and-known-issues.md
4. 03-final-runtime-domain-contract.md
5. 04-final-parameter-contract.md
6. 04a-parameter-ownership-and-layered-presentation-contract.md
7. 05-runtime-requirements-and-capability-profile-design.md
8. 06-runplan-and-preflight-contract.md
9. 07-ui-and-api-contract.md
10. 08-api-first-e2e-and-automation-requirements.md
11. 09-implementation-plan.md
12. 11-final-closeout-template.md
13. 13-codex-review.md，如果该文件已经由 Codex 生成

## 阶段主目标

完成 LightAI Go Runtime 架构、模型元数据、运行能力定义、RuntimeRequirements、BackendCapabilityProfile、参数体系、RunPlan、Preflight、UI/API 行为的最终收敛。

自动化运行、API-first E2E、无人干预是验收要求。

## 核心硬约束

### Runtime 架构

1. NodeBackendRuntime 是唯一部署入口。
2. Deployment 只接受 `node_backend_runtime_id`。
3. Deployment 拒绝 `backend_runtime_id`。
4. 不自动创建 NodeBackendRuntime。
5. Backend / BackendVersion 保持硬件无关。
6. GPU/vendor/hardware 逻辑放在 BackendRuntime / NodeBackendRuntime / DeviceBinding / RunPlan / Agent。
7. 具体模型路径只属于 ModelLocation，不写入通用 catalog / metadata。
8. discovered_metadata_json 不保存本机路径作为通用定义。

### 参数体系

1. 一个参数只有一个 owner。
2. 一个参数只有一个 schema 定义位置。
3. 其他层级只能保存 override value。
4. override 必须引用原始 owner + key 或 definition id。
5. override 不能重新定义 schema。
6. UI 不能为了展示复制 schema。
7. Deployment 可以覆盖最终运行参数，但不能定义 schema。
8. 每一层创建时 copy-on-create 上一层当时有效视图。
9. 每一层只叠加自己拥有的数据或 override。
10. 上层后续修改不反向污染已有下层。
11. 下层修改不反向污染上层。
12. 只有 ResolvedRunPlan 阶段合成全部参数。
13. RunPlan preview 必须显示最终值和来源。

### 参数展示

1. 每个页面只展示自己拥有或允许覆盖的内容。
2. Model 页面只展示模型 metadata、格式、能力、上下文、量化、模型文件信息。
3. Model 页面不展示 Docker args、Docker env、容器镜像、GPU runtime、节点运行环境参数。
4. Backend / BackendVersion 页面展示后端能力和版本能力。
5. BackendRuntime 页面展示运行模板自己的参数和默认运行配置。
6. NodeBackendRuntime 页面展示节点运行环境配置、节点覆盖参数、check-request evidence。
7. Deployment 页面展示部署 override、最终有效参数预览、RunPlan preview。
8. Instance 页面只展示运行事实、状态、日志、健康检查、实际 Docker spec 摘要。
9. Instance 页面不编辑运行参数。

### checked / enabled

1. enabled=true 只表示当前层级显式启用或覆盖。
2. default value 不等于 enabled。
3. required 不等于用户 checked。
4. required/default-applied 参数可以最终生效，但 UI 不显示成用户 checked。
5. optional 参数默认不 checked。
6. advanced 参数默认折叠、不 checked。
7. disabled input 仍显示当前值、默认值或继承值。
8. 未 enabled 的 optional 参数不进入当前层级 override。
9. clone 不扩大 checked 范围。
10. 保存、刷新、clone 后 category、value、enabled、source 不丢失。

## 执行步骤

### 1. Baseline

执行：

```bash
pwd
git status --short
git branch --show-current
git log --oneline -15
find docs/reports/runtime-architecture-parameter-final-state -maxdepth 3 -type f | sort
```

### 2. Reconciliation

读取历史相关文档和当前代码，生成当前差距审查。重点确认：

1. Codex review 中哪些建议必须采纳；
2. 哪些代码问题仍存在；
3. 哪些文档要求已覆盖；
4. 哪些问题必须本阶段修复。

### 3. 按 09-implementation-plan.md 执行

从 Batch 0 到 Batch 7 连续推进。发现可定位、可修复、可验证的问题，直接修复、验证、提交、推送。

## 验收要求

必须通过：

```bash
go test ./internal/server/...
go test ./internal/agent/...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm run build
cd web && npm test
```

必须补充 API-first E2E 证据，尤其是：

1. parameter ownership；
2. copy-on-create；
3. checked/default/required/optional；
4. Deployment override；
5. RunPlan source map；
6. preview 与 Docker spec 一致。

## 输出要求

每个 Batch 输出：

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

最终输出：

```text
Runtime Architecture and Parameter Final-State Report

1. Final status
2. Completed batches
3. Runtime domain contract result
4. Parameter ownership result
5. Copy-on-create result
6. Parameter display result
7. RuntimeRequirements result
8. BackendCapabilityProfile result
9. RunPlan / Preflight result
10. UI/API result
11. API-first E2E evidence
12. Test results
13. Commit list
14. Push result
15. git status
16. Open issues
```

## 禁止事项

1. 不新建分支；
2. 不保留历史兼容逻辑；
3. 不为了旧数据保留复杂 fallback；
4. 不自动创建 NodeBackendRuntime；
5. 不把 GPU/vendor 写入 Backend / BackendVersion；
6. 不把具体模型路径写入通用 metadata/catalog；
7. 不让 RunPlan preview 与实际 Docker spec 分裂；
8. 不把 default value 当 checked；
9. 不让所有参数默认 checked；
10. 不让 Deployment 重新定义 schema；
11. 不让 UI 复制 schema；
12. 不让 Instance 页面编辑运行参数；
13. 不把无法验证的问题静默留在代码或文档外。
