# LightAI Go Documentation

> Status: CURRENT
> Last reviewed: 2026-06-18
> Scope: Documentation entrypoint
> Read order: Start with `docs/CURRENT.md`

This directory is the source of truth for LightAI Go design, acceptance status, and formal open issues.

## Current State

LightAI Go is currently on the Phase 4 model/runtime wizard line. The validated local NVIDIA Docker path is:

```text
node model root -> file browse -> model scan -> ModelArtifact/ModelLocation
-> Backend -> BackendVersion -> BackendRuntime snapshot -> NodeBackendRuntime snapshot
-> preflight -> Server RunPlan preview -> Agent Docker start
-> /v1/models -> Docker logs -> stop -> cleanup
```

Snapshot boundaries are current design:

```text
BackendVersion -> BackendRuntime: copy defaults at creation; independent after creation.
BackendRuntime -> NodeBackendRuntime: copy config at creation/check; independent after creation.
BackendRuntimesPage manages templates only.
RunnerConfigsPage manages NodeBackendRuntime add/edit/check/delete.
Backend / BackendVersion catalog files are the source of truth; DB rows are reload/sync projections.
System BackendVersion catalog: configs/backend-catalog/versions/ (runtime read-only).
User BackendVersion catalog: data/backend-catalog.d/user/ by default, or LIGHTAI_BACKEND_CATALOG_USER_DIR.
BackendVersion user add/edit/clone/delete writes the user catalog file first, then reloads DB.
BackendVersion is hardware/node independent; GPU indexes, device mounts, node host paths, image_present, and ready/needs_check belong outside BackendVersion.
```

Current official system BackendVersion catalog baseline:

```text
vLLM v0.23.0
SGLang v0.5.12.post1 and v0.5.13.post1 (v0.5.13.post1 tag verified by git ls-remote)
llama.cpp b9700 build tag
```

The current node model directory policy is scheme B:

```text
Server persists node_model_roots.
Agent keeps denied_roots, path traversal, and symlink escape checks as final protection.
allowed roots default to empty.
Users must explicitly add a model directory before browse / scan / save.
```

## Recommended Reading Order

1. `docs/CURRENT.md`
2. `docs/design/backend-runtime-runplan-docker.md`
3. `docs/design/runtime-template-node-runtime-snapshot.md`
4. `docs/design/model-runtime-node-wizard.md`
4. `docs/reports/backend-runtime-runplan/acceptance-report.md`
5. `docs/reports/model-runtime-node-wizard/acceptance-report.md`
6. `docs/reports/model-runtime-node-wizard/full-run-chain-review.md`
7. `docs/reports/model-runtime-node-wizard/open-issues-closeout.md`
8. `docs/reports/documentation-governance/cleanup-report.md`

## Current Design Documents

| Document | Purpose |
| --- | --- |
| `docs/design/backend-runtime-runplan-docker.md` | Current Backend / BackendVersion / BackendRuntime / RunPlan Docker design |
| `docs/design/runtime-template-node-runtime-snapshot.md` | BackendVersion/BackendRuntime/NodeBackendRuntime snapshot and user catalog boundary |
| `docs/design/model-runtime-node-wizard.md` | Current model root, model wizard, runtime wizard, and deployment wizard design |
| `docs/design/tenant-rbac-resource-ownership-design.md` | Tenant/RBAC ownership reference |
| `docs/backend-catalog-vendor-extension.md` | Backend catalog vendor extension reference |

## Current Reports

| Report | Purpose |
| --- | --- |
| `docs/reports/backend-runtime-runplan/acceptance-report.md` | BackendRuntime / RunPlan / Docker lifecycle acceptance |
| `docs/reports/backend-runtime-runplan/open-issues-closeout.md` | BackendRuntime formal blockers and external validation items |
| `docs/reports/model-runtime-node-wizard/acceptance-report.md` | Phase 4 wizard acceptance and E2E evidence |
| `docs/reports/model-runtime-node-wizard/full-run-chain-review.md` | Page-to-Docker run chain review |
| `docs/reports/model-runtime-node-wizard/open-issues-closeout.md` | Phase 4 formal open issues closeout |
| `docs/reports/documentation-governance/cleanup-report.md` | Documentation governance closeout for this cleanup |

## Reference Documents

The numbered legacy documents (`00-*.md` through `10-*.md`) remain in place for compatibility with existing agent instructions. They are reference material, not the current Phase 4 execution entrypoint. If they conflict with `docs/CURRENT.md`, use `docs/CURRENT.md`.

GPUStack review documents are reference-only and must not be used to copy or translate GPUStack code.

## Archive Policy

Archived documents live under:

```text
docs/archive/
docs/reports/archive/
```

Archive documents are historical evidence only. Do not use archive documents as current implementation guidance. If an archived document conflicts with `docs/CURRENT.md`, `docs/CURRENT.md` wins.

## Guidance For Future Agents

1. Read `docs/CURRENT.md` first.
2. Use current design documents under `docs/design/`.
3. Use current reports under `docs/reports/<topic>/`.
4. Treat `docs/archive/` and `docs/reports/archive/` as historical evidence only.
5. Do not infer unresolved work from old phase plans; check the topic `open-issues-closeout.md` files.
6. Do not mark MetaX or Huawei runtime paths ready unless they have real hardware validation evidence.

## Backend Runtime Testing

- [Backend Runtime E2E Matrix and Parameter Propagation](testing/backend-runtime-e2e-matrix-and-param-propagation.md)
