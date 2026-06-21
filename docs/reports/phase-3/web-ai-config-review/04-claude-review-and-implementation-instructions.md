# 04 - 给 Claude/Codex 的审阅与实施说明

你现在在 LightAI Go 仓库中继续当前分支工作，不要新建分支。

本轮任务：审阅并逐步实施 Web AI 页面与流程重构。重点是展现方式、页面组织、导航层级、表单结构、i18n、测试入口和诊断体验。

## 绝对约束

1. 不修改数据库 schema。
2. 不新增 migration。
3. 不新增持久化数据结构。
4. 不改变 Backend / BackendVersion / BackendRuntime / NodeBackendRuntime / ModelDeployment / ModelInstance 的核心语义。
5. 不为了页面展示便利修改底层数据模型。
6. 本轮只允许使用现有字段、现有 API、现有 metadata 做展示和组织优化。
7. 如果发现某需求无法在现有字段/API 下实现，必须写入文档作为后续事项，不得擅自改 schema。
8. 不要新建分支。
9. 发现本范围内可修、可测、风险可控的问题，直接修复并验证；但不能越界改数据结构。

## 需要先阅读的文档

请先阅读本目录所有文档：

```text
docs/reports/phase-3/web-ai-config-review/README.md
docs/reports/phase-3/web-ai-config-review/00-current-issues-and-product-goals.md
docs/reports/phase-3/web-ai-config-review/01-web-ai-flow-and-navigation-design.md
docs/reports/phase-3/web-ai-config-review/02-page-configuration-design-no-schema-change.md
docs/reports/phase-3/web-ai-config-review/03-staged-implementation-and-acceptance.md
```

如果这些文档尚未存在，请先创建它们，内容按本任务说明和现有项目实际情况整理。

## 第一步：审阅现状并补充 Review

先不要直接大改代码。先完成审阅并更新文档：

1. 找出现有 Web AI 页面：
   - 模型库
   - 模型部署
   - 模型实例
   - 节点运行配置
   - 推理后端
   - 运行模板
   - 测试弹窗
   - 日志/诊断弹窗
2. 列出每个页面当前展示字段。
3. 列出每个页面当前使用的 API。
4. 判断哪些目标能仅通过展示修改完成。
5. 判断哪些目标需要数据结构支持但本轮不能做。
6. 更新 review 文档。

## 第二步：给出实施计划

在文档中给出阶段计划，至少包括：

1. 导航与页面层级调整。
2. 模型能力展示与测试入口修复。
3. NBR 结构化运行参数页面。
4. 部署页面信息增强。
5. 实例详情中文化和 stopped 列表处理。
6. 测试与验收。

计划中必须明确：

```text
哪些可以本轮实现
哪些只能记录为后续
哪些页面需要修改
哪些测试需要补充
```

## 第三步：经确认后逐步实施

如果执行方已经获得确认，可以按 Phase 1 → Phase 6 逐步执行。

每个 Phase 完成后必须：

1. 运行相关测试。
2. 更新文档。
3. 报告修改文件和验证结果。
4. 再进入下一阶段。

## 本轮重点修复方向

### 1. 导航调整

把 Backend / BackendVersion / BackendRuntime 相关入口移入配置/高级配置区域。

主流程突出：

```text
模型库
运行配置
模型部署
模型实例
测试与诊断
```

### 2. 模型能力展示

使用现有字段/metadata 展示模型能力。

如果已有 capabilities 字段和更新 API，可以提供编辑。

如果没有可持久化字段：

- 不新增 schema。
- 展示自动推断能力。
- 标记为“推断”。
- 文档记录后续能力配置持久化需求。

### 3. 测试入口修复

`Qwen3-0.6B-Instruct-2512` 或名称包含 Instruct/Chat 的模型，不应默认只走 Completion。

默认逻辑：

```text
capabilities 包含 chat → Chat Completion
模型名包含 Instruct/Chat → Chat Completion，标记为推断
completion only → Text Completion
unknown → Auto，允许手动选择
```

测试失败必须显示具体 endpoint 和错误原因。

### 4. NBR 页面结构化

NBR 页面不要以 JSON 快照为主入口。

需要展示：

```text
镜像
命令
args
env
volumes
ports
devices
privileged
ipc
shm_size
ulimits
health check
```

只读/可编辑取决于现有 API 是否支持。

### 5. 部署页面增强

部署页面展示：

```text
模型
后端
后端版本
NBR
镜像
节点
GPU/accelerator
endpoint
状态
最近错误
```

如果现有字段支持 deployment-level extra volumes/env/args，则展示并可编辑；若不支持，记录为后续。

### 6. 实例详情中文化

实例详情必须客户可读：

```text
基础信息
运行信息
资源信息
测试
日志
诊断
```

不得直接展示英文内部字段和 raw JSON。

### 7. stopped 实例处理

模型实例主列表默认不显示 stopped 实例。

failed/exited 保留用于诊断。

audit/log/operation 历史不得丢失。

## 验收命令

完成后执行：

```bash
gofmt -w cmd/ internal/
go test ./internal/server/api/...
go test ./internal/server/runplan/...
go vet ./...
npm --prefix web build
npm --prefix web test
bash -n scripts/e2e/*.sh scripts/e2e/lib/*.sh
git diff --check
git status --short
```

如脚本路径不同，按实际路径调整。

## 提交要求

完成并验收通过后：

```bash
git add .
git commit -m "feat(web): improve AI workflow presentation"
git push
```

如果改动很大，可以拆多个 commit，但最终必须 push，并保持工作区干净。

## 最终报告必须包含

1. Review 文档路径。
2. 是否修改数据结构：必须明确说明没有。
3. 导航调整结果。
4. 哪些页面被移动到配置/高级区域。
5. 模型能力展示逻辑。
6. 能力配置是否基于已有字段实现；若未实现持久化，后续事项是什么。
7. Qwen3 Instruct 默认测试方式。
8. NBR 页面结构化字段。
9. 部署页面新增字段。
10. 实例详情中文化结果。
11. stopped 实例处理结果。
12. 本轮无法实现但已记录的后续事项。
13. 修改文件清单。
14. 测试命令和结果。
15. commit id。
16. push 结果。
17. `git status --short` 是否为空。
