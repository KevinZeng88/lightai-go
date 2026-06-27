# UI and API Contract

## 1. 目的

UI 不再按“所有参数平铺大表”展示，而是展示 ConfigSet 生成的 ConfigView / ConfigPanel。

## 2. API 输出原则

每个领域对象 API 应返回 config_bundle、config_views、local_edits_summary、effective_view。Deployment preview API 还必须返回 resolved_run_plan、parameter_source_map、plan_hash。

## 3. ConfigView API shape

```json
{
  "config_set_key": "deployment",
  "title": "部署配置",
  "summary_view": {},
  "edit_view": {"sections": []},
  "preview_view": {},
  "child_slots": []
}
```

UI 不直接解析内部旧 config_set_json 混合结构。

## 4. Renderer

默认使用 GenericConfigSetRenderer。少数复杂项使用 CustomRendererRegistry。custom renderer 必须遵守 ConfigItem 字段分级，不得绕过 value/state/provenance。

## 5. 展示规则

页面展示 Config / ConfigPanel，不直接暴露内部 ConfigSet 原始结构；ConfigSet 自己定义 required/common/advanced/readonly/local_edits 等 sections；父 ConfigSet 定义 child_slots，并调用 child ConfigSet view；child ConfigSet 自己处理内部展示；advanced 默认折叠；required/common 显眼展示；local edits 单独可见；inherited value、local value、effective value 必须可区分；checked/enabled 只表示当前层显式修改；disabled input 仍显示 effective value。

## 6. 页面契约

Model 页面展示 ModelArtifactConfigSet / ModelLocationConfigSet，不展示 Docker/runtime/GPU/deployment override 参数。

Backend / BackendVersion 页面展示 BackendCapabilityConfigSet / BackendParameterConfigSet / BackendEndpointConfigSet，不展示节点检测结果、本机模型路径、部署覆盖。

BackendRuntime 页面展示 inherited BackendVersion summary、RuntimeTemplateConfigSet、RuntimeDockerConfigSet、RuntimeHealthCheckConfigSet、local edits。

NodeBackendRuntime 页面展示 inherited BackendRuntime view、NodeRuntimeEnvironmentConfigSet、NodeDeviceBindingConfigSet、NodeRuntimeCheckEvidenceConfigSet、local edits。NBR check evidence 是节点运行配置自己的 ConfigSet，不应混入 BackendVersion。

Deployment 页面展示 deployment required/common/advanced、ModelArtifact / ModelLocation child panels、NodeBackendRuntime child panel、DeploymentPortConfigSet、DeploymentVolumeConfigSet、DeploymentHealthCheckConfigSet、local edits summary、RunPlan preview。Deployment 不应直接编辑 BackendVersion schema。

Instance 页面只展示 status、health、logs、actual Docker spec summary、ResolvedRunPlan summary、errors。Instance 不编辑 ConfigSet。

## 7. API 行为

Save 只保存当前层 local edits 和 own ConfigSet 修改；Refresh 不应从上游 live config 覆盖当前层 snapshot；Clone 必须复制当前对象 effective bundle snapshot，不扩大 checked/enabled 范围；Delete 不应影响 parent snapshot；Check-request 只更新 NBR check evidence ConfigSet；Deployment start 只使用 DeploymentConfigBundle effective snapshot。

## 8. 测试要求

Web unit/component tests 必须覆盖 ConfigSetRenderer sections、child_slots 调用 child ConfigSet view、required/common/advanced 分组、advanced 默认折叠、local edits summary、inherited/local/effective value 展示、checked/enabled 只表示当前层 local edit、custom renderer obeys ConfigItem contract、Model 页面不展示 Docker 参数、Instance 页面不编辑 ConfigSet。
