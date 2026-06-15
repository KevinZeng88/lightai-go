# LightAI Go 模型运行与服务管理 — 阶段实施总览

> 基于 `docs/design/12-model-runtime-serving-design.md`（修订版）
> 最后更新：2026-06-15

## 阶段总览

| Phase | 名称 | 周期 | 核心交付 | 依赖 |
|---|---|---|---|---|
| 1 | 数据模型与 Dry Run | 2-3 周 | CRUD + Dry Run + ResolvedRunSpec + 权限 + 脱敏 | 无 |
| 2 | Agent Docker Runtime | 2-3 周 | DockerRuntimeDriver + Start/Stop/Logs + GpuLease 流转 | Phase 1 |
| 3 | Web 模型服务 | 2-3 周 | 模型库/环境/模板/部署/实例页面 + GPU 占用展示 | Phase 2 |
| 4 | Gateway + API Key + Usage | 3-4 周 | OpenAI-compatible API + API Key + 调用审计 | Phase 3 |
| 5 | 基础自动调度 | 3-4 周 | auto schedule_mode + vendor/显存过滤 + best-fit | Phase 2 |

## 各阶段详细文档

- [Phase 1：数据模型与 Dry Run](./12-phase-1-model-runtime-foundation.md)
- [Phase 2：Agent Docker Runtime](./12-phase-2-agent-docker-runtime.md)
- [Phase 3：Web 模型服务](./12-phase-3-web-model-serving.md)
- [Phase 4：Gateway + API Key + Usage](./12-phase-4-gateway-api-key-usage.md)
- [Phase 5：基础自动调度](./12-phase-5-basic-scheduler.md)

## 关键设计决策（已确认）

1. RunTemplate 保留独立 CRUD，Phase 1 简化（结构化，无通用模板引擎）
2. ModelArtifact 与 ModelDeployment 允许 1:N
3. GPU 可见变量由 RuntimeEnvironment 提供默认值，RunTemplate 引用，Deployment 可覆盖
4. docker stop 后默认不 rm；delete instance/deployment 时才 rm -f
5. Agent 离线时不立即释放 GpuLease，先标记实例 unknown
6. Phase 1 复用 audit_logs，不新增 ModelEvent 表
7. Phase 1 不建 ModelRoute 和 ModelUsageRecord 表
8. API 路径不加 /v1，沿用现有 `/api/` 风格
