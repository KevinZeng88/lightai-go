# Model Runtime Node Wizard Open Issues Closeout

> Status: CURRENT_REPORT
> Last reviewed: 2026-06-18
> Scope: Phase 4 formal closeout and documented blockers
> Read order: See `docs/CURRENT.md`

Date: 2026-06-18

All P0/P1 problems found in this round are FIXED and verified by the NVIDIA wizard E2E. Remaining items are product-depth work that should not block the validated local NVIDIA Docker wizard path.

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| MRW-P0-001 | Model wizard root not allowed after adding a root | `RemoteFileBrowser.vue` previously kept roots in front-end state while Agent scan used a different allowed root policy | Model browse could succeed but scan/save could fail | FIXED | `internal/server/api/model_browser_handlers.go`, `internal/server/api/agent_proxy_handlers.go`, `cmd/agent/main.go`, `web/src/components/RemoteFileBrowser.vue`, `web/src/pages/ModelArtifactsPage.vue` | `scripts/e2e-model-runtime-wizard-nvidia-api.sh` PASS in `e2e-run-20260618-115214` | Closed |
| MRW-P0-002 | Browse/scan/save path semantics were not unified | Absolute paths could bypass the intended root_id + relative_path flow | Inconsistent validation and harder auditability | FIXED | `internal/server/api/artifact_handlers.go`, `internal/server/api/agent_proxy_handlers.go`, `web/src/pages/ModelArtifactsPage.vue` | E2E scan/save used `root_id` and `relative_path`; negative traversal test rejected `/tmp/../etc` | Closed |
| MRW-P0-003 | Denied root validation needed negative coverage | User could attempt `/`, `/etc`, `/etc/xxx`, or traversal values | Dangerous directory exposure if accepted | FIXED | `internal/server/api/model_browser_handlers.go`, `cmd/agent/main.go`, `internal/server/api/model_root_policy_test.go`, `scripts/e2e-model-runtime-wizard-nvidia-api.sh` | Unit test and E2E negative tests PASS | Closed |
| MRW-P1-001 | Start wizard needed explicit BackendVersion selection | Previous main flow could hide BackendVersion selection behind runtime internals | User could not reason about backend version capability from the launch wizard | FIXED | `web/src/pages/ModelDeploymentsPage.vue` | Web build/test PASS; E2E start path PASS | Closed |
| MRW-P1-002 | Main flow should not require hand-entered internal IDs | Manual IDs are not a wizard-style product flow | Users could not complete deployment from normal UI affordances | FIXED | `web/src/pages/ModelDeploymentsPage.vue` | Web build/test PASS; E2E API flow validates backend selections | Closed |
| MRW-P2-001 | Deep model consistency comparison is not implemented | Scanner identifies config/format basics but does not deep hash every weight/tokenizer artifact | Cross-node model equivalence has limited assurance | DOCUMENTED_BLOCKER | Future scanner/attestation in `internal/agent/collector` and model location APIs | Add fixture models with mismatched tokenizer/weights and verify exact/probable/mismatch statuses | Keep out of current P0/P1 closure |
| MRW-P2-002 | GPU auto/manual controls are basic in Web | Preflight and selected node exist, but UI does not expose a full GPU lease picker | Advanced placement workflows need richer controls | DOCUMENTED_BLOCKER | Future `web/src/pages/ModelDeploymentsPage.vue` GPU selection panel | Browser test selecting automatic/manual GPU and verifying resulting RunPlan lease | Keep out of current P0/P1 closure |
| MRW-P2-003 | Health-check detail panel is not complete | Instance status and logs exist, but detailed health-check history is not a dedicated panel | Operators have less context for slow/failed health checks | DOCUMENTED_BLOCKER | Future `web/src/pages/ModelInstancesPage.vue` health-check panel | Start a failing deployment and verify health-check timeline/errors render | Keep out of current P0/P1 closure |
| MRW-P2-004 | Instance detail command preview link can be richer | Server preview exists and start response displays it, but instance detail deep link is basic | Operators may need extra navigation to inspect the exact command after start | DOCUMENTED_BLOCKER | Future `web/src/pages/ModelInstancesPage.vue` and node run plan detail route | Open instance detail and verify Server `/node-run-plans/{id}/command-preview` renders | Keep out of current P0/P1 closure |
| MRW-P2-005 | Non-Docker runner types are not implemented | Current validated path is Docker-first | Command/systemd/external runner use cases are not available | DOCUMENTED_BLOCKER | Future runner abstraction in BackendRuntime/NodeBackendRuntime UI and Agent executor | Add non-Docker runner fixture and verify preflight/start lifecycle | Keep out of current P0/P1 closure |
| MRW-P2-006 | Node detail does not show Docker readiness as a first-class panel | Docker images and runtime checks exist, but Node detail does not summarize Docker availability | Operators may need to open runtime flows to infer Docker status | DOCUMENTED_BLOCKER | Future `web/src/pages/NodesPage.vue` Docker readiness section | Bring Docker down and verify Node detail shows unavailable status | Keep out of current P0/P1 closure |
| MRW-P2-007 | Model location disable UX is basic | API supports patch/delete and Web exposes delete, but there is no dedicated disable action in the location table | Operators have fewer non-destructive location lifecycle controls | DOCUMENTED_BLOCKER | Future `web/src/pages/ModelArtifactsPage.vue` location actions | Disable a location from Web and verify it is excluded from preflight candidates | Keep out of current P0/P1 closure |
| MRW-P2-008 | BackendVersion capability display in start wizard is minimal | BackendVersion is explicit, but detailed schema/capability explanation is not shown inline | Users may need to inspect runtime/catalog pages for deeper details | DOCUMENTED_BLOCKER | Future `web/src/pages/ModelDeploymentsPage.vue` version detail panel | Select BackendVersion and verify supported formats/protocols render | Keep out of current P0/P1 closure |
| MRW-P2-009 | NodeBackendRuntime disable action is not separated from delete in all Web surfaces | Node runtime management exists, but richer disable/delete distinction can be improved | Operators may prefer non-destructive node runtime removal | DOCUMENTED_BLOCKER | Future `web/src/pages/BackendRuntimesPage.vue` node runtime actions | Disable a NodeBackendRuntime and verify preflight excludes it while history remains | Keep out of current P0/P1 closure |
| MRW-P2-010 | Port auto/manual UI is basic | Start wizard has host port input and preflight, but no auto-port suggestion/conflict picker | Users may need to choose ports manually | DOCUMENTED_BLOCKER | Future `web/src/pages/ModelDeploymentsPage.vue` port control | Occupy a port and verify wizard suggests an alternative | Keep out of current P0/P1 closure |
| MRW-P2-011 | GPU lease display in Web is limited | E2E verifies lease release through cleanup, but Web does not expose a detailed lease panel | Operators have less direct visibility into lease lifecycle | DOCUMENTED_BLOCKER | Future `web/src/pages/ModelInstancesPage.vue` or node run plan detail lease panel | Start and stop a deployment, then verify allocated/released lease transitions render in Web | Keep out of current P0/P1 closure |

No problems from this round are left only in chat history.

---

## 2026-06-18 Polish Round: i18n, UX, and Concept Cleanup

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| MRW-POL-001 | Bare English strings displayed across Web UI (toasts, status, alerts, dialogs) | `ElMessage.success('Created')`, raw `row.status` display, hardcoded `'Confirm'`/`'Failed'` etc. | Users on zh-CN see English UI text | FIXED | `web/src/locales/*.ts`, `web/src/utils/status.ts`, all page `.vue` files | `npm run build` PASS, `npm test` PASS, 577 keys both locale files | Closed |
| MRW-POL-002 | `model root is not allowed` error toast was raw English | `RemoteFileBrowser.vue` mapped `root_not_allowed` error code but not all paths | User sees English error | FIXED | Already mapped via `fileBrowser.rootNotAllowed` i18n key; confirmed working | Existing `root_not_allowed` → `fileBrowser.rootNotAllowed` mapping verified | Already fixed before this round |
| MRW-POL-003 | Check result displayed raw `ready` status and `runtime verified for node` reason | `RunnerConfigsPage.vue` alert title and `BackendRuntimesPage.vue` tag showed raw API strings | User sees English status labels | FIXED | `web/src/utils/status.ts` added `translateStatus()` and `translateStatusReason()`, applied to all pages | Build/test PASS | Closed |
| MRW-POL-004 | Runtime config template created with same name as system template | Clone step in `RunnerConfigsPage.vue` didn't distinguish user config name | User confuses user config with system template | FIXED | Auto-suffix ` - 用户配置` / ` - Custom` with auto-increment on conflict, applied in both `RunnerConfigsPage.vue` and `BackendRuntimesPage.vue` | Build/test PASS | Closed |
| MRW-POL-005 | "启动实例" button inaccurate for deployment wizard | Button text implied single action, wizard is multi-step | Misleading UX | FIXED | `startWizard.title` changed to `部署向导` / `Deployment Wizard` in both locale files | Build/test PASS | Closed |
| MRW-POL-006 | Wizard single-select steps required manual "Next" click | Steps with only one select control needed extra click | Poor UX | FIXED | `web/src/composables/useWizardAutoAdvance.ts` shared helper, applied to ModelArtifacts, RunnerConfigs, Deployment wizards | Build/test PASS | Closed |
| MRW-POL-007 | Runtime config list showed node_count=0, ready_count=0 when configs existed | Frontend hardcoded `node_count: 0, ready_count: 0`; backend didn't return aggregate counts | Incorrect display | FIXED | Backend: added JOIN query to enrich `HandleListBackendRuntimes` with node_count/ready_count. Frontend: reads actual API values | `go test ./...` PASS, build PASS | Closed |
| MRW-POL-008 | Deployment wizard showed no runtime options when configs existed | Filter `r.backend_version_id === wizardVersionId` correct but no feedback when empty | User sees blank select | FIXED | Added `noRuntimeForVersion` alert when filtered list empty; verified filter logic matches backend | Build/test PASS | Closed |
| MRW-POL-009 | Enabling runtime on node created BackendRuntime clone → polluted template list | `RunnerConfigsPage.doCreateConfig` called `/clone` then `/enable`, creating a new BackendRuntime per node enablement | User templates appeared in "运行模板" list after node enable | FIXED | Removed clone step; `doCreateConfig` now only calls `/enable` with existing template ID. `RunnerConfigsPage` now shows NodeBackendRuntime records (node-level configs). Template list shows BackendRuntime only. | Build/test PASS | Closed |

No problems from this round are left only in chat history.

---

## 2026-06-18 Design Round: Template / Node-Config Boundary Formalization

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| MRW-DES-001 | BackendRuntime / NodeBackendRuntime boundary not formally documented | No design doc described the template vs node-config relationship, edit semantics, or status lifecycle | Agents and future developers could conflate the two concepts | FIXED | `docs/design/runtime-template-node-runtime-snapshot.md` (new), `docs/design/README.md`, `docs/README.md`, `docs/CURRENT.md` | Design doc reviewed; all doc links updated | Closed |
| MRW-DES-002 | Editing NodeBackendRuntime image fields did not invalidate ready status | `HandlePatchNodeBackendRuntime` updated image_ref/image_present but left status='ready' | Node config could be edited and still show ready without re-check | FIXED | `internal/server/api/node_runtime_handlers.go` — added `needsRecheck` flag that sets `status='needs_check'` when image_ref/image_id/image_digest/image_present change | `go test ./...` PASS; `go vet ./...` PASS | Closed |
| MRW-DES-003 | `needs_check` status not in i18n or status type mapping | New status value from DES-002 not translatable | Users would see raw English status text | FIXED | `web/src/locales/zh-CN.ts` (+`needs_check: '需重新检测'`), `web/src/locales/en-US.ts` (+`needs_check: 'Needs Check'`), `web/src/utils/status.ts` (warning type) | `npm run build` PASS; `npm test` PASS | Closed |
| MRW-DES-004 | Template re-apply / template change not implemented | Design describes these as explicit user actions with diff UI | Operators cannot switch NodeBackendRuntime source template | DOCUMENTED_BLOCKER | Future: NodeBackendRuntime edit UI + diff display + explicit confirmation | Add diff UI and re-apply/change buttons with status invalidation | P2 future enhancement |

No problems from this round are left only in chat history.


---

## 2026-06-18 Snapshot Round: config_snapshot_json, RunPlan Independence, Per-Node Model Mount

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| MRW-SNAP-001 | RunPlan resolver used live BackendRuntime config at every start, contradicting snapshot independence | `runplan.Resolve` `buildArgs`/`buildEnv`/`buildMounts`/`buildHealthCheck` all read `in.BackendRuntime.*` directly; template edits silently changed next deployment start | Template edits silently affected running deployments on next start | FIXED | `internal/server/db/db.go` (migrateV16: `config_snapshot_json`, `source_runtime_name`, `source_runtime_revision`), `internal/server/api/runtime_handlers.go` (snapshot capture in `upsertNodeBackendRuntime`), `internal/server/api/deployment_lifecycle_handlers.go` (snapshot read in `preflightDeployment` overrides RuntimeInfo before `runplan.Resolve`) | `go test ./...` PASS; snapshot captured on enable; preflight uses snapshot; older BackendRuntime edits do not change RunPlan | Closed |
| MRW-SNAP-002 | Model mount host path used only model_root, not model_root + relative_path — wrong for multi-node with different roots | `buildMounts` used `modelHostRoot` which returned just `ModelRoot` directory, not the full model path | Multiple nodes with different root directories could not have per-node model paths | FIXED | `internal/server/runplan/resolver.go` — `buildMounts` now constructs `hostPath = modelRoot + "/" + relativePath`, container path standardized to `/models/<relativePath>`; `buildVarMap` computes per-node `MODEL_HOST_PATH` from root+relativePath | `go test ./...` PASS; per-node mount test updated; MetaX and LlamaCpp tests verify new mount format | Closed |
| MRW-SNAP-003 | BackendRuntime edits silently affected NBR RunPlan output | No snapshot existed; preflight read live template args/env/docker/health_check at start time | Operators could not reason about what config a deployment would use after template edits | FIXED | `config_snapshot_json` frozen at enable/check time; `preflightDeployment` overrides RuntimeInfo from snapshot; NBR `image_ref` takes precedence for image resolution | `go test ./...` PASS; `go vet ./...` PASS | Closed |
| MRW-SNAP-004 | `needs_check` status invalidation only covered image fields, not snapshot config edits | `HandlePatchNodeBackendRuntime` only tracked image_ref/image_present changes | Editing snapshot fields could leave NBR as ready with stale config | FIXED | `node_runtime_handlers.go` — `needsRecheck` flag expanded to cover config_snapshot_json edits | `go test ./...` PASS | Closed |
| MRW-SNAP-005 | `NodeRuntimeOverride` was never passed to RunPlan resolver in preflight | `preflightDeployment` did not build `NodeRuntimeOverride`; NBR image_ref was ignored at start time | Node-level image override not used during deployment start | FIXED | `preflightDeployment` now builds `NodeRuntimeOverride` from NBR `image_ref` when non-empty, passed to `runplan.Resolve` | `go test ./...` PASS | Closed |
| MRW-SNAP-006 | Design doc claimed "full config snapshot is P2" | Previous `docs/design/runtime-template-node-runtime-snapshot.md` §2.2 described snapshot as "future upgrade" | Documentation contradicted the required implementation state | FIXED | Design doc updated: §1.2, §2.1, §2.2, §6 rewritten to document current snapshot implementation; P2 is now template re-apply/change UI and revision visualization | Design doc reviewed | Closed |

No problems from this round are left only in chat history.


---

## 2026-06-18 Hardening Round: Container Path Safety, Error Messages, Design Verification

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| MRW-HARD-001 | Container model path generated from untrusted relative_path without validation | `buildMounts` constructed `/models/<relPath>` without checking for `..`, absolute prefix, or emptiness | Path traversal could escape `/models` directory | FIXED | `internal/server/runplan/resolver.go` — `buildMounts` now returns `([]MountMapping, error)`; validates: no empty, no `..`, no absolute prefix; `cleanPath` ensures no escape; `Resolve` returns error on invalid path. `buildVarMap` falls back to safe default. | `TestContainerPathSafety` 6 cases PASS; `go test ./...` PASS | Closed |
| MRW-HARD-002 | Preflight error messages not i18n-ready for ModelLocation missing, NBR not ready, node offline | `preflightDeployment` returned free-form English error strings | Frontend displays raw English errors to zh-CN users | FIXED | `web/src/locales/zh-CN.ts` (+`preflight.reason.modelLocationMissing`/`nbrNotReady`/`nodeOffline`), `web/src/locales/en-US.ts` (same) | `npm run build` PASS; `npm test` PASS; 581 keys both locales | Closed |
| MRW-HARD-003 | config_snapshot_json verified not to contain model host paths | Snapshot captures `rt["model_mount_json"]` which is `{"container_path":"/models","readonly":true}` from templates; model host path resolved per-node at RunPlan time from ModelLocation | Verified safe — no contamination | VERIFIED | `internal/server/api/runtime_handlers.go` lines 276-292; `internal/server/runplan/resolver.go` `buildMounts` uses `modelHostRoot(in.Artifact)` not snapshot for host path | Code review: snapshot contains only container mount rules; host path resolved per-node | Closed |
| MRW-HARD-004 | Preflight auto-select node does not iterate candidates — documented limitation | `preflightDeployment` picks first online node when placement unspecified; if that node lacks ModelLocation, preflight fails without trying others | Single-node auto-select cannot fall back to other candidates | DOCUMENTED_BLOCKER | Future scheduler enhancement: iterate candidate nodes, exclude those without ModelLocation/NBR/GPU; report per-node reasons. Current single-node behavior is correct for specified-node path. | Implement multi-candidate iteration with per-node reason reporting | Documented: single-node auto-select uses first online node; multi-candidate iteration is future scheduler feature |
| MRW-HARD-005 | Deployment wizard runtime visibility: filtered by backend_version_id only; preflight does NBR+ModelLocation validation | Frontend `filteredRuntimes` computed on BackendRuntime list (by `backend_version_id`), not NodeBackendRuntime readiness. Preflight does the actual validation. | Wizard Step 3 shows all matching BackendRuntime templates; preflight Step 4 validates actual readiness + ModelLocation. This is correct behavior — the template is selected first, then preflight validates. | VERIFIED | `web/src/pages/ModelDeploymentsPage.vue` line 182: `filteredRuntimes` filters `runtimes` by `backend_version_id`; `preflightDeployment` validates NBR ready + ModelLocation on specific node | Code review confirmed; E2E evidence from previous round | Closed |

No problems from this round are left only in chat history.


---

## 2026-06-18 Structured Preflight Errors Round

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| MRW-SPE-001 | Preflight returned free-form English error strings directly to frontend | `preflightDeployment` appended raw strings like `"model location is not available on target node..."` to `pf.errs []string` | zh-CN users saw untranslated English errors; frontend had no reliable way to i18n-map them | FIXED | `internal/server/api/deployment_lifecycle_handlers.go` — added `PreflightError{Code,Message,Context}` struct, changed `errs []string` to `errs []PreflightError`, added `addErr()` helper, all error paths emit structured codes: `model_location_missing`, `node_backend_runtime_not_ready`, `node_offline`, `unknown` | `go build`, `go test ./...`, `go vet` all PASS | Closed |
| MRW-SPE-002 | Frontend preflight display used raw error strings | `ModelDeploymentsPage.vue` line 101-103 used `<el-alert :title="e">` directly on string errors | Raw English or Go format strings displayed to user | FIXED | `web/src/pages/ModelDeploymentsPage.vue` — added `preflightErrorText()` helper mapping error.code → i18n key via codeMap; displays error context (node_id, artifact_id, runtime_id) as detail | `npm run build` PASS; `npm test` PASS; 585 keys both locales | Closed |
| MRW-SPE-003 | Missing i18n keys for backendVersionMismatch, dockerImageMissing, runtimeDisabled, unknown | Only modelLocationMissing, nbrNotReady, nodeOffline existed | New or future error codes would display as fallback `[code] message` | FIXED | `web/src/locales/zh-CN.ts` (+4 keys), `web/src/locales/en-US.ts` (+4 keys), `ModelDeploymentsPage.vue` codeMap includes all 7 codes | `npm test` — 585 keys consistent | Closed |
| MRW-SPE-004 | Wizard Step 3 label said "选择运行配置" but data source is BackendRuntime templates | `startWizard.selectRuntime` was "选择运行配置" / "Select Runtime" | User might expect node configs, not templates | FIXED | `zh-CN.ts`: "选择运行模板"; `en-US.ts`: "Select Runtime Template" | `npm test` PASS | Closed |

No problems from this round are left only in chat history.


---

## 2026-06-18 Model Smoke Test Round

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| MRW-TEST-001 | No way to verify model inference after instance starts | Only health check (/v1/models) existed; no chat/completions test | Operators cannot confirm the model actually works beyond API liveness | FIXED | `internal/server/api/deployment_lifecycle_handlers.go` (+`HandleModelInstanceTest`), `internal/server/api/router.go` (+`POST /api/v1/model-instances/{id}/test`), `web/src/pages/ModelInstancesPage.vue` (+test button, result dialog, reason_code→i18n mapping), `web/src/locales/zh-CN.ts` (+13 keys), `web/src/locales/en-US.ts` (+13 keys) | `go build/test/vet` PASS; `npm build/test` PASS; 598 keys both locales | Closed |
| MRW-TEST-002 | Model test uses resolved model name, not user-entered ID | Test reads `model_artifacts.name` from deployment→artifact join; sends in chat/completions request body | Correct model is tested | VERIFIED | `HandleModelInstanceTest` — `SELECT COALESCE(ma.name,'') FROM model_deployments JOIN model_artifacts` | Code review confirmed | Closed |
| MRW-TEST-003 | Audit log entries for test actions | `WriteAudit` called for `model_instance.test.started`, `.succeeded`, `.failed` | Test events are traceable in audit log | VERIFIED | `deployment_lifecycle_handlers.go` — 3 WriteAudit calls | Code review confirmed | Closed |

No problems from this round are left only in chat history.


---

## 2026-06-18 Model Test Hardening Round

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| MRW-TEST-004 | model id used `model_artifacts.name` only, doesn't match runtime's /v1/models | `HandleModelInstanceTest` read only artifact name; vLLM/llama.cpp may expose different model ids at runtime | Test could use wrong model id, causing false failures | FIXED | `internal/server/api/deployment_lifecycle_handlers.go` — new `resolveModelID()` function: 1) RunPlan model_name, 2) query /v1/models, 3) single model → use directly, 4) multi-model → exact match on artifact name/path basename, 5) alias/substring match, 6) fail with `model_id_not_resolved` | `go build/test/vet` PASS | Closed |
| MRW-TEST-005 | Only chat/completions attempted; no completions fallback | `HandleModelInstanceTest` hardcoded `/v1/chat/completions`; if runtime only supports completions, test falsely fails | Models behind completions-only APIs (e.g., llama.cpp server) always fail | FIXED | `internal/server/api/deployment_lifecycle_handlers.go` — new `tryInference()` function: tries chat/completions first; if 404/405 (endpoint not supported), falls back to /v1/completions. Real errors (OOM, auth, model load fail) do NOT trigger fallback. Mode returned in response. | `go build/test/vet` PASS | Closed |
| MRW-TEST-006 | New reason codes not i18n-mapped | `model_id_not_resolved`, `chat_endpoint_failed`, `completion_endpoint_failed` had no frontend translation | Users see raw codes | FIXED | `web/src/locales/zh-CN.ts` (+8 keys), `web/src/locales/en-US.ts` (+8 keys), `web/src/pages/ModelInstancesPage.vue` (extended `testReasonI18n` map + display mode and `model_resolution_method`) | `npm build` PASS; `npm test` PASS; 606 keys both locales | Closed |

No problems from this round are left only in chat history.

---

## 2026-06-18 Real-Machine Verification: Instance Test API

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| MRW-VER-001 | Instance test API real-machine E2E verification | Fresh deployment + `POST /api/v1/model-instances/{id}/test` on running vLLM+NVIDIA RTX 5090 instance | Confirms test API works end-to-end on real hardware | VERIFIED | N/A (verification only) | `docs/reports/model-runtime-node-wizard/e2e-run-20260618-201241/` (full E2E), `e2e-run-20260618-202641-instance-test/` (instance test API) | Closed |
| MRW-VER-002 | Test API resolved model via single_model_fallback | Only one model in `/v1/models`; test correctly used `single_model_fallback` | Model resolution method works correctly when single model exists | VERIFIED | `deployment_lifecycle_handlers.go` `resolveModelID()` | Response: `model_resolution_method: single_model_fallback`, `model: e2e-itest-202321` | Closed |
| MRW-VER-003 | Chat/completions mode confirmed working | Test used `/v1/chat/completions` with 200 response; Qwen3 returned Chinese text | Chat mode correctly preferred and working | VERIFIED | `deployment_lifecycle_handlers.go` `tryInference()` | Response: `mode: chat`, `endpoint: /v1/chat/completions`, `latency_ms: 177` | Closed |
| MRW-VER-004 | Audit log records test.started + test.succeeded | Two audit entries with entity_id, endpoint, model, resolution method | Traceability confirmed in audit trail | VERIFIED | `deployment_lifecycle_handlers.go` `WriteAudit()` | Audit API: `model_instance.test.started` and `model_instance.test.succeeded`, both `result=success` | Closed |
| MRW-VER-005 | Pending instance correctly blocked from test | Test on pending instance returned 400 with `reason_code: instance_not_running` | Correct guard prevents testing non-running instances | VERIFIED | `deployment_lifecycle_handlers.go` `HandleModelInstanceTest()` | Response: `{"ok":false,"reason_code":"instance_not_running"}`, i18n-ready message | Closed |

### Real-Machine Verification Evidence

- **Date**: 2026-06-18 20:12-20:26 CST
- **Environment**: WSL2, Docker 29.5.3, NVIDIA RTX 5090 (24GB, nvidia-smi 610.43.02, CUDA 13.3)
- **Model**: Qwen3-0.6B-Instruct-2512 (huggingface)
- **Backend**: vLLM (vllm/vllm-openai:latest), runtime: `runtime.vllm.nvidia-docker`
- **Git**: branch `main`, commit `48ee190`
- **Basic verification**: `go test ./...` PASS, `go vet ./...` PASS, `go build ./...` PASS, `npm build` PASS, `npm test` PASS (all 11 tests, 606 i18n keys)
- **E2E result**: PASS (exit code 0), `/v1/models` returned model `e2e-wizard-*`
- **Instance test API**: PASS (200 OK, 177ms latency)
- **Instance ID**: `6380d872-9a24-4c07-9b8f-976ddc35a8a2`
- **Model resolved**: `e2e-itest-202321` via `single_model_fallback`
- **Mode**: `chat` (endpoint `/v1/chat/completions`)
- **Response preview**: `"Ping 是一种网络测试工具，用于"`
- **Log paths**:
  - `docs/reports/model-runtime-node-wizard/e2e-run-20260618-201241/` — full E2E output, server/agent logs, exit code
  - `docs/reports/model-runtime-node-wizard/e2e-run-20260618-202641-instance-test/` — instance test API response, audit logs, summary

No problems from this round are left only in chat history.
