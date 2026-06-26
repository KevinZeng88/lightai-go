AUTORUN

仓库：

`~/projects/ai-platform-study/lightai-go`

请在当前 main 上继续开发，不新建分支。

目标：

基于现有 ConfigSet / config_set_json，设计并实现一套通用的 ConfigEditView 抽象，把内部存储模型与外部用户编辑模型彻底分离。不要继续在每个页面硬编码不同表单。将来新增配置项后，只要 ConfigItem metadata 完整，页面应能自动生成合理的编辑界面。

先阅读：

`docs/reports/phase-3/runtime-template-catalog-redesign/`

并新增/阅读本轮设计文档：

`docs/reports/phase-3/runtime-template-catalog-redesign/config-edit-view-design/`

核心原则：

1. ConfigSet / config_set_json 是内部 canonical storage。
2. UI 不直接编辑 `launcher.xxx` / `runtime.xxx`。
3. UI 渲染 ConfigEditView。
4. UI 保存 ConfigEditPatch。
5. 后端负责 ProjectConfigSetToEditView / ApplyEditPatchToConfigSet / ValidateEditPatch / NormalizeConfigSet。
6. BackendVersion、BackendRuntime、NodeBackendRuntime、Deployment 共用同一套转换逻辑。
7. 大多数可选参数必须有 enabled checkbox。
8. 少数 required 参数默认 enabled=true，且不允许取消启用。
9. object/json 参数普通区必须拆成结构化字段或进入高级原始配置，不得直接作为普通 JSON textarea。
10. Raw ConfigSet 只出现在高级诊断或高级原始配置，默认折叠。
11. copy-on-create 与 RunPlan snapshot-only 规则保持不变。

分组顺序：

```text
010 basic
020 model_serving
030 backend_runtime
040 container_resources
050 devices_mounts
060 environment
070 service
080 health_check
090 advanced_raw
```

字段归组：

- `launcher.image` -> basic
- `launcher.command`, `launcher.entrypoint` -> backend_runtime
- `launcher.docker_options.shm_size`, `privileged`, `ipc_mode`, `uts_mode`, `security_options`, `ulimits` -> container_resources
- `launcher.docker_options.devices`, `optional_devices`, `group_add`, `runtime.model_mount`, `volumes` -> devices_mounts
- `runtime.env` -> environment
- `runtime.health` -> health_check
- `backend.arg.*` -> model_serving by default
- `service.*` / `deployment.service_json` -> service
- `source_metadata.*`, `internal.*`, `resolver.*` -> advanced_raw

后端实现：

新增：

```text
internal/server/configedit/
```

实现：

- types.go
- taxonomy.go
- project.go
- apply.go
- validate.go
- docker_options.go
- configset_adapter.go

至少提供：

- ProjectConfigSetToEditView
- ApplyEditPatchToConfigSet
- ValidateEditPatch
- NormalizeConfigSet

API：

新增或扩展：

```text
POST /api/v1/config-edit/view
POST /api/v1/config-edit/apply
```

支持 object_kind：

- backend_version
- backend_runtime
- node_backend_runtime
- deployment

并让 NBR enable / Deployment create / Deployment preview 支持 `editable_config_patch`。

前端实现：

新增：

```text
web/src/components/config/ConfigEditView.vue
web/src/components/config/ConfigSection.vue
web/src/components/config/ConfigField.vue
web/src/components/config/fields/
web/src/utils/configEditView.ts
```

替换普通编辑入口：

- BackendsPage BackendVersion 参数编辑
- BackendRuntimesPage 运行模板编辑
- NodeRuntimeConfigWizard 节点运行配置编辑
- DeploymentOverrideEditor 部署参数编辑

RuntimeParameterEditor 可保留为过渡 fallback 或高级诊断，不得继续作为普通用户主编辑入口。

运行模板 UX 同步修复：

1. 内置模板 Backend Version 显示 `*`。
2. 运行模板主标题显示 display_name，不显示 raw runtime id。
3. Built-in / User config 清晰区分。
4. 复制为用户配置时弹窗输入 display_name/name。
5. clone API 支持 display_name/name，后端保证 name 唯一。
6. hidden/reference/disabled/template-only/runtime.xxx 不进入普通选择器。

测试要求：

必须运行：

```bash
go build ./cmd/server/...
go build ./cmd/agent/...
go test ./internal/server/...
go test ./internal/agent/...
cd web && npm run build
cd web && npm test
```

新增/更新后端测试：

1. ProjectConfigSetToEditView 不输出 `launcher.xxx`/`runtime.xxx` 作为普通 label。
2. `launcher.docker_options` 被拆成结构化字段。
3. ApplyEditPatchToConfigSet 能把结构化字段合并回 `launcher.docker_options`。
4. required 字段强制 enabled=true。
5. optional 字段有 enabled checkbox。
6. BackendRuntime / NBR / Deployment 使用同一 apply 逻辑。
7. NBR enable 接收 `editable_config_patch` 并落入 `config_set_json`。
8. Deployment override 接收 `editable_config_patch` 并生成 snapshot。
9. protected fields 在 Deployment 层不可覆盖。
10. raw/internal 字段只进入 advanced_raw。

新增/更新前端测试：

1. ConfigEditView section 顺序正确。
2. required 字段不可取消启用。
3. optional 字段显示 enabled checkbox。
4. disabled 字段仍显示值但输入 disabled。
5. Docker options 显示为结构化字段。
6. 普通区不显示 `launcher.docker_options`。
7. 高级原始配置默认折叠。
8. BackendRuntimesPage 使用 ConfigEditView。
9. NodeRuntimeConfigWizard selector 主标题不显示 raw id。
10. clone dialog 传 display_name/name。

文档：

更新：

`docs/reports/phase-3/runtime-template-catalog-redesign/final-closeout.md`

新增章节：

`Post-closeout ConfigEditView abstraction`

写明实现内容、测试结果、commit id、push result、git status。

完成后：

```bash
git status --short
git add .
git commit -m "runtime: add user-facing config edit abstraction"
git push
```

最终只输出：

- PASS/FAIL
- commit id
- push result
- test summary
- closeout path
- remaining blocked items, if any
- git status
