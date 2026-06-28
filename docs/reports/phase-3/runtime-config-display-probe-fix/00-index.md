# Runtime Devices / Model Deployment Regression Fix — Index

This package defines a controlled repair scope for the current LightAI Go runtime template and model deployment regressions.

## Documents

1. `11-metax-device-mount-template-review-v2.md`
   - Revised review for MetaX Docker device / mount semantics.
   - Supersedes earlier wording that introduced `Optional devices`.

2. `12-model-deployment-regression-review.md`
   - Review of the uploaded `LightAI Go1.mhtml` model deployment page snapshot.
   - Records confirmed and user-observed deployment regressions.

3. `13-runtime-device-volume-design.md`
   - Final product and data-model semantics for Devices / Model mount / Additional volumes.
   - Includes MetaX, NVIDIA, Huawei catalog expectations.

4. `14-model-deployment-fix-design.md`
   - Final design for model deployment wizard reset, detail/edit UX, raw JSON handling, runtime_type resolution, port fields, and parameter layering.

5. `15-implementation-steps.md`
   - Concrete implementation sequence and expected code areas.

6. `16-validation-and-tests.md`
   - Required tests and acceptance checklist.

7. `17-claude-execution-prompt.md`
   - Self-contained execution prompt for Claude.

## Scope summary

This repair combines these user-reported problems:

- Runtime template device fields look like volume mounts.
- Current template has `Devices` and `Optional devices`; the final model should use one `Devices` field only.
- NVIDIA templates should leave Devices disabled/empty by default.
- MetaX templates should enable Devices and prefill `/dev/mxcd`, `/dev/dri`, `/dev/mem`.
- Device path existence is diagnostic-only and must not block deploy.
- Model mount should stay read-only; broad `/mnt:/mnt` belongs to Additional volumes if needed.
- Runtime template list page needs visible View / Edit / Copy-as-user-config actions.
- Model deployment fails with `[resolve_error] unsupported runtime_type: (only docker is supported)`.
- Model deployment details show raw config JSON / source metadata JSON instead of structured detail and edit operations.
- Create deployment wizard reuses previous saved/cancelled state until page refresh.
- Container listen port shows container port but host port is blank without explanation.
- `model_runtime.port`, `model_runtime.host`, `model_runtime.model` and overly specialized backend args are shown as ordinary deployment fields.

## Non-goals

- No new branch.
- No full runtime/config architecture rewrite.
- No `metax-docker` runtime mode.
- No real MetaX hardware validation requirement.
- No Playwright requirement.
- No handling of pre-existing `VERSION` modification.
