# 04 - MiMo Review and Comments

> Status: review response
> Date: 2026-06-25
> Reviewer: MiMo (mimo-v2.5-pro)

## 1. 合理的设计

### 1.1 命令模板不是编辑入口（§2.1）

完全同意。`command_template`/`default_command` 是生成模板，不是用户参数编辑入口。当前代码已经将 `--host`/`--port` 从 `default_args` 移到 `default_args_schema`，通过 `parameter_values_json` 提供，方向正确。

### 1.2 required/optional 分离（§2.2）

required 参数 locked-on、optional 参数 enabled/value 的模型清晰，可实现。

### 1.3 vendor 正交性（§2.4）

Backend serve args 和 vendor runtime settings 是不同维度，这个分离是正确的。NVIDIA 用 DeviceRequests，MetaX 用 raw devices，Huawei 用 Ascend devices，三者不应串台。

### 1.4 分层优先级（§3.7）

`Deployment override > NBR > BackendRuntime > BackendVersion default` 的优先级模型清晰。

### 1.5 Clean DB 策略（§7）

不兼容旧 DB、允许重建的策略务实。

## 2. 可能过度复杂的设计

### 2.1 参数 schema 字段过多（§4）

设计文档列出了 30+ 个 schema 字段（name, label, group, type, required, default, value, enabled, user_editable, cli_flag, env_name, docker_field, backend, vendor, layer, advanced, risk_level, source, allow_override, visible_when, applies_to, help_ref, placeholder, validation, conflict_keys, redaction, copy_policy）。

**风险**：一次性实现所有字段会导致：
- YAML catalog 文件变得极其冗长
- 前端渲染逻辑复杂度指数增长
- 测试覆盖困难
- 实际使用中大部分字段为空

**建议**：分批实现。Phase 1 只用 `name, required, value, default, optional, alias`。Phase 2 加 `type, group, label, placeholder`。Phase 3 加 `advanced, risk_level, vendor, backend`。Phase 4 加 `help_ref, conflict_keys, redaction`。

### 2.2 VendorRuntimeProfile 独立实体（§3.3）

设计提出 `VendorRuntimeProfile` 作为独立概念，但当前代码中 vendor 信息嵌入在 BackendRuntime 的 `docker_json` 中（如 `gpu_driver`, `gpu_capabilities`, `devices`）。

**风险**：引入独立 VendorRuntimeProfile 需要：
- 新 DB 表或新 YAML 结构
- BackendRuntime 创建逻辑重构
- RunPlan resolver 增加 vendor profile 合并层
- 所有 E2E 脚本更新

**建议**：当前阶段保持 vendor 信息在 BackendRuntime `docker_json` 中，通过 catalog 文件的 `vendor` 字段区分。后续需要多 vendor 动态切换时再引入独立实体。

### 2.3 Source Map（§2.5）

设计提出 RunPlan 应包含每个参数的来源追踪（`{value, source, layer}`）。

**风险**：
- 增加 resolver 复杂度
- 增加 RunPlan JSON 体积
- API response 需要脱敏处理
- 前端展示逻辑复杂

**建议**：Phase 1 不实现 source map，只在 debug log 中记录关键参数来源。Phase 2 在 RunPlan 的 `debug_info` 字段中添加可选 source map。

### 2.4 Help 文档外置方案（§5）

设计提出 `configs/backend-catalog/help/{backend}/{version}.{locale}.yaml` 的外置帮助文档方案。

**风险**：
- 增加 catalog 维护成本
- help 内容需要与参数 schema 同步更新
- 多语言维护

**建议**：Phase 1 在 schema 中直接用 `help` 字段存储简短帮助文本（单行）。Phase 2 引入外置帮助文件。

## 3. 会影响当前实现的设计

### 3.1 required 参数 locked-on（§2.2）

当前 RuntimeParameterEditor 中 required 参数仍有普通 checkbox 可取消勾选。需要修改为：
- required 参数：显示 locked switch 或无 checkbox
- UI 层面阻止 disable

**影响**：RuntimeParameterEditor.vue 组件需要修改 `backendParams` 渲染逻辑。

### 3.2 Layer 3 缺少模板替换

当前 resolver Layer 3（Deployment overrides）不做 `substituteVars()`。如果用户在 Deployment override 中输入 `{{container_port}}`，会作为字面字符串传递。

**影响**：resolver.go 需要在 Layer 3 也调用 `substituteVars()`。

### 3.3 Deployment edit 不显示 Docker 选项

当前 `ModelDeploymentsPage` 的 `editParameterModel` 只传 `parameter_values_json`，`docker_json` 为空。设计说 Deployment override 应能覆盖部分 Docker 选项。

**影响**：需要决定哪些 Docker 选项允许 Deployment 级覆盖。

## 4. 与当前代码结构冲突的设计

### 4.1 RuntimeParameterEditor 的静态 scalarOptions/listOptions

当前 RuntimeParameterEditor 内部硬编码了 `scalarOptions`（privileged, ipc_mode 等）和 `listOptions`（devices, group_add 等）。这些是 Docker runtime 参数，不是 backend serve args。

设计要求 backend serve args 从 `backendSchema` 动态渲染，Docker runtime args 从 vendor profile 渲染。但当前代码两者混在同一个组件中。

**冲突**：如果要完全按设计实现，需要：
- 将 Docker runtime 参数从 RuntimeParameterEditor 移出
- 创建独立的 DockerRuntimeEditor 组件
- 或者让 RuntimeParameterEditor 接受两套 schema（backend schema + docker schema）

### 4.2 BackendRuntimesPage 的 command preview

BackendRuntimesPage 有自己的 `commandPreview` computed，RuntimeParameterEditor 也有。两者逻辑不一致（页面版本不包含 backend serving args）。

**冲突**：需要统一为单一来源。

### 4.3 args_override_json vs parameter_values_json 的语义

当前 `args_override_json` 存储原始 CLI 参数字符串数组（Layer 1），`parameter_values_json` 存储结构化参数（Layer 2）。设计要求所有参数都结构化，但 `args_override_json` 作为"高级附加参数"仍然需要保留。

**冲突**：需要明确 `args_override_json` 的定位——是"额外附加参数"还是"遗留兼容字段"。

## 5. 需要调整的字段/表/API/UI

### 5.1 DB 字段

| 当前字段 | 问题 | 建议 |
|---------|------|------|
| `backend_versions.default_args_schema_json` | 缺少 type/group/label 等元数据 | 分批扩展 |
| `backend_runtimes.parameter_schema_json` | 与 BackendVersion schema 重复 | 保持，作为 runtime 级 schema |
| `node_backend_runtimes.parameter_values_json` | 当前从 BackendVersion schema 初始化 | 改为从 BackendRuntime schema 初始化 |
| `model_deployments.parameter_values_json` | 只有 structured params，没有 Docker options | 按设计决定是否扩展 |

### 5.2 API DTO

| 端点 | 问题 | 建议 |
|------|------|------|
| `PATCH /backend-runtimes/{id}` | 接受 `parameter_schema_json` 和 `parameter_values_json` | 保持 |
| `PATCH /nodes/{id}/backend-runtimes/{id}` | 接受 `config_snapshot_json` 中的参数 | 保持 |
| `PATCH /deployments/{id}` | 接受 `parameter_values_json` | 保持，考虑扩展 Docker override |

### 5.3 Web UI

| 组件 | 问题 | 建议 |
|------|------|------|
| RuntimeParameterEditor | required 参数有可取消 checkbox | 改为 locked switch |
| RuntimeParameterEditor | 缺少参数分组 | 按 group 字段分组渲染 |
| RuntimeParameterEditor | 缺少 help `?` | 添加 popover 帮助 |
| BackendRuntimesPage | 重复 command preview | 统一为 RuntimeParameterEditor 的 preview |
| ModelDeploymentsPage | 不显示 Docker options | 按设计决定 |

## 6. 需要澄清的参数分层规则

### 6.1 container_port 的来源

当前 `container_port` 通过 `{{container_port}}` 模板变量在 resolver 中替换，值来自 `effectiveContainerPort()`。但设计说 container_port 是"required schema field, user_editable"。

**澄清需求**：用户能否在 NBR/Deployment 层修改 container_port？如果能，修改后是否影响 health check URL 和 port mapping？

### 6.2 served_model_name 的默认值

设计建议默认值为 `lightai-{model_slug}`。但当前代码中 `served_model_name` 的默认值逻辑较复杂（从 deployment name → artifact name → model directory name 派生）。

**澄清需求**：默认值规则是否需要简化？

### 6.3 extra_args 与 structured args 的冲突处理

设计说 `extra_args` 不能重复 structured args。但当前 resolver 的 `deduplicateArgs` 只是静默保留最后一个。

**澄清需求**：冲突时应报错、警告还是静默覆盖？

## 7. BackendVersion/BackendRuntime/NBR/Deployment 边界

边界基本清晰，但有以下模糊点：

1. **BackendRuntime 的 `parameter_schema_json` vs BackendVersion 的 `default_args_schema_json`**：两者是否始终一致？如果用户在 BackendRuntime 层修改了 schema（添加/删除参数），NBR 应使用哪个？

2. **NBR 的 `config_snapshot_json` 冻结时机**：当前在 NBR 创建时冻结。如果 BackendRuntime 后续修改了参数，已存在的 NBR 不受影响。这是正确的，但需要文档明确。

3. **Deployment override 的范围**：当前 Deployment 只能 override `parameter_values_json` 和 `env_overrides_json`。设计说还应能 override 部分 Docker 选项。需要明确哪些 Docker 选项允许 Deployment 级覆盖。

## 8. command_template 与 schema fields 的关系

当前 `default_args` 存储 model path（如 `["--model", "{{model_container_path}}"]`），`default_args_schema` 存储所有参数的 schema。resolver Layer 1 使用 `default_args`，Layer 2 使用 `parameter_values_json`。

**关系合理**：`default_args` 是"必须传递的最低参数集"（model path），`default_args_schema` 是"所有可配置参数的 schema"。两者不重复（host/port 从 default_args 移到了 schema）。

**但需要注意**：`default_args` 中的参数（如 `--model`）和 schema 中的 required 参数（如 `--host`）在 resolver 中会合并。如果 `default_args` 中已有 `--model`，schema 中的 `--model` 不应重复添加。当前 resolver 通过 `collectExistingFlags` 去重，这是正确的。

## 9. required/optional/enabled/value 规则可实现性

**可实现**，但需要注意：

1. **required 参数的 value 来源**：可以是静态 default、模板 value、或系统生成。当前 `buildDefaultParamValuesFromSchema` 正确处理了这三种情况。

2. **optional 参数的 value 保留**：当前 `loadFromModel` 在 `RuntimeParameterEditor` 中已实现"不清空 disabled value"的逻辑。但 `BackendRuntimesPage` 的 `buildPayload` 需要确保 disabled value 也被保存。

3. **copy/clone 时的 enabled/value 保留**：当前 `showClone` 使用 `cloneParameterEditorModel` 直接复制整个对象，正确保留了 enabled/value。

## 10. vendor profile 与 backend schema 的组合

**建议方案**：

```
BackendVersion schema = backend-specific serve args (host, port, model path, max_model_len, etc.)
VendorRuntimeProfile = vendor-specific Docker runtime args (devices, env, privileged, etc.)
BackendRuntime = BackendVersion schema + VendorRuntimeProfile defaults
NBR = BackendRuntime + node-specific overrides
```

当前代码中 vendor 信息嵌入在 BackendRuntime 的 `docker_json` 中，没有独立的 VendorRuntimeProfile。这个方案在当前阶段可接受，但需要在 catalog 文件中明确标注哪些字段是 vendor-specific。

## 11. custom args/env/docker options 如何避免绕过结构化参数

**当前状态**：`extra_args` textarea 直接传递原始 CLI 参数，不做冲突检测。

**建议**：
1. Phase 1：在 resolver 的 `deduplicateArgs` 中，如果 `extra_args` 包含与 structured args 相同的 flag，记录 warning log。
2. Phase 2：在 UI 层面，输入 `--host` 到 extra_args 时显示警告。
3. Phase 3：在 preflight 中，检测 extra_args 与 structured args 的冲突，返回结构化错误。

## 12. 参数来源追踪 source map

**必要性**：对于调试和 E2E 验证有价值，但不是 MVP 必需。

**实现建议**：
1. Phase 1：在 resolver 中，每个参数添加 debug log 记录来源。
2. Phase 2：在 RunPlan 的 `debug_info` 字段中添加可选 source map。
3. Phase 3：在 Web UI 中展示参数来源（高级模式）。

## 13. 当前文档遗漏的场景

### 13.1 多实例部署

当前设计未讨论同一模型的多实例部署时，`served_model_name` 是否需要区分实例。

### 13.2 模型热更新

当前设计未讨论模型热更新（切换模型但保持容器运行）时参数如何处理。

### 13.3 参数版本兼容

当 BackendVersion 升级（如 vLLM v0.23.0 → v0.24.0）时，已有 NBR 的 `parameter_values_json` 中的参数是否仍然有效？

### 13.4 参数验证时机

当前 preflight 在启动时验证。但参数在保存时是否也需要验证（如端口范围、内存大小格式）？

### 13.5 并发修改

多个用户同时修改同一个 NBR 的参数时，如何处理冲突？

## 14. 反对意见和替代方案

### 14.1 反对：完全结构化所有参数

设计要求所有参数都结构化（schema fields），但实际中后端参数非常多（vLLM 有 100+ 参数）。完全结构化不现实。

**替代方案**：核心参数（~10 个）结构化，其余通过 `extra_args` 传递，但 `extra_args` 需要与核心参数做冲突检测。

### 14.2 反对：VendorRuntimeProfile 独立实体

当前阶段引入独立 VendorRuntimeProfile 会增加大量重构工作，且 MetaX/Huawei 无法真实验证。

**替代方案**：保持 vendor 信息在 BackendRuntime `docker_json` 中，通过 catalog 文件的 `vendor` 字段区分。后续需要时再引入独立实体。

### 14.3 反对：外置 help 文档

当前阶段引入外置 help 文档会增加维护成本，且大部分参数的帮助文本可以从 `--help` 输出中提取。

**替代方案**：Phase 1 在 schema 中直接用 `help` 字段存储简短帮助文本。Phase 2 引入外置帮助文件。
