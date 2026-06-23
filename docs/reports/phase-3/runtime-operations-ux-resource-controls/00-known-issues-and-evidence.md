# Runtime Operations UX & Resource Controls — Known Issues and Evidence

> Date: 2026-06-23  
> Status: Known issue inventory for review. Do not implement from this file alone.

## 1. Scope

This document records observed runtime operations and UX problems that should be addressed as one coherent batch, not as isolated fixes.

Batch name:

```text
Runtime Operations UX & Resource Controls
```

## 2. Known issues

### ISSUE-001: SGLang dependency warning is not classified

Observed log:

```text
/usr/local/lib/python3.12/dist-packages/torchao/quantization/quant_api.py:1731: SyntaxWarning: invalid escape sequence '\.'
  """Configuration class for applying different quantization configs to modules or parameters based on their fully qualified names (FQNs).
```

Initial classification:

- Backend: SGLang
- Severity: `noise` or `advisory`
- Category: dependency_warning
- Runtime impact: should not mark instance failed
- Required action: classify and display as non-fatal diagnostic event

Risk if unfixed:

- Operators must manually inspect raw Docker logs.
- Non-fatal warnings may be confused with runtime failure.

### ISSUE-002: SGLang attention backend default message is not classified

Observed log:

```text
[2026-06-23 06:01:40] Attention backend not specified. Use flashinfer backend by default.
```

Initial classification:

- Backend: SGLang
- Severity: `advisory`
- Category: default_selection
- Runtime impact: should not mark instance failed
- Suggested UI message: SGLang selected default attention backend. If flashinfer-related failures occur, try another supported backend.

Required design:

- Add runtime log classifier rule.
- Add optional advanced SGLang parameter `attention_backend`, default `auto`.

### ISSUE-003: Model instance page status does not auto-refresh

Observed behavior:

- Model instance page requires manual refresh to see status transitions.

Required design:

- Add state-sensitive frontend polling.
- Avoid immediate WebSocket complexity.
- Transitional states should refresh faster than stable states.
- Document hidden/inactive tabs should pause or slow down polling.

Risk if unfixed:

- Operator may think instance is stuck or stale.
- Runtime lifecycle improvements are not visible without manual refresh.

### ISSUE-004: Advanced diagnostic JSON is complete when copied but not readable in place

Observed behavior:

- "Advanced Diagnostic JSON" copy output is complete.
- Current page area cannot show full content clearly.

Required design:

- Use a shared JsonViewer component.
- Support scroll, fullscreen, copy, download, search, wrap toggle.
- Reuse for all diagnostic JSON locations.

Risk if unfixed:

- Operators must copy JSON elsewhere to read it.
- Long diagnostics degrade page layout.

### ISSUE-005: Health check JSON and advanced diagnostic JSON boundaries are unclear

Observed behavior:

- Runtime configuration page exposes "Health Check JSON" and "Advanced Diagnostic JSON" without a clear distinction between user configuration and system-generated evidence.

Required design:

- User-configurable health-check fields:
  - path
  - method
  - timeout_seconds
  - interval_seconds
  - expected_status
  - expected_body_contains
  - readiness_grace_seconds
- Generated diagnostics should be read-only:
  - health result JSON
  - advanced diagnostic JSON
  - RunPlan JSON
  - Docker inspect JSON
  - preflight evidence JSON

Risk if unfixed:

- Users may edit diagnostic JSON thinking it changes runtime behavior.
- Configuration form becomes hard to understand and maintain.

### ISSUE-006: Configuration pages are not consistent with "Copy as user configuration" layout

Observed concern:

- Runtime configuration and model deployment pages need a clearer layout similar to "Copy as user configuration".

Required design:

- Introduce `ConfigEditorLayout`.
- Use structured common fields.
- Move advanced JSON into read-only or expert-mode areas.
- Show RunPlan preview, Docker command preview, lint results, and diff from base.

Risk if unfixed:

- Users cannot see all meaningful runtime knobs at once.
- Editing remains fragmented and error-prone.

### ISSUE-007: llama.cpp env/CLI parameter conflict

Observed log:

```text
warn: LLAMA_ARG_HOST environment variable is set, but will be overwritten by command line argument --host
```

Root cause category:

- RunPlan argument conflict.
- Same logical parameter represented by both env and CLI.

Required design:

- RunPlan lint must detect env/CLI conflicts.
- Platform-owned serving args should not be duplicated in env.
- For llama.cpp:
  - `host`: `--host` and `LLAMA_ARG_HOST`
  - `port`: `--port` and `LLAMA_ARG_PORT`
- Conflict policy: reject by default.

Risk if unfixed:

- Generated command contains redundant or conflicting settings.
- Operators must interpret backend warnings manually.
- Future config templates can accidentally reintroduce the same issue.

### ISSUE-008: GPU memory/resource controls are unclear

Observed concern:

- UI does not clearly expose GPU memory control settings.
- It is unclear whether one GPU can run multiple Docker containers.

Required design:

- Distinguish GPU visibility from memory isolation.
- Add backend-specific resource controls:
  - vLLM: memory fraction → `--gpu-memory-utilization`
  - SGLang: memory fraction → `--mem-fraction-static`
  - llama.cpp: no fake memory fraction; use offload/context/batch/KV cache controls
- Add shared-GPU admission rules.

Risk if unfixed:

- Users may overcommit GPU memory without warning.
- Platform may incorrectly suggest Docker GPU binding provides VRAM isolation.
- llama.cpp may expose misleading controls.

## 3. Evidence fixtures to preserve in tests

Create fixture files under the implementation's chosen testdata path, for example:

```text
internal/server/runplan/testdata/runtime-logs/sglang-torchao-syntax-warning.log
internal/server/runplan/testdata/runtime-logs/sglang-attention-backend-default.log
internal/server/runplan/testdata/runtime-logs/llamacpp-env-host-overwritten.log
```

Fixture contents:

```text
# sglang-torchao-syntax-warning.log
/usr/local/lib/python3.12/dist-packages/torchao/quantization/quant_api.py:1731: SyntaxWarning: invalid escape sequence '\.'
  """Configuration class for applying different quantization configs to modules or parameters based on their fully qualified names (FQNs).
```

```text
# sglang-attention-backend-default.log
[2026-06-23 06:01:40] Attention backend not specified. Use flashinfer backend by default.
```

```text
# llamacpp-env-host-overwritten.log
warn: LLAMA_ARG_HOST environment variable is set, but will be overwritten by command line argument --host
```

## 4. Out of scope for the first implementation batch

The following should not be implemented unless the review finds they are necessary:

- full arg abstraction layer;
- full RuntimeRequirements structure;
- WebSocket-based status updates;
- mandatory real GPU E2E;
- llama.cpp VRAM estimator;
- MIG/vGPU/HAMi integration;
- DB-based log rule editor;
- full rewrite of every configuration page.

## 5. Success definition

This batch is successful when:

- every known issue above is fixed, classified, or documented as a blocker;
- observed log samples are fixture-tested;
- parameter conflicts are detected before container start;
- shared-GPU behavior is explicit and testable;
- JSON diagnostics are readable in the UI;
- model instance statuses update automatically;
- configuration pages separate user-editable fields from generated diagnostics.
