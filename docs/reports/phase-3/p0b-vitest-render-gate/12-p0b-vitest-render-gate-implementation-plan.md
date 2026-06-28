# P0-B Vitest Render Gate Implementation Plan

## 1. Implementation Principle

Implement the best P0-B rendered-test set, not the smallest possible set and not a page-by-page test rewrite.

The target is to protect the actual regressions discovered in runtime/config/probe UI while keeping the suite maintainable.

## 2. Step-by-Step Plan

### Step 1: Inspect Current Frontend Test Setup

Check:

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go/web
cat package.json
find src -path '*__tests__*' -type f -maxdepth 5
find tests -type f
```

Identify whether `vitest`, `@vue/test-utils`, and DOM environment dependencies already exist in `package.json` or lock files.

### Step 2: Add Vitest Dependencies If Missing

If missing, add only the minimal dependencies required:

- `vitest`
- `@vue/test-utils`
- `jsdom` or `happy-dom`

Prefer `jsdom` unless the repository already indicates `happy-dom`.

Do not add Playwright dependencies.

### Step 3: Add Vitest Config

Create:

`web/vitest.config.ts`

Config requirements:

- Vue plugin compatible with existing Vite setup.
- DOM environment enabled.
- setup file: `web/tests/setup/vitest.setup.ts`.
- include only intended tests if legacy TS tests are incompatible.

Example include strategy:

```ts
include: [
  'src/components/**/*.render.test.ts',
  'src/pages/**/*.integration.test.ts',
  'src/**/__tests__/*.test.ts'
]
```

If existing legacy tests are incompatible, use a narrower include and document it in the completion report.

### Step 4: Add Test Setup

Create:

`web/tests/setup/vitest.setup.ts`

It may include:

- global mocks for `ResizeObserver`, `matchMedia`, `IntersectionObserver` if needed;
- Element Plus transition/stub setup if needed;
- helper to flush promises;
- default console warning filtering only if absolutely necessary.

Do not suppress errors broadly.

### Step 5: Update npm Scripts

Update `web/package.json`:

- Add `test:unit`.
- Ensure `npm test` runs `test:unit` after existing `.mjs` tests.

Do not remove current `.mjs` tests.

### Step 6: Add ConfigEditView Render Test

Create:

`web/src/components/config/__tests__/ConfigEditView.render.test.ts`

Implementation guidance:

- Use Vue Test Utils `mount()`.
- Use real `ConfigEditView.vue`.
- Stub only heavy external UI components if needed.
- Provide a realistic `ConfigEditView` object matching backend shape.
- Keep test data small but semantically representative.

Suggested test names:

- `renders sections and structured runtime fields`
- `does not leak docker parent object into missing subfields`
- `renders structured widgets instead of raw json`
- `keeps raw config hidden by default`
- `does not show vendor visible device placeholder as user env`

### Step 7: Extract ProbeSummaryView If Needed

If `RunnerConfigsPage.vue` currently contains inline probe summary rendering, extract a pure display component:

`web/src/components/runtime/ProbeSummaryView.vue`

Requirements:

- Props-only display component.
- No API calls.
- No routing.
- Summary visible by default.
- Raw diagnostics collapsed by default.

Do not refactor unrelated RunnerConfigs logic.

### Step 8: Add ProbeSummaryView Render Test

Create:

`web/src/components/runtime/__tests__/ProbeSummaryView.render.test.ts`

Suggested test names:

- `renders user-facing probe summary by default`
- `keeps raw image env hidden until diagnostics are expanded`
- `does not show development wording in default summary`

### Step 9: Add BackendRuntimesPage Integration Test

Create:

`web/src/pages/__tests__/BackendRuntimesPage.integration.test.ts`

Guidance:

- Mock API modules used by `BackendRuntimesPage.vue`.
- Mock router/i18n/store only as needed.
- Verify state flow: detail -> edit -> cancel/save.
- Verify clone default display name and version display.

Do not repeat the full ConfigEditView field assertions here.

Suggested test names:

- `opens runtime in readonly detail mode before edit`
- `supports edit cancel and save state transitions`
- `uses product display name for clone default and shows wildcard version`

### Step 10: Add RunnerConfigsPage Integration Test

Create:

`web/src/pages/__tests__/RunnerConfigsPage.integration.test.ts`

Guidance:

- Mock API modules used by `RunnerConfigsPage.vue`.
- Provide fake NBR list/detail and config edit view.
- Verify page integrates `ProbeSummaryView` and canonical port display.

Suggested test names:

- `shows probe summary instead of raw evidence by default`
- `shows service container port and hides empty model runtime port`

### Step 11: Run Tests Incrementally

Run:

```bash
cd web
npm run test:unit
npm test
npm run build
```

Then run Go tests:

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
go test ./internal/server/...
go test ./internal/agent/...
```

### Step 12: Document Results

Update or append a closeout note in:

`docs/reports/phase-3/test-inventory-and-gap-review.md`

or create:

`docs/reports/phase-3/p0b-vitest-render-gate-closeout.md`

Keep it short. Include:

- added test infrastructure;
- tests added;
- historical regressions covered;
- command results;
- commit ID.

### Step 13: Commit And Push

Commit all relevant files.

Do not include unrelated `VERSION` change unless explicitly requested.

Expected status after commit:

```text
 M VERSION
```

## 3. Verification Checklist

Before final output, verify:

- `npm test` includes Vitest.
- Static `.mjs` tests still run.
- New render tests fail if raw Docker parent object is shown.
- New render tests fail if `NVIDIA_REQUIRE_CUDA` appears in default probe summary DOM.
- New render tests fail if clone default uses `runtime.vllm.nvidia-docker` as display name.
- New render tests fail if `model_runtime.port` appears as required/readonly/empty.

## 4. Completion Report Format

Output:

1. Added dependencies/config/scripts.
2. Added or extracted components.
3. Added test files and test names.
4. Historical issues covered by each test.
5. Whether existing orphan TS tests were included or left out.
6. Test command results.
7. Commit ID.
8. Push result.
9. `git status --short`.
