# 05 - 实施计划与验收标准

## 执行原则

- 当前分支继续。
- 不做旧数据兼容；必要时允许重建 DB。
- 不在每个页面继续硬编码编辑表单。
- 先实现通用 configedit 层，再替换页面。
- 可定位、可验证的问题直接修复。
- API-first 测试优先。
- UI 测试覆盖主要显示和交互路径。

## Batch 0：现状确认

执行：

```bash
grep -R "RuntimeParameterEditor" -n web/src
grep -R "config_set" -n web/src/pages web/src/components
grep -R "launcher.docker_options" -n .
grep -R "applyConfigOverrides" -n internal/server
```

## Batch 1：后端 configedit 基础

新增 `internal/server/configedit/`，实现：

- ProjectConfigSetToEditView
- ApplyEditPatchToConfigSet
- ValidateEditPatch
- NormalizeConfigSet
- Docker options 拆分和合并
- enabled/required 规则
- section/order 规则

## Batch 2：API 接入

新增或扩展：

```text
POST /api/v1/config-edit/view
POST /api/v1/config-edit/apply
```

接入对象：backend_version、backend_runtime、node_backend_runtime、deployment。并在 NBR enable、deployment create/preview 路径支持 `editable_config_patch`。

## Batch 3：前端 ConfigEditView

新增 `web/src/components/config/` 和 `web/src/utils/configEditView.ts`。实现 string、integer、number、boolean、select、multi_select、string_list、key_value_list、device_list、raw_json。

## Batch 4：替换页面

替换：

- BackendsPage BackendVersion 编辑
- BackendRuntimesPage 用户运行模板编辑
- NodeRuntimeConfigWizard Step 2
- DeploymentOverrideEditor

保留 JsonViewer / raw ConfigSet 只在高级诊断。

## Batch 5：运行模板 UX

修复：

- Backend Version 显示 `*`
- Runtime selector 主标题用 display_name
- Built-in / User config 分组
- clone runtime dialog 输入 display_name/name
- raw id 弱化到 tooltip/高级信息
- hidden/reference/disabled/template-only 不进入普通 selector

## Batch 6：测试

后端测试：

1. `ProjectConfigSetToEditView` 不输出 `launcher.xxx`/`runtime.xxx` 作为普通 label。
2. `launcher.docker_options` 拆出 `shm_size`、`privileged`、`devices`、`group_add` 等字段。
3. `ApplyEditPatchToConfigSet` 能把结构化字段合并回 `launcher.docker_options`。
4. required 字段强制 enabled=true。
5. optional 字段有 enabled checkbox。
6. BackendRuntime / NBR / Deployment 使用同一 apply 逻辑。
7. NBR enable 接收 `editable_config_patch` 并写入 `config_set_json`。
8. Deployment override 接收 `editable_config_patch` 并生成 snapshot。
9. protected fields 在 Deployment 层不可覆盖。
10. hidden/internal 字段只进入 advanced_raw。

前端测试：

1. ConfigEditView section 顺序正确。
2. required 字段无可关闭 checkbox 或 checkbox disabled。
3. optional 字段有 enabled checkbox。
4. disabled 字段仍显示值但输入 disabled。
5. Docker options 显示为结构化字段。
6. 普通区不显示 `launcher.docker_options`。
7. 高级原始配置默认折叠。
8. BackendRuntimesPage 使用 ConfigEditView。
9. NodeRuntimeConfigWizard selector 主标题不显示 raw id。
10. clone dialog 传 display_name/name。

常规验证：

```bash
go build ./cmd/server/...
go build ./cmd/agent/...
go test ./internal/server/...
go test ./internal/agent/...
cd web && npm run build
cd web && npm test
```

## Batch 7：文档和 closeout

更新：

```text
docs/reports/phase-3/runtime-template-catalog-redesign/final-closeout.md
```

新增章节：

```text
Post-closeout ConfigEditView abstraction
```

必须写明实现内容、测试结果、commit id、push result、git status。
