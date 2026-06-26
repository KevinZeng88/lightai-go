# 11. Batch 1 Registry Normalizer Closeout

> Status: PASS
> Scope: Batch 1 semantic registry and normalizer foundation.
> Date: 2026-06-27

## Summary

Batch 1 adds `internal/server/semanticconfig` with the first common semantic registry and normalizer. This creates the shared source of truth for canonical keys, owners, display tiers, legacy key normalization and direct legacy patch rejection.

## Implemented

| Requirement | Evidence |
| --- | --- |
| Add `internal/server/semanticconfig` | Added `types.go`, `registry.go`, `normalizer.go`, and package tests. |
| Semantic registry | `DefaultRegistry()` registers canonical definitions for runtime image/command/env, service host/port, deployment host/service name, model runtime, health, mount and Docker keys. |
| Owner correctness | `model_runtime.max_model_len` owner is `model_runtime`; `deployment.served_model_name` owner is `deployment_service`. |
| Legacy normalizer | Legacy keys such as `backend.common.port`, `backend.arg.max_model_len`, `backend.common.served_model_name`, and grouped `launcher.docker_options` normalize to canonical semantic keys. |
| Direct legacy patch hard error | `ValidatePatchKeys()` rejects direct legacy keys and unknown canonical keys. |
| Conflict warning | Normalizer emits `conflict` warnings when canonical and legacy aliases provide different values; canonical key value wins. |

## Canonical Coverage Added

- `runtime.image_ref`
- `runtime.command`
- `runtime.entrypoint`
- `runtime.env`
- `service.listen_host`
- `service.container_port`
- `deployment.host_port`
- `deployment.served_model_name`
- `model_runtime.context_length`
- `model_runtime.max_model_len`
- `model_runtime.gpu_memory_utilization`
- `runtime.health`
- `runtime.model_mount`
- `docker.shm_size`
- `docker.ipc_mode`
- `docker.privileged`
- `docker.network_mode`
- `docker.devices`
- `docker.optional_devices`
- `docker.group_add`

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

No unresolved Batch 1 blocker remains.

Batch 1 intentionally does not yet change catalog materialization, ConfigEdit projection, API write paths, RunPlan resolver or web entrypoints. Those are assigned to Batch 2 through Batch 6 in the execution plan, and this closeout records them as implementation scope for those batches rather than open design issues.
