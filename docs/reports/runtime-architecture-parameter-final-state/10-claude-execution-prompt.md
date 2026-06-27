# Claude Execution Prompt — Runtime Architecture and Parameter Final-State

## 1. Role

You are the implementation agent for LightAI Go Runtime architecture and parameter final-state convergence. Read all required docs before editing code. Do not assume prior chat context.

## 2. Working directory

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
```

## 3. Required reading

Read in this order:

```text
docs/reports/runtime-architecture-parameter-final-state/00-index.md
docs/reports/runtime-architecture-parameter-final-state/01-execution-policy-and-scope.md
docs/reports/runtime-architecture-parameter-final-state/02-current-context-and-known-issues.md
docs/reports/runtime-architecture-parameter-final-state/13-codex-review.md
docs/reports/runtime-architecture-parameter-final-state/14-codex-review-fix-plan.md
docs/reports/runtime-architecture-parameter-final-state/03-final-runtime-domain-contract.md
docs/reports/runtime-architecture-parameter-final-state/04-final-parameter-contract.md
docs/reports/runtime-architecture-parameter-final-state/04a-parameter-ownership-and-layered-presentation-contract.md
docs/reports/runtime-architecture-parameter-final-state/05-runtime-requirements-and-capability-profile-design.md
docs/reports/runtime-architecture-parameter-final-state/05a-configset-bundle-composition-and-presentation-contract.md
docs/reports/runtime-architecture-parameter-final-state/06-runplan-and-preflight-contract.md
docs/reports/runtime-architecture-parameter-final-state/07-ui-and-api-contract.md
docs/reports/runtime-architecture-parameter-final-state/08-api-first-e2e-and-automation-requirements.md
docs/reports/runtime-architecture-parameter-final-state/09-implementation-plan.md
docs/reports/runtime-architecture-parameter-final-state/11-final-closeout-template.md
```

## 4. Core decisions

Do not preserve old compatibility. Fresh DB rebuild is allowed. ConfigSet is a final-domain concept, not seed-only and not an interim compatibility bridge. Current old `config_set_json` / `config_overrides_json` mixed semantics must not be preserved if they conflict with the final model. Each domain layer owns a ConfigSetBundle. A ConfigSetBundle contains inherited bundle snapshots, own ConfigSets, local edits, and effective view. ConfigSet is self-describing and composable. Parent ConfigSet defines child slots; child ConfigSet renders/explains itself. ConfigItem fields are schema/value/state/provenance/snapshot/presentation. schema/snapshot are read-only after copy. value/state can be changed in the current layer. Do not use complex `overridable_at` as the core rule. Use `schema.read_only` or `state.editable=false` for special read-only cases. checked/enabled means current-layer local edit, not default/required/inherited. RunPlan reads only DeploymentConfigBundle effective snapshot. preview/preflight/dry-run/start must share one builder. parameter_source_map must be persisted/returned and cover all final spec fields. Instance does not edit ConfigSet.

## 5. Execution policy

Do not run full AUTORUN until the plan is split into batches and the user has approved. Start with Batch 0 inventory. Do not touch unrelated working-tree changes, especially `deploy/observability/grafana/provisioning/dashboards/dashboards.yaml` unless explicitly in scope. Prefer clean deletion/refactor over compatibility fallback. Commit and push each verified batch. Update closeout and evidence.

## 6. Required output before implementation

Before changing functional code, produce an implementation plan that maps each batch to files to inspect, files likely to change, data model changes, API changes, UI changes, tests, commands, evidence paths, risks. Do not start implementation until the user approves the plan.
