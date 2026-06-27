# Parameter Editor Surfaces Matrix

This matrix records which pages/surfaces currently reuse the shared ConfigEdit stack and how they should be tested.

| Surface | Frontend file | Server object_kind | Layer | Shared editor? | Playwright role | Notes |
|---|---|---:|---:|---:|---|---|
| Backend Version | `web/src/pages/BackendsPage.vue` | `backend_version` | `backend_version` | Yes | Thin readonly/edit smoke | Mostly developer/catalog config; not first target. |
| Backend Runtime | `web/src/pages/BackendRuntimesPage.vue` | `backend_runtime` | `backend_runtime` | Yes | First contract surface | Clone from system template to editable user config, then test enabled/value round-trip. |
| Node Backend Runtime | `web/src/pages/RunnerConfigsPage.vue` | `node_backend_runtime` | `node_backend_runtime` | Yes | Second contract surface | Saving should mark `needs_check`; current page should reload edit view after save. |
| Node Runtime Wizard | `web/src/components/deployments/NodeRuntimeConfigWizard.vue` | NBR creation payload | `node_backend_runtime` | Yes | Workflow integration | Verify create/enable + editable_config_patch is applied to NBR snapshot. |
| Deployment Override | `web/src/components/deployments/DeploymentOverrideEditor.vue` | `deployment` | `deployment` | Yes | Third contract surface | Verify deployment override patch, protected-field filtering, and RunPlan preview consistency. |
| Model Artifact | model pages | model artifact/location | model facts/hints | Partial/separate | Negative test | Model page should not expose Docker/runtime launcher params. |
| Legacy RuntimeParameterEditor | `web/src/components/common/RuntimeParameterEditor.vue` | old schema/value | legacy/dev | Not primary | No business coverage | File comment says normal flows use ConfigEditView and semantic projection. |
| HumanRuntimeParameterForm | `web/src/components/runtime/HumanRuntimeParameterForm.vue` | view model | uncertain | Not primary | Audit only | Appears not central; confirm before adding tests. |

## Testing implication

Do not write a standalone full persistence spec for each page. Add one ConfigEdit contract runner and one adapter per surface.

## First three implementation surfaces

1. `BackendRuntime` — safest first target because it can clone a user-managed config without involving node check/deployment.
2. `NodeBackendRuntime` — covers actual deployable runtime config and status side effect.
3. `DeploymentOverride` — covers deployment-level override and protected-field filtering.

## Required shared selectors

Add these to shared components:

```text
config-edit-view
config-edit-section
config-field
config-field-enabled
config-field-value
```

Add these attributes:

```text
data-object-kind
data-layer
data-object-id
data-section-key
data-field-key
data-internal-key
```

Once these exist, most surface adapters should be very small.
