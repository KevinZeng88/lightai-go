# Validation and Smoke Review

## Validation Adequacy

| Area | Current Plan | Adequate? | Required Amendment |
| --- | --- | --- | --- |
| Go tests | Global `go test ./...`, focused package tests. | Yes | Keep focused tests before full suite. |
| Go builds | Server and agent builds in each code batch. | Yes | Add `git diff --check` before commit, already in commit strategy. |
| Web tests | `cd web && npm test`, build. | Partial | Browser workflow changes need Playwright or explicit component coverage with blocker evidence. |
| API-first E2E | New dry-run script and current contract docs. | Partial | Must require a current active-script manifest and command that fails on stale active scripts. |
| Docker dry-run | Covered through preflight/dry-run/start convergence. | Partial | Dry-run must include final AgentRunSpec/Docker command sample assertion. |
| NVIDIA real smoke | Smoke plan exists with SKIP rules. | Partial | Needs deterministic server/agent startup and cleanup. |
| OpenAPI | Batch 3 updates routes. | Partial | Add validation of YAML syntax, absence of stale paths, and sample request/response checks. |
| Playwright | Conditional if dependency exists. | Partial | Current dirty `web/package*.json` suggests Playwright may already be introduced. Make the plan decide how to handle it. |
| Tenant/RBAC negative matrix | Batch 5 lists tests. | Partial | Add explicit route x role x tenant matrix and required forbidden responses. |
| Docker policy negative tests | Batch 5 lists tests. | Partial | Must prove policy at save, preview, and start. |
| Agent token negative tests | Batch 5 lists tests. | Partial | Must include cross-node token reuse and mismatched node_id/task result. |
| Performance | Bundle warning and API audit. | Partial | If accepting larger chunk threshold, require written threshold and measured build output. |

## Commands That Should Be Added

```bash
git diff --check
rg -n "backend_runtime_id|parameters_json|/backend-runtimes/check|image_present|docker_available" scripts docs/api docs/testing
rg -n "/runtime-environments|/run-templates|/model-deployments" docs/api/openapi.yaml
python3 - <<'PY'
import yaml
yaml.safe_load(open('docs/api/openapi.yaml'))
PY
```

If the project does not want PyYAML as a dependency, replace the YAML command with the repo's chosen OpenAPI validation tool and document the install/SKIP rule.

## Smoke Harness Requirements

The smoke plan should define:

- Temp DB/data/log directories.
- Exact server command and flags.
- Exact agent command and flags.
- Port availability checks for `18080`, `19091`, `19090`, `13000`, and any model ports.
- PID files and cleanup trap.
- Readiness polling for `/healthz` and agent online.
- Maximum wait times for start/running/healthy.
- Docker cleanup label/name prefix and final `docker ps` check.
- Evidence directory naming.

Without these, the smoke plan can hang, mutate an existing dev server, or leave containers/processes behind.

## Validation Verdict

The validation matrix is a strong base, but it does not yet prove enough for AUTORUN. The main missing pieces are deterministic runtime smoke automation, OpenAPI sample validation, browser smoke policy, and hard negative security tests.
