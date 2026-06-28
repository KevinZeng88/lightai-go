# P0-B Vitest Render Gate Design

## 1. Background

The Phase 3 test inventory found that Go/API tests are comparatively strong, while frontend runtime/config/probe UI tests are mostly static or contract-style checks. The current `npm test` gate runs Node-based `web/tests/*.mjs` scripts, but it does not provide real DOM rendering coverage for runtime/config/probe pages and components.

Recent manual validation found real UI regressions that static source checks did not fully prevent:

1. Runtime template detail rendered no parameters because `getConfigEditView()` did not unwrap the backend envelope.
2. Docker option subfields such as UTS mode and Network mode displayed the whole parent `launcher.docker_options` object.
3. Environment variables, model mount, and health check fell back to raw JSON instead of structured widgets.
4. Raw probe evidence was displayed by default, including `NVIDIA_REQUIRE_CUDA`, `PATH`, and `LD_LIBRARY_PATH`.
5. Runtime template click behavior entered edit mode directly instead of detail/read-only mode.
6. Clone default display name and version display regressed.
7. Node runtime config showed duplicated/invalid port fields.

P0-A deterministic tests already covered non-rendered boundaries:

- `getConfigEditView()` envelope unwrap.
- Docker image `Config.Env` does not enter NBR config set.
- Docker image `Config.Env` does not enter `ResolvedRunPlan.env`.

P0-B now covers the rendered UI gap.

## 2. Objective

Introduce a maintainable frontend rendered-test gate using Vitest + Vue Test Utils.

The goal is not to test every page. The goal is to cover the actual high-risk shared components and page integration points where regressions occurred.

## 3. Recommended Test Shape

Use four layers:

1. Test infrastructure.
2. Shared component rendered tests.
3. Probe summary rendered tests.
4. Two page integration tests for accident-prone runtime pages.

The optimal P0-B set is:

| Layer | File | Purpose |
| --- | --- | --- |
| Infrastructure | `web/vitest.config.ts` | Vitest config. |
| Infrastructure | `web/tests/setup/vitest.setup.ts` | Vue Test Utils setup, Element Plus stubs/mocks. |
| Shared component | `web/src/components/config/__tests__/ConfigEditView.render.test.ts` | ConfigEditView real DOM rendering and structured widgets. |
| Shared component | `web/src/components/runtime/__tests__/ProbeSummaryView.render.test.ts` | Probe summary display and raw evidence collapse. |
| Page integration | `web/src/pages/__tests__/BackendRuntimesPage.integration.test.ts` | Runtime detail/view/edit/clone user-visible flow. |
| Page integration | `web/src/pages/__tests__/RunnerConfigsPage.integration.test.ts` | Node runtime config probe summary and port field integration. |

## 4. Why This Is Not Page-by-Page Testing

Do not add a rendered test for every page.

Only add page integration tests where the page itself caused or exposed a regression:

- `BackendRuntimesPage`: detail/edit mode, clone display name, version display.
- `RunnerConfigsPage`: probe summary integration, raw evidence default behavior, runtime port field display.

Most field rendering should be covered by `ConfigEditView.render.test.ts`, not repeated in each page.

## 5. Test Infrastructure Requirements

### 5.1 Dependencies

Use Vitest and Vue Test Utils.

Preferred packages if not already present:

- `vitest`
- `@vue/test-utils`
- `jsdom` or `happy-dom`

Prefer `jsdom` unless the project already has `happy-dom` or a strong reason to choose it.

### 5.2 package.json Scripts

Add:

```json
{
  "scripts": {
    "test:unit": "vitest run"
  }
}
```

`npm test` must include `test:unit` so that rendered tests are not orphaned.

Acceptable pattern:

```json
{
  "scripts": {
    "test": "node tests/apiClientPaths.test.mjs && ... && npm run test:unit",
    "test:unit": "vitest run"
  }
}
```

Do not remove existing `web/tests/*.mjs` tests.

### 5.3 Existing Orphan Tests

The repository already contains `web/src/**/__tests__/*.ts` tests that are not currently run by `npm test`.

If they run cleanly under Vitest, include them.

If they are stale or incompatible, do not spend this batch rewriting all of them. Document the reason and include only the P0-B tests in the Vitest include pattern.

## 6. Component Test Design: ConfigEditView

### 6.1 File

`web/src/components/config/__tests__/ConfigEditView.render.test.ts`

### 6.2 Test Data

Use a representative `ConfigEditView` object with these sections/items:

- Docker options:
  - `launcher.docker_options.shm_size = "16gb"`
  - `launcher.docker_options.ipc_mode = "host"`
  - `launcher.docker_options.uts_mode = null`
  - `launcher.docker_options.network_mode = null`
  - parent object contains `gpu_capabilities`, `gpu_driver`, `ipc_mode`, `shm_size`
- Environment variables:
  - empty object or safe example values
  - no `CUDA_VISIBLE_DEVICES={{vendor_visible_devices}}`
- Model mount:
  - `container_path = "/models"`
  - `readonly = true`
- Health check:
  - path/interval/timeout/retries or equivalent current schema
- Raw config JSON area, if exposed by component props/slots.

Use the actual component props expected by `ConfigEditView.vue`. Avoid inventing a parallel DTO that the real component does not consume.

### 6.3 Required Assertions

Must assert:

1. Sections render in the DOM.
2. `shm_size` displays `16gb`.
3. `ipc_mode` displays `host`.
4. `uts_mode` and `network_mode` do not display the parent Docker object.
5. DOM does not contain `"gpu_capabilities"` in the ordinary detail view.
6. Empty/null fields render as not configured, hidden, or an equivalent current product phrase. They must not render the parent object.
7. Environment variables are not expanded by default if the component supports section collapse.
8. DOM does not contain `CUDA_VISIBLE_DEVICES={{vendor_visible_devices}}`.
9. Model mount does not render as raw JSON containing `"container_path":"/models"`.
10. Health check does not render as raw JSON.
11. Raw config JSON is not visible by default.
12. Read-only mode does not expose editable inputs for protected/detail-only fields.
13. Editable mode exposes editable controls where appropriate.

### 6.4 Negative Assertions

The test should explicitly fail if the ordinary rendered view contains:

- `{"gpu_capabilities"`
- `"container_path":"/models"`
- `CUDA_VISIBLE_DEVICES={{vendor_visible_devices}}`
- `deferred to future design`
- `not yet implemented`

## 7. Component Test Design: ProbeSummaryView

### 7.1 Component Boundary

If no dedicated component exists, extract a small pure display component from `RunnerConfigsPage.vue`.

Suggested component:

`web/src/components/runtime/ProbeSummaryView.vue`

It should receive the probe data/summary as props and render:

- user-facing summary by default;
- raw evidence only inside collapsed/hidden diagnostics area.

Do not move data fetching into this component.

### 7.2 File

`web/src/components/runtime/__tests__/ProbeSummaryView.render.test.ts`

### 7.3 Test Data

Use a representative probe result containing:

- `level1.image_present = true`
- `level1.image_ref = "vllm/vllm-openai:latest"`
- `level2.env` containing:
  - `NVIDIA_REQUIRE_CUDA=...very long...`
  - `PATH=/usr/local/nvidia/bin:...`
  - `LD_LIBRARY_PATH=/usr/local/nvidia/lib64:...`
- `level3.backend_match_status = "confirmed_match"`
- `process_start_detection.selected_profile_id = "vllm.image_default"`
- `compatibility_check_status` or equivalent current field with `not_run` / non-blocking product message.

### 7.4 Required Assertions

Must assert:

1. Summary renders by default.
2. Image reference is visible.
3. Backend match status or equivalent user-facing status is visible.
4. Process start profile or start mode is visible.
5. Raw JSON/evidence is collapsed or hidden by default.
6. Default DOM does not contain `NVIDIA_REQUIRE_CUDA`.
7. Default DOM does not contain `PATH=/usr/local/nvidia`.
8. Default DOM does not contain `LD_LIBRARY_PATH`.
9. Default DOM does not contain `deferred to future design`.
10. Default DOM does not contain `not yet implemented`.
11. When raw diagnostics are explicitly expanded, raw evidence may become visible.

## 8. Page Integration Test: BackendRuntimesPage

### 8.1 File

`web/src/pages/__tests__/BackendRuntimesPage.integration.test.ts`

### 8.2 Scope

Only test page-level integration and state flow.

Do not duplicate all field rendering assertions from `ConfigEditView.render.test.ts`.

### 8.3 Mocking

Mock API modules used by `BackendRuntimesPage.vue`, including runtime list/detail/config-edit/clone/save methods as needed.

Use stable fake data:

- one vLLM runtime template;
- `display_name = "vLLM NVIDIA Docker"`;
- `name = "runtime.vllm.nvidia-docker"`;
- version display should be `*`;
- config edit view includes at least one section.

### 8.4 Required Assertions

Must assert:

1. Clicking runtime row opens detail/read-only view, not edit mode.
2. Detail view includes a clear Edit button.
3. Save/Cancel are not shown in read-only view.
4. Clicking Edit shows Save and Cancel.
5. Clicking Cancel returns to read-only view.
6. Successful Save exits edit mode.
7. Clone dialog default display name uses product display name, e.g. `vLLM NVIDIA Docker - 用户配置`, not `runtime.vllm.nvidia-docker - 用户配置`.
8. Version display is `*`, not `v0.23.0` or other concrete backend version.

## 9. Page Integration Test: RunnerConfigsPage

### 9.1 File

`web/src/pages/__tests__/RunnerConfigsPage.integration.test.ts`

### 9.2 Scope

Only test page integration points that previously regressed:

- probe summary vs raw evidence;
- canonical port field display;
- model runtime port removal from user-facing required/readonly empty field.

### 9.3 Mocking

Mock API modules used by `RunnerConfigsPage.vue`.

Use one ready/checked NBR with:

- `display_name` user-facing name;
- `probe_results_json` containing raw Docker image env;
- config edit view containing `service.container_port = 8000` and no user-facing `model_runtime.port` field.

### 9.4 Required Assertions

Must assert:

1. Detail view uses probe summary; default DOM does not show raw `probe_results_json`.
2. Default DOM does not contain `NVIDIA_REQUIRE_CUDA`.
3. Default DOM does not contain `PATH=/usr/local/nvidia`.
4. Default DOM does not contain `LD_LIBRARY_PATH`.
5. `service.container_port` or user-facing label such as `Container listen port` displays `8000`.
6. Page does not display `Model runtime port` as a required + readonly + empty field.

## 10. Boundaries And Non-goals

Do not do the following in this batch:

1. Do not add a test for every page.
2. Do not introduce Playwright or modify Playwright setup.
3. Do not depend on running server, agent, Docker, GPU, or model files.
4. Do not rewrite pages to make tests easier.
5. Do not refactor the full frontend architecture.
6. Do not remove existing `web/tests/*.mjs` tests.
7. Do not process `VERSION`.
8. Do not make broad UI changes unrelated to the rendered test gate.

Functional code changes are allowed only if required to extract a pure `ProbeSummaryView` component or to expose stable, testable component boundaries.

## 11. Required Commands

After implementation, run:

```bash
go test ./internal/server/...
go test ./internal/agent/...
cd web && npm test
cd web && npm run build
```

`npm test` must run both existing `web/tests/*.mjs` checks and the new Vitest tests.

## 12. Acceptance Criteria

P0-B is accepted only if:

1. Vitest is configured.
2. Vitest tests are included in `npm test`.
3. `ConfigEditView.render.test.ts` exists and covers the required field rendering regressions.
4. `ProbeSummaryView.render.test.ts` exists or an equivalent extracted component test exists.
5. `BackendRuntimesPage.integration.test.ts` exists and covers detail/edit/clone/version state flow.
6. `RunnerConfigsPage.integration.test.ts` exists and covers probe summary and canonical port display.
7. All Go tests pass.
8. `cd web && npm test` passes.
9. `cd web && npm run build` passes.
10. Changes are committed and pushed.
11. `git status --short` leaves only pre-existing `M VERSION`.

## 13. Output Required From Implementer

The implementer must output:

1. Added dependencies/config/scripts.
2. Added test files and test names.
3. Which historical regression each test protects.
4. Whether old `web/src/**/__tests__/*.ts` tests were included or intentionally left out, with reason.
5. Test command results.
6. Commit ID.
7. Push result.
8. `git status --short`.
