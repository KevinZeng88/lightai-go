# Runtime Parameter Editing Design

> Date: 2026-06-23
> Status: Design Document — NOT implemented yet
> **No code changes were made. No database modifications. No commits.**

---

## Document List

| # | Document | Purpose |
|---|----------|---------|
| 0 | `00-current-state-analysis.md` | Current capabilities, gaps, existing design principles |
| 1 | `01-runtime-parameter-editing-design.md` | Core design: parameter types, layers, copy-on-create, UI |
| 2 | `02-implementation-plan.md` | Batch A-F implementation plan |
| 3 | `03-open-decisions.md` | Decisions requiring user confirmation |

---

## Recommended Reading Order

1. `00-current-state-analysis.md` — understand what exists today
2. `01-runtime-parameter-editing-design.md` — understand the design
3. `02-implementation-plan.md` — understand how to implement
4. `03-open-decisions.md` — understand what needs your confirmation

---

## Core Conclusions

1. **NBR is source of truth** — RunPlan must only read NBR snapshot, not BackendVersion/BackendRuntime at resolution time
2. **8 parameter types** — capability metadata, schema, values, env, args, container config, runtime requirements, deployment overrides
3. **Copy-on-create** — each layer deep-copies from parent at creation, independent after
4. **Enabled/disabled state** — every parameter can be disabled without removing it
5. **Backend-specific memory params** — vLLM (gpu-memory-utilization), SGLang (mem-fraction-static), llama.cpp (n-gpu-layers/ctx-size)
6. **Reusable UI component** — `RuntimeParameterEditor` for all parameter editing

---

## Questions Requiring Your Confirmation

1. Parameter value storage format (flat map vs structured array)?
2. NBR snapshot scope (values only vs schema + values)?
3. RunPlan resolver migration strategy (big bang vs gradual)?
4. Deployment parameter merge strategy (replace vs merge)?
5. Disabled parameter persistence (store disabled vs absent)?
6. DB rebuild timing (with Batch B vs after Batch F)?

See `03-open-decisions.md` for details.

---

## Review Checklist (Must Confirm Before Implementation)

- [ ] **Parameter JSON fields use structured arrays**: Default `[]`, not `{}`; all parameter_schema_json, parameter_values_json, disabled_parameters_json, parameter_defaults_json
- [ ] **BackendRuntime stores schema + values snapshot**: BackendRuntime has both `parameter_schema_json` and `parameter_values_json`; NBR deep-copies from BR at creation
- [ ] **NBR schema + values snapshot**: NBR saves both parameter schema and values; RunPlan does NOT query BV/BR at resolution time
- [ ] **Deployment disabled override**: Explicit disabled overrides saved as structured tombstone array; "absent" ≠ "disabled"
- [ ] **ModelArtifact/ModelLocation boundary**: Only model-side defaults/requirements; never override NBR container config
- [ ] **Backend-specific memory params**: Schema-driven, not hardcoded UI; no unified gpu_memory_limit; llama.cpp uses gpu_layers/ctx/batch
- [ ] **Big bang resolver migration**: Rewrite resolver to only read NBR snapshot; no backward compatibility shim
- [ ] **DB rebuild timing**: Rebuild with Batch B (early) vs after Batch F (end)

**Status**: Design pending review. Implementation NOT approved.

---

## Important Notes

- **This is a design document only.** No code has been modified.
- **No database changes have been made.**
- **No commits have been made.**
- **The design is subject to change based on your feedback.**
- Implementation will begin only after you approve this design.
