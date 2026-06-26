# Missing or Ambiguous Items

| ID | Area | Ambiguity | Why It Matters | Proposed Clarification |
| -- | ---- | --------- | -------------- | ---------------------- |
| A-001 | Workspace baseline | Current dirty files are recorded, but commit isolation is not strict. | AUTORUN could accidentally commit `web/package*.json`, `.mimocode/`, old evidence, or untracked reports. | Add pathspec-limited commit rules and a baseline dirty-file allowlist. |
| A-002 | `/check` endpoint | Batch 1 allows either wrapper or removal. | Different paths have different API/UI/OpenAPI/test impacts. | Default: keep route only as server-to-Agent probe wrapper; ignore request body evidence. |
| A-003 | UI NBR change | Batch 4 allows either removing selector or implementing full flow. | Full flow is larger and can delay fixing the P1 misleading UI. | Default: remove selector now; create explicit NBR change only if time remains in a separate sub-batch. |
| A-004 | Batch 5 scope | One batch includes several high-risk architectural changes. | Failures will be hard to isolate; commit review becomes too large. | Split Batch 5 into Agent credentials, Docker policy, tenant/schema/RBAC. |
| A-005 | Runtime smoke startup | Plan does not define exact server/agent commands. | Smoke may attach to stale local processes or hang. | Add temp data dirs, exact commands, readiness polling, timeouts, and cleanup traps. |
| A-006 | DB/schema changes | Plan allows clean mainline schema changes but no backup/rollback. | Schema work can break local data or tests. | Use temp DB for tests; before any real DB smoke, copy DB to evidence backup path. |
| A-007 | OpenAPI proof | "Update OpenAPI" is not tied to a validator. | YAML can be syntactically valid but semantically stale. | Add stale path scan and sample validation command. |
| A-008 | Playwright availability | Browser smoke is conditional. | UI P1/P2 fixes can pass without real workflow coverage. | If Playwright dependency exists, run it; if browser binary missing, document install blocker and add component tests. |
| A-009 | Docker policy allowlist | "platform admin + policy allow" lacks exact field matrix. | Codex may implement inconsistent gates. | Define option-level default deny matrix and enforce at save, preview, and start. |
| A-010 | Agent token migration | Bootstrap/global token transition is underspecified. | Agent registration can break, or old token may remain too powerful. | Define bootstrap token only for registration; node token required after registration; cross-node reuse negative test. |
| A-011 | Active E2E scripts | Inventory categories are listed but no canonical active list file is required. | Archived scripts may still be invoked by operators. | Generate `docs/testing/active-e2e-scripts.md` and a grep gate for forbidden fields. |
| A-012 | Product-scope status | `INTENTIONALLY_DEFERRED_WITH_OWNER_AND_ACCEPTANCE` is allowed. | Can conflict with the instruction to avoid parking problems. | Use `CLOSED_BY_SCOPE_REDUCTION` or `BLOCKED_BY_EXTERNAL_DEPENDENCY`. |
| A-013 | Push failure | Plan says push and record result. | Network/credential failure can leave local commits not published. | Add push failure status and exact final reporting rules. |
| A-014 | Real smoke cleanup | Docker cleanup criteria are listed but not automated. | Containers or ports can be left behind. | Require trap cleanup and final `docker ps` filtered by run prefix. |
| A-015 | Manifest drift | `manifest.json` exists but is not in required read list or generated output list. | AUTORUN may ignore it or fail to update it when plan docs change. | Include manifest maintenance in Batch 0 and final closeout. |
