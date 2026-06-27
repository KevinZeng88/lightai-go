# Runtime Architecture & Parameter Final-State — 文档索引

## 目录标准

本专题所有新增文档、审查结果、执行证据、closeout 统一放在：

```text
docs/reports/runtime-architecture-parameter-final-state/
```

该目录采用专题生命周期命名，不再绑定历史阶段编号。历史报告可以作为输入材料读取；本专题新增输出、执行证据、Codex review、Claude closeout 全部进入本目录。

## 当前目录结构

```text
docs/reports/runtime-architecture-parameter-final-state/
├── 00-index.md
├── 01-execution-policy-and-scope.md
├── 02-current-context-and-known-issues.md
├── 03-final-runtime-domain-contract.md
├── 04-final-parameter-contract.md
├── 04a-parameter-ownership-and-layered-presentation-contract.md
├── 05-runtime-requirements-and-capability-profile-design.md
├── 06-runplan-and-preflight-contract.md
├── 07-ui-and-api-contract.md
├── 08-api-first-e2e-and-automation-requirements.md
├── 09-implementation-plan.md
├── 10-claude-execution-prompt.md
├── 11-final-closeout-template.md
├── 12-codex-review-instructions.md
├── evidence/
├── templates/
└── manifest.json
```

`13-codex-review.md` 由 Codex 审核后生成并提交，不随初始文档包提供。

## 安装方式

把 zip 拷贝到 `/tmp` 后，在项目根目录执行：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
unzip -o /tmp/runtime-architecture-parameter-final-state-docs-revised.zip
find docs/reports/runtime-architecture-parameter-final-state -maxdepth 2 -type f | sort
```

检查 zip 未写入历史阶段目录：

```bash
unzip -l /tmp/runtime-architecture-parameter-final-state-docs-revised.zip | grep 'docs/reports/runtime-architecture-parameter-final-state'
unzip -l /tmp/runtime-architecture-parameter-final-state-docs-revised.zip | grep 'docs/reports/phase-' && exit 1 || true
```

## 阅读顺序

Codex 审核和 Claude 执行前按以下顺序阅读：

1. `01-execution-policy-and-scope.md`
2. `02-current-context-and-known-issues.md`
3. `03-final-runtime-domain-contract.md`
4. `04-final-parameter-contract.md`
5. `04a-parameter-ownership-and-layered-presentation-contract.md`
6. `05-runtime-requirements-and-capability-profile-design.md`
7. `06-runplan-and-preflight-contract.md`
8. `07-ui-and-api-contract.md`
9. `08-api-first-e2e-and-automation-requirements.md`
10. `09-implementation-plan.md`
11. `10-claude-execution-prompt.md`
12. `11-final-closeout-template.md`
13. `12-codex-review-instructions.md`

## 阶段主目标

完成 LightAI Go Runtime 架构、模型元数据、运行能力定义、RuntimeRequirements、BackendCapabilityProfile、参数体系、RunPlan、Preflight、UI/API 行为的最终收敛。

自动化运行是验收要求：用户预设模型、运行配置、节点运行配置、部署参数后，系统通过 API 和状态机自动完成检查、预检、RunPlan 生成、启动、健康检查、日志采集、状态判断和失败归因。

## 本次修订新增硬契约

本次文档修订补充以下硬契约：

1. 参数单一属主：一个参数只属于一个 owner。
2. 参数单一定义：一个参数只有一个 schema 定义位置。
3. 分层快照：每一层创建时 copy-on-create 上一层当时的有效视图。
4. 层级叠加：每一层只新增自己拥有的数据或 override，不重定义上层 schema。
5. 分层展示：每个页面只展示自己拥有或允许覆盖的内容。
6. 最终合成：只有 ResolvedRunPlan 阶段合成全部参数。
7. 来源可见：RunPlan preview 必须显示最终值和来源。
8. checked 语义：default value 不等于 enabled；required 不等于用户 checked；optional 默认不 checked。

## 输出原则

1. 新增文档进入本专题目录。
2. 证据进入 `evidence/`。
3. Codex review 输出为 `13-codex-review.md`。
4. Claude final closeout 按 `11-final-closeout-template.md` 生成。
5. 不新建分支。
6. 不保留历史兼容逻辑。
7. 数据库 schema 变化允许重建数据库。
8. 所有可定位、可修复、可验证的问题在本阶段处理。
9. 无法验证的问题进入 closeout open issues，并说明验证条件。
