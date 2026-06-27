# RunPlan and Preflight Contract

## 1. 目的

RunPlan 只读取 DeploymentConfigBundle effective snapshot。preview / preflight / dry-run / start 必须共用同一个 builder。ResolvedRunPlan 必须输出 parameter_source_map。

## 2. 单一 builder

必须建立单一服务端构建路径，例如 `BuildDeploymentRunPlanInput()` / `BuildResolvedRunPlan()`。deployment preview、deployment preflight、deployment dry-run、deployment start、API-first E2E 必须共用。禁止 preview 和 start 分别组装参数。

## 3. RunPlan 输入

RunPlan builder 的输入：DeploymentConfigBundle effective snapshot、Deployment desired state、ModelArtifact snapshot、ModelLocation snapshot、system/runtime context。禁止读取上游 live BackendRuntime / NBR 配置来覆盖 Deployment snapshot。

## 4. RunPlan 输出

ResolvedRunPlan 必须包含 image、command、args、env、mounts、ports、devices、docker_options、health_check、resource_controls、parameter_source_map、plan_hash、audit_refs。

## 5. parameter_source_map

source map 必须覆盖 args、env、mounts、ports、devices、docker_options、health_check、resource_controls、system_generated。

```json
{
  "args": [
    {
      "key": "gpu_memory_utilization",
      "target": "args",
      "arg": "--gpu-memory-utilization",
      "value": 0.82,
      "effective_source": "deployment_local_edit",
      "config_set_key": "BackendParameterConfigSet",
      "last_value_layer": "DeploymentConfigBundle",
      "source_chain": [
        {"layer": "BackendVersionConfigBundle", "value": 0.9, "reason": "schema default"},
        {"layer": "NodeBackendRuntimeConfigBundle", "value": 0.8, "reason": "node local edit"},
        {"layer": "DeploymentConfigBundle", "value": 0.82, "reason": "deployment local edit"}
      ]
    }
  ],
  "env": [],
  "mounts": [],
  "ports": [],
  "devices": [],
  "docker_options": [],
  "health_check": []
}
```

Storage/API rule：`resolved_run_plans.plan_json.parameter_source_map`；preview API response 返回 `resolved_run_plan.parameter_source_map`；instance actual spec summary references plan_hash。

## 6. Docker options

Docker 子字段必须来自 ConfigItem，例如 docker.shm_size、docker.group_add、docker.devices、docker.extra_hosts、docker.ipc_mode。state.enabled=false 的 optional Docker item 不进入 final Docker spec；value 存在但 checked=false 不代表进入 final Docker spec；required/system_generated Docker item 可以进入 final spec，但 source 必须清楚；旧 enabled_fields 应清理或转换为 ConfigItem.state，不作为长期兼容模型。

## 7. Preflight

Preflight 使用同一份 RunPlan input 和 resolved plan，检查 image evidence、model location、mounts、ports、devices、runtime requirements、backend capability profile、health check、resource controls、required ConfigItem、invalid ConfigItem、unchecked optional exclusion。

Preflight 输出 errors、warnings、resolved_run_plan_preview、parameter_source_map。

## 8. Required / default / checked

RunPlan builder 必须执行：required 参数缺失时返回 blocking error；default 不等于 enabled；inherited 不等于 checked；optional unchecked 不进入 args/env/docker_options；local_edit checked/enabled 进入 final spec；system_generated 可以进入 final spec，但 source 标记为 system_generated。

## 9. 验收

必须证明：preview/start 同 builder；preview plan_hash 与 start plan_hash 一致；preview Docker spec 与实际 Docker create spec 一致；source_map 覆盖所有 final fields；Docker optional unchecked 被过滤；required 缺失阻断；copy-on-create 后上游修改不影响 Deployment RunPlan；Preflight 和 start 使用同一 resolved plan。
