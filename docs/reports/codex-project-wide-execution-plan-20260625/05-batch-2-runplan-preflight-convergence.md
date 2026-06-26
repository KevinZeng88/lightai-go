# Batch 2 — RunPlan and Preflight Convergence

## 目标

让 `/deployments/preflight`、`/deployments/{id}/dry-run`、`/deployments/{id}/start` 在 deployability、resolver、错误结构、warnings 方面一致。

覆盖：

- R-003
- Q-002
- R-005 中 snapshot boundary 的 resolver 相关部分
- RunPlan final contract

## 设计决策

默认决策：`/deployments/preflight` 应升级为 final RunPlan preflight。  
如果仍需要“候选节点/候选模型”轻量检查，应新建或改名为 candidate check，不再叫 final preflight。

## 任务

### 2.1 抽取统一 resolver input builder

建立一个共享函数或服务层，例如：

```go
BuildDeploymentRunPlanInput(ctx, tenantID, deploymentID, opts) (...)
PreflightDeploymentFinal(ctx, deploymentID, opts) (...)
```

要求：

- dry-run 使用它。
- start 使用它。
- preflight 使用它。
- error schema 一致。
- warnings 一致。
- deployability 使用同一 helper，例如 `isNBRDeployable()`。

### 2.2 统一状态语义

必须接受：

- `ready`
- `ready_with_warnings`

必须阻断：

- `needs_check`
- `missing_image`
- `disabled`
- `error`
- node offline 或 Agent 不可达（如果 final plan 需要 Agent evidence）

### 2.3 统一错误结构

建议统一：

```json
{
  "can_run": false,
  "errors": [
    {
      "code": "nbr_not_ready",
      "message": "...",
      "field": "node_backend_runtime_id",
      "severity": "error"
    }
  ],
  "warnings": [
    {
      "code": "version_probe_warning",
      "message": "...",
      "field": "backend_version"
    }
  ],
  "resolved_run_plan": null
}
```

不要让 preflight 返回纯 string errors，而 dry-run/start 返回 structured errors。

### 2.4 RunPlan final preview contract

如果 UI 需要预览最终命令，应暴露：

- deployment create 后 preview。
- deployment patch 后 preview。
- NBR/deployment parameter override 后 preview。
- final AgentRunSpec 或 Docker command preview。

必须声明：preview 与 start 使用相同 resolver input class。

### 2.5 多副本/单实例语义

当前 start 是 single instance。  
本批至少要使 preflight/dry-run/start 对 replicas > 1 一致：

- 未支持前：API 400，UI 禁用或隐藏。
- 或明确生成多 run plans 并有测试。

如果实现多副本风险太大，默认先拒绝 replicas > 1，进入 Batch 8 设计。

### 2.6 测试

新增 contract matrix：

| Case | preflight | dry-run | start |
| --- | --- | --- | --- |
| ready NBR | pass | pass | pass |
| ready_with_warnings | pass with warnings | pass with warnings | pass or queued with warnings |
| needs_check | block | block | block |
| missing image | block | block | block |
| model location missing | block | block | block |
| context overflow | block | block | block |
| invalid port | block | block | block |
| stale snapshot | deterministic behavior | deterministic behavior | deterministic behavior |
| replicas > 1 unsupported | same 400 | same 400 | same 400 |

建议文件：

```text
internal/server/api/deployment_preflight_contract_test.go
internal/server/api/deployment_runplan_consistency_test.go
internal/server/runplan/runplan_contract_matrix_test.go
```

## 验证命令

```bash
go test ./internal/server/api ./internal/server/runplan
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
```

## 验收

- R-003 CLOSED。
- preflight/dry-run/start 对同一 deployment 得到一致结果。
- `ready_with_warnings` 不被 preflight 误阻断。
- preflight 不再只是轻量 candidate check，除非另有新 endpoint 明确命名。
- Batch closeout 记录 API 样例和测试输出。
