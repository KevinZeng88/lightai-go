# 04 - Claude Review and Implementation Plan

Date: 2026-06-19
Branch: main
Starting commit: c63cef6

## 1. Files Reviewed

- `00-known-issues-and-evidence.md` ✅ Reviewed
- `01-formal-e2e-requirements.md` ✅ Reviewed
- `02-acceptance-criteria-and-parameter-matrix.md` ✅ Reviewed
- `03-implementation-steps-recommendation.md` ✅ Reviewed

## 2. Review of 00-known-issues-and-evidence.md

### 2.1 Confirmed Issues

| Issue | Status | Evidence |
|-------|--------|----------|
| 2.1 vLLM port propagation not tested | **CONFIRMED** | Wizard scripts pass only `host_port`, never `container_port/app_port` custom combos. No `--port` assertion in command preview. |
| 2.2 vLLM positional model duplicate | **CONFIRMED** | No reverse assertion to check `--model` is absent. Scripts don't check args at all. |
| 2.3 default args override user args | **CONFIRMED** | `deduplicateArgs` was indeed keeping first (now fixed). Scripts don't verify flag count/existence. |
| 2.4 Matrix is runner not verifier | **CONFIRMED** | `e2e-model-runtime-wizard-nvidia-matrix.sh` runs sub-scripts, collects payloads, but doesn't assert runplan/command correctness. |
| 2.5 common.sh false pass risk | **CONFIRMED** | `e2e_instance_test()` (line 230-235) saves response but never checks `parsed_summary != ""`. `e2e_stop_deployment()` (line 247-251) uses `|| true` for the formal stop step itself. |
| 2.6 Instance stop not covered | **CONFIRMED** | All scripts call `POST /api/v1/deployments/{id}/stop`. None call instance-level stop. |
| 2.7 Inference parser no semantic assertion | **CONFIRMED** | `e2e_instance_test()` records status/duration but doesn't check that content/reasoning/text exists. |
| 2.8 Clone template not tested | **CONFIRMED** | No script calls `POST /api/v1/backend-runtimes/{id}/clone` with modified payload. No clone→deploy→dry-run chain. |
| 2.9 GGUF file/directory semantics | **CONFIRMED** | Wizard scripts pass `path_type=directory` even for `.gguf` files in some cases. |
| 2.10 Legacy/local scripts | **CONFIRMED** | `e2e-model-runtime-local.sh` uses direct `DB=...` and process management. `e2e-model-runtime-api.sh` references old API paths. |

### 2.2 Corrections to 00

- **Section 4 table**: `e2e-ui-persistence-runplan-selected.sh` is more a UI persistence smoke than dry-run smoke. Its primary assertions are about port values and instance creation, not about RunPlan parameter source correctness.
- **Section 2.10**: The claim about "old API paths like `/model-deployments`" needs clarification. I searched all current scripts and found NO uses of `/model-deployments` (old path). All current wizard scripts use `/api/v1/deployments`. The legacy `e2e-model-runtime-api.sh` uses a different path pattern (`/api/v1/model-types/...`). This point is **mostly historical** — the actual risk is lower than stated.

### 2.3 Accuracy Assessment

Overall accuracy: **HIGH**. All 10 issues are either confirmed by code inspection or previously fixed (e.g., 2.3 dedup priority). The scripts genuinely lack strong assertions.

## 3. Review of 01-formal-e2e-requirements.md

### 3.1 Confirmed Requirements

All 11 sections (E2E classification, script safety, assertions, artifacts, parameter propagation, user chains, backend-specific, vendor-specific, preview/create consistency, state/diagnostics, output status) are well-defined and appropriate.

### 3.2 Gaps in 01

**Gap A**: No requirement for `container_port == app_port` validation in health check assertions. If the product doesn't support sidecar/proxy, DryRun E2E should catch this mismatch.

**Gap B**: No requirement for `model_container_path` correctness verification. The E2E should verify that the host path is NOT passed directly to app args.

**Gap C**: The "last_error on failure" requirement should also specify `failure_reason_code` enumeration values (e.g., `container_exited`, `health_timeout`, `image_pull_failed`).

**Gap D**: No requirement for `--gpus` vs `CUDA_VISIBLE_DEVICES` consistency. For NVIDIA, both should match. If they don't, the E2E should catch it.

## 4. Review of 02-acceptance-criteria-and-parameter-matrix.md

### 4.1 Accuracy

The matrices are **accurate and comprehensive**. Each parameter has expected inputs, expected command output, and reverse assertions.

### 4.2 Minor Corrections

- **Section 4.2 SGLang model path**: The table says "HF model: usually path_type=directory" but SGLang can also load GGUF or safetensors single files. The E2E should be flexible about format.
- **Section 5.1 GGUF**: "volume: mount file or parent directory" — for GGUF, the file MUST be a single file mount. Mounting just the parent directory means `-m` can't resolve without correct relative path construction. The E2E should specifically verify `-m /models/model.gguf`, not `-m /models/`.
- **Section 6.1 Devices**: Line 315 says "默认不得作为 simple device 出现" — this is accurate but needs the E2E to actually extract Docker device specs and check for colons.

### 4.3 Additional Parameters Not Covered

These should be added:

| Backend | Missing Param | Priority |
|---------|--------------|----------|
| vLLM | `--dtype` (float16/bfloat16) | LOW |
| vLLM | `--quantization` (if user sets) | LOW |
| SGLang | `--disable-cuda-graph` (MetaX commonly enables) | MEDIUM |
| SGLang | `--attention-backend` (flashinfer/triton) | LOW |
| llama.cpp | `--mlock` (memory lock) | LOW |
| llama.cpp | `--numa` (NUMA config) | LOW |
| All | `startup_timeout_seconds` value verification | MEDIUM |
| All | health check uses `host_port` not `container_port` | HIGH |

## 5. Review of 03-implementation-steps-recommendation.md

### 5.1 Phase Assessment

| Phase | Feasibility | Risk | Recommended Priority |
|-------|------------|------|---------------------|
| 0 (classification) | ✅ Easy | Low | **First** ✅ |
| 1 (fix false pass) | ✅ Moderate | Medium | **Second** ✅ |
| 2 (parameter DryRun) | ✅ Easy | Low-Medium | **Third** ✅ |
| 3 (clone template) | ✅ Easy | Low-Medium | **Fourth** ✅ |
| 4 (deployment visibility) | ✅ Easy | Low-Medium | **Fifth** ✅ |
| 5 (instance stop) | ⚠️ Needs runtime | Medium-High | **Sixth** ⚠️ |
| 6 (inference parser) | ✅ Mixed | Medium | **Seventh** |
| 7 (matrix verifier) | ⚠️ Depends on 1-6 | Medium | **Eighth** |
| 8 (legacy governance) | ✅ Documentation | Low | **Last** |

### 5.2 Recommendation Approval

The Phase 0→1→2→3 sequence is **correct and should be approved for first batch**. Phase 5 (instance stop) requires a real running container and should be gated behind Phase 2 passing first.

## 6. Claude's Additional Findings (Not in 00/01/02/03)

### NF1: Missing `set -euo pipefail` enforcement

`scripts/e2e-model-runtime-wizard-nvidia-api.sh` has `set -euo pipefail` at line 2 but then `set -e` at line 142 (downgrading from `-euo pipefail`). This is likely unintentional — the script should maintain `set -euo pipefail` throughout.

### NF2: Hardcoded artifact paths may collide

`scripts/e2e-ui-persistence-runplan-selected.sh` uses `ARTIFACT_DIR=/tmp/lightai-ui-persistence-runplan-selected-final`. If two runs happen in parallel or close succession, artifacts collide. Should use timestamped or UUID-based directories.

### NF3: `e2e_run_default` return code not always checked

`e2e_run_default()` returns 0 on success (line 286). But looking at how it's called in wizard scripts (`scripts/e2e-model-runtime-wizard-nvidia-api.sh`), the call is:
```bash
e2e_run_default
```
without `|| { ... fail ... }`. The script continues even if `e2e_run_default` returns non-zero. This is a **false pass risk** — the function can fail internally but the outer script still exits 0.

### NF4: No artifact directory uniqueness enforcement

`model-runtime-common.sh` creates `$ARTIFACT_DIR` but doesn't enforce that it's unique. Multiple runs overwrite each other's artifacts. The directory should include a timestamp or run-id.

### NF5: `json_get` helper silently conflates missing/null/empty/parse-error

The `json_get` function uses `python3 -c '...' 2>/dev/null || echo ''`. When a key is missing, null, empty string, or the parser crashes, it returns the same empty result. The caller cannot distinguish:

- field does not exist (`KeyError`);
- field exists but value is `null`;
- field exists but value is `""`;
- Python parser error.

All four collapse to an empty string, and `[ -n "$(json_get ...)" ]` passes for the empty result because command substitution strips trailing newlines — an empty output becomes a truly empty string that `[ -n "" ]` correctly rejects. However, callers that don't check at all (e.g., just assign and continue) silently proceed with empty values. The real fix is a strict JSON helper that fails loudly on missing/null/parse-error, plus callers must explicitly opt-in when empty is acceptable.

### NF6: No DryRun-only mode in any E2E script

None of the scripts support a `DRY_RUN_ONLY=1` mode that stops after Docker command preview without launching a container. This makes parameter propagation testing unnecessarily expensive (must always run the full container lifecycle).

### NF7: `e2e_model_runtime_common.sh` wraps model path in `file://`

Looking at the wizard scripts, model roots are created with `file://` prefix hardcoded. This may not work on all environments and masks path validation.

## 7. Summary: Issue Classification

### CONFIRMED (by code inspection)

| # | Issue | Source Doc | Evidence |
|---|-------|-----------|----------|
| C1 | Port custom values not tested | 00 §2.1 | Scripts only set host_port |
| C2 | Positional model duplicate not checked | 00 §2.2 | No reverse assertion for `--model` |
| C3 | Matrix is runner not verifier | 00 §2.4 | No runplan assertions in matrix |
| C4 | common.sh false pass (test/stop) | 00 §2.5 | `e2e_instance_test` doesn't check summary; `e2e_stop_deployment` uses `|| true` |
| C5 | Instance stop not covered | 00 §2.6 | Only deployment-level stop tested |
| C6 | Inference parser no semantic check | 00 §2.7 | No content/reasoning/text assertion |
| C7 | Clone template not tested | 00 §2.8 | No clone→deploy→dry-run chain |
| C8 | GGUF semantics not strict | 00 §2.9 | `path_type=directory` for `.gguf` files |
| C9 | `set -euo pipefail` downgrade in wizard script | NF1 | Line 142 drops to `set -e` |
| C10 | `e2e_run_default` return not checked | NF3 | Outer script doesn't `|| fail` |
| C11 | `json_get` masks missing keys | NF5 | Returns empty string that `-n` treats as true |
| C12 | No DryRun-only mode | NF6 | Every test must launch container |

### SUSPECTED (needs runtime verification)

| # | Issue | Source | How to Confirm |
|---|-------|--------|---------------|
| S1 | `file://` prefix breaks on some platforms | NF7 | Run on fresh env without existing model roots |
| S2 | Matrix sub-scripts fail but matrix reports PASS | 00 §2.4 | Run matrix with intentional error in one sub-script |
| S3 | Cleanup `|| true` masks real errors in deployment/artifact deletion | NF3 | Run with read-only DB to trigger actual deletion errors |

### NEEDS RUNTIME TO CONFIRM

| # | Issue | Source | Required |
|---|-------|--------|----------|
| R1 | Instance stop actually returns 405 | 00 §2.6 | Real running container |
| R2 | Response parser misses llama.cpp thinking models | 00 §2.7 | Real llama.cpp with reasoning model |
| R3 | LLAMA_ARG_HOST warning in Docker logs | 03 §5.3 | Real llama.cpp container |

### NO LONGER APPLICABLE

| # | Issue | Reason |
|---|-------|--------|
| N1 | `deduplicateArgs` keeps first (00 §2.3) | Fixed in commit 015180c. Now keeps last. |
| N2 | CLI/snake_case name mismatch (00 §2.7 item) | Fixed in commit 015180c. Naming normalized. |
| N3 | GGUF directory accepted (00 §2.9) | Fixed in commit c63cef6. Backend rejects GGUF dir paths. |
| N4 | `/model-deployments` old API paths (00 §2.10) | Not found in any current script. Historical only. |

## 8. Implementation Plan (Phased)

### Phase 0: Script Classification & Documentation

**Goal**: Establish classification for all E2E scripts. No functional changes.

**Files to modify**: None (read-only audit documented in this plan)

**Script classification table** (confirmed by code review):

| Script | Category | GPU? | Container? | Assertions | Status |
|--------|----------|------|-----------|------------|--------|
| `e2e-ui-persistence-runplan-selected.sh` | smoke / dry-run | No | No | Weak | Needs Phase 1 |
| `e2e-model-runtime-wizard-nvidia-api.sh` | runtime smoke | Yes | Yes | Medium | Current baseline |
| `e2e-model-runtime-wizard-nvidia-vllm.sh` | runtime | Yes | Yes | Medium | Current baseline |
| `e2e-model-runtime-wizard-nvidia-sglang.sh` | runtime | Yes | Yes | Medium | Current baseline |
| `e2e-model-runtime-wizard-nvidia-llamacpp.sh` | runtime | Yes | Yes | Medium | Current baseline |
| `e2e-model-runtime-wizard-nvidia-matrix.sh` | matrix runner | Yes | Yes | Weak | Upgrade to verifier |
| `e2e-backend-runtime-nvidia-api.sh` | backend API smoke | Yes | Yes | Medium | Current baseline |
| `e2e-model-runtime-api.sh` | legacy | No | No | Weak | Legacy |
| `e2e-model-runtime-local.sh` | legacy/local | No | No | Weak | Legacy |
| `e2e-model-runtime-failed-instance-logs.sh` | failed-state | Yes | Yes | Good | Keep |
| `e2e/lib/model-runtime-common.sh` | helper | N/A | N/A | N/A | Needs Phase 1 |

**Risk**: None (documentation only)
**Verification**: N/A (no code changes)
**Run command**: Not applicable

---

### Phase 1: Fix False Pass & Add Assertion Helpers

**Goal**: Make existing scripts fail when they should. Add shared assertion library.

**Files to modify**:
- `scripts/e2e/lib/model-runtime-common.sh` — add assertion functions, fix `e2e_instance_test`, `e2e_stop_deployment`, `e2e_run_default`
- `scripts/e2e-model-runtime-wizard-nvidia-api.sh` — fix `set -euo pipefail` downgrade at line 142
- `scripts/e2e/lib/e2e-assert.sh` (NEW) — shared assertion library

**Changes**:

1. **Add `scripts/e2e/lib/e2e-assert.sh`** with:
   - `assert_eq`, `assert_nonempty`, `assert_contains`, `assert_not_contains`
   - `assert_http_ok` (check 2xx)
   - `assert_json_field_nonempty`
   - `assert_exactly_one_flag` (for command dedup checks)
   - `assert_no_flag` (reverse assertion)

2. **Fix `e2e_instance_test`** (common.sh line 230-235):
   - After saving response, assert `parsed summary` is non-empty
   - If empty but raw_response has data → FAIL
   - Save `raw_response` as separate artifact

3. **Fix `e2e_stop_deployment`** (common.sh line 247-251):
   - Remove `|| true` from the formal stop call
   - Assert HTTP status is 2xx
   - Add separate `e2e_instance_stop()` that tests instance-level stop
   - Keep `|| true` only on `e2e_cleanup()`

4. **Fix `e2e_run_default`** (common.sh line 268-287):
   - Keep `set +e` for stage-by-stage error handling
   - But ensure the RETURN CODE propagates correctly:
   ```bash
   e2e_run_default || { log "E2E FAILED"; exit 1; }
   ```

5. **Fix wizard script `set -e` downgrade**:
   - Change line 142 from `set -e` back to full `set -euo pipefail`

**Risk**: Medium. Scripts that currently pass may start failing, revealing actual product issues or script bugs.
**GPU/Container needed**: No (these are assertion/safety changes)
**Run command**:
```bash
bash -n scripts/e2e/lib/e2e-assert.sh scripts/e2e/lib/model-runtime-common.sh
bash -n scripts/e2e-model-runtime-wizard-nvidia-api.sh
```
**Acceptance**: All scripts pass `bash -n`. Existing wizard script still passes `bash -n`. Assertion helpers testable in isolation.

---

### Phase 2: New DryRun Parameter Source Audit E2E

**Goal**: Create a new script that tests parameter propagation WITHOUT launching containers.

**Files to create**:
- `scripts/e2e-runplan-parameter-source-audit.sh` (NEW)

**Files to modify**: None (new script only)

**Coverage** (all DryRun only, no container start):

1. **vLLM custom ports**: Create deployment with host_port=8111, container_port=8022, app_port=8022 → DryRun → assert `--port 8022`, `-p 8111:8022`, no `--port 8000`
2. **vLLM served_model_name**: Set custom name → assert `--served-model-name <custom>`
3. **vLLM positional model**: Assert no `--model` flag in args
4. **vLLM gpu_memory_utilization**: Set 0.85 → assert in args
5. **SGLang custom port**: Set app_port=30111 → assert `--port 30111`, no `--port 30000`
6. **SGLang model-path**: Assert `--model-path` uses container path not host path
7. **llama.cpp GGUF**: Assert `-m` points to `.gguf` file, format=gguf
8. **llama.cpp custom port**: Set app_port=18082 → assert `--port 18082`
9. **Docker devices classification**: MetaX template → assert `/dev/dri` as `--device`, not volume-style
10. **Vendor visible env**: NVIDIA → `CUDA_VISIBLE_DEVICES`, MetaX → `MACA_VISIBLE_DEVICE`
11. **Dedup check**: Assert each flag appears exactly once
12. **Reverse assertions**: No default values leaking through

**Risk**: Low-Medium. Creates/deletes test objects via API, no container starts.
**GPU/Container needed**: No
**Run command**:
```bash
SERVER_URL=http://127.0.0.1:18080 bash scripts/e2e-runplan-parameter-source-audit.sh
```
**Acceptance**: All 12 parameter assertions pass. Artifacts saved to timestamped directory. Returns PASS/FAIL correctly.

---

### Phase 3: New Clone Template Parameter Persistence E2E

**Goal**: Verify that clone template modifications are saved and used in deployments.

**Files to create**:
- `scripts/e2e-clone-template-parameter-persistence.sh` (NEW)

**Files to modify**: None

**Coverage**:
1. GET builtin runtime (e.g., vLLM NVIDIA Docker)
2. Clone with modified payload (name, image, env, devices, ports, args)
3. GET clone detail → assert modified values present
4. Create deployment using cloned runtime
5. DryRun → assert command uses clone values
6. Assert original builtin unchanged
7. Assert cloned runtime is user-managed (`is_editable=true`)
8. Assert cloned runtime appears in deployment runtime selection

**Risk**: Low-Medium. API/DB operations only, no container.
**GPU/Container needed**: No
**Run command**:
```bash
SERVER_URL=http://127.0.0.1:18080 bash scripts/e2e-clone-template-parameter-persistence.sh
```
**Acceptance**: All clone assertions pass. Clone modifications survive round-trip.

---

### Phase 4: New Deployment Visibility E2E

**Goal**: Prevent deployment list regression (SELECT/Scan column mismatch was caught in commit 8d68afe but has no E2E guard).

**Files to create**:
- `scripts/e2e-deployment-visibility-selected.sh` (NEW)

**Files to modify**: None

**Coverage**:
1. Create deployment → assert list contains id
2. Get detail → assert fields non-empty (status, display_name, backend_runtime_id)
3. DryRun → assert runplan exists
4. If runtime available: start → assert list STILL contains id
5. Assert status updated after start
6. Assert `active_instance_id` and `current_run_plan_id` present in detail

**Risk**: Low (API-only) to Medium (if starting a container).
**GPU/Container needed**: Optional (can run API-only version without container)
**Run command**:
```bash
# API-only mode:
LIGHTAI_E2E_SKIP_REAL_RUNTIME=1 SERVER_URL=http://127.0.0.1:18080 bash scripts/e2e-deployment-visibility-selected.sh
```
**Acceptance**: List does not go empty after create/start. Detail has all required fields.

---

### Phase 5: New Instance Stop E2E (gated)

**Pre-condition**: Phase 2 must pass first. Requires real running container.

**Goal**: Verify instance-level stop works (caught HTTP 405 regression in commit 3006fc1).

**Files to create**:
- `scripts/e2e-instance-stop-selected.sh` (NEW)

**Files to modify**: None

**Coverage**:
1. Start llama.cpp instance (selected, GGUF)
2. Wait for healthy
3. Call instance stop → assert NOT 405
4. Assert instance state → stopped
5. Assert container stopped/removed
6. Assert deployment state synced
7. Assert GPU lease released

**Risk**: Medium-High. Requires real Docker + GPU + GGUF model.
**GPU/Container needed**: YES
**Run command**:
```bash
# Only if GPU + model available:
bash scripts/e2e-instance-stop-selected.sh
# Otherwise:
echo "SKIPPED_ENV: no GPU/model available" && exit 0
```
**Acceptance**: Instance stop returns 2xx. All state transitions verified.

---

### Phase 6: New Inference Response Parser E2E

**Goal**: Ensure response parser handles all formats (caught empty summary regression in commit 8d68afe).

**Files to create**:
- `scripts/e2e-inference-response-parser-selected.sh` (NEW)

**Files to modify**: None

**Coverage** (fixture-based, no container needed for most cases):
1. Test with fixture JSON: `choices[0].message.content = "hello"` → summary non-empty
2. Test with fixture JSON: `choices[0].message.reasoning_content = "think..."` → summary non-empty, not fail
3. Test with fixture JSON: `choices[0].text = "hello"` → summary non-empty
4. Test with fixture JSON: `choices[0].delta.content = "hello"` → summary non-empty
5. Test with fixture JSON: top-level `response = "hello"` → summary non-empty
6. Test with fixture JSON: `choices[0].message.content = ""` + no other text → summary empty → should FAIL
7. If real llama.cpp available: test with real inference → save raw_response → verify parser

**Risk**: Low (fixture-based) to Medium (real inference).
**GPU/Container needed**: No (fixture mode) / Yes (real inference mode)
**Run command**:
```bash
# Fixture mode (always available):
LIGHTAI_E2E_FIXTURE_ONLY=1 bash scripts/e2e-inference-response-parser-selected.sh
```
**Acceptance**: All 6 fixture cases pass. Real inference case saves raw_response + parsed summary.

---

### Phase 7: Matrix Verifier Upgrade (deferred)

**Goal**: Convert matrix from runner to verifier with strong assertions per case.

**Risk**: Depends on Phases 1-6 completion. Deferred until earlier phases pass.
**Recommendation**: Do NOT start until Phase 2 and Phase 3 pass consistently.

---

### Phase 8: Legacy Script Governance (deferred)

**Goal**: Tag legacy scripts, migrate valuable assertions.

**Risk**: Low. Documentation and file moves.
**Recommendation**: Last phase, after all new scripts are stable.

---

## 9. Revised Execution Order

The environment has NVIDIA GPU, Docker, vLLM/SGLang/llama.cpp images, HF test models, and GGUF test models. The plan should use all available resources, not default to skipping real smoke.

### Phase execution sequence

```
Phase 0 (classify) ──► Phase 1 (false pass fix + assert selftest)
                           │
                           ▼
                      Phase 2 (DryRun parameter audit)
                           │
                           ▼
                      Phase 3 (clone template persistence)
                           │
                           ▼
                      Phase 4 (deployment visibility)
                           │
                    ┌──────┴──────┐
                    ▼              ▼
              Phase 5           Phase 6
         (instance stop)   (inference parser)
         [REAL DOCKER]     [fixture first, then real]
                    │              │
                    └──────┬───────┘
                           ▼
                      Phase 7 (matrix verifier)
                           │
                           ▼
                      Phase 8 (legacy governance)
```

### Phase characteristics

| Phase | Type | GPU/Container | Auto-advance? |
|-------|------|--------------|---------------|
| 0 | Classification | No | Yes |
| 1 | Safety + assert selftest | No | Yes |
| 2 | DryRun parameter audit | No | Yes |
| 3 | Clone template API | No | Yes |
| 4 | Deployment visibility API | No (API mode) | Yes |
| 5 | Instance stop | **YES** | Yes (short timeout, self-cleaning) |
| 6-fixture | Parser fixture | No | Yes |
| 6-real | Real inference test | **YES** | Yes (short timeout) |
| 7 | Matrix verifier | Mixed | Stop if sub-item failures unclear |
| 8 | Legacy governance | No | Yes |

## 10. Auto-Advance Principle

You will auto-advance from one Phase to the next without waiting for human approval, EXCEPT when:

1. **Destructive operation**: deleting user data, dropping tables, removing model roots, `docker system prune`
2. **Wide architectural refactor**: changes spanning >5 files with structural impact
3. **External dependency**: needs external account, external network, external hardware not in current env
4. **Real GPU smoke with unbounded timeout**: if a container fails to start and timeout is >5 minutes with no clear failure reason, stop and report
5. **Cannot determine fix direction**: if a test failure has multiple plausible causes and no clear root cause after 3 attempts

In all other cases — including discovering product bugs — you will:
- Record the actual behavior
- Record the expected behavior
- Locate the root cause in product code
- Fix the product code
- Re-run the verification
- Document the fix
- Continue to the next Phase

## 11. Product Bug vs Test Bug Policy

If a new or updated E2E exposes a product code defect, **do not**:
- Weaken the assertion
- Skip the test case
- Mark it `SKIPPED_ENV` when the environment is available
- Add `|| true` to silence the failure

Instead:
1. Record actual behavior and expected behavior
2. Locate the root cause in product code
3. Fix the product code
4. Re-run the E2E to confirm fix
5. Document the fix in commit message and relevant report

The E2E is the guard — not the thing to be guarded against.

## 12. Artifact Directory Uniqueness

Every script must use a unique artifact directory. Format:

```
${ARTIFACT_BASE:-docs/reports/model-runtime-node-wizard/e2e-artifacts}/${SCRIPT_NAME}-$(date +%Y%m%d-%H%M%S)-$$-${RANDOM}/
```

At minimum each script must save:
- `request-payloads/` — all API request bodies
- `responses/` — all API response bodies with HTTP status
- `runplan.json` — extracted RunPlan (if DryRun or start)
- `docker-command-preview.txt` — equivalent Docker command
- `assertion-report.txt` — pass/fail per assertion
- `cleanup-result.txt` — cleanup success/failure
- `summary.txt` — final PASS/FAIL/SKIPPED_ENV/WEAK_PASS

Real container smoke additionally saves:
- `docker-inspect.json` — Docker inspect output
- `docker-logs-tail.txt` — last 200 lines of container logs
- `lightai-logs.txt` — relevant server/agent log excerpts

## 13. Confirmation

This document was generated in **read-only mode**. No product code, E2E scripts, server, agent, or containers were modified or executed. No commits were made. No pushes were performed.

All findings are based on code/documentation review and static analysis only. Items marked as "CONFIRMED" have been verified against the actual source code. Items marked as "SUSPECTED" or "NEEDS RUNTIME" require live testing to verify.
