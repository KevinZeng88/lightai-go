# LightAI Go — ConfigEdit Object Model & Template-Driven Configuration Design

This package defines the target design and Codex review/execution input for fixing the current class of problems in LightAI Go configuration editing and RunPlan generation.

## Why this package exists

Recent fixes improved field labels, wizard action bars, RunPlan source visibility, and preview diagnostics. However, the latest UI screenshots still show a deeper problem:

- Parameters such as `--gpus "device=0"` and `CUDA_VISIBLE_DEVICES=0` appear in the final Docker command but are not editable in the deployment edit/wizard pages.
- Some final runtime behaviors are still injected or assembled outside the editable configuration object.
- The user cannot clearly tell which parameters are copied from parent layers, which are defaults, which are overridden, and where to edit them.
- Some low-level raw/template/debug information is displayed in normal user views.
- Tooltips/help do not consistently show value range, default, recommended values, effect, or edit scope.
- The issue is not vLLM-only and not deployment-page-only; it affects vLLM/SGLang/llama.cpp and the whole chain from template → BackendRuntime → NodeBackendRuntime → Deployment → RunPlan.

## Files

1. `01-design.md`  
   Target architecture and design principles.

2. `02-configedit-template-contract.md`  
   Proposed external ConfigEdit component template contract and object model.

3. `03-codex-audit-prompt.md`  
   Self-contained Codex prompt for read-only architecture audit and implementation plan.

4. `04-acceptance-checklist.md`  
   Acceptance criteria for later implementation.

## Recommended workflow

1. Give `03-codex-audit-prompt.md` to Codex first.
2. Codex should perform a read-only audit and produce a concrete implementation plan.
3. Review the plan before allowing code changes.
4. Use `04-acceptance-checklist.md` as the gate for implementation closeout.

This package intentionally starts with audit/design validation rather than immediate code edits because the scope is architectural and previous local fixes only solved parts of the problem.
