# 06 — Workstream D: OpenAI-Compatible Entry, Audit, and Metering

## Goal

Add a minimal, tenant-scoped, audited OpenAI-compatible gateway.

This is not a full billing engine. It is the product boundary required before later billing.

## Step D1 — Inspect current serving and audit code

Run:

```bash
rg -n "openai|chat/completions|completions|embeddings|models|api[_-]?key|Bearer|usage|token|meter|audit|endpoint_url|HandleModelInstanceTest|proxy" internal web docs configs
```

Inspect:

```text
internal/server/api/router.go
internal/server/api/*audit*
internal/server/api/*instance*
internal/server/api/*deployment*
internal/server/db/db.go
internal/server/auth/*
internal/common/types/*
docs/api/openapi.yaml
```

## Step D2 — Define product contract

Add external gateway routes outside `/api/v1`:

```text
GET  /v1/models
POST /v1/chat/completions
```

Optional if straightforward:

```text
POST /v1/completions
POST /v1/embeddings
```

Authentication:

```text
Authorization: Bearer <lightai_api_key>
```

Routing policy for first implementation:

- API key belongs to tenant.
- API key can be scoped to:
  - all active deployments in tenant; or
  - explicit deployment IDs.
- `model` in request maps to:
  1. deployment served model name;
  2. deployment name;
  3. model artifact name/display name if unambiguous.
- If ambiguous, return OpenAI-compatible error object.

## Step D3 — DB schema

Modify clean schema in:

```text
internal/server/db/db.go
```

Add tables:

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
```

Add indexes:

```sql
CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_gateway_usage_tenant_created ON gateway_usage_records(tenant_id, created_at);
CREATE INDEX idx_gateway_usage_deployment_created ON gateway_usage_records(deployment_id, created_at);
CREATE INDEX idx_gateway_usage_key_created ON gateway_usage_records(api_key_id, created_at);
```

Clean DB policy:

- no legacy migration needed unless current project policy changes;
- document DB rebuild.

## Step D4 — API key management

Add `/api/v1` management routes:

```text
GET    /api/v1/api-keys
POST   /api/v1/api-keys
POST   /api/v1/api-keys/{id}/disable
DELETE /api/v1/api-keys/{id}
```

Optional:

```text
POST /api/v1/api-keys/{id}/rotate
```

Rules:

- full key shown only once at creation;
- store hash, not plaintext;
- expose prefix only;
- audit create/disable/delete/rotate;
- redact in logs.

Permissions:

- tenant admin or specific permission such as `api_key:read`, `api_key:write`.

## Step D5 — Gateway middleware

Add middleware:

```text
internal/server/api/gateway_auth.go
```

Responsibilities:

- parse Bearer key;
- hash and lookup active key;
- verify tenant;
- verify expiry/status;
- attach tenant/api_key/scopes to request context;
- update last_used_at;
- never log full key.

## Step D6 — Model routing

Add resolver:

```text
internal/server/gateway/model_resolver.go
```

Inputs:

- tenant_id;
- requested model string;
- API key scopes;
- active deployments;
- running healthy instances;
- deployment service_json/config_overrides.

Outputs:

```go
type GatewayTarget struct {
    DeploymentID string
    InstanceID string
    ModelArtifactID string
    RequestedModel string
    ResolvedModel string
    BackendURL string
    Route string
}
```

Rules:

- only tenant-owned deployments;
- only running/healthy instance unless explicit degraded policy;
- if multiple candidates match, return ambiguous error;
- if no running instance, return unavailable error;
- do not start deployments implicitly.

## Step D7 — Proxy implementation

Add handler file:

```text
internal/server/api/gateway_handlers.go
```

Implement:

```text
GET /v1/models
POST /v1/chat/completions
```

Behavior:

- preserve OpenAI-compatible request/response body as much as possible;
- set timeout;
- forward to backend instance endpoint;
- capture HTTP status;
- parse response usage if present:
  - `usage.prompt_tokens`
  - `usage.completion_tokens`
  - `usage.total_tokens`
- if usage missing, store null token counts and `usage_source=missing`;
- create audit log and usage record for every request;
- return backend error body if safe; otherwise wrap in OpenAI-style error.

## Step D8 — Usage query API

Add:

```text
GET /api/v1/gateway/usage
```

Filters:

- time range;
- deployment_id;
- model_artifact_id;
- api_key_id;
- success;
- route.

Return:

```json
{
  "items": [],
  "summary": {
    "requests": 0,
    "success": 0,
    "errors": 0,
    "total_tokens_known": 0,
    "total_tokens_unknown_count": 0,
    "latency_ms_avg": 0
  }
}
```

## Step D9 — UI

Add minimal pages or sections:

```text
web/src/pages/ApiKeysPage.vue
web/src/pages/GatewayUsagePage.vue
```

Router/menu:

```text
/system/api-keys
/observability/gateway-usage
```

UI requirements:

- create key;
- copy key once;
- list key prefix/status/last used;
- disable key;
- usage table;
- usage summary;
- no full secrets shown after creation.

## Step D10 — Tests

Go tests:

```bash
go test ./internal/server/api/... -run 'Gateway|APIKey|Usage|Audit'
```

Required cases:

- missing bearer rejected;
- invalid key rejected;
- disabled key rejected;
- expired key rejected;
- cross-tenant deployment not accessible;
- ambiguous model rejected;
- no running instance returns unavailable;
- successful chat request proxied;
- backend usage parsed and recorded;
- missing usage recorded as unknown;
- audit record written on success and failure;
- full key never returned after creation.

Frontend tests:

```text
web/tests/apiKeys.test.mjs
web/tests/gatewayUsage.test.mjs
```

## Acceptance

- `GET /v1/models` works with valid API key.
- `POST /v1/chat/completions` proxies to a running deployment.
- Usage records are created.
- Audit logs are created.
- Token counts are stored when backend returns usage.
- Unknown usage is explicit when backend omits usage.
- Secrets are redacted.
- OpenAPI docs updated.
- Tests pass.
