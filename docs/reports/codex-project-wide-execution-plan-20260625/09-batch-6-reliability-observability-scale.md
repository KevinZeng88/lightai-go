# Batch 6 — Reliability, Observability, and Operational Hardening

## 目标

解决启动/停止/任务/租约/日志/节点离线等可靠性问题，使单机和小规模多节点行为可预测。

覆盖：

- stability/reliability/observability findings
- GPU lease conflict
- task timeout cleanup
- delete live deployment cleanup
- logs endpoint limits
- R-013 的部分 observability 一致性

## 任务

### 6.1 GPU lease 唯一性与并发 start

实现/验证：

- 同一 GPU/accelerator 同一时间不能被两个 running/reserved lease 占用，除非支持共享策略。
- 同一 host port 不能被两个实例占用。
- 并发 start 使用事务和唯一索引/条件检查。
- 冲突返回 deterministic error。

测试：

```text
internal/server/api/gpu_lease_concurrency_test.go
internal/server/api/host_port_conflict_test.go
```

### 6.2 task timeout 与 node offline

实现/验证：

- Agent claimed task 后宕机。
- node heartbeat 停止。
- server 能把 in-progress task 和 starting instance 进入一致终态。
- GPU lease 释放或标记 orphan。
- operation_id 可追踪。

测试：

```text
internal/server/api/node_offline_task_recovery_test.go
internal/server/api/agent_restart_task_idempotency_test.go
```

### 6.3 delete deployment with live container

当前风险：DB 删除可能丢失真实 container cleanup context。

要求：

- 删除 deployment 前必须停止或标记 cleanup task。
- 如果 Agent 不可达，deployment 进入 deleting/pending_cleanup，不直接丢失 instance/container id。
- 后续 reconcile 可继续清理。
- UI 显示 pending cleanup。

### 6.4 Stop failed/starting cleanup

要求：

- stop running/starting/failed 都幂等。
- missing container 视作 stopped，但记录 warning。
- lease release 幂等。
- task stale result 不破坏 terminal state。

### 6.5 Logs endpoint 限制

实现：

- 默认 tail 行数。
- 最大 tail 行数。
- 最大 bytes。
- 大日志截断标记。
- UI 自动刷新退避。
- 错误时保留诊断摘要。

### 6.6 Observability status

如果 Prometheus/Grafana 仍由脚本管理，文档和 UI 不应声称 server-managed。

选择：

- 实现 `/observability/status` 反映外部脚本模式。
- 或明确文档：Prom/Grafana supervision 不在 Go server 内。

### 6.7 operation_id 贯通审计

检查：

- deployment start/stop。
- NBR check。
- preflight/dry-run。
- Agent task claim/result。
- Docker create/start/logs/stop/remove。

要求关键路径日志都能按 operation_id 追踪。

## 验证命令

```bash
go test ./internal/server/api ./internal/server/runplan ./internal/agent/runtime
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
```

必要时运行 real smoke：

```bash
scripts/e2e-current-contract-nvidia-llamacpp-smoke.sh
```

## 验收

- 并发 GPU/port 冲突测试通过。
- node offline/task timeout 终态一致。
- delete live deployment 不丢 cleanup context。
- logs endpoint 有硬限制。
- observability claim 与实现一致。
