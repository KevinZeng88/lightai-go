# 12. Batch 2 Snapshot Builder Closeout

> Status: PASS
> Scope: Config snapshot builder foundation.
> Date: 2026-06-27

## Summary

Batch 2 adds semantic snapshot builder primitives in `internal/server/semanticconfig`. The builder centralizes copy-on-create behavior for BackendVersion -> BackendRuntime, BackendRuntime -> NodeBackendRuntime, and NodeBackendRuntime + ModelArtifact/service input -> Deployment.

## Implemented

| Requirement | Evidence |
| --- | --- |
| BackendVersion -> BackendRuntime | `BuildBackendRuntimeSnapshot()` normalizes source ConfigSet and creates copied semantic items. |
| BackendRuntime -> NodeBackendRuntime | `BuildNodeBackendRuntimeSnapshot()` copies source snapshot and applies canonical values such as `runtime.image_ref`. |
| NBR + ModelArtifact + service input -> Deployment | `BuildDeploymentSnapshot()` adds `deployment.host_port`, `service.container_port`, `deployment.served_model_name`, `model_runtime.context_length`, and `model_runtime.max_model_len`. |
| Copy-on-create lineage | Snapshot items receive `copied_from`, `source_snapshot`, `copied_at`, and `dirty=false`. |
| Downstream edit independence | `ApplyPatch()` clones the snapshot before mutation and marks changed items dirty. Tests assert upstream snapshots are not mutated. |
| `service_json` derived/transition | `DerivedServiceJSON()` derives service response data from semantic snapshot instead of treating service JSON as source authority. |

## Validation

Commands run:

```bash
go test ./internal/server/semanticconfig
```

Result:

```text
ok  	lightai-go/internal/server/semanticconfig	0.001s
```

## Closeout State

No unresolved Batch 2 blocker remains.

API handlers still call their existing ConfigSet copy and service JSON paths until Batch 3 through Batch 5 integrate the semantic builder into projection, validation, RunPlan and web entrypoints.
