# Codex Second Review — ConfigSetBundle Final-State Docs

## 1. Review Scope

This review covers the revised documentation package under:

```text
docs/reports/runtime-architecture-parameter-final-state/
```

I read the required documents: `00-index.md`, `13-codex-review.md`, `14-codex-review-fix-plan.md`, `03-final-runtime-domain-contract.md`, `04-final-parameter-contract.md`, `04a-parameter-ownership-and-layered-presentation-contract.md`, `05a-configset-bundle-composition-and-presentation-contract.md`, `06-runplan-and-preflight-contract.md`, `07-ui-and-api-contract.md`, `08-api-first-e2e-and-automation-requirements.md`, `09-implementation-plan.md`, `10-claude-execution-prompt.md`, `11-final-closeout-template.md`, and `manifest.json`.

I also ran a targeted code-reality check for the old paths named by the docs: `config_set_json`, `config_overrides_json`, `ConfigEdit`, `RuntimeParameterEditor`, `RunPlan`, `parameter_source_map`, `enabled_fields`, and shared builder naming. This review is documentation-only; no functional code was changed.

## 2. Overall Verdict

ACCEPT

The ConfigSetBundle revision closes the seven first-review documentation blockers at the design level. It makes a clear product/architecture decision instead of preserving an interim compatibility model: ConfigSet is now a final-domain concept, each layer owns a ConfigSetBundle, ConfigItems have explicit field tiers, copy-on-create copies readonly schema snapshots, and RunPlan generation is constrained to one shared builder over the DeploymentConfigBundle effective snapshot.

Claude can proceed only after user approval of the batch plan required by `10-claude-execution-prompt.md`. The current code still uses the old implementation shape, so this verdict accepts the documentation as execution guidance, not the implementation state.

## 3. Executive Summary

The revised package is substantially clearer than the first version. `14-codex-review-fix-plan.md` records the design decision that ConfigSet is not seed-only and not a compatibility bridge. `03`, `04`, `04a`, and `05a` now define ConfigSetBundle, ConfigSet, ConfigItem field tiers, child slots, ConfigView/ConfigPanel, and GenericConfigSetRenderer well enough for an implementation agent to produce a concrete batch plan.

The RunPlan contract is also materially stronger. `06-runplan-and-preflight-contract.md` now requires one server builder for preview, preflight, dry-run, and start; defines the storage/API location for `parameter_source_map`; covers args/env/mounts/ports/devices/docker_options/health_check; and states the Docker optional unchecked filtering rule.

The remaining risk is execution size, not document ambiguity. This is a broad replacement of the current `config_set_json`/ConfigEdit/runplan resolver paths. The docs correctly require Batch 0 inventory and user approval before functional edits.

## 4. Closure of First Review Issues

1. ConfigSet to final parameter model path: CLOSED. The docs reject `ParameterDefinition` as the mandatory conceptual root and replace it with ConfigSetBundle plus ConfigItem field tiers. `04-final-parameter-contract.md` states that ParameterDefinition/ParameterValue may exist as code names, but final semantics obey ConfigItem schema/value/state/provenance/snapshot/presentation.

2. Exact `parameter_source_map`: CLOSED. `06-runplan-and-preflight-contract.md` defines covered targets, sample JSON, storage in `resolved_run_plans.plan_json.parameter_source_map`, preview API response behavior, and source-chain requirements.

3. Docker subfield enabled/checked filtering: CLOSED. `04` and `06` require Docker subfields to be ConfigItems or structured items in DockerOptionsConfigSet. `state.enabled=false` filters optional Docker items from the final Docker spec; old `enabled_fields` must be cleaned or converted, not preserved as long-term compatibility.

4. Shared RunPlan builder: CLOSED. `06` requires `BuildDeploymentRunPlanInput()` / `BuildResolvedRunPlan()` or equivalent, shared by preview, preflight, dry-run, start, and E2E.

5. Test matrix: CLOSED. `08-api-first-e2e-and-automation-requirements.md` separates Go/API contract tests, web tests, and API-first E2E with specific assertions.

6. Fresh DB/rebuild and catalog policy: CLOSED. `08` explicitly allows deleting and rebuilding `/tmp/lightai/data/lightai.db`; `09` and `10` state no old DB/API/snapshot compatibility and require catalog/registry rebuild into the final model.

7. Closeout open issue status format: CLOSED. `11-final-closeout-template.md` requires open issues to use only `FIXED`, `DOCUMENTED_BLOCKER`, or `INVALID`, with evidence, impact, validation command, condition, and owner.

## 5. ConfigSetBundle Model Review

The ConfigSetBundle model is now sufficiently defined for planning and implementation. The key shape is repeated consistently:

```text
ConfigSetBundle =
  inherited_bundle_snapshots[]
  own_sets[]
  local_edits[]
  effective_view
```

The layer chain is clear: BackendVersion, BackendRuntime, NodeBackendRuntime, Deployment, ResolvedRunPlan, Instance. The docs also clarify that RunPlan reads only DeploymentConfigBundle effective snapshot and must not read live upstream BackendRuntime or NBR data to override it.

ConfigSet is now clearly defined as a final-domain concept. The revision explicitly says ConfigSet is not the old mixed `config_set_json` object and not a seed-only format. That is enough to prevent Claude from treating the current implementation as already final.

## 6. ConfigItem Field-Tier Review

The six field tiers are implementable:

```text
schema
value
state
provenance
snapshot
presentation
```

The docs define each tier with responsibilities and examples. The owner rule is now practical: schema can be copied during copy-on-create, but inherited schema/snapshot fields are readonly and owner remains unchanged. Current-layer value/state edits update provenance rather than redefining schema.

Removal of `overridable_at` is safe as a final rule. The replacement is simpler and testable: inherited items are editable by default at value/state level; special restrictions use `schema.read_only=true` or `state.editable=false`. That should be easier to implement and reason about than a separate override matrix.

## 7. Presentation Contract Review

The self-describing/self-presenting ConfigSet presentation is clear enough for implementation. `05a` defines ConfigView / ConfigPanel and establishes that external pages should not parse raw internal ConfigSet structures.

`child_slots` is sufficiently specified for first implementation. The parent determines placement, view mode, display mode, expansion, title, and ordering; the child renders/explains its own internal items. This avoids page-private schema duplication.

GenericConfigSetRenderer is specified at the right level: it consumes ConfigView, renders summary, own sections, child slots, local edits, and preview. Custom renderers are allowed only through the same ConfigView schema and cannot bypass ConfigItem rules.

One implementation caution: Claude should produce concrete TypeScript/Go shapes in Batch 0 or Batch 1 before writing broad UI code, because the docs intentionally define the contract rather than exact component props.

## 8. RunPlan / Source Map Review

The RunPlan contract is now clear. It requires:

1. Single shared builder for preview, preflight, dry-run, start, and E2E.
2. Builder input limited to DeploymentConfigBundle effective snapshot plus deployment/model/system/runtime context.
3. ResolvedRunPlan output including `parameter_source_map`, `source_chain`, `plan_hash`, and audit refs.
4. Source-map coverage for args, env, mounts, ports, devices, docker_options, health_check, resource_controls, and system_generated.
5. Storage under `resolved_run_plans.plan_json.parameter_source_map`.
6. Preview/start plan_hash and Docker spec consistency tests.

The current code still lacks this shape: `ResolvedRunPlan` has no source map and preview/start assemble inputs separately in existing handlers. That is now an implementation gap with clear documentation, not a documentation blocker.

## 9. No-Compatibility Policy Review

The no-compatibility policy is now explicit and actionable. The docs allow fresh DB rebuild, removal of old API fields, removal of old UI branches, cleanup of old resolver fallbacks, and conversion/removal of `enabled_fields`.

This policy is coherent with the target architecture. It also increases implementation blast radius, so the batch gate in `10-claude-execution-prompt.md` is necessary: Claude must first present a file-by-file implementation plan and wait for user approval.

## 10. Remaining Required Fixes

No additional documentation blocker remains before user approval of a batch implementation plan.

Required implementation preconditions remain:

1. Claude must run Batch 0 inventory before functional edits.
2. Claude must present the requested implementation plan before changing functional code.
3. User must approve batch execution.
4. Implementation commits must stay scoped and must not include unrelated files such as `deploy/observability/grafana/provisioning/dashboards/dashboards.yaml`.

## 11. Claude Execution Recommendation

Claude may proceed batch-by-batch after user approval. It should not run a broad AUTORUN directly. The correct next step is the pre-implementation plan required by `10-claude-execution-prompt.md`, mapping each batch to files, data model changes, API changes, UI changes, tests, commands, evidence paths, and risks.

Recommended execution gate:

```text
APPROVE_DOCS_FOR_BATCH_PLANNING
DO_NOT_START_FUNCTIONAL_IMPLEMENTATION_WITHOUT_USER_APPROVAL_OF_PLAN
```

## 12. Final Status

Review verdict: ACCEPT

Review document path:

```text
docs/reports/runtime-architecture-parameter-final-state/16-codex-second-review.md
```

Files changed:

```text
docs/reports/runtime-architecture-parameter-final-state/00-index.md
docs/reports/runtime-architecture-parameter-final-state/16-codex-second-review.md
docs/reports/runtime-architecture-parameter-final-state/manifest.json
```

Commit id: recorded in terminal completion output after commit.

Push result: recorded in terminal completion output after push.

git status --short at review creation time: clean for tracked functional code; final status is recorded in terminal completion output.
