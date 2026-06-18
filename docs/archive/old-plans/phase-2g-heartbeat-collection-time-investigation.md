> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# Phase 2G: Heartbeat / Collection Time Investigation

> Status: In progress
> Created: 2026-06-16

## 1. Observed Issue

Web pages show "采集时间" (collected at) and "最后心跳" (last heartbeat) with timestamps
frequently showing 50+ seconds in the past. Users may interpret this as Agent collection
delay, heartbeat anomaly, or page refresh anomaly.

## 2. Complete Data Flow (Source-Level Trace)

### 2.1 Agent Side

| Component | Interval | Config Key | Default |
|-----------|----------|-----------|---------|
| Heartbeat | 2s | `heartbeat.interval` | `2s` |
| System collect | 5s | `collectors.system.interval` | `5s` |
| GPU collect (external timeout) | 5s per script | `collectors.gpu_external.timeout` | `5s` |
| Resource report | follows system collect | `collectors.report_interval` | `5s` (not independently implemented) |
| Request timeout | 5s | `request_timeout` | `5s` |

Key files:
- `cmd/agent/main.go` — main loop with independent heartbeat ticker and collect ticker
- `internal/agent/register/register.go` — `SendHeartbeat` sends only `{node_id, agent_id}`
- `internal/agent/collector/collector.go` — `ResourceReport` struct, `NormalizeGPUs`
- `internal/agent/collector/registry.go` — `Registry.Collect` orchestrates system + GPU
- `internal/agent/collector/system.go` — `SystemCollectorImpl.Collect` sets `CollectedAt: time.Now()`
- `internal/agent/collector/external.go` — `ExternalCommandCollector` runs discover.sh + metrics.sh
- `internal/agent/collector/protocol.go` — `ParseProtocolOutput` sets `CollectedAt` from parameter

**Heartbeat payload (POST /api/v1/agent/heartbeat):**
```json
{"node_id": "node-xxx", "agent_id": "agent-xxx"}
```
No time fields in the heartbeat request.

**Resource report payload (POST /api/v1/agent/resources/report):**
```json
{
  "agent_id": "agent-xxx",
  "collected_at": "2026-06-16T12:00:00+08:00",
  "system": { "...": "...", "collected_at": "..." },
  "gpu_resources": [
    { "...": "...", "collected_at": "2026-06-16T12:00:00+08:00" }
  ]
}
```
`collected_at` at top level and per GPU resource is set by the Agent at collection start time.

### 2.2 Server Side

**Heartbeat handler** (`HandleHeartbeat` in `internal/server/api/agent_handlers.go:180`):
```sql
UPDATE nodes SET last_heartbeat_at = ?, status = 'online', updated_at = ?
WHERE id = ?
```
- `last_heartbeat_at` = Server's `time.Now().Format(time.RFC3339)` — **Server time, not Agent time**
- `status` = `'online'`
- `updated_at` = Server's current time

**Resource report handler** (`HandleResourceReport` in `internal/server/api/resource_handlers.go:181`):

For GPU devices (INSERT and UPDATE):
```go
collectedAt := g.CollectedAt       // Agent's time string from payload
if collectedAt == "" {
    collectedAt = now              // Fallback to Server's time
}
```
- `gpu_devices.collected_at` = **Agent's collected_at time** (from the payload), or Server time if empty
- `gpu_devices.updated_at` = Server's current time

For nodes (when system data present):
```sql
UPDATE nodes SET updated_at = ? WHERE id = ?
```
- Does **NOT** update `nodes.last_heartbeat_at`
- Does **NOT** update `nodes.status`

For system snapshots:
- `node_system_snapshots.collected_at` = Server's `now` (not Agent's time)

**Node offline check** (`MarkOfflineNodes` in `agent_handlers.go:531`):
```sql
UPDATE nodes SET status = 'offline', updated_at = datetime('now')
WHERE status = 'online' AND last_heartbeat_at < ?
```
- Runs every 10s in a background goroutine
- Default threshold: 20s (configurable via `node_offline_threshold`)
- Only checks `last_heartbeat_at` — resource reports do not prevent offline marking

### 2.3 Database Schema

| Table | Time Field | Semantics |
|-------|-----------|-----------|
| `nodes` | `last_heartbeat_at` | Server time when last heartbeat received |
| `nodes` | `created_at` | Row creation time |
| `nodes` | `updated_at` | Last row update time (heartbeat or resource report) |
| `gpu_devices` | `collected_at` | Agent's collection time (or Server time as fallback) |
| `gpu_devices` | `created_at` | Row creation time |
| `gpu_devices` | `updated_at` | Last row update time |
| `node_system_snapshots` | `collected_at` | Server time when snapshot was persisted |
| `node_filesystem_snapshots` | `collected_at` | Server time when snapshot was persisted |
| `node_network_snapshots` | `collected_at` | Server time when snapshot was persisted |

### 2.4 API Responses

**GET /api/v1/nodes** returns per node:
- `last_heartbeat_at` — from `nodes.last_heartbeat_at` (nullable)
- `created_at`, `updated_at`

**GET /api/v1/gpus** returns per GPU:
- `collected_at` — from `gpu_devices.collected_at` (nullable)
- `created_at`, `updated_at`

### 2.5 Web Frontend

| Page | Field | Display Label (zh-CN) | API Field | DB Field |
|------|-------|----------------------|-----------|----------|
| NodesPage table | 最后心跳 | `nodes.lastHeartbeat` | `last_heartbeat_at` | `nodes.last_heartbeat_at` |
| GpusPage table | 采集时间 | `gpus.collectedAt` | `collected_at` | `gpu_devices.collected_at` |
| Dashboard node table | 最后心跳 | `nodes.lastHeartbeat` | `last_heartbeat_at` | `nodes.last_heartbeat_at` |
| Dashboard diagnostics | Agent 最近上报 | `dashboard.agentLastReport` | `last_heartbeat_at` | `nodes.last_heartbeat_at` |
| Dashboard diagnostics | 最近采集时间 | `dashboard.latestCollection` | `collected_at` (max) | `gpu_devices.collected_at` |

**Relative time function** (`web/src/utils/format.ts:42`):
```typescript
export function formatRelativeTime(iso: string | undefined | null, locale?: string): string {
  if (!iso) return 'N/A'
  const diff = Date.now() - new Date(iso).getTime()
  const s = Math.floor(diff / 1000)
  if (s < 60) return isZh ? '刚刚' : 'just now'
  const m = Math.floor(s / 60)
  if (m < 60) return isZh ? `${m} 分钟前` : `${m}m ago`
  // ...
}
```

**Auto-refresh interval**: 5s (default in `useAutoRefresh.ts`)

**Stale detection thresholds** (DashboardPage.vue):
- Heartbeat stale: 20s (`STALE_HEARTBEAT_MS = 20000`)
- GPU stale: 30s (`30000`)

## 3. Root Cause Analysis

### 3.1 Primary Root Cause: `gpu_devices.collected_at` Uses Agent Time, Not Server Receive Time

The `gpu_devices.collected_at` field is set to the **Agent's collection timestamp** from the
resource report payload, NOT the Server's current time. This creates a critical problem:

1. When GPU collection fails, the Registry keeps the **last successful** GPU data in cache
   (`r.lastGPUDevices`, `r.lastGPUResources`)
2. The cached data retains its **original** `CollectedAt` timestamp from when it was first collected
3. On each subsequent cycle, if collection continues to fail, the Agent sends the same cached
   data with the same **old** `CollectedAt` timestamp
4. The Server stores this old timestamp in `gpu_devices.collected_at`
5. The Web UI displays this old timestamp, showing "50+ seconds ago" or more

**Why this happens frequently:**
- External GPU scripts (`discover.sh`, `metrics.sh`) have a 5-second timeout each
- Sequential execution means each cycle waits up to 10s for GPU collection
- With a 5s collect interval, the collect cycle is always behind, causing ticker backpressure
- GPU scripts can intermittently fail (nvidia-smi hangs, driver issues, etc.)
- Each failure extends the staleness of the cached `CollectedAt`

### 3.2 Secondary Issue: Resource Report Does Not Update `last_heartbeat_at`

The resource report handler updates `nodes.updated_at` but does **NOT** update
`nodes.last_heartbeat_at`. This means:

1. A node that sends resource reports but has heartbeat failures will appear offline
2. The "最后心跳" time only reflects the last heartbeat, not the last resource report
3. If heartbeat fails for any reason (network, server load), `last_heartbeat_at` becomes stale
   even though resource reports are successfully received

### 3.3 Tertiary Issue: Dashboard Label Confusion

The Dashboard's "Agent 最近上报" (Agent Last Report) actually displays `last_heartbeat_at`,
not the resource report time. This label is misleading because:
- "Agent 最近上报" suggests the last resource report
- But it shows the last heartbeat time (updated every 2s, not tied to actual data collection)

### 3.4 Summary of Root Causes

| # | Issue | Impact |
|---|-------|--------|
| 1 | `gpu_devices.collected_at` uses Agent time (from possibly cached data) | GPU "采集时间" can show stale timestamps |
| 2 | Resource report does not update `nodes.last_heartbeat_at` | Node heartbeat time can diverge from actual activity |
| 3 | Dashboard label confusion | Users misinterpret what the time represents |
| 4 | Server time fallback never triggers (Agent always sends non-empty `CollectedAt`) | No server-side freshness guarantee |

## 4. Fix Plan

### Fix 1: Use Server Time for `gpu_devices.collected_at`

Change the resource report handler to **always** use the Server's `now` time for
`gpu_devices.collected_at`, instead of the Agent's `CollectedAt` from the payload.

**Rationale:**
- `collected_at` in the database should reflect "when did the server last receive data for this GPU"
- The Agent's original collection time is useful metadata but should not be the primary freshness indicator
- Using server time guarantees that a successful resource report always produces a fresh timestamp
- Eliminates the stale-cache timestamp problem entirely

**Alternative considered:** Keep Agent time but add a separate `last_reported_at` field.
Rejected because it adds complexity without clear benefit for the current use case.

### Fix 2: Update `last_heartbeat_at` on Resource Report

Update `nodes.last_heartbeat_at` in the resource report handler, since receiving a resource
report is also proof that the Agent is alive and communicating.

**Rationale:**
- A resource report is as much a sign of life as a heartbeat
- If heartbeat fails but resource report succeeds, the node should not become "offline"
- Makes node online status more robust

### Fix 3: Fix Dashboard Labels

| Current Label (zh-CN) | Current Field | New Label (zh-CN) | New Field |
|----------------------|---------------|-------------------|-----------|
| Agent 最近上报 | `last_heartbeat_at` | Agent 最后通信 | `last_heartbeat_at` |
| 最近采集时间 | max(`collected_at`) | GPU 最后采集 | max(`collected_at`) |

### Fix 4: Adjust Default Intervals (Recommended)

| Interval | Current Default | Recommended |
|----------|----------------|-------------|
| `heartbeat.interval` | 2s | 5s (reduce unnecessary traffic, resource report is sufficient) |
| `collectors.system.interval` | 5s | 5s (keep) |
| `collectors.gpu_external.timeout` | 5s | 5s (keep) |
| `web_refresh_interval` | 5s | 5s (keep) |

Note: heartbeat interval should ideally be >= collect interval, since resource reports
will also update `last_heartbeat_at` after Fix 2.

## 5. Files to Modify

| File | Change |
|------|--------|
| `internal/server/api/resource_handlers.go` | Use server `now` for `gpu_devices.collected_at`; update `nodes.last_heartbeat_at` |
| `web/src/locales/zh-CN.ts` | Fix label text |
| `web/src/locales/en-US.ts` | Fix label text |
| `web/src/pages/DashboardPage.vue` | Update `latestHeartbeat` computation to use `collected_at` where appropriate |
| `configs/agent.yaml` | Optionally adjust default intervals |

## 6. Verification Plan

### 6.1 Build & Test
```bash
go test ./...
go vet ./...
go build ./cmd/server
go build ./cmd/agent
cd web && npm run build
```

### 6.2 API Verification
```bash
# Continuous monitoring of node heartbeat
watch -n 2 "curl -s http://127.0.0.1:18080/api/v1/nodes | jq '.[0].last_heartbeat_at'"

# Continuous monitoring of GPU collection time
watch -n 2 "curl -s http://127.0.0.1:18080/api/v1/gpus | jq '.[0].collected_at'"
```

### 6.3 Expected Behavior After Fix
1. `gpu_devices.collected_at` updates to server's current time on every successful resource report
2. `nodes.last_heartbeat_at` updates on both heartbeat and resource report
3. Web UI labels accurately describe what each time represents
4. Timestamps should typically be within 0-5s of current time
5. If Agent is not running or reports fail, times will show when data was last successfully received

## 7. Implementation Results

### 7.1 Status: Completed

### 7.2 Modified Files

| File | Change Summary |
|------|---------------|
| `internal/server/api/resource_handlers.go` | Fix 1: `collected_at` now uses server `now` instead of Agent `CollectedAt`. Fix 2: resource report now updates `nodes.last_heartbeat_at` and `status = 'online'` |
| `web/src/locales/zh-CN.ts` | Changed `agentLastReport` from `Agent 最近上报` to `Agent 最后通信` |
| `web/src/locales/en-US.ts` | Changed `agentLastReport` from `Agent Last Report` to `Agent Last Communication` |
| `web/src/pages/DashboardPage.vue` | Added P2-001 comments clarifying field semantics for `latestHeartbeat` and `latestCollected` |
| `docs/plan/phase-2g-heartbeat-collection-time-investigation.md` | This investigation document |

### 7.3 Verification Results

| Check | Result |
|-------|--------|
| `go test ./...` | ✅ All tests pass (including resource_handlers_test.go, agent_identity_test.go, tenant_isolation_test.go) |
| `go vet ./...` | ✅ No warnings |
| `go build ./cmd/server` | ✅ Build successful |
| `go build ./cmd/agent` | ✅ Build successful |
| `cd web && npm run build` | ✅ Build successful (3.22s) |

### 7.4 Before/After Behavior

**Before (Problem):**
- `gpu_devices.collected_at` = Agent's collection timestamp from payload (possibly stale cached data)
- `nodes.last_heartbeat_at` updated ONLY by heartbeat requests (every 2s)
- Resource report does NOT update `last_heartbeat_at` or `status`
- Dashboard label "Agent 最近上报" misleadingly suggests resource report time

**After (Fix):**
- `gpu_devices.collected_at` = Server receive time (`now`) — always reflects when data was last received
- `nodes.last_heartbeat_at` updated by BOTH heartbeat AND resource report
- Resource report sets `status = 'online'` — node stays online as long as any communication succeeds
- Dashboard label "Agent 最后通信" accurately describes last contact time

### 7.5 Field Semantics After Fix

| Field | Updated By | Meaning |
|-------|-----------|---------|
| `nodes.last_heartbeat_at` | Heartbeat AND Resource Report | Agent last communication time (any contact) |
| `nodes.status` | Heartbeat AND Resource Report sets `online`; health checker sets `offline` | Node online/offline status |
| `nodes.updated_at` | Heartbeat AND Resource Report | Last row modification time |
| `gpu_devices.collected_at` | Resource Report (server `now`) | When GPU data was last received by server |
| `gpu_devices.updated_at` | Resource Report (server `now`) | Last GPU row modification time |
| `node_system_snapshots.collected_at` | Resource Report (server `now`) | When host snapshot was persisted |

### 7.6 Web Display Mapping

| Page | Display Label (zh-CN) | API Field | Semantics |
|------|----------------------|-----------|-----------|
| Nodes | 最后心跳 | `last_heartbeat_at` | Agent last communication time |
| GPUs | 采集时间 | `collected_at` | GPU data last received by server |
| Dashboard | Agent 最后通信 | `last_heartbeat_at` (max) | Latest agent contact across all nodes |
| Dashboard | 最近采集时间 | `collected_at` (max) | Latest GPU data received across all GPUs |

## 8. Remaining Risks

1. **Server time vs Agent time**: Using server `now` for `collected_at` loses the distinction between "when Agent collected" and "when Server received." If future requirements need fine-grained timing analysis, a separate `agent_collected_at` field could be added.

2. **Heartbeat interval not adjusted**: The heartbeat interval remains 2s (default). With resource report also updating `last_heartbeat_at` (every 5s), the 2s heartbeat is redundant for freshness but still serves task assignment. Consider adjusting to 5s in a future update to reduce unnecessary traffic.

3. **No additional `collected_at` on nodes**: Host system snapshots have `collected_at` in the `node_system_snapshots` table but there is no top-level `collected_at` on the `nodes` API response. If users want to see "host metrics last collected" at a glance, this could be added in a future update.

4. **Clock skew**: If the server clock is significantly different from the browser clock, relative time display could be inaccurate. Server timestamps are stored in RFC3339 with timezone offsets, and browser `Date.parse` handles them correctly, so this risk is minimal under normal conditions.

## 9. Recommended Follow-up

1. **Config consolidation**: Consider increasing default `heartbeat.interval` from 2s to 5s since resource reports now also confirm agent life.
2. **Add `host_collected_at` to node API**: Query the latest `node_system_snapshots.collected_at` and include it in `GET /api/v1/nodes` responses for host-level collection time visibility.
3. **Prometheus scrape interval display**: If ObservabilityTargetsPage shows scrape-related times, ensure it uses Prometheus scrape timestamps, not agent collection timestamps.
