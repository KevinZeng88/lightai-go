# LightAI Go Current State

> Status: CURRENT
> Last reviewed: 2026-06-18
> Scope: Current implementation and documentation entrypoint
> Read order: This file first, then `docs/README.md`

## Branch Baseline

Current branch verified during documentation governance:

```text
main
```

Current relevant baseline:

```text
89bdf68 fix: add node model root policy and harden wizard flow
```

The working tree may contain a user-owned `VERSION` modification. Documentation or code tasks must not touch, stage, or commit that unrelated change.

## Accepted Runtime Chain

The NVIDIA BackendRuntime / RunPlan / Docker lifecycle path has been accepted. Current reports:

```text
docs/reports/backend-runtime-runplan/acceptance-report.md
docs/reports/backend-runtime-runplan/open-issues-closeout.md
```

The model/runtime wizard path has been accepted with documented P2 gaps. Current reports:

```text
docs/reports/model-runtime-node-wizard/acceptance-report.md
docs/reports/model-runtime-node-wizard/full-run-chain-review.md
docs/reports/model-runtime-node-wizard/open-issues-closeout.md
```

## Phase 4 Model/Runtime Wizard State

Phase 4 is now based on scheme B:

```text
Server persists node_model_roots.
Agent keeps denied_roots, path traversal, and symlink escape checks as final protection.
Server passes only an authorized root to Agent browse/scan requests.
```

Default model root policy:

```text
allowed roots default to empty.
The user must explicitly add a node model directory before browse / scan / save.
```

Default denied roots:

```text
/
/etc
/root
/boot
/proc
/sys
/dev
/run
/var/run
/var/lib/docker
```

Root not allowed was fixed by replacing front-end temporary roots with persisted node model roots:

```text
GET    /api/v1/nodes/{node_id}/model-roots
POST   /api/v1/nodes/{node_id}/model-roots
PATCH  /api/v1/nodes/{node_id}/model-roots/{root_id}
DELETE /api/v1/nodes/{node_id}/model-roots/{root_id}
```

Browse / scan / save now use the same path semantics:

```json
{
  "root_id": "node-root-id",
  "root": "/home/kzeng/models",
  "relative_path": "Qwen3-0.6B-Instruct-2512",
  "path_type": "directory"
}
```

The wizard main flow must not require users to hand-enter internal IDs.

## Current Page-To-Docker Chain

The currently validated local NVIDIA single-node flow is:

```text
1. Add node model root.
2. Browse the root.
3. Scan model metadata.
4. Create ModelArtifact.
5. Create ModelLocation.
6. Select Backend.
7. Select BackendVersion.
8. Select BackendRuntime.
9. Enable/check NodeBackendRuntime.
10. Run deployment preflight.
11. Generate Server RunPlan command preview.
12. Start deployment through Agent Docker executor.
13. Verify /v1/models.
14. Read Docker logs through Server -> Agent.
15. Stop deployment.
16. Cleanup resources and release GPU lease.
```

Server command preview must come from the Server RunPlan resolver, not from front-end Docker string concatenation.

## E2E Evidence

Current Phase 4 NVIDIA wizard E2E evidence:

```text
docs/reports/model-runtime-node-wizard/e2e-run-20260618-115214/
```

Result recorded there:

```text
PASS
```

This documentation cleanup did not change runtime code paths and therefore did not rerun E2E.

## Formal Open Issues

Current formal open issue locations:

```text
docs/reports/backend-runtime-runplan/open-issues-closeout.md
docs/reports/model-runtime-node-wizard/open-issues-closeout.md
docs/reports/documentation-governance/open-issues.md
```

Known external/future items:

```text
MetaX real hardware validation remains external validation required.
Huawei vendor adapter remains template-only / future adapter work.
Model consistency deep comparison is documented as P2.
GPU index mapping real multi-GPU validation is documented as product/runtime validation work.
Runtime edit UX, Backend Catalog productization, and Node Runtime tab depth are documented as P2.
```

## Archive Rule

Do not use these directories as current implementation guidance:

```text
docs/archive/
docs/reports/archive/
```

They contain historical plans, superseded designs, closed review records, and old evidence. If an archive document conflicts with this file, this file wins. Restoring an archived design requires a new review.

## Required Reading For Future Agents

1. `docs/CURRENT.md`
2. `docs/README.md`
3. Current topic design under `docs/design/`
4. Current topic acceptance/open issue reports under `docs/reports/<topic>/`

Do not start from archived or historical documents.
