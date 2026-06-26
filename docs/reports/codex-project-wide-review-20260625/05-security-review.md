# Security Review

## Strengths

- Session cookies are HttpOnly and SameSite=Lax.
- CSRF token plus Origin/Referer validation is applied to state-changing session routes.
- Password reset and must-change-password enforcement exist.
- Agent token is separated from user session auth.
- Tenant scope checks are present for core node/GPU/model/deployment/NBR APIs.
- Logging has redaction helpers for sensitive env/key names.

## High-risk findings

| Finding | Evidence | Impact | Recommendation |
| --- | --- | --- | --- |
| Client-trusted NBR readiness endpoint. | `POST /api/v1/nodes/{id}/backend-runtimes/check` uses session auth and trusts `image_present/docker_available` when `checkOnly=true`. | False-ready NBR can produce wrong deployment attempts. | Remove this endpoint from UI/session route or make it Agent-only/server-probed. |
| Shared Agent bearer token. | `auth.AgentAuthMiddleware(cfg.AgentToken)` compares one global token; agent payload carries `node_id/agent_id`. | Token leak allows access to all agent endpoints and task result submission attempts. | Move to per-agent credentials or signed node-bound tokens. |
| Agent node-side admin endpoints expose Docker/image/file information behind only shared token. | `cmd/agent/main.go` `/docker-images`, `/docker-image-inspect`, `/files`, `/model-paths/scan`. | Token leak leaks image inventory and authorized filesystem roots. | Per-node token, mTLS/TLS, audit, and stricter network binding. |
| Docker dangerous options are user-configurable. | `RuntimeParameterEditor` exposes privileged/devices/security options; `docker_real.go` applies them directly. | Misconfigured runtime can mount host devices/paths, use privileged, or weaken isolation. | Add server-side policy gate and tenant/platform-admin separation for dangerous options. |

## Medium-risk findings

- TLS/HTTPS is not implemented; docs list reverse proxy TLS but server defaults are HTTP.
- API error bodies are mostly generic, which is good, but Docker inspect errors may propagate agent-side command errors in check responses.
- `GET /healthz`, `/metrics`, and `/metrics/targets` are unauthenticated. This may be acceptable for local deployment but should be explicit.
- File browser safety is well designed with roots/denied paths, but runtime root addition depends on server-provided `extra_roots`; keep Agent final validation as non-negotiable.

## Tenant/RBAC assessment

Tenant enforcement exists in many handlers, but it is not uniformly generated or centralized. The project should add a negative API test matrix for every route family:

- regular tenant cannot read/write other tenant node/NBR/model/deployment/GPU;
- platform admin can see all;
- tenant transfer updates related GPUs/NBR/model roots consistently or explicitly blocks unsafe transfer.
