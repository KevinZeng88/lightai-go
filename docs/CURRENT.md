# LightAI Go Current State

> Status: CURRENT
> Last reviewed: 2026-06-18
> Scope: Current implementation and documentation entrypoint
> Read order: This file first, then `docs/README.md`

## Branch Baseline

Current branch and commit verified during 2026-06-18 real-machine verification:

```text
Branch: main
Baseline before this round: 13698f3 chore: ignore runtime pid files
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
Backend / BackendVersion = software capability layer; keep hardware/node state outside BackendVersion.
System Backend / BackendVersion rows are read-only catalog entries.
System BackendVersion can be cloned to user BackendVersion; user BackendVersion can be added, edited, deleted, and synced.
BackendVersion -> BackendRuntime copies defaults at creation; later BackendVersion edits do NOT mutate existing BackendRuntime.
BackendRuntime stores source_backend_id/source_backend_version_id/source_version_revision and version_snapshot_json.
NodeBackendRuntime = node-level config with frozen config_snapshot_json.
NBR snapshot captured at creation time (args, env, docker, mounts, health_check).
BackendRuntime -> NodeBackendRuntime copies config at creation only; later BackendRuntime edits do NOT mutate existing NodeBackendRuntime. NodeBackendRuntime check/validate only verifies the snapshot against node state and does NOT refresh the snapshot from BackendRuntime.
RunPlan resolver reads BackendRuntime version_snapshot_json and NBR config_snapshot_json, not live mutable defaults.
BackendRuntime template edits do NOT affect existing NodeBackendRuntime RunPlans.
Editing NodeBackendRuntime image/snapshot fields invalidates ready status → needs_check.
Model mount resolved per-node: host = model_root + / + relative_path.
Container model path standardized: /models/<relative_path>.
Template list shows BackendRuntime only; RunnerConfigsPage shows NodeBackendRuntime.
BackendRuntimesPage only manages templates; RunnerConfigsPage manages add/edit/check/delete for NodeBackendRuntime.
Backend / BackendVersion catalog files are the source of truth.
DB backend catalog rows are reload/sync projections for query and references.
System BackendVersion catalog files live under `configs/backend-catalog/versions/` and are read-only at runtime.
User BackendVersion catalog files live under `data/backend-catalog.d/user/` by default, or `LIGHTAI_BACKEND_CATALOG_USER_DIR` when configured.
BackendVersion add/edit/clone/delete writes user catalog files first, then reloads/syncs DB projection.
Catalog reload/sync never mutates existing BackendRuntime or NodeBackendRuntime snapshots.
```

## Backend Catalog Baseline

Current system BackendVersion baseline follows the official software-version boundary:

```text
vLLM: v0.23.0, OpenAI-compatible, image candidates vllm/vllm-openai:v0.23.0, v0.23.0-cu129-ubuntu2404, latest.
SGLang: v0.5.12.post1 and v0.5.13.post1, OpenAI-compatible, launch module python3 -m sglang.launch_server.
llama.cpp: b9700 build tag, OpenAI-compatible subset, GGUF focused, llama-server.
```

SGLang `v0.5.13.post1` was verified with:

```bash
git ls-remote --tags https://github.com/sgl-project/sglang.git | grep 'refs/tags/v0.5.13'
```

BackendVersion must not store node IDs, GPU indexes, `image_present`, `ready` / `needs_check`, host model paths, device mounts, or vendor runtime checks. Those belong to BackendRuntime presets, NodeBackendRuntime, Node/GPU discovery, or RunPlan.

User catalog files can be added by scripts by writing YAML under `data/backend-catalog.d/user/<backend>/` and calling `POST /api/v1/backend-catalog/reload`. Export/share is file-based: copy the YAML files from that user directory to another LightAI Go deployment's user catalog directory and reload there.

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
Advanced node detail Docker readiness, GPU lease picker, and non-Docker runners remain formal documented blockers.
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
