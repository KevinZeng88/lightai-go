Please implement the P0-B runtime/config/probe rendered UI test gate on the current `main` branch. Do not create a new branch.

Read first:

- `docs/reports/phase-3/test-inventory-and-gap-review.md`
- `docs/reports/phase-3/runtime-config-display-probe-fix/05-closeout.md`
- `docs/reports/phase-3/p0b-vitest-render-gate/11-p0b-vitest-render-gate-design.md`
- `docs/reports/phase-3/p0b-vitest-render-gate/12-p0b-vitest-render-gate-implementation-plan.md`

Goal:

Introduce Vitest + Vue Test Utils as the best P0-B rendered UI test gate for the actual runtime/config/probe regressions already observed. Do not add one test per page. Do not do a full frontend test rewrite.

Required scope:

1. Add minimal Vitest setup and ensure `npm test` runs Vitest.
2. Add `ConfigEditView.render.test.ts` for rendered ConfigEdit sections, Docker subfields, structured widgets, raw JSON collapse, and vendor device env placeholder absence.
3. Add or extract `ProbeSummaryView.vue`, then add `ProbeSummaryView.render.test.ts` to ensure probe summary is visible by default and raw Docker image env is hidden unless diagnostics are expanded.
4. Add `BackendRuntimesPage.integration.test.ts` for readonly detail mode, edit/save/cancel, clone default display name, and wildcard version display.
5. Add `RunnerConfigsPage.integration.test.ts` for probe summary integration and canonical `service.container_port` display while preventing empty required `model_runtime.port` display.

Limits:

- Do not introduce Playwright.
- Do not depend on server, agent, Docker, GPU, or model files.
- Do not rewrite frontend architecture.
- Do not remove existing `web/tests/*.mjs` tests.
- Do not process `VERSION`.
- If old `web/src/**/__tests__/*.ts` tests are incompatible, do not rewrite them broadly; document whether they were included or left out.

Run:

```bash
go test ./internal/server/...
go test ./internal/agent/...
cd web && npm test
cd web && npm run build
```

Commit and push. Output:

1. Added dependencies/config/scripts.
2. Added or extracted components.
3. Added test files and test names.
4. Which historical issue each test protects.
5. Whether old orphan TS tests were included or left out.
6. Test results.
7. Commit ID.
8. Push result.
9. `git status --short`.
