# Backend Runtime RunPlan Gap Fix Plan

Date: 2026-06-17

## Phase A: Backend Catalog And Seed

目标: Add target `configs/backend-catalog/` structure and seed built-in Backend, BackendVersion, and BackendRuntime rows idempotently.

现状: Current seed is hardcoded in `internal/server/db/db.go:seedBuiltInBackends`; current YAML config is under `configs/model-runtime/`.

差异: Target requires stable IDs/slugs, `catalog_version`, `checksum`, `managed_by`, system readonly items, external override directory, and Ollama/Huawei templates.

修改文件:

- `internal/server/db/db.go`
- `configs/backend-catalog/**`
- `configs/backend-catalog.d/**`
- `docs/backend-catalog-vendor-extension.md`

数据库变更:

- Add metadata columns to `inference_backends`, `backend_versions`, `backend_runtimes`.
- Seed system-managed rows with stable target IDs.

API 变更:

- Expose target paths `GET /api/v1/backends`, `GET /api/v1/backend-versions`.

Agent 变更: None.

Web 变更: Use target aliases where safe.

测试方法:

- `go test ./internal/server/api ./internal/server/runplan`
- API smoke via E2E script.

验收标准:

- Built-in `backend-version.llamacpp.server`, `backend-version.llamacpp.server-metax`, `backend-version.vllm.openai-latest`, `backend-version.sglang.openai-latest`, and `backend-version.ollama.latest` exist.
- Built-in runtimes are `managed_by=system` and readonly.

兼容影响:

- Strict target IDs are seeded and used by target APIs; legacy IDs are not treated as acceptance criteria.

## Phase B: BackendVersion And BackendRuntime Templates

目标: Ensure BackendVersion is preserved and all runtime templates are distinct from versions.

现状: BackendVersion exists. Runtime templates exist as YAML but are not seeded as system BackendRuntime rows.

差异: Target Runtime templates must include NVIDIA, MetaX, Huawei, and CPU variants with complete Docker options.

修改文件:

- `internal/server/db/db.go`
- `internal/server/api/runtime_handlers.go`
- `internal/server/runplan/types.go`
- `internal/server/runplan/resolver.go`
- `configs/backend-catalog/runtimes/**`

数据库变更:

- Seed system BackendRuntime rows for built-in variants.

API 变更:

- Prevent patch/delete of `managed_by=system` or `is_editable=false` rows.

Agent 变更: None yet; Agent already supports most Docker options.

Web 变更:

- Display BackendVersion and readonly status.

测试方法:

- RunPlan resolver tests for MetaX and Huawei template-only.

验收标准:

- MetaX runtime generates command preview with tested Docker options.
- Huawei runtime is present but never marked ready by default.

兼容影响:

- User-created runtimes continue to work.

## Phase C: NodeBackendRuntime

目标: Add node-scoped runtime readiness records and API.

现状: `node_runtime_overrides` exists but is only override config, not readiness.

差异: Target requires NodeBackendRuntime status: ready, missing_image, driver_mismatch, toolkit_missing, adapter_missing, template_only, unsupported_device, invalid, unknown.

修改文件:

- `internal/server/db/db.go`
- `internal/server/api/runtime_handlers.go`
- `internal/server/api/router.go`
- `internal/server/api/phase3_rbac_test.go`

数据库变更:

- Add `node_backend_runtimes` table.

API 变更:

- `GET /api/v1/nodes/{node_id}/backend-runtimes`
- `POST /api/v1/nodes/{node_id}/backend-runtimes/enable`
- `POST /api/v1/nodes/{node_id}/backend-runtimes/check`

Agent 变更:

- No direct Agent call in first pass; Server infers from DB GPU vendor and node docker image query if available.

Web 变更:

- Runtime page can show node readiness status.

测试方法:

- Handler tests for NVIDIA ready/missing image and Huawei template_only.

验收标准:

- Huawei returns `template_only`/`adapter_missing`, not ready.
- NVIDIA can be marked ready when enabled with image present or explicitly enabled.

兼容影响:

- New table and endpoints are additive.

## Phase D: ModelArtifact / ModelLocation

目标: Add node-scoped ModelLocation without breaking existing ModelArtifact.path.

现状: `model_artifacts.path` is the only model location field.

差异: Target requires `model_locations` with node_id, model_root, relative_path, absolute_path, size, checksum, match and verification status, manual attestation.

修改文件:

- `internal/server/db/db.go`
- `internal/server/api/artifact_handlers.go`
- `internal/server/api/router.go`
- `internal/server/runplan/resolver.go`
- `internal/server/api/deployment_lifecycle_handlers.go`

数据库变更:

- Add `model_locations`.
- Backfill first location only when node is known via API.

API 变更:

- `POST /api/v1/model-artifacts/discover`
- `POST /api/v1/model-artifacts/{id}/locations`
- `POST /api/v1/model-artifacts/{id}/locations/{location_id}/rescan`
- `POST /api/v1/model-artifacts/{id}/locations/{location_id}/attest`

Agent 变更:

- First pass uses server-side path metadata only; actual remote scanning remains documented blocker if Agent file scan protocol is absent.

Web 变更:

- Model detail can show locations.

测试方法:

- API tests and RunPlan test verifying ModelLocation mount path.

验收标准:

- Start selects ModelLocation for target node when present; fallback to artifact.path is compatibility only.

兼容影响:

- Existing artifact path remains supported.

## Phase E: DeploymentPlan / RunPlanGroup / NodeRunPlan Resolver

目标: Align current deployment and resolved_run_plans with target aliases.

现状: Current DB uses `model_deployments` and `resolved_run_plans`.

差异: Target names are DeploymentPlan, RunPlanGroup, NodeRunPlan.

修改文件:

- `internal/server/api/router.go`
- `internal/server/api/deployment_lifecycle_handlers.go`
- `internal/server/runplan/*`

数据库变更:

- Add lightweight `run_plan_groups`.
- Reuse `resolved_run_plans` as NodeRunPlan-compatible storage.

API 变更:

- Add `/api/v1/deployments` aliases.
- Add `/api/v1/node-run-plans/{id}` and command-preview/logs endpoints.

Agent 变更: None for aliases.

Web 变更:

- Command preview uses current dry-run/start response.

测试方法:

- Handler tests and E2E script.

验收标准:

- Target paths are the acceptance surface; legacy paths are not required for closure.

兼容影响:

- Additive aliases.

## Phase F: Agent DockerExecutor

目标: Ensure Agent receives all Docker options from NodeRunPlan.

现状: Agent supports most Docker options, but Server payload omits devices/group_add and hardcodes GPU env.

差异: MetaX runtime requires raw devices, group_add, security_opt, ulimits, privileged, ipc, uts, shm_size, and env from template.

修改文件:

- `internal/server/runplan/types.go`
- `internal/server/runplan/resolver.go`
- `internal/server/runplan/preview.go`
- `internal/server/api/deployment_lifecycle_handlers.go`
- `internal/agent/runtime/runplan_adapter.go`
- `internal/agent/runtime/docker.go`

数据库变更: None beyond runtime/template seed.

API 变更:

- Command preview must include enabled options.

Agent 变更:

- Ensure `GroupAdd`, `Devices`, `SecurityOptions`, `UTSMode`, `Ulimits` flow to Docker create options.

Web 变更:

- Preview displays the same command string.

测试方法:

- Unit tests for MetaX command preview and Agent create options.

验收标准:

- MetaX preview includes the user-tested Docker options except `/dev/mem`.

兼容影响:

- Existing NVIDIA plans continue to use DeviceRequests.

## Phase G: Web Pages

目标: Productize Runtime parameters and prevent i18n key leakage.

现状: Runtime page is CRUD-lite with raw JSON detail.

差异: Target requires enabled blocks, textarea list blocks, Custom Args, Custom Env, Custom Docker Options, readonly system behavior, command preview.

修改文件:

- `web/src/pages/BackendRuntimesPage.vue`
- `web/src/api/runtimes.ts`
- `web/src/locales/en-US.ts`
- `web/src/locales/zh-CN.ts`
- `web/tests/i18nMissingKeys.test.mjs`

数据库变更: None.

API 变更:

- Runtime PATCH must accept JSON fields.

Agent 变更: None.

Web 变更:

- Runtime page includes enabled blocks and preview rendering.

测试方法:

- `npm --prefix web run build`
- `npm --prefix web test -- --runInBand || true`
- i18n tests.

验收标准:

- No new hardcoded user-facing English text in runtime page.
- i18n keys exist in zh-CN and en-US and resolve to strings.

兼容影响:

- Existing runtime list remains available.

## Phase H: E2E, Docs, Commit, Push

目标: Verify and document acceptance.

现状: Existing `scripts/e2e-model-runtime-api.sh` exists, but requested NVIDIA script path does not.

差异: Need `scripts/e2e-backend-runtime-nvidia-api.sh` with skip behavior for missing server/image/model.

修改文件:

- `scripts/e2e-backend-runtime-nvidia-api.sh`
- `docs/reports/backend-runtime-runplan-acceptance-report.md`
- `docs/reports/backend-runtime-runplan-current-state-audit.md`
- `docs/plan/backend-runtime-runplan-gap-fix-plan.md`
- `docs/backend-catalog-vendor-extension.md`

数据库变更: None.

API 变更: None beyond previous phases.

Agent 变更: None beyond previous phases.

Web 变更: None beyond previous phases.

测试方法:

```bash
go test ./...
go vet ./...
npm --prefix web run build
npm --prefix web test -- --runInBand || true
bash -n scripts/*.sh
git diff --check
```

验收标准:

- All known fixable problems are fixed.
- Any environment-only unverified items are documented as `DOCUMENTED_BLOCKER` in `docs/reports/backend-runtime-runplan/open-issues-closeout.md`.

兼容影响:

- Final commit and push as requested.

