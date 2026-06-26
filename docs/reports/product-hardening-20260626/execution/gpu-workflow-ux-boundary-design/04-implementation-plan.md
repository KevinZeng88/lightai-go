# 04 — Implementation Plan

## 1. Execution policy

- Continue on current branch.
- Do not create a new branch unless explicitly requested.
- Keep DB/API/schema clean; do not add legacy compatibility fallback.
- Do not implement Gateway/API Key/Usage/Billing in this round.
- Prefer view-model adapters and UI component boundaries over exposing raw ConfigSet.
- Every fix must include tests and evidence.

---

## 2. Recommended commit plan

### Commit 1 — Documentation and workflow boundary review

Create:

```text
docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-review.md
```

Content:

- Three-line model: Model line / Runtime line / Deployment line
- Current UI mapping
- Current leakage points
- Target UX
- Implementation plan
- Acceptance contract

### Commit 2 — Shared selectors and wizard state hygiene

Add or refactor:

```text
web/src/components/common/NodeSelectorTable.vue
web/src/components/deployments/NodeRuntimeConfigWizard.vue
web/src/pages/ModelArtifactsPage.vue
web/src/pages/RunnerConfigsPage.vue
```

Tasks:

- Add shared NodeSelectorTable.
- Use it in Model Library wizard and Node Runtime Config wizard.
- Reset NodeRuntimeConfigWizard on every open.
- Add `destroy-on-close` to relevant dialogs or expose/reset wizard state.
- Add config name field and default generation.
- Fix save/check lifecycle so failed operations keep the dialog open.

### Commit 3 — Runtime template presentation model

Modify:

```text
web/src/pages/BackendRuntimesPage.vue
web/src/api/runtimes.ts
web/src/locales/zh-CN.ts
web/src/locales/en-US.ts
```

Possible new files:

```text
web/src/utils/runtimeDisplay.ts
web/src/components/runtime/RuntimeTemplateCard.vue
web/src/components/runtime/RuntimeTemplateDetails.vue
```

Tasks:

- Create runtime template display name: `<vendor>.<backend> [version]`.
- Group or visually simplify duplicate/internal catalog variants.
- Hide raw IDs from primary table.
- Move ConfigSet and source metadata to collapsed Advanced Diagnostics.
- Keep system built-in templates read-only.
- Keep user-managed templates editable.

### Commit 4 — Human-facing runtime parameter form

Add:

```text
web/src/components/runtime/HumanRuntimeParameterForm.vue
web/src/utils/runtimeParameterViewModel.ts
```

Modify:

```text
web/src/components/deployments/NodeRuntimeConfigWizard.vue
web/src/components/common/RuntimeParameterEditor.vue
```

Tasks:

- Do not show all ConfigSet items in normal user forms.
- Map internal config items to human fields.
- Hide internal keys by default.
- Preserve advanced diagnostics with raw ConfigSet.
- Support backend-specific fields for vLLM, SGLang, llama.cpp.

Recommended human parameter view model:

```ts
type HumanRuntimeField = {
  key: string
  label: string
  group: 'basic' | 'gpu' | 'backend' | 'advanced'
  backend?: 'vllm' | 'sglang' | 'llamacpp' | 'common'
  type: 'string' | 'number' | 'boolean' | 'select' | 'list' | 'kv' | 'volume' | 'port'
  placeholder?: string
  unit?: string
  defaultValue?: unknown
  value?: unknown
  enabled?: boolean
  required?: boolean
  help?: string
  mapsTo: Array<{
    internalKey: string
    target: 'config_set' | 'parameter_values' | 'docker' | 'env' | 'service_json'
  }>
  visibility?: {
    backend?: string[]
    vendor?: string[]
  }
}
```

Initial field mapping examples:

```text
shared memory          → docker.shm_size or equivalent config key
health timeout         → health.check.timeout or equivalent
GPU memory utilization → vllm gpu-memory-utilization
SGLang memory fraction → sglang mem-fraction-static
llama ctx size         → llama.cpp ctx-size
llama n gpu layers     → llama.cpp n-gpu-layers
```

Internal keys hidden from normal form:

```text
launcher.command
launcher.args
launcher.*
runtime_env.*
internal.*
resolver.*
source_metadata
{{MODEL_CONTAINER_PATH}}
{{MODEL_CONTAINER_DIR}}
```

### Commit 5 — Deployment compatibility and error behavior polish

Modify:

```text
web/src/components/deployments/DeploymentWizard.vue
web/src/components/deployments/NodeRuntimeSelector.vue
web/src/components/deployments/DeploymentPreviewPanel.vue
web/src/pages/ModelDeploymentsPage.vue
```

Tasks:

- Keep non-deployable NBR visible but disabled.
- Verify model location and NBR node compatibility before save/start.
- Improve preview error display.
- Keep dialog open on errors.
- Ensure preview uses `/deployments/preview`.
- Ensure payload uses `node_backend_runtime_id` only.

### Commit 6 — Tests, evidence, and closeout

Update tests and create evidence report.

---

## 3. Page-level technical notes

### BackendRuntimesPage

Current page uses raw runtime rows. Add a display adapter:

```ts
function toRuntimeTemplateDisplay(row: any): RuntimeTemplateDisplay {
  return {
    id: row.id,
    displayName: formatRuntimeTemplateName(row),
    vendor: row.vendor,
    backend: row.backend_id || row.backend?.name,
    version: formatVersion(row),
    image: row.image_ref,
    formats: extractSupportedFormats(row),
    readyCount: row.deployable_count,
    managedBy: row.is_editable ? 'user' : 'system',
    raw: row,
  }
}
```

Use displayName in main table. Raw JSON stays in diagnostics.

### NodeRuntimeConfigWizard

Add state reset:

```ts
function resetWizard() {
  activeStep.value = 0
  selectedNode.value = null
  selectedRuntime.value = null
  form.display_name = ''
  form.image_ref = ''
  paramOverrides.value = {}
  checkResult.value = null
  error.value = ''
}
```

Parent should either:

```vue
<el-dialog destroy-on-close ...>
```

or call reset on open/close.

### Save/check lifecycle

Replace auto-close pattern with explicit state machine:

```text
idle
saving
save_failed
saved_needs_check
checking
check_failed
checked_not_ready
checked_ready
```

Emit `completed` only when user clicks Finish or when the business flow clearly reaches ready.

### HumanRuntimeParameterForm

Inputs should produce a payload adapter result, not raw displayed fields:

```ts
type RuntimeParamFormOutput = {
  config_set_patch?: Record<string, any>
  parameter_values?: Array<{key: string; value: any; enabled: boolean}>
  docker_options?: Record<string, any>
  env?: Record<string, string>
  service_json?: Record<string, any>
}
```

### ModelArtifactsPage

Replace node dropdown with shared NodeSelectorTable.

The label should be model-specific:

```text
选择模型所在节点
```

### DeploymentWizard

Add compatibility explanation:

```text
This model must have a location on the same node as the selected node runtime config.
```

If incompatible:

```text
The selected runtime is on node X, but this model has no verified location on node X.
```

---

## 4. i18n requirements

All user-facing strings must go into zh-CN and en-US.

Required new keys include:

```text
runtimeTemplates.userName
runtimeTemplates.vendor
runtimeTemplates.backend
runtimeTemplates.version
runtimeTemplates.supportedFormats
runtimeTemplates.advancedDiagnostics
runtimeTemplates.systemBuiltin
runtimeTemplates.userManaged

nodeSelector.selectModelNode
nodeSelector.selectRuntimeNode
nodeSelector.hostname
nodeSelector.agentStatus
nodeSelector.gpuSummary
nodeSelector.refresh

runnerConfigs.configName
runnerConfigs.autoGeneratedName
runnerConfigs.basicSettings
runnerConfigs.gpuSettings
runnerConfigs.backendSettings
runnerConfigs.advancedSettings
runnerConfigs.sharedMemory
runnerConfigs.healthTimeout
runnerConfigs.visibleGpus
runnerConfigs.saveOnly
runnerConfigs.saveAndCheck
runnerConfigs.finish
runnerConfigs.checkPending
runnerConfigs.checkFailed
runnerConfigs.checkNotReady
runnerConfigs.checkReady
runnerConfigs.internalDiagnostics
runnerConfigs.hideInternalFields

runtimeParams.vllmGpuMemoryUtilization
runtimeParams.vllmMaxModelLen
runtimeParams.sglangMemFractionStatic
runtimeParams.sglangContextLength
runtimeParams.llamaCtxSize
runtimeParams.llamaGpuLayers

deployments.nodeMismatch
```

---

## 5. Avoid these implementation mistakes

- Do not simply rename internal keys and keep showing them.
- Do not hide errors by closing dialogs.
- Do not make Model Library configure runtime args.
- Do not make Runtime Templates depend on a model path.
- Do not allow deployment with `backend_runtime_id`.
- Do not allow non-ready NBR deployment.
- Do not let raw JSON become the primary UI.
- Do not create compatibility fallback for old DB/data.

