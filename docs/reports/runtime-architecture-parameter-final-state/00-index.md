# Runtime Architecture & Parameter Final-State — 文档索引

## 目录标准

本阶段所有新增文档统一放在：

```text
docs/reports/runtime-architecture-parameter-final-state/
```

该目录采用专题生命周期命名，避免继续绑定历史阶段编号。目录内结构如下：

```text
docs/reports/runtime-architecture-parameter-final-state/
├── 00-index.md
├── 01-execution-policy-and-scope.md
├── 02-current-context-and-known-issues.md
├── 03-final-runtime-domain-contract.md
├── 04-final-parameter-contract.md
├── 05-runtime-requirements-and-capability-profile-design.md
├── 06-runplan-and-preflight-contract.md
├── 07-ui-and-api-contract.md
├── 08-api-first-e2e-and-automation-requirements.md
├── 09-implementation-plan.md
├── 10-claude-execution-prompt.md
├── 11-final-closeout-template.md
├── evidence/
├── templates/
└── manifest.json
```

历史报告如果存在，只作为输入材料读取；本阶段新增文档、执行证据、closeout 全部进入本专题目录。

## 安装方式

把 zip 拷贝到 `/tmp` 后，在项目根目录执行：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
unzip -o /tmp/runtime-architecture-parameter-final-state-docs.zip
find docs/reports/runtime-architecture-parameter-final-state -maxdepth 2 -type f | sort
```

检查目录没有写入历史阶段目录：

```bash
unzip -l /tmp/runtime-architecture-parameter-final-state-docs.zip | grep 'docs/reports/runtime-architecture-parameter-final-state'
unzip -l /tmp/runtime-architecture-parameter-final-state-docs.zip | grep 'docs/reports/phase-' && exit 1 || true
```

## 阅读顺序

Claude 执行前按以下顺序阅读：

1. `01-execution-policy-and-scope.md`
2. `02-current-context-and-known-issues.md`
3. `03-final-runtime-domain-contract.md`
4. `04-final-parameter-contract.md`
5. `05-runtime-requirements-and-capability-profile-design.md`
6. `06-runplan-and-preflight-contract.md`
7. `07-ui-and-api-contract.md`
8. `08-api-first-e2e-and-automation-requirements.md`
9. `09-implementation-plan.md`
10. `10-claude-execution-prompt.md`

## 阶段主目标

完成 LightAI Go Runtime 架构、模型元数据、运行能力定义、参数体系、RunPlan、Preflight、UI/API 行为的最终收敛。

自动化运行是验收要求：用户预设模型、运行配置、节点运行配置、部署参数后，系统应通过 API 和状态机自动完成检查、预检、RunPlan 生成、启动、健康检查、日志采集、状态判断和失败归因。

## 输出原则

1. 新增文档进入 `docs/reports/runtime-architecture-parameter-final-state/`。
2. 证据进入 `docs/reports/runtime-architecture-parameter-final-state/evidence/`。
3. 执行计划、审查结论、closeout 进入同一个专题目录。
4. 不新建分支。
5. 不保留历史兼容逻辑。
6. 不为了旧数据保留复杂 fallback。
7. 数据库 schema 变化允许重建数据库。
8. 所有可定位、可修复、可验证的问题在本阶段处理。
9. 无法验证的问题进入 closeout open issues，并说明验证条件。
