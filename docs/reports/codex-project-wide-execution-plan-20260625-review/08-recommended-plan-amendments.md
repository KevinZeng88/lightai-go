# Recommended Plan Amendments

## Amendment Table

| Target Document | Section | Current Issue | Suggested Change |
| --------------- | ------- | ------------- | ---------------- |
| `01-execution-policy-and-scope.md` | Execution mode | Dirty workspace and commit isolation are not hard-gated. | Add the workspace gate text below. |
| `01-execution-policy-and-scope.md` | Failure handling | Failed batch push policy is unclear. | Add "failed implementation batches must not push code". |
| `02-risk-to-workstream-map.md` | Status values | Allows `INTENTIONALLY_DEFERRED_WITH_OWNER_AND_ACCEPTANCE`. | Remove for R-001 to R-015; use only `CLOSED`, `CLOSED_BY_SCOPE_REDUCTION`, `BLOCKED_BY_EXTERNAL_DEPENDENCY`. |
| `04-batch-1-contract-readiness-hardening.md` | 1.1 | Allows route removal or wrapper. | Set default to wrapper-only with ignored request body, or explicitly remove; do not leave choice open. |
| `07-batch-4-ui-workflow-repair.md` | 4.1 | Allows full NBR change flow in same batch. | Default to remove selector; full change flow is separate optional sub-batch after R-004 is closed. |
| `08-batch-5-security-tenant-hardening.md` | Whole file | Batch too broad. | Split into 5A Agent credentials, 5B Docker policy, 5C tenant/schema/RBAC. |
| `08-batch-5-security-tenant-hardening.md` | Docker policy | Policy matrix not exact. | Add option-level role/policy matrix and enforcement points. |
| `12-validation-matrix.md` | Batch 3 | OpenAPI validation not concrete. | Add YAML validation, stale path grep, and sample contract validation. |
| `12-validation-matrix.md` | Batch 4 | Browser smoke conditional. | Require Playwright if dependency exists; otherwise component tests plus blocker evidence. |
| `13-autonomous-codex-execution-prompt.md` | Execution authorization | Does not include amended workspace gate/split batches. | Update after amendments so AUTORUN prompt is self-contained. |
| `15-runtime-smoke-plan.md` | Smoke 1 | Startup/cleanup vague. | Add deterministic server/agent harness text below. |
| `16-commit-and-push-strategy.md` | Worktree pollution | Pathspec-limited commit not required. | Add exact `git add` and baseline-diff rules. |

## Insertable Text: Workspace Gate

Add to `01-execution-policy-and-scope.md` and `16-commit-and-push-strategy.md`:

```markdown
## Pre-AUTORUN Workspace Gate

Before Batch 0 implementation work, record:

```bash
git status --short
git diff --stat
git diff -- web/package.json web/package-lock.json
```

Create `docs/reports/codex-project-wide-execution-plan-20260625/workspace-baseline.md`.

Rules:

- Do not include baseline unrelated files in any batch commit.
- Use pathspec-limited `git add` for each batch.
- `.mimocode/` is never committed unless the user explicitly asks.
- Existing untracked E2E evidence directories are not committed unless a batch explicitly reclassifies or marks them.
- If a batch needs to touch a baseline-modified file, closeout must show before/after diff and explain why it became in-scope.
- If `git status --short` contains a new unexplained path before commit, stop that batch and document it.
```

## Insertable Text: Failed Batch Push Rule

Add to `01-execution-policy-and-scope.md`:

```markdown
## Failed Batch Push Rule

If implementation validation fails because of code, tests, contract, or build errors, do not push that batch. Fix in-place and rerun validation.

If validation is blocked only by an external dependency such as Docker daemon, GPU, model file, browser binary, network, or GitHub credentials, commit only documentation/evidence that records the blocker. Do not push partial implementation code as complete.
```

## Insertable Text: Batch 1 Default Decision

Replace the optional wording in `04-batch-1-contract-readiness-hardening.md` with:

```markdown
Default implementation for AUTORUN:

- Keep `POST /api/v1/nodes/{id}/backend-runtimes/check` only as a compatibility route name.
- The handler must ignore `image_present`, `docker_available`, and any readiness evidence in the request body.
- The handler must call the same server-to-Agent probe path as `/check-request`.
- The response and persisted NBR status must be derived only from Agent/server probe evidence.
- Add a regression test proving a session caller cannot set ready by sending `image_present=true,docker_available=true`.
```

## Insertable Text: Batch 4 Default Decision

Add to `07-batch-4-ui-workflow-repair.md`:

```markdown
AUTORUN default: remove the deployment edit runtime selector in this batch. Do not implement full NBR change flow unless the removal, tests, and build have already passed and it can be done as a separate sub-batch with its own closeout.
```

## Insertable Text: Batch 5 Split

Replace Batch 5 execution order with:

```markdown
Batch 5A - Agent node-bound credentials:
- Bootstrap token may only register.
- Node-bound token is required for heartbeat, task claim/result, docker inspect, file browse, and model scan.
- Cross-node token reuse fails.

Batch 5B - Docker dangerous options policy:
- Default deny `privileged`, raw `devices`, host mounts outside model roots, `network_mode=host`, `pid_mode=host`, `ipc_mode=host`, `cap_add`, `security_opt`, arbitrary entrypoint/command override, and raw env injection.
- Enforcement must happen at save, preview/preflight/dry-run, and start.

Batch 5C - Tenant/schema/RBAC:
- Fresh DB schema has no `tenant_id DEFAULT 'default'`.
- Handler-side table creation is removed or proven unreachable in normal server init.
- Tenant negative matrix covers nodes, GPUs, NBRs, models, deployments, instances, aggregate endpoints, and policy routes.
```

## Insertable Text: Runtime Smoke Harness

Add to `15-runtime-smoke-plan.md`:

```markdown
## Deterministic Local Harness

Smoke must run against an isolated local server/agent unless an existing process is explicitly selected and recorded.

Required harness behavior:

- Create a temp data directory under `run/codex-smoke-YYYYMMDDHHMMSS/`.
- Pick ports or verify `18080` and `19091` are free before starting.
- Start server and agent with logs redirected to the evidence directory.
- Record PIDs.
- Poll health endpoints with a maximum timeout.
- Use a run-specific deployment/container name prefix.
- Install a shell trap that stops server, agent, and any matching containers on exit.
- Record final `docker ps --filter name=<prefix>` and `git status --short`.
- If a required port is occupied by an unrelated process, SKIP with `ss -ltnp` evidence.
```

## Insertable Text: OpenAPI Validation

Add to `12-validation-matrix.md` Batch 3:

```bash
rg -n "/runtime-environments|/run-templates|/model-deployments" docs/api/openapi.yaml && exit 1 || true
rg -n "backend_runtime_id|parameters_json" docs/api/openapi.yaml && exit 1 || true
python3 - <<'PY'
import yaml
with open('docs/api/openapi.yaml') as f:
    doc = yaml.safe_load(f)
assert '/api/v1/deployments' in doc.get('paths', {})
assert '/api/v1/deployments/preflight' in doc.get('paths', {})
PY
```

If PyYAML is unavailable, add a repo-local validator script or record an external dependency blocker.
