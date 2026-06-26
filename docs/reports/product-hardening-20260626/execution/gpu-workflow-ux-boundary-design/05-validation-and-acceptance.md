# 05 — Validation and Acceptance Criteria

## 1. Required commands

Run from repo root:

```bash
go test ./...
go build ./cmd/server/... ./cmd/agent/...
cd web && npm test
cd web && npm run build
cd ..
git diff --check
git status --short
```

All must pass before commit.

---

## 2. Functional acceptance

### Runtime Templates

- Main table shows user-facing names like `nvidia.sglang`, `nvidia.vllm`, `nvidia.llama.cpp b9700`.
- Main table does not show internal IDs as the primary name.
- Main table does not show `launcher.*`, `runtime_env.*`, or template placeholders.
- ConfigSet and Source Metadata are available only in collapsed Advanced Diagnostics.
- System templates are read-only.
- User-managed templates can be edited.

### Node Runtime Configs

- New wizard always starts from Step 1.
- Cancel then reopen starts from Step 1.
- Config name field is visible and saved as `display_name`.
- Default config name is generated from node + vendor + backend.
- Image field is visible and editable.
- Normal parameter form shows user-facing fields, including shared memory.
- Normal parameter form does not display `launcher.command`, `launcher.args`, `{{MODEL_CONTAINER_PATH}}`, `runtime_env.*`, or other internal keys.
- Save failure keeps wizard open and shows error.
- Check failure keeps wizard open and shows error.
- Not-ready check result keeps wizard open and shows status/reason/warnings.
- Ready/ready_with_warnings result allows finish and refreshes the list.

### Model Library

- Node selection uses the shared table-style node selector.
- Label clearly says the node is where model files are stored.
- Model wizard remains about model path scan and model facts.
- Model wizard does not expose Docker/backend runtime args.

### Model Deployments

- Only `ready` and `ready_with_warnings` NBRs are selectable.
- `needs_check`, `missing_image`, `error`, and `unknown` are visible but disabled when show-all is enabled.
- Deployment preview calls `/deployments/preview`.
- Payload uses `node_backend_runtime_id` only.
- If the selected model has no location on the NBR's node, preview/save shows a clear error.
- Errors keep the dialog open.

### Model Instances

- Instance list uses readable model/runtime/node labels where available.
- Logs remain accessible.
- Stopped/failed state behavior remains unchanged unless explicitly improved with tests.

---

## 3. Frontend tests

Add or update tests to cover:

1. Runtime Templates page generates user-facing names.
2. Runtime Templates main table hides raw ConfigSet/internal keys.
3. Runtime Templates diagnostics can show raw JSON in a collapsed section.
4. NodeRuntimeConfigWizard resets on open.
5. NodeRuntimeConfigWizard resets after cancel.
6. NodeRuntimeConfigWizard has config name input.
7. NodeRuntimeConfigWizard normal form includes shared memory.
8. NodeRuntimeConfigWizard normal form hides `launcher.command`.
9. NodeRuntimeConfigWizard normal form hides `{{MODEL_CONTAINER_PATH}}`.
10. Save failure does not emit completion and does not close parent dialog.
11. Check failure does not emit completion and shows error.
12. Ready check result allows finish.
13. ModelArtifactsPage uses shared NodeSelectorTable.
14. DeploymentWizard only allows ready / ready_with_warnings NBR selection.
15. DeploymentWizard blocks non-ready NBR in buildPayload.
16. Deployment payload uses `node_backend_runtime_id` and does not include `backend_runtime_id`.
17. i18n key test passes.

---

## 4. API and resolver tests

Keep or add Go tests for:

- Deployment preview rejects legacy `backend_runtime_id`.
- Deployment preview accepts `node_backend_runtime_id`.
- Deployment preview requires model location compatible with NBR node.
- ready_with_warnings NBR is deployable.
- missing_image / needs_check / error NBR is not deployable.
- Preview and start share the same resolver path.

---

## 5. Manual Web verification

Record notes in evidence directory.

### Runtime Templates

Verify:

```text
- Template names are understandable.
- Count is not inflated by raw implementation variants.
- Advanced JSON is not the main view.
```

### Node Runtime Configs

Verify:

```text
- Open New wizard.
- Select node.
- Cancel.
- Reopen: starts from Step 1.
- Select node and runtime template.
- Enter config name.
- Step 3 shows human runtime parameters.
- launcher.command is not visible.
- {{MODEL_CONTAINER_PATH}} is not visible.
- Try save/check with an invalid image: error remains visible and dialog stays open.
- Save/check with valid image: status is shown and list refreshes.
```

### Model Library

Verify:

```text
- New model wizard uses table-style node selector.
- Node selector label is model-location oriented.
- File browser still works.
- Scan and save still work.
```

### Model Deployments

Verify:

```text
- New deployment wizard can select a model.
- ready/ready_with_warnings NBR selectable.
- needs_check/missing_image/error visible but disabled.
- Preview shows Run Plan.
- Node mismatch is reported clearly.
- Save errors keep dialog open.
```

---

## 6. Evidence output

Create:

```text
docs/reports/product-hardening-20260626/evidence/<TS>/gpu-workflow-ux-boundary/
```

Required files:

```text
review-summary.md
runtime-templates-verification.md
node-runtime-config-wizard-verification.md
model-library-node-selector-verification.md
deployment-wizard-verification.md
npm-test.log
npm-build.log
go-test.log
go-build.log
git-diff-check.log
manual-web-verification-notes.md
```

Screenshots are useful but not mandatory if browser automation is unavailable. If screenshots are unavailable, write explicit manual observations with exact page names and tested actions.

---

## 7. Commit acceptance

A commit is acceptable only if it reports:

```text
- root cause
- files changed
- tests run
- manual verification
- evidence path
- commit id
- push result
- final git status
```

Final git status must be clean.

