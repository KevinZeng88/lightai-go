# 03 - 分阶段实施建议与验收标准

## 总体实施策略

本轮属于 Web 展示和流程重组，不应一次性大改底层。

实施原则：

1. 先做 review 和页面设计确认。
2. 再做导航和页面重组。
3. 再做关键体验修复。
4. 最后补测试和 closeout。
5. 全程不改数据结构，不新增 migration。

## Phase 0：现状审计与文件沉淀

### 目标

确认当前页面、字段、API、i18n、测试入口的真实状态。

### 任务

1. 创建/更新本目录文档。
2. 全局审计 Web AI 相关页面：
   - 模型库
   - 模型部署
   - 模型实例
   - 节点运行配置
   - 推理后端
   - 运行模板
   - 测试弹窗
   - 日志/诊断弹窗
3. 列出现有可用字段。
4. 标记哪些需求可以仅通过展现修改完成。
5. 标记哪些需求需要后续数据结构支持，但本轮不做。

### 验收

- Review 文档中有当前页面清单。
- 明确本轮“不改数据结构”的边界。
- 明确每个需求是否可用现有字段实现。
- 没有代码变更或只有文档变更。

## Phase 1：导航与流程重组

### 目标

把主流程从内部对象导向，调整为用户任务导向。

### 任务

1. 将 Backend / BackendVersion / BackendRuntime 移入配置/高级配置入口。
2. 主导航突出：
   - 模型库
   - 运行配置
   - 模型部署
   - 模型实例
   - 测试与诊断
3. 不删除原页面，只调整入口和命名。
4. 修复导航 i18n。

### 验收

- 普通主流程能按“模型 → 运行配置 → 部署 → 实例 → 测试”理解。
- 推理后端和运行模板不再占据主流程显眼位置。
- 所有菜单中文正常。
- npm build/test 通过。

## Phase 2：模型能力展示与测试入口修复

### 目标

让模型测试不再默认错误调用 Completion；让模型能力在页面上可见。

### 任务

1. 模型详情展示能力：
   - 优先使用现有 capabilities/metadata 字段。
   - 若无字段，基于 metadata/name 做只读推断展示。
2. 支持能力标签：
   - Chat
   - Completion
   - Embedding
   - Rerank
   - Vision
3. 若已有字段/API 支持编辑，则提供能力配置编辑。
4. 若无可写字段，不新增 schema，只记录后续事项。
5. 测试弹窗支持：
   - Auto
   - Chat Completion
   - Text Completion
6. 默认选择逻辑：
   - capabilities 包含 chat → Chat
   - 模型名包含 Instruct/Chat → Chat，标记为推断
   - completion only → Completion
7. `Qwen3-0.6B-Instruct-2512` 默认走 `/v1/chat/completions`。
8. 错误提示具体化。

### 验收

- Qwen3 Instruct 类模型测试默认不是 Completion。
- UI 能看到模型能力或推断能力。
- 测试失败能看到 endpoint、HTTP 状态或具体原因。
- 不新增 schema。
- 前端测试覆盖默认测试类型选择。

## Phase 3：NBR 结构化运行参数页面

### 目标

把“配置快照 JSON”降级为高级只读诊断，把主要运行参数结构化展示和编辑。

### 任务

1. NBR 编辑页按分区展示：
   - 基础信息
   - 镜像与命令
   - env
   - volumes
   - ports
   - devices
   - 权限
   - 健康检查
   - 预览
   - 高级 JSON
2. 只展示/编辑现有 API 支持的字段。
3. 不支持保存的字段只读或不展示。
4. JSON 快照默认折叠，只读。
5. 提供等价 docker 命令预览，如现有功能支持。

### 验收

- NBR 页面不再只显示镜像和 JSON。
- 客户能看到主要运行参数。
- JSON 不作为主编辑入口。
- 保存后 dry-run/preflight 能体现现有可编辑字段变化。
- 不新增 schema。

## Phase 4：模型部署页面信息增强与 overrides 呈现

### 目标

让部署页面展示完整上下文，并在现有字段支持范围内提供部署级覆盖配置。

### 任务

1. 部署列表新增展示：
   - 后端
   - 后端版本
   - NBR
   - 镜像
   - 节点
   - GPU/accelerator
   - endpoint
   - 最近错误
2. 部署创建/详情按步骤展示：
   - 选择模型
   - 选择运行配置
   - 选择资源
   - 部署级覆盖
   - RunPlan 预览
3. 如果现有字段支持 extra volume/env/args/port override，则暴露。
4. 如果不支持，不新增 schema，在文档记录。
5. RunPlan 预览展示最终合并结果摘要。

### 验收

- 部署页面能看清模型跑在哪个后端、哪个镜像、哪个节点、哪张卡。
- 如已有字段支持，部署时可添加额外卷。
- 如不支持，页面不伪造能力，文档记录后续需求。
- RunPlan 预览可读，不直接扔 raw JSON。

## Phase 5：实例详情中文化与停止后列表语义

### 目标

让模型实例页面对客户友好，停止后不污染 active 列表。

### 任务

1. 实例详情分区：
   - 基础信息
   - 运行信息
   - 资源信息
   - 测试
   - 日志
   - 诊断
2. 所有字段中文化。
3. 状态中文化。
4. Raw JSON 移入高级折叠区。
5. 主列表默认不显示 stopped。
6. 提供筛选项显示 stopped，如实现成本低。
7. failed/exited 保留用于诊断。

### 验收

- 实例详情没有英文内部字段直出。
- 主列表不显示用户主动停止后的 stopped 实例。
- failed/exited 可诊断。
- 日志/audit 历史不丢。

## Phase 6：全链路验收

### 目标

验证 Web AI 主流程可走通。

### 建议最小验收场景

```text
添加/查看 Qwen3-0.6B-Instruct-2512
→ 能看到模型能力或推断能力 Chat
→ 进入运行配置，能看到结构化运行参数
→ 创建部署，能看到后端/镜像/节点/GPU
→ RunPlan 预览可读
→ 启动实例
→ 实例详情中文可读
→ 默认 Chat Completion 测试
→ 停止实例
→ 主列表不再显示 stopped
```

### 验收命令

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

### 最终报告要求

Claude/Codex 最终必须报告：

1. 是否修改数据结构：必须为否。
2. 导航和页面调整清单。
3. 推理后端/运行模板是否已移入配置/高级区域。
4. 模型能力展示/编辑使用了哪些现有字段。
5. 如果能力编辑无法持久化，记录了什么后续事项。
6. Qwen3 Instruct 默认测试方式。
7. NBR 结构化参数页面覆盖哪些字段。
8. Deployment 页面新增展示哪些信息。
9. stopped instance 列表处理方式。
10. i18n 泄露检查结果。
11. 修改文件清单。
12. 测试命令和结果。
13. commit id。
14. push 结果。
15. `git status --short` 是否为空。
