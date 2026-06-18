> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# Phase 2H: Web Node Visibility Investigation

> Status: In progress
> Created: 2026-06-16
> Version: v0.1.15 → v0.1.16

## 1. Problem

After v0.1.15 update, Web pages show no nodes or GPUs, even though the backend API
returns data correctly.

## 2. Confirmed Facts

| Fact | Evidence |
|------|----------|
| `nodes` table has node `node-56a34816-...` with `status=online` | DB verified |
| `last_heartbeat_at` continuously updating | Agent running |
| `GET /api/v1/nodes` returns 200 with 1 node (array) | curl verified |
| `GET /api/v1/gpus` returns 200 with 8 GPUs (array) | curl verified |
| Unauthenticated requests return 401 | Auth works |
| `GET /api/v1/dashboard` returns 404 (endpoint does not exist) | curl verified |
| Server routes are all at `/api/v1/*` | router.go confirmed |
| Server has SPA fallback: any non-API path returns `index.html` | main.go confirmed |

## 3. Root Cause Analysis

### 3.1 Primary Root Cause: `client.ts` `fetch()` Uses Wrong URL

**File:** `web/src/api/client.ts`, line 38

**Bug:** In commit `86ab1d4` (phase-2f), the `ApiClient.request()` method was updated to
compute `fullUrl` with auto-prepended `/api/v1` prefix:

```typescript
// Line 27: Full URL computed correctly
const fullUrl = url.startsWith('/api/v1') ? url : (url.startsWith('http') ? url : this.apiBase + url)
```

But the actual `fetch()` call on line 38 still uses the ORIGINAL `url`:

```typescript
// Line 38: BUG — uses url instead of fullUrl
const resp = await fetch(BASE + url, {
```

`BASE` is `''`, so the actual request goes to `/nodes` instead of `/api/v1/nodes`.

**Impact chain:**
1. `fetchNodes()` calls `apiClient.get('/nodes')`
2. `client.ts` computes `fullUrl = '/api/v1/nodes'` but never uses it
3. `fetch('' + '/nodes')` = `fetch('/nodes')` → hits the Go server
4. `/nodes` is not an API route → Go server's SPA fallback returns `index.html`
5. Browser parses HTML as JSON → gets a non-array object
6. `Array.isArray(data) ? data : []` returns `[]` (empty array)
7. Web page shows "no nodes"

**Why `fetchNode(id)` and `fetchGPU(id)` work but `fetchNodes()` and `fetchGPUs()` don't:**
- `fetchNode(id)` uses absolute path `/api/v1/nodes/${id}` → passes through unchanged → WORKS
- `fetchNodes()` uses relative path `/nodes` → should get `/api/v1` prepended → BROKEN (fullUrl unused)
- Same pattern for GPU functions

### 3.2 Secondary Bug: Sweep SQL Parameter Count Mismatch

**Files:**
- `internal/server/api/sweep.go` lines 74, 87
- `internal/server/api/agent_handlers.go` line 364

**Bug:** Three SQL UPDATE queries on `gpu_leases` have 4 `?` placeholders but only 3 arguments:

```sql
UPDATE gpu_leases SET status = ?, updated_at = ?   -- 2 placeholders
WHERE expires_at IS NOT NULL AND expires_at < ? AND status = ?  -- 2 more placeholders
-- Total: 4 placeholders
```

But the Go code passes only 3 args: `LeaseFailed, now, LeaseReserved`
Missing: the 4th arg for `status = ?` in the WHERE clause.

Wait, looking at this more carefully:

```go
database.Exec(
    `UPDATE gpu_leases SET status = ?, updated_at = ?
     WHERE expires_at IS NOT NULL AND expires_at < ? AND status = ?`,
    LeaseFailed, now, LeaseReserved,
)
```

Placeholders:
1. `status = ?` → LeaseFailed ✓
2. `updated_at = ?` → now ✓
3. `expires_at < ?` → LeaseReserved ✗ (should be `now`)
4. `status = ?` → MISSING ✗ (should be `LeaseReserved`)
```

**Error:** The `now` value is being used for `updated_at = ?` (correct) but also being treated as the comparison for `expires_at < ?`. The `LeaseReserved` is being used for `expires_at < ?` (wrong — `LeaseReserved` is a string, not a time). And there's NO argument for the last `status = ?`.

The correct order is: `LeaseFailed, now, now, LeaseReserved`:
1. `status = ?` → LeaseFailed ✓
2. `updated_at = ?` → now ✓
3. `expires_at < ?` → now ✓
4. `status = ?` → LeaseReserved ✓

Same bug in all three locations.

**Impact:** The sweep loop fails every 30s with `"not enough args to execute query: want 4 got 3"`. Expired leases are never cleaned up. The `agent_handlers.go` call in heartbeat task claiming also fails silently.

### 3.3 Secondary Bug: `/api/v1/dashboard` Does Not Exist

The Dashboard page does NOT call `/api/v1/dashboard` — it composes data from `fetchNodes()` +
`fetchGPUs()`. But if any observability or debug page calls it, it returns 404. Not a direct
cause of the node visibility issue.

## 4. Fixes

### Fix 1: Use `fullUrl` in `client.ts` fetch call

**File:** `web/src/api/client.ts`, line 38
**Change:** `fetch(BASE + url, ...)` → `fetch(fullUrl, ...)`

### Fix 2: Fix sweep SQL parameter count (3 locations)

| File | Line | Change |
|------|------|--------|
| `sweep.go` | 74 | `LeaseFailed, now, LeaseReserved` → `LeaseFailed, now, now, LeaseReserved` |
| `sweep.go` | 87 | `LeaseFailed, now, LeaseActive` → `LeaseFailed, now, now, LeaseActive` |
| `agent_handlers.go` | 364 | `LeaseFailed, now, LeaseReserved` → `LeaseFailed, now, now, LeaseReserved` |

## 5. Verification

### 5.1 Build
```bash
go test ./...          # All pass
go vet ./...           # Clean
go build ./cmd/server  # Success
go build ./cmd/agent   # Success
npm run build          # Success
```

### 5.2 Browser Verification
1. F12 → Network → Preserve log
2. Navigate to Nodes page
3. Verify `GET /api/v1/nodes` is called (NOT `/nodes`)
4. Verify response is 200 with node array
5. Verify nodes display in table
6. Repeat for GPUs page
7. Verify no more sweep SQL errors in server logs

## 6. Modified Files

| File | Change |
|------|--------|
| `web/src/api/client.ts` | Line 38: use `fullUrl` instead of `BASE + url` |
| `internal/server/api/sweep.go` | Lines 74, 87: add missing `now` arg |
| `internal/server/api/agent_handlers.go` | Line 364: add missing `now` arg |
| `docs/plan/phase-2h-web-node-visibility-investigation.md` | This document |
