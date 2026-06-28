# LightAI Go Runtime Config UI Regression Review

## Source

- Uploaded MHTML snapshot: `LightAI Go.mhtml`
- Snapshot URL: `/runner-configs`
- Visible page state: Node runtime configuration creation wizard, step `镜像与参数`.

## Executive conclusion

The new issues are not one isolated UI label bug. They show that the runtime configuration UI is mixing four different concepts:

1. Runtime template default parameters.
2. Node runtime configuration overrides.
3. Deployment/runtime placeholders such as `{{model_container_path}}` and `{{vendor_visible_devices}}`.
4. Final resolved RunPlan values.

The next repair should be scoped to runtime configuration UI semantics and RunPlan preview recovery, not a full runtime architecture redesign.

## Confirmed from MHTML

### 1. Environment variables section is expanded by default

In the MHTML, the `环境变量 / Environment variables` section is expanded and editable by default.

Observed structure:

- Section title: `环境变量`
- Field: `Environment variables`
- Table columns: `键 / 值`
- Button: `+ 添加行`

Issue:

- Environment variables are advanced runtime options and should not be expanded by default in normal template/NBR creation flow.
- If no user-configured env exists, it should display `未配置` or stay collapsed.
- It should not create a visually empty editable row by default.

Required behavior:

- Section default collapsed, or field displays a compact summary.
- Empty env value means `未配置`, not a blank editable row.
- Raw/key-value editor is shown only in edit mode or after user expands advanced section.

### 2. `Model runtime port` is marked required but disabled/empty

In the MHTML:

- Field: `Model runtime port`
- Tag: `required`
- Control class: `readonly`
- Input is disabled and empty.

Issue:

A required field that cannot be edited and has no visible value is invalid UI semantics.

Likely cause:

- `model_runtime.port` is being shown as a required model runtime parameter.
- But the actual configured service port is represented elsewhere as `service.container_port`.
- The UI currently shows both:
  - `model_runtime.port` as required/read-only/empty.
  - `service.container_port` as an editable port form with value 8000 visible in DOM.

Required behavior:

- Avoid duplicate/conflicting port concepts in the same flow.
- Prefer one canonical user-facing field: `service.container_port`.
- If `model_runtime.port` is an internal placeholder/derived value, hide it from normal UI.
- If it must be shown, it should be read-only with resolved value, not `required`.

### 3. Service port form exists and uses structured widget

In the MHTML:

- `service.container_port` renders as structured port form.
- It shows container port input with `aria-valuenow="8000"`.

This is the likely correct user-facing port field.

Required behavior:

- `service.container_port` should be the canonical normal UI field for container listen port.
- `model_runtime.port` should not compete with it in normal template/NBR flow.

### 4. Environment values from MHTML are not visible in text

The user reports `CUDA_VISIBLE_DEVICES` and `{{vendor_visible_devices}}` are present. The MHTML text extraction did not show those strings, likely because Element Plus input values are not preserved as text in the saved MHTML or the user observed another state/page.

Design conclusion still stands:

- Runtime template should not default-inject device-specific env values.
- `CUDA_VISIBLE_DEVICES={{vendor_visible_devices}}` belongs to final RunPlan device binding resolution, not user-visible runtime template defaults.
- If a placeholder is retained internally, it must be hidden from normal template detail/edit and never presented as a user-editable default env.

### 5. Raw advanced config is still exposed

The MHTML shows `高级原始配置` expanded, with raw values such as:

- `Command ["--model","{{model_container_path}}"]`
- `Entrypoint ["vllm","serve"]`
- `Devices []`
- `Ports []`
- `Volumes []`

Issue:

The previous fix moved some widgets away from raw JSON, but advanced raw config is still visible in the normal wizard step.

Required behavior:

- Raw advanced config should be diagnostic/advanced and collapsed by default.
- Normal user path should show structured summary, not raw arrays.
- Placeholders like `{{model_container_path}}` may exist internally but should not dominate normal flow.

## User-reported issues not fully covered by this MHTML

### 6. Runtime template click behavior: click opens edit directly

User reports:
- Current behavior: clicking a runtime template opens edit.
- Expected behavior: clicking should open view/detail.
- Edit should require explicit `编辑` button.

Required behavior:

- List row click / primary action opens detail/read-only view.
- Detail page/drawer has explicit `编辑` button.
- Edit mode has `保存` and `取消`.
- Save should save and exit edit mode.

### 7. Save/cancel behavior

User reports:
- Runtime template edit has `保存`, but save should save and exit.
- There should be `取消`.

Required behavior:

- Edit mode footer:
  - `保存`: persist changes and exit edit mode / close drawer.
  - `取消`: discard unsaved changes and return to detail/list.
- If save fails, stay in edit mode and show error.
- Do not leave user in ambiguous half-edit state after successful save.

### 8. Model deployment final result shows unsupported runtime_type

User reports model deployment final result:

```text
可运行: No
加载失败: [resolve_error] unsupported runtime_type: (only docker is supported)
```

Issue:

This means the RunPlan resolver is receiving an empty or unsupported `runtime_type`.

Likely causes to check:

- `runtime_type`/launcher kind was lost in ConfigSet projection/save.
- Runtime template/NBR snapshot has no canonical `runtime_type=docker`.
- The deployment flow is using a wrong source field after recent ConfigEditView changes.
- The new structured config editor did not preserve an existing hidden/required runtime type field.

Required behavior:

- Docker runtime templates and NBRs must resolve to `runtime_type=docker`.
- `runtime_type` should be internal/canonical and not user-editable in normal UI.
- RunPlan preview/preflight must fail tests if runtime_type is empty.

### 9. Deployment buttons disappeared

User reports previous deployment buttons are missing, including command preview.

Required behavior:

- Deployment page should retain operational buttons:
  - Preview/查看运行命令
  - Preflight / dry run
  - Start / deploy
  - View logs/diagnostics when applicable
- If buttons are intentionally moved, UI must provide equivalent affordances.
- This should be treated as regression unless a replacement exists.

## Repair scope

### In scope

1. Runtime template/NBR UI semantics:
   - detail vs edit mode
   - save/cancel behavior
   - default collapsed advanced sections
   - env/model port/advanced raw display rules

2. ConfigEditView projection:
   - hide or demote internal/derived placeholders
   - preserve required internal fields such as `runtime_type=docker`
   - avoid duplicate port fields

3. RunPlan/deployment regression:
   - `runtime_type` must resolve to `docker`
   - deployment command preview/actions must be restored

4. Tests:
   - ConfigEditView contract tests
   - RunPlan runtime_type test
   - Frontend UI regression tests for buttons, detail/edit flow, collapsed sections, hidden internal fields

### Out of scope

- Full runtime/config architecture redesign.
- Full Docker parameter model redesign.
- Complete NVIDIA/MIG/MetaX device binding redesign.
- Full deployment UX redesign beyond restoring missing buttons/actions.

## Acceptance criteria

### Runtime template / NBR UI

1. Clicking a runtime template opens detail/read-only view, not edit mode.
2. Detail view has explicit `编辑` button.
3. Edit mode has `保存` and `取消`.
4. Successful save exits edit mode or closes drawer.
5. Environment variables are collapsed by default or shown as compact `未配置`.
6. Empty env does not render a blank editable row in normal view.
7. `CUDA_VISIBLE_DEVICES={{vendor_visible_devices}}` is not shown as a user-editable default runtime template env.
8. `model_runtime.port` is not shown as required + disabled + empty.
9. `service.container_port` remains the canonical visible port field and shows 8000 for vLLM.
10. Advanced raw config is collapsed by default.

### RunPlan / deployment

11. vLLM deployment preflight resolves `runtime_type=docker`.
12. Deployment final result does not show `unsupported runtime_type:`.
13. Command preview / 查看运行命令 or equivalent RunPlan preview action is available.
14. Deployment operational buttons are restored or replaced by equivalent visible actions.

### Tests

15. Go tests cover ConfigEditView visibility/field semantics.
16. Go tests cover RunPlan runtime_type resolution.
17. Frontend tests cover detail vs edit, save/cancel, collapsed env/raw config, no visible vendor placeholder env, required/read-only port regression, and deployment command preview button.
18. `go test ./internal/server/...`, `cd web && npm test`, and `cd web && npm run build` pass.

## Suggested Claude execution prompt

请在当前 main 分支修复 Runtime Config UI/RunPlan 回归，不新建分支。

先阅读：
- docs/reports/phase-3/runtime-config-display-probe-fix/05-closeout.md
- docs/reports/phase-3/runtime-config-display-probe-fix/06-mhtml-config-field-review.md
- docs/reports/phase-3/runtime-config-display-probe-fix/07-config-field-display-design.md
- 当前这份 MHTML review 文档

用户最新发现：
1. 运行模板 Environment variables 不应默认展开，且不应把 `CUDA_VISIBLE_DEVICES={{vendor_visible_devices}}` 作为用户可编辑默认值展示。
2. 运行模板保存后应保存并退出；编辑态应有取消。
3. 运行模板列表点击应查看详情，详情中点击编辑才进入编辑。
4. 节点运行配置“镜像与参数”中 `Model runtime port` 显示 required 且不可编辑/空值，实际应以 `service.container_port` 为 canonical 可见字段。
5. 模型部署最终显示 `[resolve_error] unsupported runtime_type: (only docker is supported)`，必须修复 runtime_type=docker 的解析/保存链路。
6. 模型部署页之前的按钮消失，包括查看运行命令/RunPlan preview 等，需要恢复或提供等价操作。

执行要求：
- 先定位根因，再修改。
- 不做完整架构重构。
- 不扩大到无关 UI 重构。
- 重点修 ConfigEditView 字段可见性、详情/编辑模式、RunPlan runtime_type、部署操作按钮回归。
- 补 Go/API 和前端测试，覆盖上述验收标准。
- 完成后运行：
  - go test ./internal/server/...
  - cd web && npm test
  - cd web && npm run build
- 提交并推送。
- 输出根因、修改文件、测试结果、commit id、push 结果、git status --short。
