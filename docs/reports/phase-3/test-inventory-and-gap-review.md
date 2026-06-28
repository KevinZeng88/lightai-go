# Phase 3 Test Inventory And Gap Review

> Date: 2026-06-28  
> Scope: test inventory only; no code changes; no commit  
> Workspace: `/home/kzeng/projects/ai-platform-study/lightai-go`

## 1. Verification Snapshot

| Command | Result | Notes |
| --- | --- | --- |
| `git status --short` | ` M VERSION` | Pre-existing workspace change; not touched by this review. |
| `go test -list . ./internal/server/... ./internal/agent/... ./cmd/...` | PASS | Compiled packages and listed active tests in requested Go scopes. |
| `go test ./internal/server/... ./internal/agent/... ./cmd/...` | PASS | Server, Agent, and cmd tests passed. `internal/server/db`, `internal/server/metrics`, `internal/server/models`, `internal/server/rbac`, and `cmd/server` have no test files. |
| `cd web && npm test` | PASS | Runs 8 Node-based frontend contract/static tests only. |
| `cd web && npm run build` | PASS | `vue-tsc --noEmit && vite build`; Rollup chunk-size warning only. |

## 2. Current Test Command Inventory

| Command | Type | Included by default | Coverage |
| --- | --- | --- | --- |
| `go test ./internal/server/...` | Go unit/API workflow | Yes, when run manually/CI | Server API, auth, catalog, config edit, runtime boundaries, RunPlan, deployment, probe, RBAC. |
| `go test ./internal/agent/...` | Go unit | Yes, when run manually/CI | Agent collector, metrics, register, runtime Docker adapter, state. |
| `go test ./cmd/...` | Go unit | Yes, when run manually/CI | Agent command/start-result helpers. `cmd/server` has no tests. |
| `go test ./...` | Full Go tree | Not shown as a documented regular gate in this review | Adds `internal/common/...` and `internal/runtimecontract/...` beyond requested server/agent/cmd scopes. |
| `cd web && npm test` | Frontend static/contract | Yes | API path lint, i18n checks, credential scan, runtime boundary static checks, model capability utility, ConfigEdit patch contract. |
| `cd web && npm run build` | Frontend type/build | Common manual gate | Vue type-check and Vite production build. |
| `cd web && npm run test:e2e` | Playwright E2E | No, separate command | Browser smoke/auth tests under `web/tests/e2e`. Requires running app or configured base URL. |
| `bash scripts/e2e-current-contract-api-dryrun.sh` | API-first E2E | No | Current deployment dry-run contract; running server/agent required. |
| `bash scripts/e2e-runtime-config-web-check-flow.sh` | API/Docker image check E2E | No | Enable NBR, check-request positive/negative row/wizard checks; real Docker/agent required. |
| `bash scripts/e2e-real-smoke-all-three.sh` | Real runtime smoke | No | vLLM, SGLang, llama.cpp platform chain; NVIDIA Docker/model prerequisites. |
| `bash scripts/e2e-model-runtime-wizard-nvidia-{vllm,sglang,llamacpp}.sh` | Real runtime E2E | No | Backend-specific model/runtime/deployment wizard API path. |
| `bash scripts/e2e-current-contract-nvidia-llamacpp-smoke.sh` | Real NVIDIA smoke | No | llama.cpp full start/stop contract; skips if prerequisites missing. |
| `bash scripts/smoke-model-backends.sh all` | Direct/backend smoke | No | Direct backend and RunPlan smoke helpers; environment dependent. |
| `bash scripts/e2e/lib/e2e-assert-selftest.sh` | Shell assertion self-test | No | Verifies E2E assertion library behavior. |

## 3. Go Test Inventory

### 3.1 Server Tests

| File | Type | Test names / coverage |
| --- | --- | --- |
| `internal/server/agentclient/client_test.go` | Unit | `TestValidateAgentAddress_*`, `TestGetJSON_*`, `TestPostJSON_Success`; SSRF/address filtering, URL encoding, timeout, JSON client behavior. |
| `internal/server/auth/bootstrap_test.go` | Unit | `TestReadPasswordFromCredentialsFile`, `TestWriteInitialCredentials_*`, `TestPasswordResolutionPriority`; admin credential bootstrap. |
| `internal/server/authz/checks_test.go`, `helpers_test.go` | Unit | `TestTenantID_*`, `TestIsPlatformAdmin_*`; session tenant/admin helpers. |
| `internal/server/catalog/*_test.go` | Unit | `TestLoadRegistryAndBackendCatalog`, `TestMaterializeConfigSets*`, `TestCopyOnCreate*`, `TestParentMutationDoesNotPolluteChild`, `TestConfigSetGenerateView*`, `TestAdvancedSectionDefaultCollapsed`, `TestTieredConfigRoundTrip*`; ConfigSet materialization, copy-on-create, view generation, immutability, tiered round-trip. |
| `internal/server/configedit/configedit_test.go` | Unit | `TestProjectConfigSetToEditView*`, `TestApplyEditPatch*`, `TestValidateEditPatch*`, `TestDockerSubfieldValueDoesNotLeakParentDefault`, `TestItemCodeSetForWidgetOverride`; ConfigEditView projection, editable patch semantics, protected-field validation, Docker subfield behavior. |
| `internal/server/semanticconfig/*_test.go` | Unit | `TestDefaultRegistryContainsCanonicalOwners`, `TestNormalizeConfigSet*`, `TestValidatePatchRejects*`, `TestSnapshotBuilderCopiesLineageAcrossRuntimeNodeAndDeployment`, `TestDerivedServiceJSONComesFromSemanticSnapshot`; semantic config normalization, validation, lineage snapshot. |
| `internal/server/runplan/compat_test.go` | Unit | `TestCompat*`, `TestParseBackendCapabilities*`; model/backend compatibility matrix and backend capability parsing. |
| `internal/server/runplan/resolver_test.go` | Unit | `TestResolveBasic`, `TestResolveServicePortSemantics`, `TestResolveImagePriority`, `TestResolveRendersConfigSetParameterStyles`, `TestResolveDoesNotFallbackToLiveBackendVersion*`, `TestResolveArgs`, `TestResolveEnv*`, `TestBuildEnv*`, `TestResolveResourceControlsNoDuplicateWithParameterDefs`; RunPlan resolver inputs, args/env/resource controls, snapshot-only behavior. |
| `internal/server/runplan/vllm_sglang_nvidia_test.go` | Unit | `TestResolveVLLMNVIDIA`, `TestResolveSGLangNVIDIA`, `TestVLLM*`, `TestDedupKeepsUserPortOverDefault`, `TestGetParamMatchesCLIFormatNames`; vendor/backend-specific RunPlan rendering. |
| `internal/server/runplan/llamacpp_nvidia_test.go` | Unit | `TestLlamaCppNvidiaRunPlan`, `TestLlamaCppGGUFFileInDirectory`, `TestLlamaCppRunPlanNoGPU`; llama.cpp RunPlan behavior. |
| `internal/server/runplan/metax_huawei_test.go` | Unit | `TestResolveMetaXRunPlanUsesRuntimeDockerOptions`, `TestResolveHuaweiRunPlanUsesAscendVisibleDevices`; vendor-specific env/device conventions. |
| `internal/server/runplan/lint_test.go` | Unit | `TestLint*`; duplicate args, env/CLI conflict, privileged/IPC warnings, lint JSON. |
| `internal/server/runplan/source_map_test.go` | Unit | `TestSourceMap*`, `TestResolvedRunPlanNowIncludesSourceMap`; source map target coverage and provenance. |
| `internal/server/runplan/resource_controls_test.go` | Unit | `TestParseResourceControls*`, `TestValidateResourceControl*`, `TestBuildResourceControlArgs`, `TestResourceControlJSON`; resource controls parsing/validation/rendering. |
| `internal/server/runplan/log_classifier_test.go` | Unit | `TestClassify*`, `TestIsNonFatal`, `TestOccurrences`, `TestFormatEventsForDisplay`; backend log classification. |
| `internal/server/api/runtime_boundary_test.go` | API contract | `TestCreateBackendRuntimeCopiesBackendVersionSnapshot`, `TestNodeBackendRuntimeCopiesTemplateSnapshotAndTemplateEditDoesNotChangeIt`, `TestNodeBackendRuntimeCheckDoesNotRefreshSnapshot`, `TestBackendRuntimeListShowsTemplatesWithNodeAggregatesOnly`, `TestPreflight*`, `TestDeploymentStartUsesNBRNotBackendRuntime`, `TestCheckRequest*`, `TestGetProbe*`, `TestBackendRuntimePatchRejects*`, `TestCloneBackendRuntime*`, `TestCatalogSeedProducersUserVisibleDisplayNames`; runtime/NBR/deployment/probe boundary and recent display/name fixes. |
| `internal/server/api/config_edit_handlers_test.go` | API contract | `TestConfigEditViewAPIProjectsRuntimeWithoutInternalOrdinaryLabels`, `TestNodeBackendRuntimeEnableAppliesEditableConfigPatch`, `TestDeploymentCreateAppliesEditableConfigPatchToSnapshot`; config-edit API projection/apply path. |
| `internal/server/api/deployment_preflight_contract_test.go` | API contract | `TestContractPreflight*`, `TestDryRunAppliesProbeProcessStartConfig`, `TestContractDryRunWithReadyWithWarnings`, `TestContractSnapshotNotMutatedByMigration`; preflight/dry-run behavior and probe-derived start config. |
| `internal/server/api/workflow_backend_runtime_test.go` | API workflow | `TestWorkflowBackendRuntimeCRUDChain`, `TestWorkflowBackendRuntimePatchPreservesFields`, `TestWorkflowBackendRuntimeDeleteCleanup`; BackendRuntime CRUD chain. |
| `internal/server/api/workflow_nbr_probe_test.go` | API workflow | `TestWorkflowNBRProbeChain`, `TestWorkflowNBRProbeMissingImageOnlyFromInspectNotFound`, `TestWorkflowNBRProbeInspectErrorIsNotMissingImage`; NBR probe chain and inspect/list authority. |
| `internal/server/api/workflow_deployment_runplan_test.go` | API workflow | `TestWorkflowDeploymentPreflightRunPlan`, `TestWorkflowDeploymentRunPlanPreservesNBRSnapshot`, `TestWorkflowDeploymentCleanup`; deployment to RunPlan workflow. |
| `internal/server/api/workflow_lifecycle_test.go` | API workflow | `TestWorkflowLifecycleStartStatusLogsStop`, `TestWorkflowLifecycleStartFailureKeepsDiagnosticsAndLogs`, `TestWorkflowLifecycleStopIsIdempotentOrExplained`; instance lifecycle and diagnostics. |
| `internal/server/api/workflow_model_wizard_test.go` | API workflow | `TestWorkflowModelWizard*`; artifact/location wizard workflow. |
| `internal/server/api/ui_persistence_runplan_test.go` | API/persistence | `TestCloneBackendRuntimePersistsIndependentDisplayName`, `TestCreateAndPatchBackendRuntimeNamePersistence`, `TestDeploymentCapturesConfigSnapshotAtCreate`, `TestDeploymentCapturesNBRConfigAtCreate`, `TestNBRConfigModificationDoesNotAffectDeploymentDryRun`, `TestBackendRuntimeEditDoesNotAffectNBRConfig`, `TestBackendVersionEditDoesNotAffectBackendRuntime`, `TestRunPlanImmutableAfterDeploymentEdit`; persistence, copy boundaries, display name, immutable RunPlan. |
| `internal/server/api/nbr_deployable_test.go` | API/unit | `TestIsNBRDeployable`, `TestNBRDisabledReason`, `TestExtractProbeWarnings`, `TestNBRListResponseIncludesDeployable`, `TestCreateDeployment*`; NBR deployability semantics. |
| `internal/server/api/node_run_plan_logs_test.go` | API contract | `TestNodeRunPlanLogs*`; server-to-agent logs proxy and classified events. |
| `internal/server/api/agent_identity_test.go` | API contract | `TestAgentRegistration*`, `TestNodeIDAgentIDBinding*`, `TestHeartbeat*`, `TestResourceReportAgentIDBinding`, `TestNodeList*`; agent identity/tenant hardening. |
| `internal/server/api/phase3_rbac_test.go`, `tenant_isolation_test.go`, `tenant_rbac_negative_test.go` | API/RBAC | `TestBackendRuntimeTenantIsolation`, `TestTenant*`, `TestDefaultEnvJSONRedaction`; tenant scoping and RBAC negative cases. |
| `internal/server/api/resource_handlers_test.go` | API/resource | `TestServerIngestMetaX8GPUToAPI`, `TestServerIngestMemoryFreeBytes`, `TestGPUInsertDoesNotOverwriteTenant`; GPU/resource ingestion. |
| `internal/server/api/model_capability_test.go`, `model_root_policy_test.go` | API/model | `TestModelCapability*`, `TestValidateNodeModelRootRejectsDeniedAndTraversal`, `TestNodeModelRootCRUDUsesPersistentRows`; model capability and path policy. |
| `internal/server/api/agent_task_result_test.go`, `middleware_*_test.go`, `api_workflow_harness_test.go`, `catalog_seed_drift_test.go`, `metax_device_binding_test.go` | API/support | Task result state/audit, logging/recovery middleware, workflow harness, catalog drift, MetaX device binding. |

### 3.2 Agent Tests

| File | Type | Test names / coverage |
| --- | --- | --- |
| `internal/agent/collector/nvidia_test.go` | Unit | `TestParseNvidiaCSV_*`, `TestParseNvidiaMetricsCSV_Success`, `TestParseFloatOrNil_*`, `TestParseUintOrZero_NA`; NVIDIA collector parsing. |
| `internal/agent/collector/probe_test.go` | Unit | `TestProbeNvidiaAvailable`, `TestProbeMetaxAvailable`, `TestProbeBothVendorsAvailable`, `TestProbeExit10NotAvailable`, `TestProbeExit30ProbeFailed`, `TestProbeNoDevices`; vendor probe command handling. |
| `internal/agent/collector/protocol_test.go` | Unit | `TestParseProtocolOutput_*`, `TestParseKeyValues*`; collector protocol parser. |
| `internal/agent/collector/model_scanner_test.go` | Unit | `TestDetect*`, `TestB2UnsupportedDeployableFalse`, `TestEmptyDirectory`, `TestMixedHFAndGGUF`, `TestScanDirectoryFullPipeline`; model scanner detection. |
| `internal/agent/collector/gguf_reader_test.go` | Unit | `TestReadGGUFMeta_*`, `TestFormatBytes`, `TestGuessQuantFromFilename`; GGUF metadata reader. |
| `internal/agent/metrics/metrics_test.go` | Unit | `TestMetaX8GPU*`, `TestLegalZeroValues`, `TestGPUResourceUniqueKey`, `TestGPUPrometheusDuplicateDedup`, `TestVendorNeutral`; metrics normalization and Prometheus output. |
| `internal/agent/register/register_test.go` | Unit | `TestDo_*`; agent registration success/failure/mismatch paths. |
| `internal/agent/runtime/docker_test.go` | Unit/fake Docker | `TestDockerRuntimeDriverStart*`, `TestStart*Diagnostics`, `TestDockerRuntimeDriverStop/Inspect/Logs`, `TestSensitiveEnvRedaction`, `TestNvidiaGpuDeviceRequest*`, `TestMetaXUsesRawDevicesNotDeviceRequest`, `TestRealDockerRuntimeDriver`; Docker lifecycle adapter using fake client, with real Docker gated by env. |
| `internal/agent/runtime/health_test.go` | Unit | `TestHealthCheckConfig*`, `TestCheckEndpointReady*`, `TestResolveHealthCheckConfig`; runtime health checks. |
| `internal/agent/runtime/runplan_adapter_test.go` | Unit | `TestConvertRunplanToAgentSpec`, `TestConvertRunplanToAgentSpecNoGPU`; RunPlan-to-AgentRunSpec conversion. |
| `internal/agent/state/state_test.go` | Unit | `TestLoad_*`, `TestSetNodeID_Persists`, `TestCheckMismatch_*`, `TestIdentity_FilePermissions`; stable node identity. |

### 3.3 cmd, common, runtimecontract Tests

| File | Type | Test names / coverage |
| --- | --- | --- |
| `cmd/agent/exec_cmd_test.go`, `cmd/agent/main_test.go` | Unit | `TestExecCmdIncludesStderrOnFailure`, `TestApplyStartFailureDiagnostics*`, `TestApplyDefaultTaskResultStatus`; agent command and task-result helpers. |
| `cmd/server` | Gap | No test files. |
| `internal/common/errors/errors_test.go` | Unit | `TestAppError_*`, `TestIsAppError`, `TestStatusCode`; common error wrapper. |
| `internal/common/token/bootstrap_test.go` | Unit | `TestIsDefault`, `TestGenerate`, `TestWriteAndRead`, `TestBootstrap*`; bootstrap token helpers. |
| `internal/common/version/version_test.go` | Unit | `TestGet`, `TestString`; version formatting. |
| `internal/runtimecontract/constants_test.go` | Unit | `TestIsValid*`, uniqueness and Ollama inclusion checks; shared runtime contract constants. |

## 4. Frontend Test Inventory

### 4.1 Tests Included In `npm test`

| File | Type | Coverage |
| --- | --- | --- |
| `web/tests/apiClientPaths.test.mjs` | Static contract | Ensures API modules do not hardcode `/api/v1` prefix. |
| `web/tests/formatters.test.mjs` | Utility | Relative time formatting in zh-CN/en-US, abnormal future dates, cross-year formatting. |
| `web/tests/i18nKeys.test.mjs` | Static contract | Locale key parity between zh-CN and en-US. |
| `web/tests/i18nMissingKeys.test.mjs` | Static contract | `$t()` references exist, resolve to strings, and do not leak i18n keys. |
| `web/tests/noHardcodedCredentials.test.mjs` | Static security | Scans source/config/docs for hardcoded credential patterns. |
| `web/tests/runtimeBoundaryUi.test.mjs` | Static UI contract | Runtime boundary UI: no old authority fields, ConfigEditView usage, current routes, display adapter behavior, raw probe evidence collapsed, selected runtime naming, deployment wizard static checks. |
| `web/tests/modelCapabilities.test.mjs` | Utility | Model capability inference/default test-mode behavior and failure formatting. |
| `web/tests/configEditContract.test.mjs` | Utility/static contract | ConfigEdit patch generation, independent enabled/value behavior, stable test selectors in ConfigEdit components. |

Observed `npm test` output confirms all 8 scripts passed.

### 4.2 Frontend Tests Present But Not Included In `npm test`

| File | Type | Status | Coverage |
| --- | --- | --- | --- |
| `web/src/stores/__tests__/auth.test.ts` | Vitest-style unit | Not executed by current `npm test`; no Vitest script found | Multi-tenant login store behavior. |
| `web/src/pages/__tests__/dashboard.test.ts` | Vitest-style unit | Not executed by current `npm test`; no Vitest script found | Dashboard GPU memory/utilization aggregation, sorting, abnormal filtering, node aggregation. |
| `web/src/composables/__tests__/useAutoRefresh.test.ts` | Vitest-style unit | Not executed by current `npm test`; no Vitest script found | Auto-refresh core logic and visibility pause. |

### 4.3 Playwright E2E Tests

| File | Type | Command | Coverage |
| --- | --- | --- | --- |
| `web/tests/e2e/smoke/app-load.spec.ts` | Browser smoke | `cd web && npm run test:e2e` | App loads without backend. |
| `web/tests/e2e/smoke/fullstack-health.spec.ts` | Browser/API smoke | `cd web && npm run test:e2e` | Web can reach backend server. |
| `web/tests/e2e/auth/login.spec.ts` | Browser auth | `cd web && npm run test:e2e` | Admin storage state can access authenticated app. |
| `web/tests/e2e/auth/login-debug.spec.ts` | Browser debug | `cd web && npm run test:e2e` | Verbose login/change-password debug flow. |
| `web/tests/e2e/global.setup.ts` | Playwright setup | `cd web && npm run test:e2e` | Auth setup unless `LIGHTAI_SKIP_AUTH_SETUP=1`. |

These Playwright tests are not run by `npm test`.

## 5. E2E / Smoke Script Inventory

| Script | Type | Coverage | Gate status |
| --- | --- | --- | --- |
| `scripts/e2e-current-contract-api-dryrun.sh` | API-first dry-run | Current deployment contract without real container start. | Manual only. |
| `scripts/e2e-current-contract-nvidia-llamacpp-smoke.sh` | Real NVIDIA smoke | llama.cpp full start/stop contract; skips if Docker/NVIDIA/model missing. | Manual only. |
| `scripts/e2e-runtime-config-web-check-flow.sh` | API/Docker check flow | Runtime config enable/check-request positive and negative paths. | Manual only. |
| `scripts/e2e-real-smoke-all-three.sh` | Real runtime matrix | vLLM, SGLang, llama.cpp product API chain. | Manual only. |
| `scripts/e2e-model-runtime-wizard-nvidia-vllm.sh` | Real/API E2E | vLLM wizard/runtime/deployment path. | Manual only. |
| `scripts/e2e-model-runtime-wizard-nvidia-sglang.sh` | Real/API E2E | SGLang wizard/runtime/deployment path. | Manual only. |
| `scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh` | Real/API E2E | llama.cpp wizard/runtime/deployment path. | Manual only. |
| `scripts/e2e-model-runtime-param-trace.sh` | API trace | Backend/runtime/deployment parameter trace evidence. | Manual only. |
| `scripts/e2e-packaged-smoke.sh` | Package smoke | Release package/container API response validation. | Manual only. |
| `scripts/smoke-model-backends.sh` | Direct/backend smoke | Direct vLLM/SGLang/llama.cpp and RunPlan checks. | Manual only. |
| `scripts/diagnose-model-runtime-spec.sh` | Diagnostic | Dumps agent spec and diffs direct smoke references. | Manual diagnostic. |
| `scripts/verify-local.sh` | Local verify | Server/agent/Prometheus/Grafana endpoint checks. | Manual only. |
| `scripts/e2e/lib/*.sh` | Shared helpers | API client, assertions, cleanup, Docker/GPU readiness, reports/resources. | Support library. |
| `scripts/archive/legacy-contract/*.sh` | Archived E2E | Historical dry-run, clone, model runtime, failed logs, inference parser, matrix verifier. | Archived; not regular gate. Some paths may be stale by design. |

No top-level `e2e/` or `web/e2e/` directory exists in the current tree. Active browser E2E lives under `web/tests/e2e`.

## 6. Phase 3 Documented Tests Vs Current Code

| Document area | What docs record | Current code status |
| --- | --- | --- |
| `docs/reports/phase-3/runtime-config-display-probe-fix/05-closeout.md` | Documents fixes for ConfigEditView envelope unwrap, clone display/name/version, probe summary/raw collapse, deployment NBR display name, and lists `go test ./internal/server/api ./internal/server/runplan`, `npm test`, `npm run build`. | Corresponding Go tests now exist: `TestCloneBackendRuntimeWithUserVisibleDisplayName`, `TestCloneBackendRuntimeNoDisplayNameUsesGeneratedName`, `TestCatalogSeedProducersUserVisibleDisplayNames`; frontend static tests now assert `extractVersion` returns `*`, raw probe evidence is collapsed, and ConfigEdit widgets exist. |
| `docs/reports/phase-3/runtime-config-display-probe-fix/07-config-field-display-design.md` | Requires widget-level display tests for Docker options, model mount, env, health, null display, and preserving prior fixes. | Partially covered by `web/tests/runtimeBoundaryUi.test.mjs` and `web/tests/configEditContract.test.mjs` through static string/contract checks. No browser/component render test verifies actual DOM rows/values. |
| Phase 3 runtime metadata/config review docs | Mention `go test` commands, `npm test`, `npm run build`, runtime/RunPlan/compat tests, and E2E scripts. | Go unit/API tests are present. E2E scripts are present but mostly manual and environment-dependent. Archived legacy scripts are not regular gates. |
| Historical verification docs under phase-3 | Reference old field names such as `parameter_schema_json`, `env_json`, or archived E2E paths. | Active web tests assert old authority fields are not referenced. Current code uses ConfigSet/ConfigEditView. Treat old docs as historical unless cross-referenced by current closeout. |

## 7. Capability Coverage Matrix

| Capability | Backend/API tests | Frontend tests | E2E/smoke | Coverage judgment |
| --- | --- | --- | --- | --- |
| Runtime / BackendRuntime | `runtime_boundary_test.go`, `workflow_backend_runtime_test.go`, `phase3_rbac_test.go`, `ui_persistence_runplan_test.go`, catalog tests | `runtimeBoundaryUi.test.mjs` | wizard/runtime scripts | Strong API coverage; frontend is mostly static, not browser-rendered. |
| NodeBackendRuntime | `runtime_boundary_test.go`, `workflow_nbr_probe_test.go`, `nbr_deployable_test.go`, `config_edit_handlers_test.go` | `runtimeBoundaryUi.test.mjs`, `configEditContract.test.mjs` | `e2e-runtime-config-web-check-flow.sh` | Strong API/probe coverage; real node check is manual. |
| ConfigEditView | `configedit_test.go`, `config_edit_handlers_test.go`, catalog view tests | `configEditContract.test.mjs`, `runtimeBoundaryUi.test.mjs` | Indirect through wizard scripts | Strong backend projection/patch coverage; weak DOM rendering coverage. |
| RunPlan | `internal/server/runplan/*_test.go`, `deployment_preflight_contract_test.go`, `workflow_deployment_runplan_test.go` | Static RunPlan UI assertions only | dry-run and real smoke scripts | Strong resolver/API dry-run coverage; real Docker execution manual. |
| Deployment | `deployment_preflight_contract_test.go`, `workflow_deployment_runplan_test.go`, `workflow_lifecycle_test.go`, `ui_persistence_runplan_test.go` | `runtimeBoundaryUi.test.mjs`, DeploymentWizard static checks | API dry-run and real smoke scripts | Strong API coverage; limited browser coverage. |
| ModelInstance | `workflow_lifecycle_test.go`, `runtime_boundary_test.go`, `agent_task_result_test.go`, `node_run_plan_logs_test.go` | ModelInstances page not included in `npm test` beyond static references | real smoke scripts | Good API/lifecycle coverage; browser instance UI not covered by regular test. |
| Probe evidence | `runtime_boundary_test.go` check-request/probe tests, `workflow_nbr_probe_test.go`, `nbr_deployable_test.go` | Static raw-collapse assertion in `runtimeBoundaryUi.test.mjs` | `e2e-runtime-config-web-check-flow.sh` | Backend evidence semantics covered; UI default summary not browser-rendered. |
| `display_name` / `name` / version | `runtime_boundary_test.go`, `ui_persistence_runplan_test.go`, catalog seed tests | `runtimeBoundaryUi.test.mjs` | Not regular | Recent regressions now partially covered; still lacks browser/API joined assertion for clone dialog visible default. |
| API auth/RBAC | `agent_identity_test.go`, `tenant_isolation_test.go`, `tenant_rbac_negative_test.go`, `phase3_rbac_test.go`, auth/authz tests | Auth store Vitest-style tests exist but not in `npm test` | Playwright auth exists but separate | Good backend coverage; frontend auth tests not included in regular npm gate. |
| Agent/Docker lifecycle | `internal/agent/runtime/*_test.go`, `cmd/agent/*_test.go`, `workflow_lifecycle_test.go` | None | real smoke scripts | Fake Docker/unit coverage strong; real Docker lifecycle manual. |
| Frontend UI display | `runtimeBoundaryUi.test.mjs`, `configEditContract.test.mjs`, i18n tests | Mostly static source inspection | Playwright smoke/auth only | Weak for real rendered runtime/config/probe pages. |

## 8. Recent Runtime / Config / Display / Probe / RunPlan Coverage

| Recent issue class | Existing coverage | Remaining gap |
| --- | --- | --- |
| Runtime detail parameters empty due to `/config-edit/view` envelope mismatch | `npm test` now includes ConfigEdit static/contract checks; backend API projection test exists. | No direct browser/component test opens runtime detail and verifies rendered sections/fields with API-shaped response. |
| Clone display_name/name/version semantics | Go tests cover clone naming and catalog seed display names; frontend static tests cover `runtimeDisplay` version `*` and tech slug normalization. | No API+frontend integration test asserts clone dialog default, clone result list title, and detail technical name/version simultaneously. |
| Probe raw evidence default display | Frontend static test asserts raw probe evidence is collapsed; backend probe tests cover storage/status. | No DOM test proves default page text excludes `NVIDIA_REQUIRE_CUDA`, `PATH`, `LD_LIBRARY_PATH`; no backend test explicitly asserts Docker `Config.Env` never enters `config_set_json`/RunPlan env. |
| RunPlan env pollution | RunPlan `buildEnv` and resolver tests cover configured env sources and filtering; dry-run tests cover probe-derived process start config. | Missing targeted regression: inspect `.Config.Env` containing `NVIDIA_REQUIRE_CUDA` is persisted only as raw evidence and absent from `ResolvedRunPlan.env` and AgentRunSpec. |
| ConfigEdit object/list rendering | ConfigEdit backend tests cover projection and patch; frontend static tests cover widgets and null fallback. | No component/browser test verifies actual rendered nested Docker options, mount, env, health values. |

## 9. Explicit Gaps

### 9.1 Real Problems That Previously Lacked Coverage

| Problem | Why previous tests missed it | Current status |
| --- | --- | --- |
| ConfigEditView received API envelope instead of `config_edit_view` | Backend test accepted both envelope and old top-level shape; frontend static tests did not call API client with real response shape. | Partially closed by later static/contract checks, but still no rendered browser test. |
| Clone displayed technical `runtime.*` name | Tests checked persistence and API fields more than user-visible default in the clone dialog/list. | Partially closed by Go clone/catalog tests and runtimeDisplay static checks. |
| User-facing version showed concrete software version | Runtime display adapter was not previously asserted for cloned user configs. | Static check now asserts version `*`, but no browser list/detail test. |
| Raw Docker inspect env shown by default | Backend correctly stored raw evidence, but UI default rendering was not browser-tested. | Static check now asserts raw probe evidence collapsed, but no text-negative browser assertion. |
| Docker inspect `.Config.Env` pollution risk | Existing tests validate env resolution from ConfigSet/overrides, not negative propagation from probe evidence. | Still a P0 test gap. |

### 9.2 Tests That Verify Flow Success But Not Field Semantics

| Test/script class | Gap |
| --- | --- |
| API workflow tests | Often assert status/code and high-level fields, but not every user-facing display field or i18n label. |
| E2E shell scripts | Many validate product chain success, but not always exact JSON field lineage, source maps, collapsed UI text, or display_name/name distinctions. |
| Playwright smoke/auth | Confirms app/auth loads, not runtime/config/probe pages. |
| Frontend static tests | Catch source-code patterns, but can pass even when rendered DOM is wrong due to runtime data shape or CSS/Element Plus behavior. |

### 9.3 E2E Scripts Not In Regular Gate

All active `scripts/e2e-*`, `scripts/smoke-model-backends.sh`, and Playwright `npm run test:e2e` are separate manual gates. They require local services, Docker, GPU, model files, or seeded data. They are not run by `npm test`, and no single root script currently runs Go + frontend + selected API-first E2E as a standard lightweight CI gate.

## 10. Recommended Test Additions

### P0: Must Add Next

1. **ConfigEdit API client unwrap test**: direct test for `getConfigEditView()` receiving `{config_edit_view, config_view}` and returning the inner view.
2. **Probe env boundary regression**: backend test where image inspect returns `Config.Env` with `NVIDIA_REQUIRE_CUDA`, `PATH`, `LD_LIBRARY_PATH`; assert it remains only in `probe_results_json.level2.env` and does not enter NBR `config_set_json`.
3. **RunPlan env pollution regression**: deployment dry-run/start preview after such probe; assert `ResolvedRunPlan.env` and AgentRunSpec env exclude Docker image `Config.Env` unless explicitly configured.
4. **Rendered runtime detail test**: browser/component test opens BackendRuntimes detail with mocked/current API data and asserts ConfigEdit sections and key fields render.
5. **Rendered probe summary negative test**: browser/component test opens RunnerConfigs detail and asserts default view does not contain full `NVIDIA_REQUIRE_CUDA`, `PATH`, `LD_LIBRARY_PATH`, while raw JSON is collapsed.

### P1: Add Soon

1. Clone runtime UI integration: clone dialog default uses product display name, list/detail show display_name, technical name appears only as auxiliary detail.
2. Deployment list/detail display: assert NBR display name is shown instead of raw `source_node_backend_runtime_id` where available.
3. Add a regular `web` test runner for existing `web/src/**/__tests__/*.ts` or remove/convert those tests so they are not silently orphaned.
4. API-first E2E lightweight gate: run dry-run contract and runtime config check flow in a documented local/CI profile with skips made explicit.
5. More source-map assertions for env and Docker options source lineage in dry-run outputs.

### P2: Later Enhancements

1. Playwright runtime/config/deployment happy-path smoke with mocked or seeded backend data.
2. Real Docker/GPU smoke scheduling profile for nightly/manual validation.
3. Coverage summary generation (`go test -cover`, frontend coverage if Vitest is adopted).
4. Script inventory health check that flags archived/stale E2E scripts separately from active scripts.
5. Visual regression snapshots for ConfigEdit widgets after Element Plus or layout changes.

## 11. Closure Status

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TEST-GAP-001 | `web/src/**/__tests__/*.ts` tests exist but are not run by `npm test`. | `web/package.json` has no Vitest script; `npm test` runs only 8 `web/tests/*.mjs` scripts. | Frontend store/dashboard/composable unit tests may silently rot. | DOCUMENTED_BLOCKER | `web/package.json` test strategy; optional Vitest setup. | Add/enable runner, then run `cd web && npm test` or new script. | Keep documented as test infrastructure gap. |
| TEST-GAP-002 | Runtime/config/probe UI tests are mostly static source checks, not rendered DOM tests. | `runtimeBoundaryUi.test.mjs` inspects source strings; Playwright only smoke/auth. | UI regressions can pass tests if source tokens remain but rendering/data shape breaks. | DOCUMENTED_BLOCKER | Add component/browser tests for BackendRuntimesPage and RunnerConfigsPage. | Run new browser/component tests plus `npm test`. | Prioritize P0 rendered runtime detail and probe summary tests. |
| TEST-GAP-003 | Docker image inspect `.Config.Env` has no explicit negative propagation regression. | Current backend tests cover probe storage and RunPlan env sources, but not `Config.Env` absent from ConfigSet/RunPlan/AgentRunSpec. | Future probe changes could pollute user config or container env. | DOCUMENTED_BLOCKER | `internal/server/api/runtime_boundary_test.go`, `deployment_preflight_contract_test.go`, possibly `internal/agent/runtime/runplan_adapter_test.go`. | Add negative assertions with `NVIDIA_REQUIRE_CUDA`, `PATH`, `LD_LIBRARY_PATH`. | Treat as P0. |
| TEST-GAP-004 | Active E2E/smoke scripts are not regular gates. | Scripts exist under `scripts/`, Playwright has separate `npm run test:e2e`; none are run by `npm test` or the Go test command. | Full product-chain regressions depend on manual execution. | DOCUMENTED_BLOCKER | Test runbook/CI profile, likely scripts plus docs. | Run a selected lightweight E2E command in a documented profile. | Treat API-first dry-run/check-flow as P1 gate candidate. |

No test gaps discovered in this review are left only in chat; the gaps above are recorded in this document.
