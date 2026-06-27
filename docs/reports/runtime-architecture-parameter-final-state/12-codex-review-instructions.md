# Codex Review Instructions

## 1. 任务定位

请在当前仓库执行一次“文档与代码现实一致性审核”。本轮只做审核文档，不做功能代码修改。

工作目录：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
```

审核对象：

```text
docs/reports/runtime-architecture-parameter-final-state/
```

目标：

1. 审核本专题文档是否足以指导 Claude 执行 Runtime 架构与参数体系最终收敛；
2. 结合当前代码现实，指出文档遗漏、不清晰、冲突、执行风险和验收缺口；
3. 重点审核参数单一属主、单一定义、copy-on-create、分层展示、最终 RunPlan 合成；
4. 生成 review 文档；
5. 提交并推送 review 文档；
6. 不修改功能代码。

## 2. 开始前检查

执行：

```bash
pwd
git status --short
git branch --show-current
git log --oneline -10
find docs/reports/runtime-architecture-parameter-final-state -maxdepth 3 -type f | sort
```

限制：

1. 不新建分支；
2. 不修改功能代码；
3. 不运行长期服务；
4. 不执行破坏性命令；
5. 不启动耗时 E2E；
6. 可以运行轻量 grep/find/git/status；
7. 可以读取代码和历史文档；
8. 可以新增 review 文档；
9. 可以更新 00-index.md 和 manifest.json，仅用于登记 review 文档；
10. 不要在历史阶段目录新增本轮 review。

## 3. 必须阅读

完整阅读：

```text
docs/reports/runtime-architecture-parameter-final-state/00-index.md
docs/reports/runtime-architecture-parameter-final-state/01-execution-policy-and-scope.md
docs/reports/runtime-architecture-parameter-final-state/02-current-context-and-known-issues.md
docs/reports/runtime-architecture-parameter-final-state/03-final-runtime-domain-contract.md
docs/reports/runtime-architecture-parameter-final-state/04-final-parameter-contract.md
docs/reports/runtime-architecture-parameter-final-state/04a-parameter-ownership-and-layered-presentation-contract.md
docs/reports/runtime-architecture-parameter-final-state/05-runtime-requirements-and-capability-profile-design.md
docs/reports/runtime-architecture-parameter-final-state/06-runplan-and-preflight-contract.md
docs/reports/runtime-architecture-parameter-final-state/07-ui-and-api-contract.md
docs/reports/runtime-architecture-parameter-final-state/08-api-first-e2e-and-automation-requirements.md
docs/reports/runtime-architecture-parameter-final-state/09-implementation-plan.md
docs/reports/runtime-architecture-parameter-final-state/10-claude-execution-prompt.md
docs/reports/runtime-architecture-parameter-final-state/11-final-closeout-template.md
docs/reports/runtime-architecture-parameter-final-state/manifest.json
```

历史文档只作为输入材料。若存在以下文件，请重点读取：

```text
docs/reports/phase-3/runtime-architecture-and-parameter-current-gap-review.md
docs/reports/phase-3/runtime-architecture-and-parameter-repair-plan.md
```

## 4. 审核重点

### 4.1 总目标

检查：

1. 文档是否清楚表达 Runtime 架构与参数体系收敛是主目标；
2. 自动化是否作为验收要求；
3. Claude 是否可能误解目标；
4. 执行计划是否可落地；
5. closeout 是否可验证。

### 4.2 目录卫生

检查：

1. 新增输出是否全部进入本专题目录；
2. 是否错误使用历史阶段目录；
3. manifest 是否完整；
4. evidence 目录是否明确；
5. closeout 模板是否完整。

### 4.3 参数硬契约

必须重点检查文档和当前代码是否覆盖：

1. 一个参数只有一个 owner；
2. 一个参数只有一个 schema 定义位置；
3. 其他层级只能保存 override；
4. override 引用原始 owner + key 或 definition id；
5. override 不能重新定义 schema；
6. UI 不能复制 schema；
7. Deployment 不能重新定义 schema；
8. 每一层创建时 copy-on-create 上一层当时有效视图；
9. 每一层只叠加自己拥有的数据或 override；
10. 上层后续修改不污染已有下层；
11. 下层后续修改不污染上层；
12. 只有 ResolvedRunPlan 阶段合成全部参数；
13. RunPlan preview 显示最终值和来源。

### 4.4 参数展示和 checked 语义

必须重点检查：

1. 每个页面只展示自己拥有或允许覆盖的内容；
2. Model 页面不展示 Docker 参数；
3. BackendRuntime / NodeBackendRuntime / Deployment 展示边界清楚；
4. Instance 页面不编辑运行参数；
5. 参数按 category 分组；
6. advanced 默认折叠；
7. default value 不等于 enabled；
8. required 不等于用户 checked；
9. optional 默认不 checked；
10. unchecked optional 不进入 override；
11. clone 不扩大 checked 范围；
12. disabled input 仍显示值。

### 4.5 RuntimeRequirements / BackendCapabilityProfile

检查：

1. 二者职责差异是否清楚；
2. 是否能驱动 Preflight / RunPlan / UI；
3. 是否覆盖 vLLM/SGLang/llama.cpp；
4. 是否覆盖 NVIDIA/MetaX/Huawei 抽象；
5. 是否避免本机模型路径；
6. 是否避免节点/部署状态混入 Backend / BackendVersion。

### 4.6 RunPlan / Preflight

检查：

1. RunPlan 是最终执行权威；
2. preview 与 Docker spec 一致；
3. parameter_source_map；
4. env 不混入 capabilities_json；
5. args 去重；
6. unchecked optional 不进入 args；
7. resource controls；
8. health check 与端口一致；
9. check-request evidence；
10. warning/blocking error。

### 4.7 UI/API

检查文档是否覆盖当前真实问题：

1. RunnerConfigsPage 双入口；
2. legacy Docker editor 与 RuntimeParameterEditor 并存；
3. RuntimeParameterEditor 数据未 populate；
4. watch → emit OOM；
5. 只显示勾选框；
6. 所有参数默认 checked；
7. 保存/刷新/clone 丢失；
8. Deployment 覆盖不足；
9. Instance logs/status；
10. container id 与 instance id 混用。

## 5. 输出文档

生成：

```text
docs/reports/runtime-architecture-parameter-final-state/13-codex-review.md
```

允许最小更新：

```text
docs/reports/runtime-architecture-parameter-final-state/00-index.md
docs/reports/runtime-architecture-parameter-final-state/manifest.json
```

仅用于登记 `13-codex-review.md`。

## 6. Review 文档结构

`13-codex-review.md` 必须使用：

```markdown
# Codex Review — Runtime Architecture and Parameter Final-State Docs

## 1. Review Scope

## 2. Overall Verdict

选择：ACCEPT / ACCEPT_WITH_FIXES / REJECT

## 3. Executive Summary

## 4. Critical Issues

每项包含：Issue / Evidence / Impact / Required Fix

## 5. Missing or Weak Requirements

每项包含：Requirement Area / Current Weakness / Why It Matters / Suggested Fix

## 6. Code-Reality Gaps

每项包含：File or Area / Current Behavior / Expected Final State / Risk / Suggested Requirement

## 7. Parameter Ownership and Copy-on-create Review

必须专门评价单一属主、单一定义、分层快照、override、source map。

## 8. Directory and Documentation Hygiene

## 9. Claude Execution Risk Review

## 10. Required Fixes Before Claude AUTORUN

## 11. Recommended Decision for ChatGPT/User

## 12. Final Status

包含：Review verdict / Review document path / Files changed / Commit id / Push result / git status --short
```

## 7. 提交和推送

完成后执行：

```bash
git status --short
git diff -- docs/reports/runtime-architecture-parameter-final-state
```

确认只修改本专题目录文档后：

```bash
git add docs/reports/runtime-architecture-parameter-final-state
git commit -m "docs: add codex review for runtime architecture parameter plan"
git push
```

提交后：

```bash
git status --short
git log --oneline -5
```

## 8. 最终输出

终端输出：

```text
CODEX_REVIEW_COMPLETED

1. Review verdict:
2. Review document:
3. Files changed:
4. Commit id:
5. Push result:
6. git status --short:
7. Most important findings:
8. Recommendation:
```

不要开始修复功能代码。不要让 Claude 执行。
