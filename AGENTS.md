# LightAI Go AI Development Instructions

This file is the root entrypoint for Codex and other coding agents working on LightAI Go.

## 1. Must read before changing code

Read these first:

1. `docs/README.md`
2. `docs/PHASE-STATUS.md`
3. `docs/RELEASE_NOTE_v0.1.9.md`

Then read the topic documents needed for the task:

- Server / Agent / node / heartbeat / metrics target: `docs/02-server-agent-design.md`
- GPU discovery, GPU metrics, Collector scripts: `docs/03-resource-monitoring-design.md` and `docs/GPU_COLLECTOR_ARCHITECTURE.md`
- Prometheus / Grafana / observability: `docs/04-observability-design.md`
- Runtime environment and Docker command generation: `docs/05-runtime-environment-design.md`
- Model definition: `docs/06-model-design.md`
- Instance lifecycle, tasks, Docker start/stop: `docs/07-instance-lifecycle-design.md`
- Cross-module contracts: `docs/08-engineering-contracts.md`
- Auth, tenant, RBAC, node transfer: `docs/09-auth-tenant-design.md`
- Development plan: `docs/10-mvp-development-plan.md`
- RC1 review / deferred risks: `docs/RC1_REVIEW_FIX_PLAN.md` and `docs/RC1_CODEX_REVIEW_TRACKING.md`
- Local verification: `docs/RUNBOOK-LOCAL-VERIFY.md`

If the task mentions GPUStack, read only the project audit documents:

- `docs/REVIEW-GPUSTACK-AUDIT.md`
- `docs/REVIEW-GPUSTACK-UI.md`

Do not independently copy or translate GPUStack code.

## 2. Document precedence

Use this order when documents appear to conflict:

1. The user's latest explicit instruction.
2. This `AGENTS.md`.
3. `docs/08-engineering-contracts.md` for field semantics, units, identity boundaries, DockerRunSpec, task lease/generation, and cross-module rules.
4. `docs/PHASE-STATUS.md` and the latest `docs/RELEASE_NOTE_*.md` for current implementation status.
5. Topic design documents under `docs/00-*.md` through `docs/10-*.md`.
6. Review and audit notes.

Important: `docs/README.md` is still the reading-order entrypoint, but some phase-window wording may be older than the current RC state. Current state is defined by `docs/PHASE-STATUS.md` plus the latest release note.

## 3. Current project state

LightAI Go is a lightweight GPU/node management platform inspired by GPUStack, built for small and medium deployments with a small number of GPU servers.

Current implementation status:

- Server / Agent skeleton: done.
- Auth, tenant, RBAC: done.
- Agent registration and heartbeat: done.
- System / registry / mock collectors: done.
- NVIDIA Collector: done.
- Stable node identity hardening: done.
- MetaX Collector: scripts ready and mock verified; real hardware validation is still required.
- Web Console MVP: done.
- Observability pages and server metrics: done.
- Credentials, password reset, file logging, and patch tooling: done.
- Tenant model fix in v0.1.9: `tenant_id` uses UUID, default tenant has `slug='default'`, and `PATCH /api/nodes/{id}/tenant` exists for node transfer.

Known current limitations:

- Prometheus/Grafana binaries may not be present in the dev repository.
- Server-managed Prometheus/Grafana supervision is not fully implemented in Go.
- MetaX real hardware validation is still required.
- Runtime Environment, Model Registry, and Instance Lifecycle may be partially documented ahead of implementation.
- TLS/HTTPS is not implemented.

## 4. Core architecture

LightAI Go uses two independent Go binaries:

- Server: control plane, API, Web, SQLite, auth/RBAC, node/GPU/resource state.
- Agent: execution plane, node registration, heartbeat, OS/GPU collection, metrics, future Docker operations.

Default ports:

- Server API + embedded Web: `18080`
- Agent metrics: `19091`
- Prometheus: `19090`
- Grafana: `13000`
- Vite dev server: `15173`

Current Web stack:

- Vue 3
- Element Plus
- vue-i18n
- zh-CN default, en-US supported
- embedded with `-tags web`

## 5. Non-negotiable engineering rules

1. Do not make broad refactors unless the user explicitly asks.
2. Prefer small, verifiable changes.
3. Do not fix Web data problems by masking symptoms in the frontend.
4. For any Web display issue, inspect the actual API response first.
5. For any API data issue, inspect Server DTO/model/upsert/persistence.
6. For any Server state issue, inspect Agent registration, heartbeat, resource report, and collector payload.
7. Keep Agent token, User Session, and future API Key strictly separated.
8. Server state comes from Agent reports and SQLite, not from Prometheus queries.
9. `/metrics` must not trigger GPU vendor tools; it reads the latest snapshot only.
10. Do not log passwords, password hashes, session IDs, CSRF tokens, agent tokens, or future API keys.
11. Do not introduce API Key, token accounting, billing, Kubernetes, Ray, multi-cluster, gateway, SSO, or resource-level ACL unless explicitly requested.
12. Mock GPU collectors are only for development/test profile and must not replace real NVIDIA or MetaX validation.
13. All capacity values in API/DB/Go structs are bytes. Percent fields are `0-100`. Missing values must be nullable/unknown, not fake zero.
14. Preserve backward compatibility unless a release note explicitly declares a breaking change.

## 6. Output required after each task

Every completion report must include:

- Summary of the root cause.
- What was changed.
- Modified files.
- Verification commands and results.
- Any skipped tests and why.
- Remaining risks or follow-up items.
- For API/UI data fixes, include a sample API response or curl command used for verification.

Do not report success if tests or verification failed.
