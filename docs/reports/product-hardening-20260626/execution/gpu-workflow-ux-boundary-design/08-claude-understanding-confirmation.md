# 08 — Claude Understanding Confirmation: GPU Workflow UX Boundary

Generated: 2026-06-26 | Design package: `gpu-workflow-ux-boundary-design/`

This report confirms that I have read and understood the design package before implementing any code changes.

---

## 1. Three Lines Understanding

I understand the product has three separate but related workflows:

### Model Line (模型线)

**Core objects:** `ModelArtifact` (model identity + facts), `ModelLocation` (model path on a specific node).

**User goal:** "Which model files exist on which node, and what model facts can the platform detect?"

**Flow:** Model Library → Add/scan model → Select node (where files are) → Browse filesystem → Select directory/file → Scan → Confirm detected facts → Save.

**Key constraint:** Model pages describe model facts and locations. They must NOT configure how the model is served. No Docker images, no backend serve args, no GPU devices, no ports, no `launcher.*`.

### Runtime Line (运行线)

**Core objects:** `Backend` (inference backend family), `BackendVersion` (capability definition), `BackendRuntime` (runtime template), `NodeBackendRuntime` (template enabled + checked on a specific node).

**User goal:** "Can this node run models with this GPU/backend environment, and what runtime options should be used?"

**Flow:** Runtime Templates (browse) → Node Runtime Configs (new) → Select node → Select template → Name config → Configure image + params → Save and Check → Review readiness.

**Key constraint:** Runtime pages configure how a node runs models. They must NOT expose resolver internals, template placeholders, or raw ConfigSet keys as ordinary user fields.

### Deployment Line (部署线)

**Core objects:** `ModelDeployment` (saved definition), `ResolvedRunPlan` (frozen runtime spec), `ModelInstance` (actual running/stopped instance).

**User goal:** "Run this model with this ready node runtime config, expose a service, and verify the final run plan before starting."

**Flow:** Deployments → New → Select model → Select ready NBR → Configure service + overrides → Preview Run Plan → Save or start.

**Key constraint:** This is where Model and Runtime lines converge. Only `ready`/`ready_with_warnings` NBRs are selectable. Preview uses `/deployments/preview`. Payload uses `node_backend_runtime_id` only.

---

## 2. Why Model and Runtime Lines Cannot Merge

Both lines reference `Node`, but their node-selection semantics are completely different:

| Aspect | Model Line node selection | Runtime Line node selection |
|---|---|---|
| **Question** | "Where are my model files?" | "Where should I set up the runtime?" |
| **Context** | Filesystem browsing, model scanning | GPU/backend environment verification |
| **Result** | `ModelLocation` record | `NodeBackendRuntime` record |
| **Relationship** | Many models can exist on one node | Many runtimes can be enabled on one node |

Merging them into one wizard would create a single flow where the user must simultaneously decide where files live AND how the runtime is configured. That conflates two independent decisions. A user might scan models on node A, then enable a runtime on node B, then deploy on whichever node has both a model location AND a ready NBR. The deployment wizard is where this convergence happens — not before.

---

## 3. Top 10 Current UX Problems

Based on document `02-current-ux-problems-and-root-causes.md` and my own inspection:

1. **Raw ConfigSet keys shown as normal fields** — `launcher.command`, `launcher.args`, `{{MODEL_CONTAINER_PATH}}`, `runtime_env.*` rendered by `RuntimeParameterEditor` without user-facing filter/mapping.
2. **Runtime Templates page shows catalog records** — table displays `backend_id`, `backend_version_id`, raw IDs instead of user-facing names like `nvidia.vllm`.
3. **Wizard state persists after cancel** — reopening the wizard resumes from the old step instead of starting fresh.
4. **No config name field** — NodeBackendRuntime gets created without user-provided identity; users can't name their config.
5. **Save/check failure closes the wizard** — enable failure or check-request failure emits `saved` and the parent closes the dialog, losing error context.
6. **Model Library uses dropdown for node selection** while Node Runtime Configs use a table — inconsistent UX, same data, different component.
7. **Too many templates shown** — catalog records that look like implementation variants are presented as separate choices instead of grouped under a single user-facing template.
8. **ConfigSet / Source Metadata shown as primary drawer content** — advanced JSON diagnostics presented too prominently, not collapsed.
9. **Raw IDs as primary labels in tables** — `backend_runtime_id`, `source_node_backend_runtime_id` shown instead of resolved display names.
10. **Non-deployable NBRs were selectable** (already fixed in `0bc4fa3`, but the design validates this requirement).

---

## 4. Page Redesign Plans

### Runtime Templates Page (BackendRuntimesPage)

**Target:** Show a curated user-facing template catalog.

- Table columns: Display Name (`nvidia.vllm`), GPU Vendor, Backend, Version, Default Image, Supported Formats, Ready Node Count, Managed By.
- Group/name templates as `<vendor>.<backend> [version]`.
- Hide raw `backend_id`, `backend_version_id`, internal ConfigSet keys.
- Move ConfigSet + Source Metadata to collapsed "Advanced Diagnostics" drawer section.
- System templates read-only; user-managed templates editable.

**New files likely:** `web/src/utils/runtimeDisplay.ts`, `web/src/components/runtime/RuntimeTemplateCard.vue`.

### Node Runtime Configs Page (RunnerConfigsPage)

**Target:** Guided enable-and-verify wizard.

- Step 1: Select node (use shared `NodeSelectorTable`, label: "选择运行节点").
- Step 2: Select runtime template (cards/table with `nvidia.vllm` style names).
- Step 3: Config name (auto-generated default: `<hostname> / <vendor> / <backend>`), image, human-facing runtime parameters (shared memory, GPU settings, backend-specific fields). Internal ConfigSet keys hidden.
- Step 4: Save & check summary. Save failure stays open. Check failure stays open. Not-ready stays open. Ready/ready_with_warnings enables Finish.

**New files likely:** `web/src/components/runtime/HumanRuntimeParameterForm.vue`.

### Model Library Page (ModelArtifactsPage)

**Target:** Consistent node selection for model scanning.

- Replace node dropdown with shared `NodeSelectorTable`.
- Label: "选择模型所在节点" (select the node where model files are stored).
- Keeps file browser, scan, and confirm flow.
- Does NOT show Docker/runtime/GPU serve parameters.

**New files likely:** `web/src/components/common/NodeSelectorTable.vue`.

### Model Deployments Page (ModelDeploymentsPage)

**Target:** Safe model + runtime convergence with preview.

- Step 1: Select model (ModelArtifact).
- Step 2: Select NBR — only `ready`/`ready_with_warnings` selectable. Others visible but disabled.
- Step 3: Service settings (host port, container port, served model name).
- Step 4: Preview Run Plan via `/deployments/preview`.
- Step 5: Save / Start. Errors keep dialog open.

**Already partially fixed in earlier commits** (`e01acd8`, `0bc4fa3`). Remaining: compatibility check (model location on same node as NBR), error behavior.

### Model Instances Page (ModelInstancesPage)

**Target:** Observable runtime state.

- Show readable labels (not raw IDs).
- Show model, deployment, node, backend, status, start time, health, logs.
- Hide raw JSON as primary view.

**Already solid.** Minor label improvements only.

---

## 5. ConfigSet Key Hiding Strategy

### Internal keys to hide from normal UI

These must NOT appear in ordinary forms (only in "Advanced Diagnostics" / "Raw JSON" collapsed sections):

```
launcher.command
launcher.args
launcher.docker_options (internal sub-keys)
launcher.entrypoint
runtime_env.*
internal.*
resolver.*
source_metadata.*
{{MODEL_CONTAINER_PATH}}
{{MODEL_CONTAINER_DIR}}
raw ConfigSet item codes
```

### Human-readable parameter mapping

I will map internal ConfigSet paths to human-facing fields using a view model adapter (`HumanRuntimeField`):

| Human Field | Internal Key | Group |
|---|---|---|
| Shared Memory | `launcher.docker_options.shm_size` or `docker.shm_size` | Basic |
| Container Memory Limit | `launcher.docker_options.memory` | Basic |
| CPU Limit | `launcher.docker_options.cpus` | Basic |
| Health Check Timeout | `runtime.health.timeout_seconds` or `health.check.timeout` | Basic |
| GPU Memory Utilization | `backend.arg.gpu_memory_utilization` → `--gpu-memory-utilization` | Backend (vLLM) |
| Max Model Length | `backend.arg.max_model_len` → `--max-model-len` | Backend (vLLM) |
| Max Num Seqs | `backend.arg.max_num_seqs` → `--max-num-seqs` | Backend (vLLM) |
| SGLang Memory Fraction | `backend.arg.mem_fraction_static` → `--mem-fraction-static` | Backend (SGLang) |
| Context Length | `backend.arg.context_length` → `--context-length` | Backend (SGLang) |
| llama.cpp Context Size | `backend.arg.ctx_size` → `--ctx-size` | Backend (llama.cpp) |
| llama.cpp GPU Layers | `backend.arg.n_gpu_layers` → `--n-gpu-layers` | Backend (llama.cpp) |
| Served Model Name | `backend.common.served_model_name` → `--served-model-name` | Backend Common |

The `HumanRuntimeParameterForm` component will:
1. Accept a `ConfigSet` input.
2. Map known internal keys to human fields via `runtimeParameterViewModel.ts`.
3. Produce `RuntimeParamFormOutput` with patches for `config_set`, `parameter_values`, `docker_options`, `env`.
4. Keep unknown/internal keys untouched (preserved but not rendered).

---

## 6. Wizard State Reset and Error Handling

### State reset

Every time the wizard dialog opens:

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

Use `destroy-on-close` on the dialog, or watch `createVisible` to trigger reset.

### Config name

Auto-generated default: `<node hostname> / <vendor> / <backend>` — e.g., `KZ-LAPTOP / NVIDIA / SGLang`.

Field is user-editable, saved as `display_name`.

### Save/check error state machine

```
idle → saving → save_failed (stay open, show error)
     → saving → saved_needs_check
              → checking → check_failed (stay open, show error)
              → checking → checked_not_ready (stay open, show status + hints)
              → checking → checked_ready (allow finish, list refresh)
```

Do NOT emit `saved`/`completed` on failure or non-ready. Only emit when user explicitly clicks Finish on a ready/ready_with_warnings status.

---

## 7. Shared NodeSelectorTable

Both Model Library and Node Runtime Configs select a node, but for different purposes. A shared component satisfies both:

```vue
<NodeSelectorTable
  :label="$t('nodeSelector.selectModelNode')"  <!-- or 'selectRuntimeNode' -->
  :nodes="nodes"
  @select="onNodeSelected"
/>
```

The label and context are passed as props. The table itself is shared.

This fixes the inconsistency (dropdown vs table) while keeping the two business lines separate.

---

## 8. Files to Modify (by Commit)

### Commit 1 — Documentation review
```
docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-review.md  (CREATE)
```

### Commit 2 — Shared selectors + wizard state hygiene
```
web/src/components/common/NodeSelectorTable.vue             (CREATE)
web/src/components/deployments/NodeRuntimeConfigWizard.vue  (MODIFY: reset, config name, error handling)
web/src/pages/ModelArtifactsPage.vue                        (MODIFY: use NodeSelectorTable)
web/src/pages/RunnerConfigsPage.vue                         (MODIFY: use NodeSelectorTable, destroy-on-close)
web/tests/runtimeBoundaryUi.test.mjs                        (MODIFY: new tests)
web/tests/nodeSelector.test.mjs                             (CREATE)
```

### Commit 3 — Runtime template presentation model
```
web/src/pages/BackendRuntimesPage.vue                       (MODIFY: display adapter)
web/src/utils/runtimeDisplay.ts                             (CREATE)
web/src/components/runtime/RuntimeTemplateCard.vue          (CREATE: optional)
web/src/components/runtime/RuntimeTemplateDetails.vue       (CREATE: optional)
web/src/api/runtimes.ts                                     (MODIFY: display helper)
web/src/locales/zh-CN.ts                                    (MODIFY: new keys)
web/src/locales/en-US.ts                                    (MODIFY: new keys)
web/tests/runtimeTemplates.test.mjs                         (CREATE)
```

### Commit 4 — Human-facing runtime parameter form
```
web/src/components/runtime/HumanRuntimeParameterForm.vue    (CREATE)
web/src/utils/runtimeParameterViewModel.ts                  (CREATE)
web/src/components/deployments/NodeRuntimeConfigWizard.vue  (MODIFY: use HumanRuntimeParameterForm)
web/src/components/common/RuntimeParameterEditor.vue        (MODIFY: keep for advanced diagnostics only, hide internal keys)
web/tests/runtimeParamForm.test.mjs                         (CREATE)
```

### Commit 5 — Deployment compatibility + error polish
```
web/src/components/deployments/DeploymentWizard.vue         (MODIFY: compatibility check)
web/src/components/deployments/DeploymentPreviewPanel.vue   (MODIFY: error display)
web/tests/deploymentCompatibility.test.mjs                  (CREATE)
```

### Commit 6 — Tests, evidence, closeout
```
docs/reports/product-hardening-20260626/evidence/<TS>/gpu-workflow-ux-boundary/  (CREATE)
```

---

## 9. Tests to Add/Update

**New test files:**

| Test File | Coverage |
|---|---|
| `web/tests/nodeSelector.test.mjs` | NodeSelectorTable rendering, label context |
| `web/tests/runtimeTemplates.test.mjs` | Display name generation, internal key hiding |
| `web/tests/runtimeParamForm.test.mjs` | Human fields rendered, internal keys hidden, shm_size, GPU fields, output adapters |
| `web/tests/deploymentCompatibility.test.mjs` | Model location + NBR node match check |

**Existing tests to update:**

| Test File | Change |
|---|---|
| `web/tests/runtimeBoundaryUi.test.mjs` | Add assertions for wizard reset, config name, internal key hiding |
| `web/tests/i18nKeys.test.mjs` | Updated key counts after new i18n additions |
| `web/tests/modelCapabilities.test.mjs` | Verify NodeSelectorTable used instead of dropdown |

**Go tests:** Keep existing; add deployment preview compatibility checks if not already covered.

---

## 10. Out-of-Scope Confirmation

I confirm that this round does NOT implement:

- ❌ OpenAI Gateway (`/v1/models`, `/v1/chat/completions`)
- ❌ API Key management (`/api/v1/api-keys`)
- ❌ Usage Metering (`gateway_usage_records` table)
- ❌ Billing
- ❌ Historical compatibility migration for old DB data
- ❌ Multi-tenant quota redesign
- ❌ Kubernetes/Ray scheduler

These are deferred to future work. The `future-openai-gateway-notes.md` document captures the gateway design for later.

---

## 11. Questions / Risks

### Q1: ConfigSet mapping completeness
The `HumanRuntimeParameterForm` must map from internal ConfigSet keys to human fields. The exact set of internal keys in each backend's ConfigSet may vary. **Risk:** Some backends may have keys not covered by the initial mapping. **Mitigation:** Start with the documented keys for vLLM/SGLang/llama.cpp. Any unmapped key goes to "Advanced Diagnostics" (visible but not the primary form). Incrementally add mappings as gaps are found during manual verification.

### Q2: Runtime template grouping
The design says to group catalog variants under a single user-facing template name (`nvidia.vllm`). The catalog currently has separate `BackendRuntime` records for each version. **Risk:** Grouping logic may be fragile if new backend versions are added. **Mitigation:** Derive the display name from `vendor` + `backend_id` + `version` fields, not hardcoded mappings. If the catalog contains only one version per vendor/backend pair, no grouping needed — just present the display name.

### Q3: NodeSelectorTable in ModelArtifactsPage
The Model Library currently loads nodes differently than the RunnerConfigsPage. **Risk:** Changing the component while keeping the scan/upload flow intact may require adjusting the page-level data flow. **Mitigation:** The table replaces only the node dropdown in the add-model wizard. The file browser, scan, and confirm flow remain unchanged. Pass nodes as a prop.

### Q4: Wizard state reset vs. parent dialog management
The `destroy-on-close` approach may conflict with the `saved` emit pattern in `NodeRuntimeConfigWizard`. **Risk:** If the wizard emits `saved` and the parent closes the dialog, the `destroy-on-close` will trigger. But the wizard also needs to stay open on non-ready. **Mitigation:** Rename the emit from `saved` to `completed` (or `finished`), and only emit it from the explicit Finish button when the NBR is ready/ready_with_warnings. Never auto-emit on save.

### R5: No risk requiring immediate code change.

---

UNDERSTANDING_READY_FOR_REVIEW
