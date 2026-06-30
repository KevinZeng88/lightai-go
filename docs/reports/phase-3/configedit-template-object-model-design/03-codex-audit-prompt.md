# Codex Prompt — ConfigEdit Object Model & External Template Architecture Audit

You are working in `/home/kzeng/projects/ai-platform-study/lightai-go`.

Do not create a new branch. Do not modify code in this pass. This is a read-only architecture audit and implementation planning task.

## Goal

Audit whether LightAI Go currently implements the intended ConfigEdit object model and external ConfigEdit component template architecture. Produce a concrete staged implementation plan.

This is not a narrow vLLM fix and not a Model Deployment page-only fix.

The target problem class:

- Final Docker command contains parameters that are not editable in the configuration chain.
- RunPlan resolver or preview injects/assembles runtime-affecting values outside the ConfigEdit object.
- Some source information is visible in preview but not connected to editable fields.
- Raw/debug/template fields are shown to normal users.
- Tooltips/help lack range/default/recommended/effect.
- vLLM/SGLang/llama.cpp backend/version parameters are partly hardcoded and may require binary changes for new versions.
- BackendRuntime → NodeBackendRuntime → Deployment → RunPlan is not consistently modeled as copy-on-create materialized editable snapshots.

## User design intent

ConfigEdit should be treated as an object, not merely a Vue form.

A ConfigEdit object should:

- contain its own fields/components;
- know how to display and validate them;
- contain value/state/provenance/snapshot metadata;
- know its parent snapshot;
- describe child ConfigEdit initialization/copy behavior;
- materialize inherited/default/generated initial values into current-layer editable fields;
- allow current-layer override and reset-to-parent/default;
- compile final snapshot into RunPlan/DockerSpec;
- keep pages free of backend/vendor/parameter-specific logic.

External ConfigEdit component templates should define:

- sections;
- components/renderers;
- labels/help/tooltips;
- default/recommended/range;
- validation;
- editability by layer;
- copy behavior;
- parent/child behavior;
- normal/advanced/developer view grouping;
- effects on CLI/env/Docker/mount/port/health check.

Runtime/backend templates describe how a backend runs. ConfigEdit component templates describe how the configuration object is edited. Do not mix these concepts.

## Files to read first

Read these design inputs if present:

- `docs/reports/phase-3/configedit-template-object-model-design/00-index.md`
- `docs/reports/phase-3/configedit-template-object-model-design/01-design.md`
- `docs/reports/phase-3/configedit-template-object-model-design/02-configedit-template-contract.md`
- `docs/reports/phase-3/configedit-template-object-model-design/04-acceptance-checklist.md`

If these files are not present, continue using this prompt as the source of truth and mention that the design package was not found.

## Audit scope

Review at least these backend areas:

- `internal/server/configedit`
- `internal/server/catalog`
- `internal/server/semanticconfig`
- `internal/server/runplan`
- deployment preview / preflight handlers
- BackendRuntime / NodeBackendRuntime persistence and snapshot logic
- Deployment config / placement / override logic
- Docker command builder
- source map / diagnostics
- health check defaulting
- device binding / accelerator selection
- migrations/schema only if required to understand current shape

Review at least these frontend areas:

- `web/src/components/config/ConfigEditView.vue`
- `web/src/components/config/ConfigField.vue`
- `web/src/utils/configEditFieldMeta.ts`
- `web/src/components/deployments/DeploymentWizard.vue`
- `web/src/components/deployments/DeploymentPreviewPanel.vue`
- `web/src/components/deployments/NodeRuntimeConfigWizard.vue`
- `web/src/pages/ModelDeploymentsPage.vue`
- `web/src/pages/RunnerConfigsPage.vue`
- `web/src/pages/BackendsPage.vue`
- `web/src/pages/ModelArtifactsPage.vue`
- any page or component that renders backend/runtime/deployment params outside ConfigEdit

Review template/catalog sources:

- `configs/backend-catalog/**`
- any config registry files
- any embedded defaults
- any i18n or semantic field registry used for parameter labels/help

## Core audit questions

### 1. ConfigEdit as object

Determine whether the current ConfigEdit model is only a rendered field list or a full object model.

Check whether it supports:

- parent snapshot reference;
- child ConfigEdit initialization contract;
- copy whole effective snapshot;
- field state: enabled/disabled, readonly, required, sensitive;
- provenance: default/inherited/copied/generated/user_override;
- override state;
- reset to parent/default;
- patch target;
- effects preview;
- normal/advanced/developer view grouping;
- nested/custom components such as device binding, port binding, health check, env, volume, command args.

### 2. Template-driven components

Determine whether fields/components are driven by external templates or hardcoded in code.

Find all places where vLLM/SGLang/llama.cpp parameters, NVIDIA/CUDA behavior, Docker options, mount/port/health defaults, or renderer choices are hardcoded in Go/Vue.

Classify each as:

- acceptable generic engine code;
- should move to ConfigEdit component template;
- should move to runtime/backend catalog;
- should become a generic renderer;
- should become a generic compiler effect.

### 3. Materialized editable snapshots

Trace the flow:

```text
BackendRuntime → NodeBackendRuntime → Deployment → RunPlan
```

For each layer, determine:

- what is copied from parent;
- what remains a live reference;
- what is recomputed by resolver;
- what is editable in the current layer;
- what is hidden;
- what appears in final Docker command;
- whether edits persist as current-layer config.

### 4. Device binding

Specifically audit `--gpus "device=0"` and `CUDA_VISIBLE_DEVICES=0`.

Determine:

- where they are generated;
- whether they exist as ConfigEdit fields/components;
- whether they are copied from parent into NBR/Deployment;
- whether deployment edit/wizard can modify/disable them;
- whether they are hardcoded for NVIDIA;
- how MetaX/Huawei/CPU would be handled;
- whether the final command can contain values not represented in editable config.

The target is a ConfigEdit component such as `runtime.device_binding`, with editable fields for enabled/mode/vendor/accelerator ids/docker GPU option/visible env key/value/device mounts, controlled by external template metadata.

### 5. Other final Docker parameters

Audit all final runtime-affecting parameters:

- image
- entrypoint/command/args
- extra args
- env/extra env
- model mount
- extra volumes
- ports
- host/container port
- IPC mode
- shm size
- health check
- served model name
- model path/container path
- vLLM/SGLang/llama.cpp runtime params

For each, answer:

- Is it a ConfigEdit field/component?
- Is it visible in normal/advanced/developer view appropriately?
- Can it be edited where expected?
- Does it have label/help/range/recommended/default/effect?
- Does it copy to child layers?
- Does RunPlan compile it from final snapshot only?
- Does source map explain it?

### 6. User-facing UI quality

Find fields currently shown in normal UI that should be advanced/developer only, such as:

- internal key display;
- raw command templates;
- unresolved template variables;
- Config JSON;
- dry-run detail;
- raw source map;
- raw empty template arrays like `Device bindings []` or `Volume mounts []` that conflict with final effective values.

Find fields lacking meaningful help metadata:

- range;
- recommended value;
- default;
- effect;
- applicability;
- warnings.

English help text is acceptable.

### 7. External template update path

Audit whether a new vLLM/SGLang/llama.cpp backend version or parameter can be added without rebuilding the binary.

Check:

- external catalog loading;
- local override support;
- user-defined template support;
- validation of external templates;
- import/export capability;
- template editor UI feasibility;
- built-in vs override precedence.

If not supported, propose a staged path.

## Required output

Create a report under:

```text
docs/reports/phase-3/configedit-object-model-template-audit.md
```

The report must include:

1. Executive summary
2. Current architecture map
3. Gap analysis against target design
4. Hidden/hardcoded parameter inventory
5. Device binding findings
6. ConfigEdit object model findings
7. External template findings
8. UI/UX findings
9. Cross-backend coverage: vLLM/SGLang/llama.cpp
10. Cross-page coverage: BackendRuntime/NBR/Deployment/Deployment override/RunPlan preview
11. Proposed staged implementation plan
12. Test strategy
13. Risks and migration considerations
14. Open questions

## Implementation plan requirements

Do not propose another narrow patch.

The staged plan should separate:

### Stage 1 — Contract audit and tests

- Add failing tests that capture current gaps.
- Establish contract fixtures for vLLM/SGLang/llama.cpp.
- Verify final command parameters must correspond to ConfigEdit fields/effects.

### Stage 2 — ConfigEdit object model

- Parent/child snapshot metadata.
- Field provenance/override/reset.
- Normal/advanced/developer view levels.
- Nested components.

### Stage 3 — External ConfigEdit component template

- Template parser/validator.
- Template materialization into ConfigEdit object.
- Built-in/local/user precedence.
- Schema lint.

### Stage 4 — Device binding component

- Move NVIDIA GPU/CUDA behavior from hidden resolver logic into template-driven component/effects.
- Support auto/manual/disabled.
- Prepare vendor-neutral MetaX/Huawei/CPU path.

### Stage 5 — RunPlan compiler cleanup

- Compile only from final materialized snapshot.
- Remove hidden runtime-affecting injection.
- Keep source map as explanation, not source of truth.

### Stage 6 — UI cleanup

- Use ConfigEdit components everywhere.
- Hide developer/debug fields from normal UI.
- Improve tooltip/help/range/recommended/effect display.
- Add or design template editor page.

### Stage 7 — Template management UI

- clone built-in;
- edit structured template;
- raw YAML/JSON advanced mode;
- validate;
- preview ConfigEdit;
- preview RunPlan/Docker command;
- publish/rollback/import/export.

## Validation commands

Because this pass is read-only, do not modify code and do not run full builds unless needed. You may run search/grep and targeted read-only commands.

At the end, include:

```bash
git status --short
```

The expected result should show only the new audit report if you create it. Do not commit in this pass.
