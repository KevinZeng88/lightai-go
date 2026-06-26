# Future OpenAI Gateway — Design Notes

Created: 2026-06-26 | Status: **DEFERRED — document only, no code implementation in current hardening scope**

## Purpose

This document captures the design boundaries, dependency conditions, and implementation suggestions for a future tenant-scoped OpenAI-compatible API gateway with API key management, usage metering, and billing integration. It serves as a reference when the gateway workstream is activated.

**Current scope (2026-06-26 hardening) does NOT include any code, DB schema, or UI for the gateway.** This document exists so the design is not lost and prerequisites are clear.

---

## 1. Design Boundaries

### 1.1 What the gateway IS

- A **tenant-scoped entrypoint** that exposes OpenAI-compatible endpoints (`/v1/models`, `/v1/chat/completions`) to external API consumers.
- A **routing layer** that maps incoming `model` requests to running LightAI deployments and healthy instances.
- An **auth layer** that validates Bearer API keys, resolves tenant + scopes, and rejects unauthorized requests.
- A **proxy layer** that forwards requests to backend inference containers and returns responses in OpenAI-compatible format.
- A **recording layer** that captures usage (token counts, latency, success/failure) and writes audit logs.

### 1.2 What the gateway IS NOT

- A billing engine (usage records are inputs TO billing, not billing itself).
- A load balancer across nodes (first implementation: single-node deployment routing).
- A model registry or model hub (model list comes from LightAI deployments, not external catalog).
- A replacement for backend-native OpenAI endpoints (backends still expose their own `/v1/*` — gateway is the managed entrypoint).

---

## 2. Dependency Conditions

The gateway cannot be built until these prerequisites are stable:

| Prerequisite | Current status (2026-06-26) | Notes |
|---|---|---|
| Deployment lifecycle (create → preview → start → stop → delete) | Implemented, Workstreams B–D hardening in progress | Gateway needs `HandleStartDeployment` and instance lifecycle to be stable |
| Running instances with `endpoint_url` populated | Implemented | `model_instances.endpoint_url` is set on start |
| Model instance health check / state tracking | Implemented | `actual_state` field has `running`/`stopped`/`error` |
| `served_model_name` in deployment config | Implemented | Set via `config_overrides.parameter_values[backend.common.served_model_name]` |
| Tenant isolation in deployment queries | Implemented | `HandleListDeployments` scopes by tenant |
| Audit logging infrastructure | Implemented | `audit_logs` table, `WriteAudit()` function, `HandleListAuditLogs` handler |
| Runtime smoke across vLLM/SGLang/llama.cpp | Last evidence 2026-06-25 all PASS | Must be re-verified after B–D hardening |

### Hard blockers
- Gateway must NOT start deployments implicitly — only proxy to already-running instances.
- Gateway must NOT expose cross-tenant deployments — tenant scoping is mandatory.
- Gateway must NOT log full API keys — prefix-only in logs and audit records.

---

## 3. Architecture Sketch

### 3.1 Routes (outside `/api/v1`)

```
GET  /v1/models              → HandleGatewayListModels
POST /v1/chat/completions    → HandleGatewayChatCompletions
```

Optional (if backend capability model supports them):
```
POST /v1/completions         → HandleGatewayCompletions
POST /v1/embeddings          → HandleGatewayEmbeddings
```

### 3.2 Authentication

```
Authorization: Bearer lak-<random>
```

- `lak-` prefix identifies LightAI API keys (distinct from agent tokens).
- Key format: `lak-` + 32 bytes of random hex/base64.
- Full key shown exactly once at creation.
- Store bcrypt hash in `api_keys.key_hash`. Never log or return full key after creation.
- `key_prefix` = first 8–12 chars of the key (e.g., `lak-a1b2c3d4`), safe to display.

### 3.3 Model Routing

Resolution order for `model` field in chat completions request:

1. **Exact match on `served_model_name`** in deployment `config_overrides` or `service_json.served_model_name`
2. **Exact match on `deployment.name`** or `deployment.display_name`
3. **Exact match on `model_artifact.name`** or `model_artifact.display_name` (if only one deployment uses that artifact)

Rules:
- Only tenant-owned deployments.
- Only deployments with at least one `actual_state = 'running'` instance.
- Only healthy instances (health check passing or health check not configured).
- If exactly one match → proxy to that instance.
- If multiple matches at same resolution tier → return 400 with `{"error": {"message": "Ambiguous model name 'X' matches deployments: [list]", "type": "ambiguous_model"}}`.
- If no match → return 404 with `{"error": {"message": "No deployment found for model 'X'", "type": "model_not_found"}}`.
- If match found but no running instance → return 503 with `{"error": {"message": "Model 'X' has no running instances", "type": "service_unavailable"}}`.

### 3.4 Proxy Behavior

- Preserve OpenAI-compatible request/response body shape.
- Set proxy timeout: 120s for chat completions, 30s for models list (configurable).
- Forward to `instance.endpoint_url + route` (e.g., `http://10.0.0.5:8000/v1/chat/completions`).
- Capture HTTP status code.
- On success: parse `usage.prompt_tokens`, `usage.completion_tokens`, `usage.total_tokens` from response body. If absent, store `usage_source = 'missing'` with NULL token counts.
- On error: return OpenAI-compatible error shape if possible; otherwise wrap backend error.
- Record `gateway_usage_records` row for every request (success and failure).
- Write `audit_logs` row for every request.
- Update `api_keys.last_used_at` on each successful auth.

---

## 4. DB Tables (Future — NOT implemented now)

### `api_keys`

```sql
CREATE TABLE api_keys (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  name TEXT NOT NULL,
  key_prefix TEXT NOT NULL,
  key_hash TEXT NOT NULL,
  scopes_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'active',
  last_used_at TEXT,
  expires_at TEXT,
  created_by TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(tenant_id, name)
);
CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
```

`scopes_json` structure (first implementation):
```json
{
  "deployment_ids": ["*"]  // "*" = all tenant deployments, or ["id1", "id2"] for scoped
}
```

### `gateway_usage_records`

```sql
CREATE TABLE gateway_usage_records (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  api_key_id TEXT NOT NULL DEFAULT '',
  deployment_id TEXT NOT NULL DEFAULT '',
  instance_id TEXT NOT NULL DEFAULT '',
  model_artifact_id TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  operation_id TEXT NOT NULL DEFAULT '',
  route TEXT NOT NULL DEFAULT '',
  requested_model TEXT NOT NULL DEFAULT '',
  resolved_model TEXT NOT NULL DEFAULT '',
  backend_url TEXT NOT NULL DEFAULT '',
  http_status INTEGER NOT NULL DEFAULT 0,
  success INTEGER NOT NULL DEFAULT 0,
  latency_ms INTEGER NOT NULL DEFAULT 0,
  prompt_tokens INTEGER,
  completion_tokens INTEGER,
  total_tokens INTEGER,
  usage_source TEXT NOT NULL DEFAULT 'unknown',
  error_code TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_gateway_usage_tenant_created ON gateway_usage_records(tenant_id, created_at);
CREATE INDEX idx_gateway_usage_deployment_created ON gateway_usage_records(deployment_id, created_at);
CREATE INDEX idx_gateway_usage_key_created ON gateway_usage_records(api_key_id, created_at);
```

`usage_source` values: `'backend_response'` (tokens from JSON), `'missing'` (backend returned no usage), `'parse_error'` (backend returned malformed usage).

---

## 5. API Routes (Future — NOT implemented now)

### Management routes (under `/api/v1`, session auth + permissions)

| Method | Path | Permission | Purpose |
|---|---|---|---|
| POST | `/api/v1/api-keys` | `api_key:write` | Create key (returns full key once) |
| GET | `/api/v1/api-keys` | `api_key:read` | List keys (prefix only, no full key/hash) |
| POST | `/api/v1/api-keys/{id}/disable` | `api_key:write` | Disable key |
| DELETE | `/api/v1/api-keys/{id}` | `api_key:write` | Delete key |
| GET | `/api/v1/gateway/usage` | `gateway_usage:read` | Query usage records with filters + summary |

### External routes (outside `/api/v1`, Bearer token auth)

| Method | Path | Auth | Purpose |
|---|---|---|---|
| GET | `/v1/models` | Bearer API key | List available models for tenant |
| POST | `/v1/chat/completions` | Bearer API key | Chat completion proxy |

### New permissions to seed

```
api_key:read, api_key:write, gateway_usage:read
```

---

## 6. UI Pages (Future — NOT implemented now)

### `ApiKeysPage.vue`
- Route: `/system/api-keys`
- Create dialog: name field → submit → shows full key once with copy button + warning "Store this key now — it will not be shown again"
- Table: name, key_prefix, status (active/disabled/expired), last_used_at, created_at, actions (disable, delete)
- No full key or hash exposed after creation

### `GatewayUsagePage.vue`
- Route: `/observability/gateway-usage`
- Filter bar: time range, deployment_id, model_artifact_id, api_key_id, success (yes/no), route
- Table: created_at, deployment, model, api_key prefix, route, http_status, latency_ms, prompt_tokens, completion_tokens, total_tokens
- Summary bar: total requests, success count, error count, known tokens, unknown count, avg latency

---

## 7. Security Considerations

1. **Key hashing:** Use bcrypt (same as existing password hashing in `internal/server/auth/`).
2. **Key format:** `lak-` + crypto/rand 32 bytes hex-encoded → 64 hex chars + prefix.
3. **Redaction:** Full key must never appear in logs, audit records, API responses (except the single creation response), or error messages.
4. **Audit:** Every gateway request writes an audit log entry. Every key CRUD operation writes an audit log entry.
5. **Rate limiting:** Not in first implementation — add before production use.
6. **Expiry:** `expires_at` column exists; gateway middleware checks expiry. No automatic cleanup of expired keys in first implementation.

---

## 8. Billing Integration Points (Future — beyond gateway scope)

The `gateway_usage_records` table is designed to feed a billing system:

| Billing need | Source in usage records |
|---|---|
| Token count per tenant | `SUM(total_tokens) WHERE tenant_id = ? AND created_at BETWEEN ? AND ?` |
| Token count per deployment/model | `SUM(total_tokens) WHERE deployment_id = ?` / `model_artifact_id = ?` |
| Request count | `COUNT(*)` with same filters |
| Error rate | `COUNT(*) WHERE success = 0` / `COUNT(*)` |
| Latency SLA | `AVG(latency_ms)`, `PERCENTILE(latency_ms, 0.95)` |

Billing is a separate workstream that consumes usage records. The gateway does not calculate cost — it records facts.

---

## 9. Files to Create (When Activated)

### Backend (Go)

| File | Purpose |
|---|---|
| `internal/server/api/gateway_auth.go` | `GatewayAuthMiddleware` — Bearer token parse + hash + lookup + tenant/scope attach |
| `internal/server/gateway/model_resolver.go` | `ResolveGatewayTarget()` — model → deployment → instance routing |
| `internal/server/api/gateway_handlers.go` | `HandleGatewayListModels`, `HandleGatewayChatCompletions` |
| `internal/server/api/api_key_handlers.go` | CRUD handlers for API keys |
| `internal/server/api/gateway_usage_handlers.go` | `HandleListGatewayUsage` |
| `internal/server/models/api_key.go` | `APIKey`, `GatewayUsageRecord` structs |
| `internal/common/types/gateway.go` | `GatewayTarget` struct |

### Backend (Modified)

| File | Change |
|---|---|
| `internal/server/db/db.go` | Add `api_keys` + `gateway_usage_records` tables + indexes |
| `internal/server/api/router.go` | Register 8 new routes |
| `internal/server/auth/bootstrap.go` | Seed `api_key:read`, `api_key:write`, `gateway_usage:read` |
| `docs/api/openapi.yaml` | Add gateway + API key + usage schemas and paths |

### Frontend (Vue/TS)

| File | Purpose |
|---|---|
| `web/src/pages/ApiKeysPage.vue` | API key management UI |
| `web/src/pages/GatewayUsagePage.vue` | Usage records browsing UI |
| `web/src/api/apiKeys.ts` | API client for API key CRUD |
| `web/src/api/gatewayUsage.ts` | API client for usage queries |
| `web/src/router/index.ts` | Add routes `/system/api-keys`, `/observability/gateway-usage` |
| `web/src/layouts/ConsoleLayout.vue` | Add menu items under System / Observability groups |
| `web/src/locales/zh-CN.ts` | Add `apiKeys.*`, `gatewayUsage.*` i18n groups |
| `web/src/locales/en-US.ts` | Add mirror i18n groups |

### Tests

| File | Purpose |
|---|---|
| `internal/server/api/gateway_test.go` | 15 test cases (auth rejection, proxy, usage parsing, audit) |
| `internal/server/api/api_key_test.go` | 5 test cases (CRUD, duplicate, disable, delete) |
| `internal/server/gateway/model_resolver_test.go` | 6 test cases (resolution order, ambiguity, scoping) |
| `web/tests/apiKeys.test.mjs` | API key UI tests |
| `web/tests/gatewayUsage.test.mjs` | Usage UI tests |

---

## 10. Activation Checklist

Before activating this workstream, verify:

- [ ] All Workstreams B–E from `01-file-level-implementation-plan.md` are complete and stable.
- [ ] Deployment create → preview → start → stop → delete flow passes E2E.
- [ ] vLLM, SGLang, and llama.cpp runtime smoke passes (or blocked backends documented).
- [ ] `model_instances.endpoint_url` is reliably populated for running instances.
- [ ] Audit logging infrastructure is verified (table, writer, handler, UI page).
- [ ] Go test suite + frontend test suite + build all pass.
- [ ] `git status --short` is clean.
