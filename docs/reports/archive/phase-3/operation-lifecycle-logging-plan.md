> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# Operation Lifecycle Logging Plan

> Phase 3 — 全操作链路可观测性规范
> 目的：不新增功能，不重构，只补日志让任何操作都能从日志定位卡在哪一步

## 1. 统一日志阶段定义

所有关键操作应包含以下阶段的日志（不要求每个操作覆盖全部阶段）：

| Stage | Level | 说明 |
|-------|-------|------|
| `operation_received` | INFO | 操作请求已接收，含 operation_id |
| `auth_checked` | INFO/DEBUG | 认证结果 |
| `tenant_resolved` | DEBUG | 租户解析结果 |
| `permission_checked` | DEBUG | 权限校验结果 |
| `input_validated` | DEBUG | 输入参数校验 |
| `resource_resolved` | INFO | 依赖资源已定位 (artifact/runtime/backend/node/gpu) |
| `plan_resolved` | INFO | RunPlan/配置已生成 |
| `db_tx_begin` | DEBUG | DB 事务开始 |
| `db_write_done` | DEBUG | DB 写入完成 |
| `task_created` | INFO | Agent 任务已创建 |
| `task_claimed` | INFO | Agent 已领取任务 |
| `external_action_started` | INFO | 外部动作开始 (Docker start/stop) |
| `external_action_completed` | INFO | 外部动作完成 |
| `state_transition` | INFO | 状态转换 (from→to) |
| `wait_started` | INFO | 开始等待 (条件+timeout) |
| `wait_progress` | INFO/DEBUG | 等待中当前状态 |
| `wait_completed` | INFO | 等待完成 (耗时) |
| `operation_completed` | INFO | 操作完成 (总耗时) |
| `operation_failed` | ERROR | 操作失败 (原因) |
| `cleanup_started` | INFO | 清理开始 |
| `cleanup_completed` | INFO | 清理完成 |
| `cleanup_failed` | WARN | 部分清理失败 |
| `timeout` | ERROR | 超时 (最后一次状态) |

## 2. 每类操作应包含的阶段

### A. Server 启动
`config_load → db_open → db_migrate → seed_init → http_listen → operation_completed`

### B. Auth / Login
`operation_received → auth_checked → tenant_resolved → operation_completed`

### C. Deployment Start
`operation_received → auth_checked → resource_resolved → plan_resolved → task_created → wait_started → wait_progress → state_transition → operation_completed`

### D. Agent Task Execution
`task_claimed → resource_resolved → plan_resolved → external_action_started → external_action_completed → state_transition → operation_completed`

### E. Docker Lifecycle
`external_action_started → external_action_completed` (per container operation: create/start/stop/remove)

### F. GPU Lease
`task_created → state_transition(reserved→active/released)`

### G. Health Check
`wait_started → wait_progress → wait_completed/timeout`

### H. E2E / Scripts
`mode_selected → operation_started → wait_progress → operation_completed/failed → cleanup_completed`

## 3. Correlation ID 设计

| ID | 生成位置 | 传递方式 | 当前状态 |
|----|---------|---------|---------|
| `operation_id` | API handler entry | request context → task payload → agent logs → task result | ❌ 未实现 |
| `request_id` | Middleware | response header + log context | ❌ 未实现 |
| `instance_id` | HandleStartDeployment | task payload + DB + agent logs | ✅ |
| `task_id` | HandleStartDeployment | agent payload + agent logs + task result | ✅ |
| `deployment_id` | API request | DB + task payload | ✅ |

**缺口**: 无统一的 operation_id 贯通 server→agent→server 全链路。当前靠 instance_id + task_id 人工关联。

## 4. Wait / Poll 日志规范

所有等待循环必须输出：

```
wait_started: condition=<条件> timeout=<秒>
wait_progress: elapsed=<秒> current_state=<当前状态>
wait_completed: elapsed=<秒> final_state=<最终状态>
wait_timeout: elapsed=<秒> last_state=<最后状态>
```

禁止 silent wait。禁止只输出 "Waiting for task"。

## 5. 待实现优先级

| Priority | Item | Effort |
|----------|------|--------|
| P0 | wait_started/wait_progress/wait_timeout 日志 (E2E + agent poll) | Low |
| P0 | Docker spec dump before create (agent) | Low |
| P0 | Container exit logging with inspect + logs tail (agent) | Low |
| P1 | operation_id generation + propagation | Medium |
| P1 | API middleware request logging (duration + status + operation) | Medium |
| P1 | RunPlan resolve full detail logging (already partially done) | Low |
| P2 | GPU lease lifecycle tracking | Low |
| P2 | Health check retry detail logging | Low |
| P3 | Package/upgrade script logging | Medium |

## 6. Success-Path Logging Requirements (补充)

### 6.1 每个关键阶段必须有 completed 日志

成功时不只写 "success"，必须包含：
- `operation` + `stage`
- `operation_id` / `request_id` / `instance_id` / `task_id`（如适用）
- `state_from` / `state_to`（如适用）
- `duration_ms`
- 关键结果摘要（image, backend, container_id, endpoint, etc.）

### 6.2 成功慢操作 WARN

| Operation | Threshold | WARN Message |
|-----------|-----------|-------------|
| API request | 1000ms | slow_api_request |
| DB transaction | 500ms | slow_db_tx |
| Agent claim | 10000ms | slow_task_claim |
| Docker create | 5000ms | slow_docker_create |
| Docker start | 10000ms | slow_docker_start |
| Health check | 30000ms | slow_health_check |
| Model load | 120000ms | slow_model_load |
| E2E wait | 60000ms | slow_e2e_wait |

### 6.3 成功日志覆盖矩阵要求

每个操作必须在 audit 中标注：
- `_started`: 是否有开始日志
- `_completed`: 是否有成功完成日志
- `_failed`: 是否有失败日志
- `_timeout`: 是否有超时日志
- `_duration`: 是否记录耗时
- `_summary`: 是否包含结果摘要

### 6.4 当前已覆盖的成功日志

| Operation | started | completed | duration | summary |
|-----------|---------|-----------|----------|---------|
| DB migrate | ✅ | ✅ | ✅ | ✅ |
| RunPlan resolve | ✅ | ✅ | ✅ | ✅ (image, args, errors) |
| Deployment start | ✅ | ✅ | ✅ | ✅ (instance_id, task_id) |
| Deployment stop | ✅ | ✅ | ✅ | ✅ (instances_stopped) |
| Deployment delete | ✅ | ✅ | ✅ | ✅ |
| Task result (success) | ❌ | ❌ | ❌ | ❌ — 缺失 |
| Task result (failure) | ❌ | ❌ | ❌ | ❌ — 缺失 |
| Agent claim task | ❌ | ❌ | ❌ | ❌ — 缺失 |
| Docker create | ❌ | ❌ | ❌ | ❌ — 缺失 |
| Docker start | ❌ | ❌ | ❌ | ❌ — 缺失 |
| Docker stop | ❌ | ❌ | ❌ | ❌ — 缺失 |
| Health check | ❌ | ❌ | ❌ | ❌ — 缺失 |
| API CRUD | ❌ | ❌ | ❌ | ❌ — 缺失 |
| E2E wait | ❌ | ❌ | ❌ | ❌ — 缺失 |

### 6.5 P0 待补（下一轮）

1. Agent Docker create/start completed success log with container_id + duration
2. Task result received success log with instance_id + state transition  
3. E2E wait_completed with elapsed + state
4. Agent claim task completed with task_id + duration
