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
-> Backend -> BackendVersion -> BackendRuntime -> NodeBackendRuntime
-> preflight -> Server RunPlan preview -> Agent Docker start
-> /v1/models -> Docker logs -> stop -> cleanup
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
3. `docs/design/model-runtime-node-wizard.md`
4. `docs/reports/backend-runtime-runplan/acceptance-report.md`
5. `docs/reports/model-runtime-node-wizard/acceptance-report.md`
6. `docs/reports/model-runtime-node-wizard/full-run-chain-review.md`
7. `docs/reports/model-runtime-node-wizard/open-issues-closeout.md`
8. `docs/reports/documentation-governance/cleanup-report.md`

## Current Design Documents

| Document | Purpose |
| --- | --- |
| `docs/design/backend-runtime-runplan-docker.md` | Current Backend / BackendVersion / BackendRuntime / RunPlan Docker design |
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
