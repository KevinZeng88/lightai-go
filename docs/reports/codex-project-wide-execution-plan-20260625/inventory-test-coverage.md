# Inventory — Test Coverage

## Go Packages
ok  	lightai-go/cmd/agent	0.005s	coverage: 4.4% of statements
	lightai-go/cmd/server		coverage: 0.0% of statements
ok  	lightai-go/internal/agent/collector	0.711s	coverage: 58.4% of statements
ok  	lightai-go/internal/agent/metrics	(cached)	coverage: 37.9% of statements
ok  	lightai-go/internal/agent/register	(cached)	coverage: 45.9% of statements
ok  	lightai-go/internal/agent/runtime	(cached)	coverage: 66.3% of statements
ok  	lightai-go/internal/agent/state	(cached)	coverage: 80.0% of statements
	lightai-go/internal/common/config		coverage: 0.0% of statements
ok  	lightai-go/internal/common/errors	(cached)	coverage: 90.5% of statements
	lightai-go/internal/common/log		coverage: 0.0% of statements
ok  	lightai-go/internal/common/token	(cached)	coverage: 29.3% of statements
?   	lightai-go/internal/common/types	[no test files]
ok  	lightai-go/internal/common/version	(cached)	coverage: 100.0% of statements
ok  	lightai-go/internal/runtimecontract	(cached)	coverage: 100.0% of statements
ok  	lightai-go/internal/server/agentclient	(cached)	coverage: 84.8% of statements
ok  	lightai-go/internal/server/api	(cached)	coverage: 56.2% of statements
ok  	lightai-go/internal/server/auth	(cached)	coverage: 3.3% of statements
ok  	lightai-go/internal/server/authz	(cached)	coverage: 17.6% of statements
	lightai-go/internal/server/db		coverage: 0.0% of statements
	lightai-go/internal/server/metrics		coverage: 0.0% of statements
	lightai-go/internal/server/models		coverage: 0.0% of statements
	lightai-go/internal/server/rbac		coverage: 0.0% of statements
ok  	lightai-go/internal/server/runplan	(cached)	coverage: 70.6% of statements
	lightai-go/web		coverage: 0.0% of statements

## Go Test Files (50)
internal/agent/collector/gguf_reader_test.go
internal/agent/collector/model_scanner_test.go
internal/agent/collector/nvidia_test.go
internal/agent/collector/probe_test.go
internal/agent/collector/protocol_test.go
internal/agent/metrics/metrics_test.go
internal/agent/register/register_test.go
internal/agent/runtime/docker_test.go
internal/agent/runtime/health_test.go
internal/agent/runtime/runplan_adapter_test.go
internal/agent/state/state_test.go
internal/common/errors/errors_test.go
internal/common/token/bootstrap_test.go
internal/common/version/version_test.go
internal/runtimecontract/constants_test.go
internal/server/agentclient/client_test.go
internal/server/api/agent_identity_test.go
internal/server/api/agent_task_result_test.go
internal/server/api/api_workflow_harness_test.go
internal/server/api/api_workflow_test_helper_test.go
internal/server/api/fake_agent_test.go
internal/server/api/metax_device_binding_test.go
internal/server/api/middleware_logging_test.go
internal/server/api/middleware_recovery_test.go
internal/server/api/model_capability_test.go
internal/server/api/model_root_policy_test.go
internal/server/api/nbr_deployable_test.go
internal/server/api/node_run_plan_logs_test.go
internal/server/api/phase3_rbac_test.go
internal/server/api/resource_handlers_test.go
internal/server/api/runtime_boundary_test.go
internal/server/api/tenant_isolation_test.go
internal/server/api/ui_persistence_runplan_test.go
internal/server/api/workflow_backend_runtime_test.go
internal/server/api/workflow_deployment_runplan_test.go
internal/server/api/workflow_lifecycle_test.go
internal/server/api/workflow_model_wizard_test.go
internal/server/api/workflow_nbr_probe_test.go
internal/server/auth/bootstrap_test.go
internal/server/authz/checks_test.go
internal/server/authz/helpers_test.go
internal/server/runplan/compat_test.go
internal/server/runplan/lint_test.go
internal/server/runplan/llamacpp_nvidia_test.go
internal/server/runplan/log_classifier_test.go
internal/server/runplan/metax_huawei_test.go
internal/server/runplan/resolver_test.go
internal/server/runplan/resource_controls_test.go
internal/server/runplan/test_helpers_test.go
internal/server/runplan/vllm_sglang_nvidia_test.go

## Web Test Files (26)
web/node_modules/de-indent/test.js
web/node_modules/path-browserify/test/test-path-basename.js
web/node_modules/path-browserify/test/test-path-dirname.js
web/node_modules/path-browserify/test/test-path-extname.js
web/node_modules/path-browserify/test/test-path-isabsolute.js
web/node_modules/path-browserify/test/test-path-join.js
web/node_modules/path-browserify/test/test-path-parse-format.js
web/node_modules/path-browserify/test/test-path-relative.js
web/node_modules/path-browserify/test/test-path-resolve.js
web/node_modules/path-browserify/test/test-path-zero-length-strings.js
web/node_modules/path-browserify/test/test-path.js
web/node_modules/playwright/test.d.ts
web/node_modules/playwright/test.js
web/node_modules/playwright/test.mjs
web/node_modules/playwright/types/test.d.ts
web/node_modules/playwright/types/testReporter.d.ts
web/src/composables/__tests__/useAutoRefresh.test.ts
web/src/pages/__tests__/dashboard.test.ts
web/src/stores/__tests__/auth.test.ts
web/tests/apiClientPaths.test.mjs
web/tests/formatters.test.mjs
web/tests/i18nKeys.test.mjs
web/tests/i18nMissingKeys.test.mjs
web/tests/modelCapabilities.test.mjs
web/tests/noHardcodedCredentials.test.mjs
web/tests/runtimeBoundaryUi.test.mjs

## Coverage Gaps (P0-P1)
- R-010: auth/authz/db/rbac/main/metrics have low/no coverage
- No browser/Playwright tests
- Static analysis tests only (string matching in .vue files)
