# GPU Workflow UX Boundary Audit

Generated: 2026-06-26 | Status: audit — no code changes

## 1. Three Lines Model

| Line | Objects | User Goal | Pages |
|---|---|---|---|
| Model | ModelArtifact, ModelLocation | Find model files and facts | ModelArtifactsPage |
| Runtime | Backend, BackendVersion, BackendRuntime, NodeBackendRuntime | Enable and verify backend/GPU runtime on a node | BackendRuntimesPage, RunnerConfigsPage |
| Deployment | ModelDeployment, ResolvedRunPlan, ModelInstance | Combine model + ready runtime → service | ModelDeploymentsPage, ModelInstancesPage |

## 2. Current UI → Object Mapping

### Model Line

| Page | Component | Objects shown | Status |
|---|---|---|---|
| ModelArtifactsPage | table + create dialog | ModelArtifact (display_name, format, task_type, path, capabilities), ModelLocation (node, path, verify status) | Mostly correct. Node selector uses dropdown — should use shared table. |

### Runtime Line

| Page | Component | Objects shown | Leaks |
|---|---|---|---|
| BackendRuntimesPage | table + drawer | BackendRuntime (name, backend_id, backend_version_id, vendor, image_ref) | `backend_id` and `backend_version_id` shown as raw IDs; `ConfigSet` and `Source Metadata` shown as primary drawer content; templates not grouped by user-facing name |
| RunnerConfigsPage | table + wizard | NodeBackendRuntime (display_name, node_id, backend_runtime, image_ref, status) | NBR list is OK after previous fixes; wizard passes raw `config_set` to RuntimeParameterEditor which renders all ConfigSet items including `launcher.*` and `runtime_env.*` |

### Deployment Line

| Page | Component | Objects shown | Status |
|---|---|---|---|
| ModelDeploymentsPage | table + wizard | ModelDeployment, NBR, preview | Wizard flow OK after previous fixes. NBR deployability gate works. Preview calls `/deployments/preview`. |
| ModelInstancesPage | table + drawer | ModelInstance (id, deployment_id, node_id, state, container, port, logs) | Generally correct. Some raw IDs as labels. |

## 3. ConfigSet/Internal Key Leakage Points (Active Issues)

| Location | Leak | Severity |
|---|---|---|
| `RuntimeParameterEditor.vue` | Renders ALL `config_set.items` including `launcher.command`, `launcher.args`, `launcher.*`, `runtime_env.*`, `model_runtime.*` as user-editable fields | **HIGH** |
| `NodeRuntimeConfigWizard.vue:170` | Passes raw `{ config_set: selectedRuntime.value.config_set }` to RuntimeParameterEditor — all internal keys exposed in Step 3 | **HIGH** |
| `BackendRuntimesPage.vue:13,37` | Table shows `backend_id`, `backend_version_id` as raw IDs. Drawer shows `ConfigSet` and `Source Metadata` as primary content | **MEDIUM** |
| `BackendRuntimesPage.vue:55` | `title="ConfigSet"` hardcoded — internal term shown to users | **MEDIUM** |
| `ModelArtifactsPage.vue` | Node selector uses `el-select` dropdown — inconsistent with other pages that use table | **LOW** |

## 4. Already Fixed (Previous Rounds)

| Issue | Fix Commit |
|---|---|
| RunnerConfigsPage thin dialog → wizard | `e01acd8` |
| DeploymentWizard empty states + deployable filter | `e01acd8` |
| Non-deployable NBR selection blocked | `0bc4fa3` |
| Naming cleanup (ConfigSet/RunPlan/NBR in labels) | `ee53d67` |
| Release packaging: config-registry missing | `6cd8a08` |

## 5. Target UX Summary

| Page | Current Problem | Target |
|---|---|---|
| Runtime Templates | Raw catalog records with internal IDs | User-facing names (`nvidia.vllm`). ConfigSet in collapsed Advanced Diagnostics. |
| Node Runtime Configs | Internal ConfigSet keys in wizard Step 3 | HumanRuntimeParameterForm with mapped fields. Internal keys hidden. |
| Model Library | Dropdown node selector | Shared NodeSelectorTable with model-location context label. |
| Model Deployments | — (already fixed) | Keep invariants: deployable-only NBR, preview endpoint, node_backend_runtime_id payload. |
| Model Instances | — (mostly OK) | Readable labels, raw JSON minimized. |

## 6. Implementation Order

1. NodeSelectorTable (shared component) + wizard reset + config naming → Commit 2
2. Runtime template display adapter → Commit 3
3. HumanRuntimeParameterForm + view model adapter → Commit 4
4. Model Library node selector swap + deployment compatibility → Commit 5
5. Tests + evidence + closeout → Commit 6
