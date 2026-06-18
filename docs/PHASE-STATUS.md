# LightAI Go Development Phase Status

> Status: CURRENT
> Last reviewed: 2026-06-18
> Scope: Current phase status summary
> Read order: See `docs/CURRENT.md`

## Current Baseline

```text
Branch: phase-4-model-runtime-wizards
Latest verified commit for Phase 4: 89bdf68
```

The detailed current entrypoint is `docs/CURRENT.md`.

## Completed Foundations

| Area | Status | Current evidence |
| --- | --- | --- |
| Server / Agent skeleton | Done | Early phase commits and current build/test |
| Auth, tenant, RBAC | Done | `docs/RELEASE_NOTE_v0.1.9.md`, current API |
| Agent registration and heartbeat | Done | Current node/agent APIs |
| System / registry / mock collectors | Done | Current collector tests |
| NVIDIA collector | Done | Current local NVIDIA validation evidence |
| Stable node identity | Done | Current node registration behavior |
| Web console MVP | Done | Current Vue console |
| Observability pages and server metrics | Done | Current reports and scripts |
| Credentials, password reset, file logging, patch tooling | Done | Current scripts and release notes |

## Runtime And Model Serving

| Area | Status | Current evidence |
| --- | --- | --- |
| Backend catalog | Implemented | `configs/backend-catalog/`, `docs/design/backend-runtime-runplan-docker.md` |
| Backend / BackendVersion / BackendRuntime | Implemented | `docs/reports/backend-runtime-runplan/acceptance-report.md` |
| NodeBackendRuntime readiness | Implemented | BackendRuntime reports and E2E evidence |
| RunPlan resolver and Server command preview | Implemented | BackendRuntime reports and Phase 4 E2E |
| Agent Docker lifecycle | Implemented | Docker start/logs/stop/cleanup reports |
| Docker logs through Server -> Agent | Implemented | `docs/reports/backend-runtime-runplan/open-issues-closeout.md` |
| ModelArtifact / ModelLocation | Implemented | `docs/reports/model-runtime-node-wizard/acceptance-report.md` |
| node_model_roots policy | Implemented | `docs/design/model-runtime-node-wizard.md` |
| Model/runtime/deployment wizard | Accepted with P2 gaps | `docs/reports/model-runtime-node-wizard/full-run-chain-review.md` |

## Phase 4 Current State

Phase 4 uses scheme B for model directory safety:

```text
Server persists node_model_roots.
Agent keeps denied_roots and path containment as final protection.
allowed roots default to empty.
Browse / scan / save use root_id + relative_path.
```

Validated local NVIDIA path:

```text
model root -> browse -> scan -> ModelArtifact/Location
-> Backend -> BackendVersion -> Runtime -> preflight
-> Server RunPlan preview -> Docker start -> /v1/models
-> logs -> stop -> cleanup
```

E2E evidence:

```text
docs/reports/model-runtime-node-wizard/e2e-run-20260618-115214/
```

## Formal Open Issues

| Area | Status | Source |
| --- | --- | --- |
| MetaX real hardware validation | DOCUMENTED_BLOCKER | `docs/reports/backend-runtime-runplan/open-issues-closeout.md` |
| Huawei vendor adapter | DOCUMENTED_BLOCKER | `docs/reports/backend-runtime-runplan/open-issues-closeout.md` |
| Model consistency deep comparison | DOCUMENTED_BLOCKER | `docs/reports/model-runtime-node-wizard/open-issues-closeout.md` |
| GPU auto/manual UX and lease display | DOCUMENTED_BLOCKER | `docs/reports/model-runtime-node-wizard/open-issues-closeout.md` |
| Documentation governance findings | Tracked | `docs/reports/documentation-governance/open-issues.md` |

## Current Limitations

1. MetaX real hardware validation is still required before marking MetaX runtime paths ready.
2. Huawei/Ascend runtime remains template-only until a vendor adapter is implemented and verified.
3. TLS/HTTPS is not implemented.
4. Prometheus/Grafana bundled binary availability still depends on local preparation scripts.

## Archive Rule

Older phase reports and RC review artifacts were moved under:

```text
docs/archive/
docs/reports/archive/
```

They are historical evidence only and must not override `docs/CURRENT.md`.
