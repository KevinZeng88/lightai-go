# Execution Policy and Scope

## 1. 本阶段定位

本阶段聚焦 Runtime 架构和参数体系的最终收敛。目标是把模型、后端、运行模板、节点运行配置、部署、RunPlan、Preflight、实例生命周期和 UI/API 展示统一到清晰、干净、可验证的主线。

自动化执行、无人干预、API-first E2E 是本阶段验收方式。自动化要求服务于架构收敛，避免把自动化脚本本身变成孤立目标。

## 2. 范围内工作

### 2.1 领域模型收敛

必须重新核对并收敛：

1. Backend；
2. BackendVersion；
3. BackendRuntime；
4. NodeBackendRuntime；
5. ModelArtifact；
6. ModelLocation；
7. RuntimeRequirements；
8. BackendCapabilityProfile；
9. RuntimeParameterSchema；
10. RuntimeParameterValues；
11. Deployment；
12. ResolvedRunPlan；
13. DeviceBinding；
14. HealthCheck；
15. ModelInstance lifecycle。

### 2.2 参数体系收敛

必须明确参数在不同层级的职责：

1. 模型参数；
2. 后端能力参数；
3. 运行模板参数；
4. 节点运行参数；
5. 部署覆盖参数；
6. 最终 Docker args；
7. 最终 env；
8. 最终 mounts；
9. 最终 ports；
10. 最终 device bindings；
11. 健康检查参数；
12. UI 展示参数；
13. API 返回参数；
14. E2E 断言参数。

### 2.3 Preflight 和 RunPlan 收敛

必须保证：

1. Preflight 使用 RuntimeRequirements 和 BackendCapabilityProfile；
2. RunPlan 使用同一套输入和合并规则；
3. RunPlan preview 与实际 Docker create spec 一致；
4. errors/warnings 语义清楚；
5. API 和 UI 展示同口径；
6. E2E 能断言关键字段。

### 2.4 UI/API 链路修复

必须覆盖：

1. 模型管理页面；
2. 运行配置页面；
3. 节点运行配置页面；
4. 部署创建页面；
5. RunPlan 预览；
6. 模型实例页面；
7. 日志页面；
8. 状态提示；
9. 参数编辑器；
10. clone / save / refresh 行为。

### 2.5 API-first 自动化验收

必须提供自动化验收能力，覆盖：

1. fresh DB；
2. server / agent 启动；
3. 登录与 CSRF；
4. BackendRuntime；
5. NodeBackendRuntime enable；
6. check-request；
7. model scan；
8. ModelArtifact / ModelLocation；
9. Deployment；
10. Preflight；
11. RunPlan preview；
12. start；
13. health check；
14. logs；
15. stop；
16. final state；
17. evidence；
18. non-zero failure。

## 3. 范围外工作

以下事项本阶段只做边界预留或文档记录，除非代码中已经存在可修复缺陷：

1. 多节点高级调度；
2. 多副本编排；
3. 租户级计费结算；
4. 生产级审计报表；
5. 真实 MetaX/Huawei 硬件运行验证；
6. 大规模压力测试；
7. UI 全量 Playwright 覆盖；
8. OpenAI compatible gateway 的完整计费闭环。

## 4. 强制原则

### 4.1 干净主线

1. 不做历史兼容迁移。
2. 表结构变化允许重建数据库。
3. 旧字段、旧接口、旧模板、旧 snapshot fallback 应删除。
4. 旧流程导致分裂时，保留当前正确模型。
5. 文档、API、UI、测试必须同口径。

### 4.2 部署入口

1. NodeBackendRuntime 是唯一部署入口。
2. Deployment 只接受 `node_backend_runtime_id`。
3. Deployment 拒绝 `backend_runtime_id`。
4. 不自动创建 NodeBackendRuntime。
5. NodeBackendRuntime 必须显式 enable。
6. check-request 必须通过 Server 代理 Agent 获取真实证据。
7. ready 和 ready_with_warnings 可部署。
8. needs_check、missing_image、failed、disabled 不可部署。

### 4.3 Backend / BackendVersion 硬件无关

Backend 和 BackendVersion 只表达推理后端和后端版本能力。

GPU vendor、Docker runtime、设备文件、驱动、节点硬件差异属于：

1. BackendRuntime；
2. NodeBackendRuntime；
3. Node；
4. Accelerator；
5. DeviceBinding；
6. RunPlan。

### 4.4 自动化验收

1. 不依赖人工手工执行 Docker 命令判断结果。
2. 不依赖前端传入的 `image_present` 作为权威。
3. 不依赖 UI 手工刷新判断状态。
4. API-first 脚本必须输出结构化证据。
5. 失败必须退出非零。
6. E2E evidence 必须可复核。

## 5. 工作区和提交策略

执行前必须确认：

```bash
pwd
git status --short
git branch --show-current
git log --oneline -15
```

默认在当前分支继续，不新建分支。

如存在用户未说明的修改：

1. 先记录文件清单；
2. 判断是否为当前任务相关；
3. 不覆盖用户改动；
4. 在输出中说明处理方式。

每个可独立验证的批次可以单独 commit。最终 closeout 必须列出 commit list、push result、git status。
