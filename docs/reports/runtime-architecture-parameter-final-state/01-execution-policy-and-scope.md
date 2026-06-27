# Execution Policy and Scope

## 1. 执行定位

本专题用于指导 Runtime 架构与参数体系最终收敛。执行端需要以当前代码和本专题文档为依据，完成文档复核、代码修复、API-first 验收、证据沉淀、最终 closeout。

主线目标包括：

1. Runtime 领域模型边界收敛；
2. 模型 metadata 与模型实例路径边界收敛；
3. RuntimeRequirements 与 BackendCapabilityProfile 定义收敛；
4. 参数体系收敛；
5. RunPlan / Preflight 收敛；
6. UI/API 行为收敛；
7. 自动化验收链路收敛。

自动化运行是验收要求。它用于证明架构和参数体系已经形成可执行闭环。

## 2. 工作目录

仓库路径：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
```

专题目录：

```text
docs/reports/runtime-architecture-parameter-final-state/
```

历史报告可作为输入材料读取。新增审查、计划、证据、closeout 不写入历史阶段目录。

## 3. 分工策略

### 3.1 ChatGPT

负责：

1. 制定目标和设计文档；
2. 审核 Codex review；
3. 判断哪些建议接受、部分接受或驳回；
4. 生成最终 Claude 执行口径；
5. 协助用户验收结果。

### 3.2 Codex

负责：

1. 审核本专题文档和当前代码现实是否一致；
2. 输出 `13-codex-review.md`；
3. 提交并推送 review 文档；
4. 不修改功能代码。

### 3.3 Claude

负责：

1. 按最终文档执行代码修复；
2. 运行测试和 E2E；
3. 生成证据；
4. 提交并推送；
5. 输出 final closeout。

## 4. 非兼容策略

1. 不保留旧配置兼容逻辑。
2. 不保留旧模板 fallback。
3. 不为了旧 DB 数据保留复杂迁移分支。
4. 表结构变化允许重建数据库。
5. 旧字段、旧接口、旧流程如果与最终设计冲突，应删除。
6. 发现真实问题时优先修复、验证、提交、推送。

## 5. 架构硬约束

### 5.1 NodeBackendRuntime 部署入口

1. Deployment 只接受 `node_backend_runtime_id`。
2. Deployment 拒绝 `backend_runtime_id`。
3. 系统不自动创建 NodeBackendRuntime。
4. NodeBackendRuntime 必须显式 enable。
5. check-request 必须由 Server 代理 Agent 获取 evidence。
6. ready 和 ready_with_warnings 可部署。
7. needs_check、missing_image、failed、disabled 不可部署。
8. UI 与 API 对可部署状态保持一致。

### 5.2 Backend / BackendVersion 硬件无关

1. Backend / BackendVersion 表达推理后端和后端版本能力。
2. GPU vendor、设备文件、Docker runtime、硬件绑定逻辑不写入 Backend / BackendVersion。
3. GPU/vendor/hardware 相关内容放入 BackendRuntime、NodeBackendRuntime、Node、Accelerator、DeviceBinding、RunPlan。

### 5.3 模型 metadata 边界

1. 模型 metadata 描述模型本身。
2. 模型实例路径归 ModelLocation。
3. 通用 catalog、Backend、BackendVersion、RuntimeRequirements、BackendCapabilityProfile 不写入 `/home/kzeng/models/...` 这类本机路径。
4. discovered_metadata_json 不沉淀本机具体模型路径为通用定义。

## 6. 参数体系硬约束

### 6.1 单一属主

1. 一个参数只能有一个 owner。
2. owner 可以是 Model / ModelArtifact、Backend / BackendVersion、BackendRuntime、NodeBackendRuntime、Deployment 或系统生成项。
3. owner 决定 schema 定义位置。
4. 其他层级不能重新定义该参数 schema。

### 6.2 单一定义

1. 一个参数只有一个 schema 定义位置。
2. UI 展示不能复制 schema。
3. Deployment 覆盖不能复制 schema。
4. NodeBackendRuntime 覆盖不能复制 schema。
5. 克隆对象时不能扩大 schema 所属范围。

### 6.3 分层快照和 copy-on-create

1. 每一层创建时拷贝上一层当时的有效视图。
2. 每一层在快照基础上叠加自己拥有的数据或 override。
3. 后续上一层变更不反向污染已经创建的下一层。
4. 下一层后续变更不反向污染上一层。
5. copy-on-create 保存的是快照和 override 边界，不是把所有 schema 改成当前层 owner。

### 6.4 分层展示

1. 每个页面只展示自己拥有或允许覆盖的内容。
2. Model 页面只展示模型 metadata、格式、能力、上下文、量化、模型文件信息。
3. Backend / BackendVersion 页面只展示后端能力和版本能力。
4. BackendRuntime 页面只展示运行模板自己的参数和默认运行配置。
5. NodeBackendRuntime 页面只展示节点运行环境配置、节点覆盖参数、check-request evidence。
6. Deployment 页面展示部署可覆盖参数、部署覆盖值、最终 RunPlan preview。
7. Instance 页面展示运行结果、状态、日志、健康检查、实际 Docker spec 摘要。
8. Instance 页面不编辑运行参数。

### 6.5 enabled / checked 语义

1. enabled=true 只表示用户在当前层级显式启用或覆盖。
2. default value 不等于 enabled。
3. required 不等于用户 checked。
4. required/default-applied 参数可以在最终 RunPlan 生效，但 UI 不把它显示成用户 checked。
5. optional 参数默认不 checked。
6. advanced 参数默认折叠、不 checked。
7. disabled input 仍显示当前值、默认值或继承值。
8. 未 enabled 的 optional 参数不进入当前层级 override。

## 7. RunPlan 硬约束

1. ResolvedRunPlan 是唯一最终运行权威。
2. 参数合成只在 RunPlan resolver 中完成。
3. RunPlan preview 必须展示最终生效值和来源。
4. 实际 Docker create spec 必须来自同一个 ResolvedRunPlan。
5. RunPlan preview 与实际 Docker spec 不一致时测试失败。
6. 每个最终参数必须带 source。
7. source 至少区分 default、model、backend_version、backend_runtime、node_backend_runtime、deployment_override、system_generated、runtime_detected。

## 8. 文档和提交约束

1. Codex review 必须提交并推送。
2. Claude 执行结果必须提交并推送。
3. 每次提交前检查 `git status --short`。
4. 文档提交和功能代码提交应尽量分离。
5. closeout 必须记录 commit id、push result、测试结果、evidence 路径、git status。
