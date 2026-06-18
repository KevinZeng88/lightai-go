# LightAI Go Current State

> Status: CURRENT
> Last reviewed: 2026-06-18
> Scope: Current implementation and documentation entrypoint
> Read order: This file first, then `docs/README.md`

## Branch Baseline

Current branch and commit verified during 2026-06-18 real-machine verification:

```text
Branch: main
Commit: 48ee190 fix: harden model smoke test resolution and fallback
```

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

## BackendRuntime / NodeBackendRuntime Boundary

Formal design for template vs node-level runtime config:

```text
docs/design/runtime-template-node-runtime-snapshot.md
```

Key rules:

```text
BackendRuntime = template (no node binding).
NodeBackendRuntime = node-level config with frozen config_snapshot_json.
NBR snapshot captured at enable/check time (args, env, docker, mounts, health_check).
RunPlan resolver reads NBR snapshot, not live BackendRuntime.
BackendRuntime template edits do NOT affect existing NodeBackendRuntime RunPlans.
Editing NodeBackendRuntime image/snapshot fields invalidates ready status → needs_check.
Model mount resolved per-node: host = model_root + / + relative_path.
Container model path standardized: /models/<relative_path>.
Template list shows BackendRuntime only; RunnerConfigsPage shows NodeBackendRuntime.
```

## E2E Evidence

Current Phase 4 NVIDIA wizard E2E evidence (most recent first):

```text
docs/reports/model-runtime-node-wizard/e2e-run-20260618-201241/          (full E2E, PASS)
docs/reports/model-runtime-node-wizard/e2e-run-20260618-202641-instance-test/  (instance test API, PASS)
docs/reports/model-runtime-node-wizard/e2e-run-20260618-115214/          (prior E2E, PASS)
```

Latest result (2026-06-18 20:14 CST on main `48ee190`):

```text
E2E: PASS (exit code 0)
Instance test API: PASS (200 OK, chat mode, 177ms, single_model_fallback)
Environment: Docker 29.5.3, NVIDIA RTX 5090 (24GB, nvidia-smi 610.43.02)
```

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
