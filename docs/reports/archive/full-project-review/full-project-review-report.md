> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# LightAI Go Full Project Review Report

## Overall Conclusion

LightAI Go has a credible lightweight platform foundation: Server/Agent split, SQLite current state, basic auth/RBAC, GPU collector abstraction, Web Console, metrics, packaging scripts, and an emerging BackendRuntime/RunPlan model runtime design. It is suitable for continued development, but not yet suitable for an unconstrained pilot delivery.

The main blocker is not breadth of features. The main blocker is that several core control-plane guarantees are incomplete: secure Agent admission, tenant isolation on direct resource APIs, Server-to-Agent runtime spec fidelity, task lease/idempotency, runtime reconciliation, database migration discipline, and up-to-date operational documentation.

## Maturity Judgment

| Area | Assessment |
|---|---|
| Node/GPU visibility | Partially mature for development and demos; needs tenant direct-ID hardening and MetaX field evidence. |
| Auth/RBAC | Good foundation, but security defaults and audit scoping need hardening. |
| Model runtime | Demo-stage. The RunPlan design is promising, but the actual Agent payload/Docker execution path has high-risk inconsistencies. |
| Observability | Useful development surface; not yet secure or supervised enough as a product default. |
| Packaging/patching | Substantial work exists, but upgrade/migration and release docs remain risky. |
| Web Console | Usable MVP, but test wiring and API/docs consistency lag implementation. |
| Testing | Go unit tests pass, but cross-component runtime, migration, tenant direct-ID, script, and E2E coverage are insufficient. |

## Most Important Risks

1. **Agent admission is not secure by default.** Release and agent configs ship with `lightai-agent-token-change-me`, and Server/Agent continue after warning. This can allow fake agents to register and report state.
2. **Tenant isolation has direct-ID gaps.** `GET /api/v1/gpus/{id}` does not check tenant scope, node transfer does not update existing GPUs, and audit logs are scoped by operator membership rather than resource tenant.
3. **Model runtime may fail despite correct-looking dry run.** Server omits key fields in `AgentRunSpec`; Agent Docker driver relies on `Vendor` for NVIDIA DeviceRequests and ignores Docker entrypoint mapping.
4. **Task/state reliability is below the engineering contract.** The code lacks lease owner/expires, operation generation, idempotent task result handling, and periodic reconciliation.
5. **Docs/scripts disagree with code.** Some ops docs still call removed `/runtime-environments` and `/run-templates` APIs, while code exposes `/backend-runtimes`.

## Findings By Direction

### Product Maturity

Already relatively strong:

- Basic Server/Agent startup, health, metrics, and resource report architecture.
- Web pages for dashboard, nodes, GPUs, observability, auth/RBAC, and model runtime objects.
- Packaging, patch, log collection, and password reset scripts.

Still demo-level or not delivery-ready:

- Model deployment start/stop path.
- MetaX hardware support claims.
- Upgrade from legacy tenant data.
- End-to-end operations documentation.
- Frontend tests and API contract docs.

### Architecture

The Server/Agent boundary is well stated in docs and mostly respected: Server owns DB/API/RunPlan; Agent owns collection and Docker execution. However, the implementation does not yet satisfy the strongest contracts in `docs/08-engineering-contracts.md`:

- Task claim lacks full lease/generation semantics.
- Agent does not reconcile existing managed containers after restart.
- DockerRunSpec/RunPlan is frozen in DB, but the payload sent to Agent is a hand-built map that omits fields.
- Resource schemas are split between migrations and handler initialization.

### Security

Critical security concerns:

- Default Agent token is allowed in release paths.
- GPU direct detail endpoint lacks tenant authorization.
- Audit scoping is not based on resource tenant.
- TLS/HTTPS is not implemented while release config binds Server to `0.0.0.0`.
- Several runtime templates use privileged containers.

Positive security foundations:

- Session and Agent token are separate.
- CSRF token and Origin validation exist for state-changing user routes.
- Password hashing and credential files are present.
- Sensitive log redaction exists in several code paths.

### Stability And Reliability

The largest reliability gaps are in runtime lifecycle:

- Claim is select-then-update, not a full conditional lease claim.
- Failed results can store `actual_state='error'`, outside documented states.
- Stop is not idempotent when a container is missing.
- No complete reconciliation loop exists to detect crashed/removed containers.
- No full E2E was run in this review because scripts can mutate Docker/runtime state.

### Observability And Logs

Strengths:

- Server and Agent expose metrics.
- Agent `/metrics` reads latest snapshot only.
- Logs include operation IDs for deployment/task paths.
- Docker start failure tries to collect inspect/log tail.

Gaps:

- Bundled Prometheus/Grafana supervision is still script-level.
- Release-mode exposure and auth defaults need hardening.
- Audit logs need tenant_id and stronger resource scoping.
- No verified "no data" operational matrix for observability modes in this review.

### Model Runtime

The new BackendRuntime/RunPlan model is directionally better than old RuntimeEnvironment + RunTemplate + Deployment merging. The implementation is not yet safe enough:

- `HandleStartDeployment` manually builds `agentSpec` and omits `vendor`, `volumes`, `devices`, and `ports`.
- Agent Docker driver only creates NVIDIA DeviceRequests when `spec.Vendor == "nvidia"`.
- Agent Docker driver maps `spec.Docker.Args` into Docker `Cmd` and does not use `spec.Docker.Command` as entrypoint.
- Create deployment accepts weak references and defers validation.

### GPU Adaptation

NVIDIA and script-based vendor protocol are reasonable for the product goal. MetaX remains unverified on hardware. GPU tenant ownership has correctness issues after node transfer, and `collected_at` currently represents server receive time in storage instead of collector sample time.

### Web / UI / API

Web build passes. API client normalizes `/api/v1` paths. However:

- `npm test` is not wired.
- `vitest` is not installed despite test files.
- Some docs and scripts call removed endpoints.
- OpenAPI is stale/incomplete.
- Build emits a large chunk warning.

### Database And Migration

SQLite is acceptable for small deployments, but migration discipline is not yet product-grade:

- `migrateV10` drops old runtime tables and does not migrate old runtime/deployment data.
- v0.1.9 release note requires deleting legacy DB for old tenant values.
- Resource tables are created from handler initialization and errors are ignored.
- Some migrations ignore ALTER errors broadly, which can mask schema drift.

### Packaging And Scripts

There is meaningful packaging and patch tooling. Remaining risks are:

- Release configs still contain default agent tokens.
- Some E2E scripts use old runtime APIs and manipulate credentials/Docker state.
- Observability start/reset scripts need clearer product support boundaries.

### Testing

Fresh verification in this review:

- `go test ./...`: PASS.
- `go vet ./...`: PASS.
- shell syntax: PASS.
- `npm run build`: PASS with warnings.
- `npm test`: FAIL, missing script.
- `vitest`: not installed.

The most important missing tests are cross-boundary tests:

- RunPlan -> AgentRunSpec -> Docker create options.
- Tenant direct-ID access.
- Node transfer with GPUs.
- Audit log tenant scoping.
- Task claim races and duplicate/stale results.
- Fresh DB and upgrade DB migrations.
- Release package install/patch apply/rollback.

## Unverified Items

- MetaX real hardware discovery/metrics/runtime.
- Full model runtime E2E with Docker and real model endpoint.
- Prometheus/Grafana bundled mode in release-like deployment.
- Legacy database upgrade.
- Patch package apply and rollback on a release directory.
- Web interaction screenshots/usability smoke.

## Next Phase Recommendation

Continue development, but run a P0/P1 hardening phase before any customer pilot. The project is not ready for pilot delivery until Critical and High issues in `issue-register.md` are addressed or explicitly accepted with formal blockers.

## Closure Status

Unresolved problems remain. All problems discovered in this review are documented in `docs/reports/full-project-review/issue-register.md` with statuses `Open` or `Not Verified`. No problems are intentionally left only in chat.
