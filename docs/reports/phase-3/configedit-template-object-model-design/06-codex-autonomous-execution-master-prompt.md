# 06 — Codex Autonomous Execution Master Prompt

You are working in:

```text
/home/kzeng/projects/ai-platform-study/lightai-go
```

Use the current branch. Do not create a new branch.

## Read first

Read these documents in order:

```text
docs/reports/phase-3/configedit-template-object-model-design/01-design.md
docs/reports/phase-3/configedit-template-object-model-design/02-configedit-template-contract.md
docs/reports/phase-3/configedit-template-object-model-design/03-codex-audit-prompt.md
docs/reports/phase-3/configedit-template-object-model-design/04-acceptance-checklist.md
docs/reports/phase-3/configedit-template-object-model-design/05-codex-audit-result-and-implementation-plan.md
docs/reports/phase-3/configedit-template-object-model-design/07-work-package-a-contract-object-model.md
docs/reports/phase-3/configedit-template-object-model-design/08-work-package-b-template-effects.md
docs/reports/phase-3/configedit-template-object-model-design/09-work-package-c-runtime-effects-runplan.md
docs/reports/phase-3/configedit-template-object-model-design/10-work-package-d-ui-template-management.md
docs/reports/phase-3/configedit-template-object-model-design/11-self-audit-evidence-and-acceptance.md
docs/reports/phase-3/configedit-template-object-model-design/12-final-closeout-template.md
```

The architecture audit in `05-codex-audit-result-and-implementation-plan.md` is the factual starting point.

## Mission

Implement the ConfigEdit Object Model and External ConfigEdit Component Template architecture as the new mainline runtime configuration path.

This must address the class of issues where final runtime behavior is produced by hidden resolver logic, page-specific code, hardcoded backend semantics, or raw service/placement JSON rather than editable ConfigEdit objects and template-defined effects.

The final system should make this chain true:

```text
External ConfigEdit Component Template
  -> materialized ConfigEdit Object
  -> parent/child effective snapshot copy
  -> editable layer snapshot
  -> final Deployment ConfigEdit Object
  -> RunPlan / DockerSpec compiled from final snapshot
  -> Docker command preview
```

## Operating mode

Run autonomously through implementation, tests, self-audit, fixes, commits, push, and final closeout.

Do not ask for human approval after each work package.

Proceed in this order:

```text
Work Package A — Contract Tests + ConfigEdit Object Foundation
Work Package B — External ConfigEdit Component Template + Effect Engine
Work Package C — Runtime Effects Components + RunPlan Compiler Cleanup
Work Package D — UI Full-Chain Integration + Template Management MVP + Final Audit
```

After each package:

1. Run package-relevant tests.
2. Self-audit using `11-self-audit-evidence-and-acceptance.md`.
3. Fix all fixable issues found by the self-audit.
4. Update `execution-log.md`.
5. Commit if the package is stable.

At the end:

1. Run full verification.
2. Write the final closeout.
3. Commit closeout.
4. Push.

## True blocker policy

Stop only when there is a true blocker that cannot be resolved in the current environment.

Examples:

- Required hardware is unavailable and no fixture/mock can represent it safely.
- Required credentials or external service are unavailable.
- A schema/storage change would destroy user data and there is no safe rebuild/backfill path.
- A security boundary requires a product decision before continuing.
- The current codebase differs so much from the audit report that the implementation plan would be unsafe.

When stopping for a blocker:

1. Do not leave the issue only in chat.
2. Create or update:

```text
docs/reports/phase-3/configedit-template-object-model-design/99-autonomous-execution-blocker.md
```

3. Include:
   - blocker id
   - exact evidence
   - affected files
   - attempted fixes
   - why continuing is unsafe
   - recommended next action
   - verification already run
   - current git status

Fixable problems are not blockers. Fix them.

## Non-negotiable architecture rules

### 1. ConfigEdit is an object model

A ConfigEdit object is not only a projected field list.

It must carry, at minimum:

- object kind
- object id
- template id
- snapshot id
- parent reference
- child initialization contract
- sections
- components
- fields
- values
- enabled state
- source/provenance
- default/current/effective values
- reset behavior
- validation
- renderer metadata
- view level
- effects preview
- diagnostics

### 2. Child layers copy whole effective snapshots

The runtime configuration chain is snapshot-based:

```text
BackendRuntime ConfigEdit
  -> copy effective snapshot
NodeBackendRuntime ConfigEdit
  -> copy effective snapshot
Deployment ConfigEdit
  -> compile
RunPlan / DockerSpec
```

After copy, the child owns an editable snapshot. Source/provenance explains the initial origin. Source/provenance must not make a field readonly unless the component explicitly says so.

### 3. RunPlan compiles from final ConfigEdit

RunPlan generation must compile from the final materialized ConfigEdit snapshot.

Resolver code must not secretly inject runtime-affecting behavior such as:

- `--gpus`
- `CUDA_VISIBLE_DEVICES`
- visible devices env for other vendors
- service port args
- served model name
- model mount
- health check values
- backend CLI flags
- Docker options
- extra env
- extra args

Platform-generated readonly facts are allowed only for platform facts such as container name, instance id, lease id, absolute safe host path, and hardware inventory evidence. They must be explicitly marked readonly/platform_generated.

### 4. Runtime template and ConfigEdit component template are separate

Runtime template describes how a backend runs.

ConfigEdit component template describes how configuration objects are edited, inherited, validated, rendered, and compiled.

Do not mix these concepts.

### 5. Backend knowledge belongs in templates, not pages

Do not add page-level hardcoding for:

- vLLM
- SGLang
- llama.cpp
- NVIDIA
- CUDA_VISIBLE_DEVICES
- `--gpus`
- model runtime parameter dictionaries
- Docker option meanings
- CLI flags

Pages should render ConfigEdit objects and call generic save/preview APIs.

### 6. Normal UI stays clean

Normal view must not show:

- technical keys such as `model_runtime.pipeline_parallel_size`
- raw source map
- raw Config JSON
- unresolved template command
- patch target internals
- system_generated internals

Those belong in developer/debug view.

Tooltips/help should show useful operator information:

- default
- recommended value
- valid range
- examples
- effect
- applicability
- source
- edit scope

English labels/help are acceptable and preferred for technical runtime parameters.

## Work package documents

Use the detailed package documents:

```text
07-work-package-a-contract-object-model.md
08-work-package-b-template-effects.md
09-work-package-c-runtime-effects-runplan.md
10-work-package-d-ui-template-management.md
```

These documents are part of this instruction. Follow them closely.

## Commit strategy

Commit after each stable package or logical milestone. Suggested messages:

```text
test(runtime): add configedit object model contract tests
feat(configedit): add object model and snapshot copy contract
feat(configedit): add external component template engine
feat(runtime): materialize runtime effect components
refactor(runtime): compile runplan from configedit effects
feat(web): render configedit object components and view levels
feat(web): add configedit template management mvp
docs(runtime): close configedit object model implementation
```

Push after the full run is complete. If the run becomes large, push safe intermediate commits after passing relevant tests to avoid data loss.

## Required final verification

Run:

```bash
git status --short
go test ./...
cd web
npm test
npm run test:unit
npm run build
```

Fix failures and rerun. Do not report success with failing tests.

## Required final closeout

Write:

```text
docs/reports/phase-3/configedit-template-object-model-design/06-configedit-object-model-full-implementation-closeout.md
```

Use `12-final-closeout-template.md` as the structure.

Final output to the user should be concise and include:

- final status
- closeout path
- commit ids
- test results
- push result
- final git status
