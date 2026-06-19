# Runtime Config Copy, First-Save, and Selection Fix

**Date**: 2026-06-19
**Status**: Pending implementation
**Scope**: Three peer issues on runtime configuration copy/edit/selection chain

---

## 1. Issue Descriptions

### Issue 1: Name Field Duplication

The clone dialog (`BackendRuntimesPage.vue`) shows two editable name fields: `name` and `display_name`. Both default to similar values (source template name + suffix). Users find this confusing.

### Issue 2: First-Save Parameter Override Lost

When cloning a runtime template and modifying `shm_size` (e.g., from default `8gb` to `6gb`) on the first save, the modification is silently discarded. After reopening the cloned runtime, `shm_size` reverts to `8gb`. However, subsequent edits (PATCH) work correctly.

### Issue 3: Copied Runtimes Not in Deployment Wizard Selector

After cloning a runtime template, the new user runtime does not appear in the deployment wizard's runtime selection dropdown. This affects vLLM and SGLang specifically; llama.cpp behavior needs verification.

---

## 2. Scope

### In Scope

- Runtime clone dialog UI (name field convergence)
- Clone save flow (first-save payload includes all user overrides)
- Deployment wizard runtime selector visibility for cloned/user runtimes
- NodeBackendRuntime creation on clone (if needed for selector visibility)
- E2E hardening against regression

### Out of Scope (DO NOT MODIFY)

- RBAC / tenant / billing / quota
- GPU scheduling / GPU lease lifecycle
- Docker runtime driver core
- Model scan / metadata
- Dashboard / Nodes / GPU / Monitoring pages
- Prometheus / Grafana
- API route renaming
- Schema migrations
- Unrelated E2E scripts
- Repository-wide formatting

---

## 3. Reproduction Steps

### Issue 1: Name Duplication

1. Open Backend Runtimes page
2. Click "Clone" on any builtin runtime (e.g., vLLM NVIDIA)
3. Observe two editable fields: "名称" (name) and "显示名称" (display_name)
4. Both contain similar values: e.g., `vllm-nvidia-docker-Custom` and `vllm NVIDIA Docker - Custom`

### Issue 2: shm_size Reversion

1. Open Backend Runtimes page
2. Clone vLLM NVIDIA runtime
3. In clone dialog, change shm_size from `16gb` to `6gb`
4. Click save
5. After save completes, open the cloned runtime for editing
6. Observe shm_size shows `16gb` (original template default), NOT `6gb`

### Issue 3: Wizard Selector Invisibility

1. Clone vLLM NVIDIA runtime (save as user config)
2. Go to Model Deployments page
3. Start new deployment wizard
4. Select vLLM backend + version
5. At runtime selection step: cloned runtime does NOT appear in dropdown

---

## 4. Root Cause Analysis

### Root Cause 1: Name Field Duplication

**Failing layer**: UI (BackendRuntimesPage.vue clone dialog)

**Exact cause**: The clone dialog `showClone()` initializes both `name` and `display_name` as separate editable fields. The `name` field is the unique identifier (UNIQUE constraint on `tenant_id, name`) while `display_name` is the human-readable label. Having both editable creates confusion, especially when both default to nearly-identical values differing only by suffix.

**Fix**: Make `name` auto-generated (read-only) in the clone dialog. Show it as "internal identifier" or hide it. Keep only `display_name` as the user-editable field. The backend already auto-generates `name` when it's empty via `uniqueRuntimeName()`.

### Root Cause 2: First-Save Override Lost

**Failing layer**: UI (BackendRuntimesPage.vue `doCloneSave`)

**Exact cause**: `doCloneSave` (line 351) sends only `{name, display_name}` to the clone API. The `buildClonePayload()` function (line 328-333) correctly builds a complete payload including `docker_json`, `args_override_json`, etc., but this function is ONLY used for the command preview — it is never passed to the save API.

```typescript
// Current (broken):
await apiClient.post(`/backend-runtimes/${cloneSource.value.id}/clone`, {
  name: cloneForm.name,
  display_name: cloneForm.display_name   // docker_json is NOT sent
})

// Should be:
await apiClient.post(`/backend-runtimes/${cloneSource.value.id}/clone`, buildClonePayload())
```

**Why subsequent edit works**: `doEdit` calls `buildPayload()` which includes `docker_json`. So PATCH after initial creation works correctly.

**Why previous tests missed this**: The clone E2E test (`scripts/e2e-clone-template-parameter-persistence.sh`) only verified `name`, `display_name`, `is_editable`, and `shm_size=20gb` in docker_json. The `20gb` value was the override, so it passed. But the test did NOT verify that a DIFFERENT value (e.g., `6gb`) persists after save — it only checked that the original override value was present. And the backend clone handler was fixed in the previous round to accept overrides, but the frontend still doesn't send them.

### Root Cause 3: Wizard Selector Invisibility

**Failing layer**: NBR binding (NodeBackendRuntime not created on clone)

**Exact cause**: `HandleCloneBackendRuntime` creates a new `backend_runtimes` row but does NOT create a `node_backend_runtimes` row. Without an NBR:
- The wizard preflight step (`HandlePreflightDeployments`) requires `nbr.status = 'ready'` to show candidate nodes
- Without any candidate nodes, the wizard shows no available runtimes
- The `can_run` field is `false`, preventing deployment start

The frontend deployment wizard runtime selector (`filteredRuntimes`) filters by `backend_version_id` — this filter works correctly and would show the cloned runtime. But the subsequent preflight step (which actually determines deployability) requires an NBR.

**Why vLLM/SGLang vs llama.cpp may differ**: llama.cpp might have an existing NBR from a previous enable action. vLLM and SGLang NBRs may not exist for the user's tenant-scoped clone. Also, the NBR for a cloned runtime must be explicitly enabled via `POST /nodes/{id}/backend-runtimes/enable` — if never done, no NBR exists.

**Fix**: After cloning, automatically create an NBR for each online node that can support this runtime (matching vendor, docker available). Or, add a clear UI indication that the runtime needs to be "enabled" on a node before it can be used in deployments.

### Recurrence Analysis — Why This Happened Again

| Previous Test | What It Covered | What It Missed |
|---|---|---|
| `e2e-clone-template-parameter-persistence.sh` | Clone creates runtime with correct name, display_name, is_editable, shm_size override | Didn't verify shm_size ≠ original default; assumed override=original means success |
| `e2e-dryrun-parameter-matrix-enhanced.sh` | DryRun for builtin runtimes | Only tested builtin templates, not cloned/user runtimes |
| `e2e-deployment-visibility-selected.sh` | Deployment CRUD visibility | Used builtin runtimes; didn't verify cloned runtime visibility |
| `e2e-real-smoke-all-three.sh` | Real container start/stop | Used builtin runtimes; didn't test cloned runtime path |
| `e2e-matrix-verifier.sh` | Cross-backend matrix | Only tested builtin runtimes |

---

## 5. Fix Plan

### Fix 1: Name Field Convergence

**File**: `web/src/pages/BackendRuntimesPage.vue`

- In clone dialog: make `name` auto-generated (read-only or hidden)
- Keep `display_name` as the primary user-editable field
- Backend already handles empty `name` via `uniqueRuntimeName()`
- Add i18n label: "内部名称" / "Internal name" for the read-only `name` field

### Fix 2: First-Save Override

**File**: `web/src/pages/BackendRuntimesPage.vue`, function `doCloneSave`

- Change the API call to use `buildClonePayload()` instead of `{name, display_name}`
- This sends `docker_json`, `args_override_json`, `default_env_json`, `entrypoint_override_json` with user modifications

### Fix 3: Wizard Selector Visibility

**Files**: `internal/server/api/node_runtime_handlers.go`, `web/src/pages/BackendRuntimesPage.vue`

- After cloning a runtime, auto-enable it on all online nodes that support the vendor
- This creates NodeBackendRuntime records so the runtime appears in deployment wizard
- Alternative: if auto-enable is too aggressive, at minimum provide a one-click "Enable on node" button after clone

The minimal fix: add auto-enable in the clone handler. After INSERT into `backend_runtimes`, for each online node matching the runtime's vendor, create a `node_backend_runtimes` record with `status='ready'` if docker is available.

---

## 6. E2E Acceptance Criteria

New script: `scripts/e2e-runtime-config-copy-first-save-selection.sh`

### Test 1: Name Field Rules

- Clone vLLM template
- Verify clone UI shows only one primary editable name field (display_name)
- Verify `name` is auto-generated or read-only
- After save, list shows `display_name`
- Deployment wizard selector shows `display_name`

### Test 2: First-Save Override Persistence

- Clone vLLM template with shm_size=6gb
- Save → immediately GET detail
- Assert `docker_json.shm_size = "6gb"`
- DryRun → assert docker command uses `--shm-size 6gb`
- Reverse: `8gb` / `16gb` NOT present

### Test 3: Subsequent-Edit Regression

- Edit the cloned runtime, change shm_size to 5gb
- Save → GET detail
- Assert `docker_json.shm_size = "5gb"`
- DryRun uses `--shm-size 5gb`

### Test 4: Wizard Selector Visibility

- Clone vLLM template
- Call deployment wizard runtime list API
- Assert cloned runtime appears in response
- Preflight with cloned runtime → candidate exists
- DryRun uses cloned runtime id

### Test 5: SGLang Same Coverage

Same as vLLM tests 1-4 but for SGLang.

### Test 6: llama.cpp Same Coverage

Same as vLLM tests 1-4 but for llama.cpp (control group).

### Test 7: Parameter Matrix

For at least vLLM, verify first-save persistence of:
- shm_size, privileged, ipc_mode
- env overrides
- args overrides
- image_name
- Each: first-save payload → GET detail → DryRun → reverse assertion

---

## 7. Test Gap Closure Table

| Gap | Previous coverage | Why insufficient | New assertion | Script |
|-----|------------------|-----------------|---------------|--------|
| First-save override persistence | Clone E2E checked shm_size=20gb | 20gb was the override value; didn't differ from "original default changed" case | shm_size=6gb → GET = 6gb | New script |
| Save then immediate GET | Clone E2E only checked clone response | Clone response ≠ persisted state if merge bug exists | GET detail after save | New script |
| Default value reverse assertion | None | No test checked that default 8gb/16gb does NOT leak | `assert_not_contains "8gb"` | New script |
| vLLM selector visibility | Deployment visibility used builtin | Only tested builtin runtime visibility | Cloned runtime in selector | New script |
| SGLang selector visibility | None | SGLang never tested for cloned runtime visibility | Cloned SGLang in selector | New script |
| NBR auto-creation on clone | None | No test verified NBR exists after clone | NBR exists, status=ready | New script |
| Clone sends docker_json | Clone E2E assumed backend fixed | Frontend never sends docker_json | Verify clone payload includes docker_json | New script |
| User vs builtin runtime in DryRun | Matrix verifier only used builtin | DryRun always used builtin template ids | DryRun with cloned runtime id | New script |

---

## 8. Verification Commands

```bash
# Syntax
bash -n scripts/e2e-runtime-config-copy-first-save-selection.sh

# Go
go vet ./internal/server/... ./internal/agent/...
go test ./internal/server/...

# Web
npm --prefix web run build

# E2E
bash scripts/e2e-runtime-config-copy-first-save-selection.sh
bash scripts/e2e-dryrun-parameter-matrix-enhanced.sh

# Governance
git diff --check
git diff --stat
git status --short
```

---

## 9. Out-of-Scope Findings

(To be populated during implementation if any unrelated issues are discovered)
