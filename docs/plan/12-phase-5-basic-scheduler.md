# Phase 5：基础自动调度

> 依赖：Phase 2（Agent Docker Runtime + GpuLease 流转）
> 周期：3-4 周

## 1. 目标

从手动选择资源（manual）过渡到系统自动选择节点和 GPU（auto）。用户不再需要自己判断哪张 GPU 空闲、哪个节点有足够显存。

## 2. 范围

- `schedule_mode: auto`
- GPU vendor 过滤（匹配 RuntimeEnvironment.vendor 或 custom 放行）
- 显存过滤（estimated_vram_bytes ≤ gpu.memory_free_bytes）
- GPU 健康状态过滤（health=healthy, status=available）
- GpuLease 冲突过滤（无 active/reserved lease）
- best-fit 策略（在满足条件的 GPU 中选显存余量最小的）
- 调度失败原因解释（列出不满足条件的具体原因）
- Web 上 schedule_mode 切换

## 3. 明确不做什么

- binpack（尽量填满一张 GPU）/ spread（尽量分散到多张 GPU）
- 多副本自动分配
- 多租户配额
- 优先级
- 抢占（preemption）
- 故障重调度
- GPU 拓扑感知（NVLink/NUMA）
- 成本/性能/SLA 策略
- 跨节点调度（第一阶段单节点内选择 GPU）

## 4. 调度算法

### 4.1 输入

```go
type ScheduleRequest struct {
    DeploymentID        string
    Vendor              string   // 来自 RuntimeEnvironment
    EstimatedVRAMBytes  int64    // 来自 ModelArtifact
    RequiredGPUCount    int      // 来自 ModelDeployment
    PreferredNodeID     string   // 可选，用户偏好
}
```

### 4.2 过滤链

```text
所有 GPU
  → 过滤：status=available AND health=healthy
  → 过滤：vendor 匹配（vendor=custom 放行）
  → 过滤：无 active/reserved GpuLease
  → 过滤：memory_free_bytes >= estimated_vram_bytes
  → 排序：按 memory_free_bytes ASC（best-fit）
  → 取前 N 张（N = required_gpu_count）
```

### 4.3 输出

```go
type ScheduleResult struct {
    Success   bool
    GPUIds    []string
    NodeID    string
    Score     int64  // best-fit 的显存余量
    Failures  []ScheduleFailure // 每张不满足条件的 GPU 的具体原因
}

type ScheduleFailure struct {
    GPUId  string
    Reason string // "health=unhealthy", "vendor=nvidia,required=metax", "free_vram=8GB,required=24GB", "lease_active"
}
```

### 4.4 失败原因示例

```text
候选 GPU 共 8 张：
  GPU-001: ✗ health=warning
  GPU-002: ✗ vendor=nvidia, 需要 metax
  GPU-003: ✗ 已被 deployment=xxx 占用 (lease active)
  GPU-004: ✗ 空闲显存 8GB, 需要 24GB
  GPU-005: ✓ (best-fit, 空闲 28GB)
```

## 5. 数据模型变更

无新增表。`model_deployments.schedule_mode` 从 `manual` 扩展到 `auto`。

## 6. API 变更

```text
POST /api/model-deployments/{id}/dry-run
  → 请求增加 schedule_mode=auto（不传 node_id/gpu_ids）
  → 响应增加 schedule_result（包含 reasoning）

POST /api/model-deployments
  → 创建时可传 schedule_mode=auto，省略 node_id/gpu_ids
```

## 7. 代码承接

| 模块 | 位置 |
|------|------|
| Scheduler 接口 + best-fit 实现 | `internal/server/scheduler/` |
| Dry Run 集成 | `internal/server/api/deployment_handler.go` |
| 创建部署集成 | 同上 |

## 8. 测试要求

- best-fit 选择显存余量最小的 GPU
- vendor 不匹配时被过滤（nvidia GPU 不能用于 metax 部署）
- GPU 已被 lease 占用时被过滤
- 显存不足时被过滤
- 所有 GPU 都不满足时返回完整的 failure reasons
- custom vendor 不进行 vendor 过滤
- schedule_mode=manual 时不触发调度器
- 权限（deployment:write 才能触发 auto schedule）

## 9. 验收标准

```bash
# auto 模式 Dry Run（有满足条件的 GPU）
curl -X POST /api/model-deployments/{id}/dry-run \
  -d '{"schedule_mode":"auto"}'
# → {"valid":true,"schedule_result":{"success":true,"gpu_ids":["gpu-005"],"node_id":"node-1","score":30064771072}}

# auto 模式 Dry Run（无满足条件的 GPU）
curl -X POST /api/model-deployments/{id}/dry-run \
  -d '{"schedule_mode":"auto"}'
# → {"valid":false,"errors":["无满足条件的 GPU"],"schedule_result":{"success":false,"failures":[...]}}

# manual 模式不受影响
curl -X POST /api/model-deployments/{id}/dry-run \
  -d '{"schedule_mode":"manual","node_id":"...","gpu_ids":["..."]}'
# → 行为与 Phase 1 完全一致
```

## 10. 风险点

- best-fit 可能导致某张 GPU 的显存碎片化（每次选余量最小的，剩余显存越来越小）。后续 Phase 需加入 binpack/spread
- 单节点内调度假设所有 GPU 在同一节点上。跨节点调度需要更复杂的网络拓扑感知
- estimated_vram_bytes 是用户填写的预估值，如果用户填写不准确（太小），会导致 OOM；如果太大，会导致 GPU 闲置
