# Logging Noise & Observability Cleanup — Closeout

**Date**: 2026-06-23  
**Status**: COMPLETE  
**Commit**: (pending)

---

## A. Audit Conclusions

Based on the Logging Noise & Observability Audit Report, the following issues were identified and addressed:

| # | Issue | Verdict | Action Taken |
|---|-------|---------|--------------|
| 1 | Static asset request log noise | Real issue (13% of server logs) | Fixed — 2xx static assets now log at DEBUG |
| 2 | slow_operation repetitive warnings | Real issue (91.7% from one endpoint) | Fixed — route-specific threshold for node-run-plans |
| 3 | model_instance_logs high frequency | Real issue (108 entries in 3 min) | Fixed — success with no stderr change now DEBUG |
| 4 | reconcile log repetition | Real issue (198 entries, same state) | Fixed — state-change detection + periodic summary |
| 5 | container_id empty | Not a bug (legitimate state) | Fixed — readability: `container_id_status=not_allocated` |
| 6 | ERROR log coverage | Mostly complete, missing panic recovery | Fixed — added RecoveryMiddleware |
| 7 | client_ip / user_agent missing | Real issue | Fixed — added to all request logs |
| 8 | JSON log format documentation | Feature exists, docs missing | Fixed — added logging-configuration.md |

---

## B. Fix Summary

### 1. Static Asset Request Log Noise

**File**: `internal/server/api/middleware_logging.go`

**Changes**:
- Added `staticAssetPrefixes` and `staticAssetSuffixes` for path detection
- Added `isStaticAsset()` function
- Static assets with 2xx status now log at DEBUG level
- Static assets with 4xx/5xx still log at WARN/ERROR

**Before**: All `/assets/*.css`, `/assets/*.js`, `/favicon.ico` requests logged at INFO  
**After**: Only non-2xx or slow static asset requests logged at INFO/WARN

### 2. slow_operation Route-Specific Threshold

**File**: `internal/server/api/middleware_logging.go`

**Changes**:
- Added `routeSpecificSlowThresholds` map
- `/api/v1/node-run-plans/{id}/logs` threshold: 3000ms (was 1000ms)
- All other routes: 1000ms (unchanged)

**Before**: 66 slow_operation warnings for node-run-plans in 27 minutes  
**After**: Expected to reduce by ~80% (threshold increased 3x)

### 3. model_instance_logs High Frequency Task

**File**: `cmd/agent/main.go`

**Changes**:
- Added `logsTaskState` to track stderr bytes per instance
- "processing logs task" now logs at DEBUG (was INFO)
- "logs task completed" logs at INFO only if stderr changed; otherwise DEBUG
- Failure cases still log at ERROR

**Before**: 108 INFO entries in 3 minutes  
**After**: Expected ~90% reduction (only stderr changes trigger INFO)

### 4. Reconcile Log Noise Reduction

**File**: `cmd/agent/main.go`

**Changes**:
- Added `reconcileState` with `ChangedTracker` for state change detection
- State change (total/exited/running differs): INFO
- No change: DEBUG
- Every 5th unlogged invocation (~5 minutes): summary INFO

**Before**: 198 INFO entries (every minute, same state)  
**After**: ~40 INFO entries (state changes + periodic summaries)

### 5. container_id Empty Readability

**File**: `cmd/agent/main.go`

**Changes**:
- Empty `container_id` now logs as `container_id_status=not_allocated`
- Non-empty `container_id` still logs as `container_id=<value>`

**Before**: `container_id=` (empty field)  
**After**: `container_id_status=not_allocated` (explicit status)

### 6. Panic Recovery Middleware

**Files**: 
- `internal/server/api/middleware_recovery.go` (new)
- `cmd/server/main.go`

**Changes**:
- Added `RecoveryMiddleware` that catches handler panics
- Logs ERROR with request_id, method, path, panic message, stack trace
- Returns HTTP 500 with `{"error":"internal server error"}`
- Registered as outermost middleware in server

**Before**: Handler panic would crash the server process  
**After**: Panic is caught, logged, and returns 500

### 7. client_ip / user_agent

**File**: `internal/server/api/middleware_logging.go`

**Changes**:
- Added `extractClientIP()` to extract IP from `RemoteAddr`
- Added `truncateString()` to limit user_agent to 200 chars
- All request logs now include `client_ip` and `user_agent` fields

**Before**: Request logs had request_id, method, path, status, duration  
**After**: Also includes client_ip and user_agent

### 8. JSON Log Format Documentation

**File**: `docs/logging-configuration.md` (new)

**Content**:
- Log format options (text vs JSON)
- Configuration parameters
- Log levels
- Output destinations
- Log rotation
- Special behaviors (high-frequency suppression, static assets, slow operations, etc.)
- Production and development recommendations

---

## C. Files Modified

| File | Status | Description |
|------|--------|-------------|
| `internal/server/api/middleware_logging.go` | Modified | Static asset filtering, client_ip/user_agent, route-specific thresholds |
| `internal/server/api/middleware_recovery.go` | New | Panic recovery middleware |
| `internal/server/api/middleware_logging_test.go` | New | Tests for static assets, thresholds, IP extraction, truncation |
| `internal/server/api/middleware_recovery_test.go` | New | Tests for panic recovery (500, nil, struct, chained) |
| `cmd/agent/main.go` | Modified | Reconcile noise reduction, logs task noise reduction, container_id readability |
| `cmd/server/main.go` | Modified | Register RecoveryMiddleware |
| `docs/logging-configuration.md` | New | Logging configuration documentation |

---

## D. Test Results

```
=== RUN   TestIsStaticAsset
--- PASS: TestIsStaticAsset (0.00s)
=== RUN   TestGetSlowThreshold
--- PASS: TestGetSlowThreshold (0.00s)
=== RUN   TestExtractClientIP
--- PASS: TestExtractClientIP (0.00s)
=== RUN   TestTruncateString
--- PASS: TestTruncateString (0.00s)
=== RUN   TestStaticAsset2xxNotLoggedAsInfo
--- PASS: TestStaticAsset2xxNotLoggedAsInfo (0.00s)
=== RUN   TestStaticAsset4xxStillLogged
--- PASS: TestStaticAsset4xxStillLogged (0.00s)
=== RUN   TestClientIPAndUserAgentInRequestLog
--- PASS: TestClientIPAndUserAgentInRequestLog (0.00s)
=== RUN   TestUserAgentTruncation
--- PASS: TestUserAgentTruncation (0.00s)
=== RUN   TestRecoveryMiddleware_PanicReturns500
--- PASS: TestRecoveryMiddleware_PanicReturns500 (0.00s)
=== RUN   TestRecoveryMiddleware_NormalRequestPasses
--- PASS: TestRecoveryMiddleware_NormalRequestPasses (0.00s)
=== RUN   TestRecoveryMiddleware_PanicWithNil
--- PASS: TestRecoveryMiddleware_PanicWithNil (0.00s)
=== RUN   TestRecoveryMiddleware_PanicWithStruct
--- PASS: TestRecoveryMiddleware_PanicWithStruct (0.00s)
=== RUN   TestRecoveryMiddleware_ChainedWithLogging
--- PASS: TestRecoveryMiddleware_ChainedWithLogging (0.00s)

All tests pass.
```

---

## E. Estimated Log Volume Reduction

Estimated ~90% INFO/WARN log volume reduction based on audit sample.

| Component | Before (per hour, estimated) | After (per hour, estimated) | Reduction |
|-----------|------------------------------|----------------------------|-----------|
| Static asset requests | ~260 INFO | ~10 DEBUG | ~96% |
| slow_operation (node-run-plans) | ~146 WARN | ~30 WARN | ~80% |
| model_instance_logs tasks | ~2160 INFO | ~200 INFO | ~91% |
| reconcile containers | ~60 INFO | ~12 INFO | ~80% |
| **Total** | ~2626 INFO/WARN | ~252 INFO/WARN | **~90%** |

Note: These are estimates based on the audit sample from 2026-06-23. Actual reduction depends on workload patterns.

---

## F. Remaining Limitations

1. **X-Forwarded-For not used**: Client IP extraction uses `RemoteAddr` only. If behind a trusted proxy, users must configure proxy headers separately.

2. **logsTaskState memory**: The `logsTaskState.lastStderrBytes` map grows unbounded with instance IDs. For long-running agents with many instances, consider periodic cleanup.

3. **reconcileState unloggedCount**: Counter resets on state change, so the 5-minute summary is approximate.

4. **Route-specific thresholds are hardcoded**: The `routeSpecificSlowThresholds` map is not configurable via config file. Future work could make this configurable.

5. **No log aggregation integration**: Documentation covers configuration but not specific integration with ELK, Loki, or Datadog.

---

## G. Commit Information

**Commit ID**: (pending)  
**Commit Message**: `ops: reduce logging noise and add recovery middleware`  
**Push Result**: (pending)

---

## H. Git Status

```
 M VERSION
 M cmd/agent/main.go
 M cmd/server/main.go
 M internal/server/api/middleware_logging.go
?? docs/logging-configuration.md
?? internal/server/api/middleware_logging_test.go
?? internal/server/api/middleware_recovery.go
?? internal/server/api/middleware_recovery_test.go
```

Note: VERSION file modification is pre-existing and not part of this change.
