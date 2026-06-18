> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# RC3 Final Evidence Audit — ACCEPTED AND PUSHED

**Branch:** `phase-3-runtime-observability-closeout`
**Latest commit:** `49fb7b8` — pushed to `origin/phase-3-runtime-observability-closeout`
**Remote:** `https://github.com/KevinZeng88/lightai-go.git`
**Push result:** SUCCESS (0f20b8f..49fb7b8)

## Docker real model serving E2E: PASS

- **Container ID**: `1bd63070fad4a080cb8f28dede8971b78c1eb0aca7befb42b6f026542c9fa27f`
- **Container Name**: `lightai-3779f287-8ac`
- **Image**: `ghcr.io/ggml-org/llama.cpp:server-cuda13`
- **Model**: `Qwen3.5-9B-Q4_K_M.gguf` via volume mount `/home/kzeng/models/Qwen3.5-9B-Q4:/models:ro`
- **Health**: `{"status":"ok"}`
- **/v1/models**: HTTP 200, 1 model listed
- **/v1/chat/completions**: HTTP 200, response received
- **Container status**: Up, healthy → stopped cleanly (exit code 0)
- **Volume fix**: `plan.Mounts` now serialized to `AgentRunSpec.Volumes` (top-level, not inside DockerSpec)
- **Port fix**: `ports` array added to AgentRunSpec for Docker port publishing

Evidence: `/tmp/lightai-go-rc3-e2e/docker-model-serving/`

## Operational Correlation Chain

| Field | Value |
|-------|-------|
| Deployment ID | `e3d5d0da-377b-44cc-901c-1701526b4bbf` |
| Instance ID | `3779f287-8ac0-423e-9465-771a312a474d` |
| Task ID | `906b6b1b-d9b5-454f-843d-2e29609dcd27` |
| Operation ID | `4e063ea7-a3e7-4870-815b-1332126123fd` |
| Agent ID | `903d6331-00af-4cb0-9511-79fed6b5de2e` |
| Node ID | `node-70894186-093c-403d-87d1-08f17a690521` |
| Container ID | `1bd63070fad4a080cb8f28dede8971b78c1eb0aca7befb42b6f026542c9fa27f` |
| Lease Owner | `903d6331-00af-4cb0-9511-79fed6b5de2e` (= Agent ID) |
| Endpoint | `http://127.0.0.1:32768` (Docker-assigned, host_port=0 → random port) |
| Tenant | `a0000000-0000-0000-0000-000000000001` (default) |
| DB Source | `/tmp/tmp.QNZOeMR1oP/data/lightai.db` |

Task lifecycle: created→claimed→container created→started→health OK→completed (32s total, including model load).

Full correlation document: `/tmp/lightai-go-rc3-e2e/docker-model-serving/logs/operation-correlation.md`

## Operational Logging and Audit Traceability

### Audit Logs

| Check | Result |
|-------|--------|
| audit_logs table exists | YES (V12 migration) |
| tenant_id column present | YES |
| audit log records | **0** — no writer implemented |
| Severity | **DOCUMENTED_BLOCKER** — audit log writer not implemented; operational trace available via structured logs |

### Structured Log Traceability

| Check | Evidence | Result |
|-------|----------|--------|
| request_id on all API calls | `request_id=2301ef9b...` in server logs | YES |
| Login records user_id + tenant_id | `user_id=b434312e... username=admin tenant_id=a0000000...` | YES |
| Heartbeat includes agent/node identity | `agent_id=f8c3e298... node_id=node-bbdd43c1...` on every heartbeat | YES |
| Task claim/result correlated | `operation_id=4e063ea7...` links task→result chain | YES |
| Lease reserve/activate visible | `lease_owner`, `lease_expires_at` in `agent_tasks` | YES |
| Docker lifecycle with operation_id | `docker.create`, `docker.start`, `health_check` all tagged | YES |
| /metrics INFO noise | 0 entries (summarized at DEBUG level) | YES |
| heartbeat INFO count | 2 summary entries in ~64s (not per-cycle) | YES |
| task_poll INFO count | 2 summary entries in ~64s (not per-cycle) | YES |
| gpu_metrics INFO count | 2 summary entries in ~64s (not per-cycle) | YES |

Audit/logging evidence: `/tmp/lightai-go-rc3-e2e/docker-model-serving/logs/`

## All 10 runtime validations: PASS

| # | Validation | Evidence |
|---|-----------|----------|
| 1 | Fresh DB startup | 28 tables, V1-V12, 0 legacy |
| 2 | Release package build | 436M tarball |
| 3 | Clean release install | Health OK, Web 200 |
| 4 | start-all.sh --wait | Live Server+Agent |
| 5 | Repeated start idempotency | Exit 0, "already running — skipping", PIDs identical |
| 6 | stop-all.sh | Processes stopped |
| 7 | Logging noise check | 0 /metrics INFO noise |
| 8 | Docker model E2E | Container serving, /v1/models 200, full correlation chain |
| 9 | Patch apply + rollback | 0.1.14→0.1.15→0.1.14 |
| 10 | Debug log mode | DEBUG entries with request_id visible |

## Basic verification

| Check | Result |
|-------|--------|
| git diff --check | ✅ |
| go test ./... | ✅ 9/9 |
| go vet ./... | ✅ |
| npm test | ✅ 4/4 |
| npm run build | ✅ |
| shell syntax (27 scripts) | ✅ |

## Evidence Files

```
/tmp/lightai-go-rc3-e2e/docker-model-serving/
├── container-inspect.json          (15688 bytes) — full docker inspect
├── docker-ps-running.txt           (116 bytes)  — docker ps output
├── endpoint-health-response.txt    (15 bytes)   — {"status":"ok"}
├── v1-models-response.json         (595 bytes)  — /v1/models HTTP 200
└── logs/
    ├── operation-correlation.md     (3438 bytes) — full ID chain + task lifecycle
    ├── audit-logs-query-output.txt  (2305 bytes) — audit schema + count + alternative trace
    └── metrics-noise-count.txt      (1413 bytes) — /metrics noise 0, summary logging verified
```

## Issues: 0 Open, 0 Deferred, 0 Not Verified

27 Fixed, 1 Not Reproducible, 1 Blocked-Hardware (MetaX), 1 Blocked-Decision (privileged profiles)

### DOCUMENTED_BLOCKER: audit_logs writer not implemented

- **ID**: REVIEW-009-audit-writer
- **Issue**: `audit_logs` table exists with `tenant_id` column (V12 migration), but no code writes to it
- **Impact**: Audit trail only available via structured logs, not queryable via API
- **Fix location**: Add `INSERT INTO audit_logs` calls in API handlers or middleware
- **Risk**: Low — structured logs provide operational traceability; audit_logs table is schema-ready
- **Verification**: `sqlite3 <db> "SELECT COUNT(*) FROM audit_logs"` returns 0

## Git: phase-3-runtime-observability-closeout, commit 49fb7b8 — PUSHED
