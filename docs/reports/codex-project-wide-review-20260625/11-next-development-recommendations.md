# Next Development Recommendations

## Batch 1: Contract and readiness hardening

Goal: remove the highest-risk deployment correctness gaps.

- Remove or lock down `POST /nodes/{id}/backend-runtimes/check`.
- Make `/check-request` the only UI/server NBR verification path.
- Reject unknown legacy deployment fields such as `parameters_json` or explicitly translate them with warnings removed after one release.
- Make `/deployments/preflight` share the final resolver path or introduce a new `/deployments/preview` contract for final RunPlan.
- Add tests for false-ready NBR, `ready_with_warnings`, missing image, mismatch image, and stale NBR snapshot.

Acceptance:

- `go test ./internal/server/api ./internal/server/runplan` covers the above.
- Active E2E scripts no longer send deprecated deployment `backend_runtime_id`.

## Batch 2: E2E and documentation convergence

Goal: make evidence trustworthy.

- Repair active scripts under `scripts/e2e-*`.
- Archive stale scripts that cannot be repaired now.
- Update `docs/api/openapi.yaml` to current routes.
- Mark historical evidence directories as historical by contract version.
- Add `docs/testing/current-e2e-contract.md`.

Acceptance:

- One current API-first E2E passes on non-GPU dry-run.
- One NVIDIA Docker smoke is runnable and records hardware skips honestly.

## Batch 3: UI workflow repair

Goal: prevent user confusion in deployment/runtime workflows.

- Remove deployment edit runtime selector until NBR change is implemented.
- Add explicit NBR change flow with warning, diff, `needs_check`, and no effect on running instances.
- Add aggregate NBR endpoint to reduce page fan-out.
- Add Playwright smoke for model wizard → runner config → deployment preview.

Acceptance:

- Browser smoke covers create preview and blocked non-ready NBR.
- UI labels use NBR terminology consistently.

## Batch 4: Security policy and tenant hardening

Goal: reduce platform risk before multi-user operation.

- Add policy gate for privileged Docker options and host mounts/devices.
- Introduce per-agent/node credentials or scoped tokens.
- Add full tenant negative route matrix.
- Remove `tenant_id DEFAULT 'default'` schema remnants.

Acceptance:

- Tenant cannot access or mutate another tenant's NBR/model/deployment/GPU.
- Dangerous Docker options require explicit platform-admin policy.

## Batch 5: Scale and reliability

Goal: stabilize small fleet behavior.

- Add GPU lease uniqueness/transactional conflict tests.
- Add node-offline and task-timeout reconciliation tests.
- Add pagination/index audit for list APIs.
- Add log API byte/tail limits.

Acceptance:

- Concurrent start of same GPU/port fails deterministically.
- Offline node with active task transitions to a consistent terminal state.
