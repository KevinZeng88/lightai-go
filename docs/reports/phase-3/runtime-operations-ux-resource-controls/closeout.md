# Runtime Operations UX & Resource Controls — Final Closeout

> Date: 2026-06-23
> Status: COMPLETE
> Scope: Phase 1 (RunPlan lint) + Phase 2a (resource_controls) + Phase 3 (log classifier) + Phase 4 (instance auto-refresh) + Phase 5 (JsonViewer) + Phase 6a (HealthCheckEditor)

## 1. 背景和目标

本批解决 LightAI Go 运行时操作和诊断展示的系统性问题，包括：

- RunPlan 参数冲突检测（lint）
- 后端资源控制建模（resource_controls）
- 运行时日志分类（log classifier）
- 模型实例自动刷新
- 诊断 JSON 可读展示（JsonViewer）
- 健康检查结构化编辑（HealthCheckEditor）

目标是建立可重复机制，而非逐条修补。

## 2. Commit Timeline

| Commit | Message | Batch |
|--------|---------|-------|
| `a07c79f` | runtime: add backend runtime diagnostics controls | A |
| `a5bc9ad` | runtime: wire resource controls and log diagnostics | A.1 |
| `770fdb6` | web: improve runtime diagnostics and refresh ux | B |
| `499af51` | web: wire instance status polling | B.1 |
| `0bb9301` | ops: reduce logging noise and add recovery middleware | Logging (separate) |

## 3. Files Changed Summary

### Backend (Batch A + A.1)

| File | Change |
|------|--------|
| `internal/server/runplan/lint.go` | RunPlan lint engine (pre-normalization + final) |
| `internal/server/runplan/lint_test.go` | Lint tests |
| `internal/server/runplan/resource_controls.go` | Resource controls model + validation + arg builder |
| `internal/server/runplan/resource_controls_test.go` | Resource controls tests |
| `internal/server/runplan/log_classifier.go` | Runtime log classifier (6 built-in rules) |
| `internal/server/runplan/log_classifier_test.go` | Log classifier fixture tests |
| `internal/server/runplan/testdata/runtime-logs/*.log` | 4 fixture log files |
| `internal/server/runplan/resolver.go` | VersionInfo.VendorOptionsJSON + Layer 4b resource_controls integration |
| `internal/server/runplan/resolver_test.go` | 7 resource_controls resolver integration tests |
| `internal/server/api/deployment_lifecycle_handlers.go` | Lint in dry-run response + bvVendorOptions + classified_log_events in logs API |
| `internal/server/api/node_run_plan_logs_test.go` | Log classifier API test |
| `configs/backend-catalog/versions/vllm/vllm-v0.23.0.yaml` | vendor_options.resource_controls |
| `configs/backend-catalog/versions/sglang/sglang-v0.5.13.post1.yaml` | vendor_options.resource_controls |
| `configs/backend-catalog/versions/sglang/sglang-v0.5.12.post1.yaml` | vendor_options.resource_controls |
| `configs/backend-catalog/versions/sglang/sglang-0.4.6-compatible.yaml` | vendor_options.resource_controls |
| `configs/backend-catalog/versions/llamacpp/llamacpp-b9700.yaml` | vendor_options.resource_controls |

### Frontend (Batch B + B.1)

| File | Change |
|------|--------|
| `web/src/components/common/JsonViewer.vue` | New: scroll/fullscreen/copy/download/search/wrap/malformed fallback |
| `web/src/components/common/HealthCheckEditor.vue` | New: structured health check fields + raw JSON advanced area |
| `web/src/composables/useInstanceStatusPolling.ts` | New: state-aware polling (transitional=3s, stable=15s) |
| `web/src/pages/ModelInstancesPage.vue` | useInstanceStatusPolling; classified_log_events display; last refreshed |
| `web/src/pages/ModelDeploymentsPage.vue` | JsonViewer for dry-run JSON |
| `web/src/pages/RunnerConfigsPage.vue` | JsonViewer for diagnostics; HealthCheckEditor for health config |
| `web/src/locales/en-US.ts` | New i18n keys (healthCheck, lastRefreshed, staleData, classifiedEvents) |
| `web/src/locales/zh-CN.ts` | New i18n keys |

### Documentation

| File | Change |
|------|--------|
| `docs/design/runtime-operations-ux-resource-controls.md` | Design doc |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/00-known-issues-and-evidence.md` | Known issues |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/01-implementation-plan.md` | Original plan |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/02-verification-and-acceptance-plan.md` | Verification plan |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/03-claude-review-prompt.md` | Review prompt |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/04-claude-review-report.md` | Review report |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/05-revised-execution-plan.md` | Revised plan |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/06-batch-execution-plan.md` | Batch execution plan |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/batch-a-backend-runtime-diagnostics-closeout.md` | Batch A closeout |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/batch-a1-repair-closeout.md` | Batch A.1 closeout |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/batch-b-frontend-runtime-ux-closeout.md` | Batch B closeout |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/closeout.md` | This file |

## 4. Batch Summaries

### Batch A: Backend Runtime Diagnostics & Resource Controls

- RunPlan lint engine with two-stage design (pre-normalization + final)
- resource_controls model with vendor_options_json storage
- Runtime log classifier with 6 built-in rules
- Lint results embedded in dry-run response

### Batch A.1: Repair

- resource_controls wired into RunPlan resolver chain (Layer 4b)
- vLLM gpu_memory_fraction → --gpu-memory-utilization
- SGLang gpu_memory_fraction → --mem-fraction-static
- SGLang attention_backend → --attention-backend
- llama.cpp gpu_memory_fraction.supported=false → no fake arg
- Log classifier wired into GET /api/v1/node-run-plans/{id}/logs
- classified_log_events returned in logs response

### Batch B: Frontend Runtime UX & Diagnostics

- JsonViewer component (scroll/fullscreen/copy/download/search/wrap)
- HealthCheckEditor component (structured fields + raw JSON)
- JsonViewer integrated into ModelDeploymentsPage + RunnerConfigsPage
- HealthCheckEditor integrated into RunnerConfigsPage
- classified_log_events displayed in ModelInstancesPage logs drawer
- i18n keys added for en-US and zh-CN

### Batch B.1: Repair

- useInstanceStatusPolling wired into ModelInstancesPage
- transitional states → 3s polling; stable states → 15s polling
- Self-managed timer with dynamic interval via watch
- Closeout commit metadata corrected

## 5. Issue-by-Issue Status

| Issue | Status | Evidence |
|-------|--------|----------|
| SGLang torchao SyntaxWarning not classified | FIXED | `log_classifier.go` rule `sglang.torchao.syntax_warning`, fixture test |
| SGLang attention backend default not classified | FIXED | `log_classifier.go` rule `sglang.attention_backend.default`, fixture test |
| Model instance page status no auto-refresh | FIXED | `useInstanceStatusPolling.ts` wired into `ModelInstancesPage.vue` |
| Advanced diagnostic JSON not readable | FIXED | `JsonViewer.vue` integrated into RunnerConfigsPage + ModelDeploymentsPage |
| Health check JSON boundaries unclear | FIXED | `HealthCheckEditor.vue` structured editing in RunnerConfigsPage |
| llama.cpp LLAMA_ARG_HOST env/CLI conflict | FIXED | Lint detects env/CLI conflict; log classifier classifies runtime warning |
| GPU memory/resource controls unclear | FIXED | resource_controls model with per-backend definitions in vendor_options_json |
| Configuration pages inconsistent layout | PARTIAL | JsonViewer + HealthCheckEditor done; complete ConfigEditorLayout deferred |
| Shared GPU multiple containers on one GPU | DOCUMENTED_BLOCKER | gpu_leases unique index enforces exclusive; shared needs schema change |

## 6. Tests Run and Results

```
$ go test ./...
ok  	lightai-go/internal/server/runplan	0.004s
ok  	lightai-go/internal/server/api	6.662s
(all packages pass)

$ go build ./...
(exit 0)

$ gofmt -l internal/ cmd/
(no output)

$ cd web && npm run build
✓ built in 3.34s

$ cd web && npm test
0 FAIL

$ git diff --check
(no output)
```

## 7. Deferred / Blockers

| Item | Status | Trigger |
|------|--------|---------|
| Shared GPU admission / budget-based GPU lease | DOCUMENTED_BLOCKER | User requests shared GPU; requires gpu_leases index change + schema |
| Complete ConfigEditorLayout | deferred | Separate project |
| Status-summary API | conditional | Current list endpoint sufficient for <50 instances |
| vitest introduction | deferred | Vue composable tests require it |
| llama.cpp VRAM estimator | deferred | User requests VRAM estimation |
| BackendsPage HealthCheckEditor | deferred | Risk of breaking raw JSON editing |
| Pre-normalization lint hook in resolver | v1 limitation | Requires exposing pre-dedup args from buildArgs() |
| Env source tracking 精细化 | v1 limitation | Requires layer metadata in buildEnv() |
| Lint error does not change dry-run valid | diagnostic-first strategy | Lint errors merged into warnings array |

## 8. Known Limitations

| Limitation | Impact | Mitigation |
|------------|--------|------------|
| Env source tracking simplified (all "platform") | User-provided env conflict may downgrade from error to warning | Acceptable for v1; worst case is warning instead of error |
| Pre-normalization lint not wired | Duplicate flag detection before dedup not in real pipeline | Final lint still catches env/CLI conflict and high-risk flags |
| Lint errors not blocking dry-run | dry-run valid=true even with lint errors | Lint findings visible in response; user can review before start |
| useInstanceStatusPolling interval not reactive to per-row state | Interval is based on aggregate states of all items | Acceptable; transitional states trigger fast polling correctly |

## 9. Out-of-Scope Items

| Item | Handled? |
|------|----------|
| Logging Noise & Observability Cleanup | Handled separately in commit `0bb9301`, not part of this closeout scope |
| VERSION file | Not modified in this batch |
| .mimocode/skills/ | Not part of this batch |
| DB schema changes | Not needed; vendor_options_json used for resource_controls |
| gpu_leases index changes | Not done; shared GPU is DOCUMENTED_BLOCKER |

## 10. Final Git Status

```
 M VERSION
?? .mimocode/skills/
```

- `VERSION`: pre-existing modification, not part of this batch
- `.mimocode/skills/`: MiMoCode internal directory, not tracked

## 11. Commit and Push

- **commit id**: (see git log below)
- **push result**: (see push output below)
