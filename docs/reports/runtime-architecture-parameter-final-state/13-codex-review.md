# Codex Review — Runtime Architecture and Parameter Final-State Docs

## 1. Review Scope

This review covers the documentation package in:

```text
docs/reports/runtime-architecture-parameter-final-state/
```

I read the required final-state documents, `manifest.json`, the root project documentation required by `AGENTS.md`, and current implementation paths in `internal/server`, `internal/agent`, `web/src`, `configs/config-registry`, and `configs/backend-catalog`. The requested historical files under `docs/reports/phase-3/runtime-architecture-and-parameter-current-gap-review.md` and `docs/reports/phase-3/runtime-architecture-and-parameter-repair-plan.md` are not present in this checkout.

No functional code was changed.

## 2. Overall Verdict

ACCEPT_WITH_FIXES

The document set is directionally strong enough to define the final architecture and prevent local UI-only fixes from being mistaken as the goal. It clearly states Runtime architecture convergence, parameter ownership, copy-on-create, layered presentation, RunPlan final-state authority, and API-first automation as acceptance requirements.

Claude should not AUTORUN from these docs without a short documentation tightening pass, because several requirements are still too broad or under-specified against current code reality. The highest-risk gaps are the migration path from current `ConfigSet`/`ConfigEdit` structures to final `ParameterDefinition`/`ParameterOverride`, the exact RunPlan source-map shape, and the contradiction between "do not keep old compatibility" and the current snapshot/catalog behavior that still needs a deliberate rebuild or transition boundary.

## 3. Executive Summary

The package correctly identifies the main final-state problem: LightAI Go currently has runtime config, parameter schema, parameter values, UI editing, and RunPlan synthesis spread across `config_set_json`, `ConfigEditView`, semantic normalization, deployment snapshots, preview handlers, and the runplan resolver. The docs also correctly emphasize that automatic E2E is acceptance evidence, not the product goal.

Current code reality still differs materially from the proposed final state. `ResolvedRunPlan` has no `parameter_source_map`; `runplan.ParameterDef` is a small CLI-oriented struct rather than a full single-owner schema; overrides are still encoded through `config_set_json`, `config_overrides_json`, and `ParameterValue` slices; and Docker options are read from `launcher.docker_options.value` before becoming typed RunPlan fields. These are not blockers to the document package, but they are execution hazards if Claude treats the final contract as already represented by existing types.

The recommended decision is to accept the package as the governing direction, require the fixes in section 10 before Claude AUTORUN, and then execute in small commits with tests proving each boundary.

## 4. Critical Issues

### Issue 1 — Final ParameterDefinition model is specified, but the migration from current ConfigSet reality is not

Evidence: The final docs require a single schema owner and `ParameterDefinition` / `ParameterOverride` references. Current code uses `config_set_json` rows and structs such as `runplan.ParameterDef{Name, CliName, Alias, Type, Default, Required}` in `internal/server/runplan/resolver.go`, while `configSetParameterDefs()` and `configSetParameterValues()` derive definitions and values from ConfigSet items in `internal/server/api/configset_helpers.go`.

Impact: Claude may either over-refactor the data model in one pass or keep treating ConfigSet items as both schema and value, leaving the single-definition contract unclosed.

Required Fix: Add an explicit implementation decision: either introduce first-class parameter definition/override tables and APIs, or define a bounded interim representation where ConfigSet item schema is treated as the definition and lower layers store definition refs plus value/enabled only. The docs must name the files and DB shape expected for Phase 1.

### Issue 2 — RunPlan source map is required but not specified enough to implement consistently

Evidence: `internal/server/runplan/types.go` defines `ResolvedRunPlan` with image, args, env, Docker fields, health check, hashes, and audit refs, but no `parameter_source_map`, `source_chain`, or per-output source object. `deployment_lifecycle_handlers.go` currently builds lint `envSources` as `"platform"` with a comment that actual source tracking needs layer metadata.

Impact: Preview/source-map acceptance can pass superficially with ad hoc source labels while Agent Docker create remains unsourced or inconsistent.

Required Fix: Define the exact JSON schema for `parameter_source_map`, including keys for args/env/mounts/ports/devices/health_check/docker options, required source labels, source-chain ordering, and whether the map is stored inside `resolved_run_plans.plan_json` or a separate column/API field.

### Issue 3 — Docker option checked/enabled semantics are still ambiguous at final RunPlan boundary

Evidence: ConfigEdit now uses `launcher.docker_options.enabled_fields` metadata, but RunPlan input currently extracts `launcher.docker_options` through `configObject()` and unmarshals it into `runplan.DockerSpecInfo`. The typed resolver then copies fields such as `ShmSize`, `Devices`, and `GroupAdd` into `ResolvedRunPlan`.

Impact: If a Docker subfield has a value but is unchecked, final RunPlan may still include it unless a dedicated filter rule is implemented. This directly conflicts with "unchecked optional does not enter final args/spec" and makes UI checked state misleading.

Required Fix: Add a documented rule for `enabled_fields`: explicit `false` filters the subfield from final RunPlan, explicit `true` keeps it, and missing metadata must have an intentional rebuild/migration policy. The recommended implementation layer is the server RunPlan input construction layer before `DockerSpecInfo` is handed to the resolver.

### Issue 4 — Non-compatibility policy conflicts with current catalog/snapshot realities

Evidence: The docs state "不保留旧配置兼容逻辑" and DB rebuild is allowed. Current code and catalog still rely on existing ConfigSet shapes, materialized runtime YAML, and snapshots copied across BackendVersion, BackendRuntime, NodeBackendRuntime, and Deployment.

Impact: Claude may delete compatibility in a way that breaks seeded vLLM/SGLang/llama.cpp templates or existing tests without first establishing a fresh-DB rebuild boundary and acceptance data.

Required Fix: Add a concrete fresh-DB/rebuild policy: which DB/data directories may be deleted, which catalog seeds are canonical, which legacy request fields must be rejected, and which legacy internal fields can remain only as migration inputs during the same batch.

## 5. Missing or Weak Requirements

### Requirement Area — DB and API schema ownership

Current Weakness: The docs define final conceptual objects, but do not say whether they must become database tables, ConfigSet schema fields, or resolver-only structs.

Why It Matters: Single owner and single definition are not enforceable without an identity boundary.

Suggested Fix: Add a table/API matrix for `ParameterDefinition`, `ParameterOverride`, `ParameterValue`, `ResolvedParameter`, and `ParameterSourceMap`, with owner identity, persistence location, unique key, and JSON/API representation.

### Requirement Area — Copy-on-create acceptance

Current Weakness: The documents require copy-on-create, but test requirements do not spell out exact mutation scenarios and DB assertions per layer.

Why It Matters: Current deployment creation copies NBR `config_set_json`, and RunPlan claims snapshot use, but template sync and patch paths can still reintroduce live upstream reads.

Suggested Fix: Add acceptance cases: mutate BackendVersion after BackendRuntime creation, mutate BackendRuntime after NBR creation, mutate NBR after Deployment creation, mutate Deployment after RunPlan creation, and assert no reverse pollution.

### Requirement Area — Preview API vs start path

Current Weakness: The docs require preview and Docker spec consistency, but do not require both paths to call the exact same builder function.

Why It Matters: Current preview and lifecycle handlers build resolver inputs separately.

Suggested Fix: Require a single `BuildDeploymentRunPlanInput` / `BuildResolvedRunPlan` server helper used by preview, preflight, dry-run, and start.

### Requirement Area — RuntimeRequirements and BackendCapabilityProfile storage

Current Weakness: The docs describe responsibilities and examples but not canonical storage, versioning, or validation.

Why It Matters: Current capabilities are partly `backend.capabilities` ConfigSet data and partly Go compatibility parsing. Without storage rules, capability and requirement data can drift.

Suggested Fix: Define canonical catalog keys, ConfigSet item codes, and DB/API projections for both objects, including vLLM/SGLang/llama.cpp minimum required fields.

### Requirement Area — UI test boundaries

Current Weakness: UI requirements list page behavior but do not say which assertions belong in Vitest/component tests versus API-first E2E.

Why It Matters: Claude may write broad Playwright/UI coverage instead of cheap contract tests for schema/value/source behavior.

Suggested Fix: State that owner/schema/default/enabled/source-map semantics are Go/API tests first; shared ConfigEdit rendering and no value clearing are Vitest/component tests; Playwright should be limited to one or two smoke paths proving the integrated user journey.

## 6. Code-Reality Gaps

### File or Area — `internal/server/runplan/types.go`

Current Behavior: `ResolvedRunPlan` has final execution fields and audit refs, but no `parameter_source_map`.

Expected Final State: RunPlan preview and persisted plan include source for each final arg/env/mount/port/device/docker option/health check field.

Risk: Final-state source visibility cannot be verified from persisted plan JSON.

Suggested Requirement: Add a mandatory source-map field and tests asserting it is present in preview and persisted `resolved_run_plans.plan_json`.

### File or Area — `internal/server/api/configset_helpers.go`

Current Behavior: `configValue()` returns `value` or `default_value` without checking enabled, while `configSetParameterValues()` copies enabled into `ParameterValue` for CLI/env items only.

Expected Final State: Value/default/required/enabled semantics are explicit and target-specific.

Risk: Optional defaults or Docker values may enter final RunPlan through helper-specific shortcuts rather than resolver policy.

Suggested Requirement: Add target-specific extraction helpers with explicit behavior for enabled, required, defaults, and legacy missing metadata.

### File or Area — `internal/server/api/deployment_lifecycle_handlers.go`

Current Behavior: Deployment creation copies NBR config into Deployment, start builds resolver input from Deployment snapshot, but lifecycle and preview have separate input construction logic. Template sync paths can rebuild deployment config from source runtime.

Expected Final State: Deployment snapshot is the authority for deployment; preview/start share a single builder; template sync is explicit user action with override preservation and source-map evidence.

Risk: Preview/start divergence and accidental live-template contamination.

Suggested Requirement: Document and implement one shared builder, and add tests for template sync not being part of normal RunPlan generation.

### File or Area — `internal/server/runplan/resolver.go`

Current Behavior: Args processing handles NBR parameter values, deployment values, disabled tombstones, required checks, deduplication, and service args, but does not produce source chains.

Expected Final State: Resolver produces final rendered output plus source-chain metadata.

Risk: The resolver can be behaviorally correct while failing the observability/debugging contract.

Suggested Requirement: Make source-map generation part of resolver output, not a UI post-processing step.

### File or Area — `web/src/components/common/RuntimeParameterEditor.vue` and `web/src/utils/runtimeParameterViewModel.ts`

Current Behavior: Legacy/diagnostic editor and human field mappings still exist and encode page-private parameter mappings.

Expected Final State: Normal runtime/deployment flows use shared ConfigEdit or the final schema-driven API; UI does not own schema.

Risk: Claude may repair the wrong editor or keep duplicate parameter schema logic.

Suggested Requirement: Explicitly classify these files as diagnostic-only or remove/replace their production references during the UI batch.

### File or Area — `configs/config-registry/items.yaml`

Current Behavior: Several foundational launcher/runtime items are enabled by default, including Docker and runtime env objects.

Expected Final State: Default value is not user checked; optional defaults do not imply enabled; required/system-generated state is distinct from user override.

Risk: Seed defaults can keep reintroducing "all checked" semantics unless final docs separate effective/default/required/user-enabled in seed format.

Suggested Requirement: Define seed fields for `default_enabled`, `effective_required`, and `user_enabled`, or an equivalent canonical encoding.

## 7. Parameter Ownership and Copy-on-create Review

The docs are strong on the conceptual contract:

1. Single owner is stated repeatedly and correctly.
2. Single schema definition is stated clearly.
3. Override must reference the original owner/key or definition id.
4. Deployment must not redefine schema.
5. UI must not copy schema.
6. Copy-on-create is explicitly defined from BackendVersion/ModelArtifact through BackendRuntime, NodeBackendRuntime, Deployment, ResolvedRunPlan, and Instance.
7. RunPlan source map is required.

The main weakness is implementability. Current code has no first-class `ParameterDefinition` or `ParameterOverride` identity; ConfigSet items still combine schema, defaults, value, enabled, render metadata, and source. That means Claude needs a concrete bridge: either normalize ConfigSet into final objects before persistence, or treat ConfigSet as the catalog definition source while lower layers store only override refs.

The copy-on-create wording is good but needs stronger test scenarios. Each layer must have a test proving parent mutation does not affect existing child snapshots and child mutation does not affect parent. Clone must also prove it preserves checked/enabled scope without expanding it.

The source-map requirement is correct but underspecified. It must cover not only CLI args, but also env, mounts, ports, devices, Docker options such as `shm_size` and `group_add`, health checks, and system-generated GPU binding.

## 8. Directory and Documentation Hygiene

The topic directory is clean and self-contained. The index and manifest identify the evidence directory and expected generated `13-codex-review.md`. No current output was written to a historical phase directory.

Required historical files named by the review prompt were not found at:

```text
docs/reports/phase-3/runtime-architecture-and-parameter-current-gap-review.md
docs/reports/phase-3/runtime-architecture-and-parameter-repair-plan.md
```

The closeout template is useful but should be tightened to require a formal issue table for any unresolved item, matching the repository problem-closure policy. It currently has an "Open Issues" section, but should require status values `FIXED`, `DOCUMENTED_BLOCKER`, or `INVALID` and evidence for each.

## 9. Claude Execution Risk Review

Claude can understand the target, but AUTORUN risk is high unless the user first resolves the documentation gaps above.

Primary risks:

1. Over-broad rewrite: The docs describe a full final architecture, but Batch 1 through Batch 7 can touch DB, API, resolver, Agent, Web, catalog, and E2E in one run.
2. Wrong editor focus: The repo has shared ConfigEdit components and legacy RuntimeParameterEditor/HumanRuntimeParameterForm artifacts.
3. Preview/start divergence: Separate construction paths can let tests pass in preview while start still emits a different plan.
4. Source-map superficiality: Claude may add display-only source labels instead of resolver-owned persisted source chains.
5. Docker disabled semantics: Without a precise enabled-fields rule, unchecked Docker values can still enter HostConfig.
6. Evidence burden: The E2E evidence list is large; it needs staged evidence requirements or Claude may spend effort on automation before the core contract is testable.

## 10. Required Fixes Before Claude AUTORUN

1. Add a concrete parameter persistence decision: first-class DB/API objects vs bounded ConfigSet-backed interim model.
2. Add exact `parameter_source_map` schema and storage/API location.
3. Add exact Docker subfield enabled/disabled final RunPlan rule.
4. Add one shared RunPlan builder requirement for preview, preflight, dry-run, and start.
5. Add a test matrix assigning owner/default/enabled/source semantics to Go tests, web unit/component tests, and minimal Playwright smoke only.
6. Add a fresh-DB/rebuild and catalog seed policy for the non-compatibility strategy.
7. Amend closeout open-issue format to use repository-approved statuses.

## 11. Recommended Decision for ChatGPT/User

Accept the document package as the final-state direction, but do not let Claude start the full AUTORUN yet. First apply a documentation-only revision that resolves section 10. After that, execute in small commits:

1. Backend contract and DB/API shape.
2. ConfigSet-to-parameter bridge and copy-on-create tests.
3. Shared RunPlan builder and source-map resolver output.
4. UI shared editor cleanup.
5. API-first E2E and closeout.

The current docs are strong enough for design alignment, but not yet precise enough for safe unattended implementation.

## 12. Final Status

Review verdict: ACCEPT_WITH_FIXES

Review document path:

```text
docs/reports/runtime-architecture-parameter-final-state/13-codex-review.md
```

Files changed:

```text
docs/reports/runtime-architecture-parameter-final-state/13-codex-review.md
docs/reports/runtime-architecture-parameter-final-state/00-index.md
docs/reports/runtime-architecture-parameter-final-state/manifest.json
```

Commit id: recorded in the terminal completion output after commit.

Push result: recorded in the terminal completion output after push.

git status --short at review creation time:

```text
 M deploy/observability/grafana/provisioning/dashboards/dashboards.yaml
```

The Grafana file is outside this review scope and was not touched.
