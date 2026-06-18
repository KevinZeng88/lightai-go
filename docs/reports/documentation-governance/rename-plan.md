# Documentation Rename And Archive Plan

> Status: CURRENT_REPORT
> Last reviewed: 2026-06-18
> Scope: Documentation governance rename/archive mapping
> Read order: See `docs/CURRENT.md`

| Old path | New path | Reason | Status | Link impact | Action |
| --- | --- | --- | --- | --- | --- |
| `docs/lightai-backend-runtime-runplan-docker-design.md` | `docs/design/backend-runtime-runplan-docker.md` | Current design belongs under `docs/design/` and needs kebab-case name | rename | Current links updated through `docs/CURRENT.md` and `docs/README.md`; old path recorded in link report | `git mv` |
| `docs/model-runtime-node-wizard-design.md` | `docs/design/model-runtime-node-wizard.md` | Current Phase 4 design belongs under `docs/design/` | rename | Current links updated through `docs/CURRENT.md` and `docs/README.md`; old path recorded in link report | `git mv` |
| `docs/reports/backend-runtime-runplan-acceptance-report.md` | `docs/reports/backend-runtime-runplan/acceptance-report.md` | Current acceptance report belongs under topic directory | rename | Current links updated | `git mv` |
| `docs/reports/backend-runtime-runplan-current-state-audit.md` | `docs/reports/backend-runtime-runplan/current-state-audit.md` | Current audit belongs under topic directory | rename | Current links updated | `git mv` |
| `docs/plan/model-runtime-node-wizard-implementation-plan.md` | `docs/plan/phase-4-model-runtime-wizard-implementation.md` | Current plan needs phase-specific kebab-case name | rename | Current plan README points to new path | `git mv` |
| `docs/design/12-model-runtime-serving-design.md` | `docs/archive/superseded/12-model-runtime-serving-design.md` | Superseded by current Phase 4 wizard and BackendRuntime designs | archive | Archived with status header; not current guidance | `git mv` |
| `docs/design/13-backend-runplan-runtime-design.md` | `docs/archive/superseded/13-backend-runplan-runtime-design.md` | Superseded by `docs/design/backend-runtime-runplan-docker.md` | archive | Archived with status header; not current guidance | `git mv` |
| `docs/plan/12-*.md` | `docs/archive/old-plans/12-*.md` | Old staged model-serving plans are no longer current execution guidance | archive | Archived with status header | `git mv` |
| `docs/plan/phase-2f-*.md` | `docs/archive/old-plans/phase-2f-*.md` | Closed Phase 2F planning artifacts | archive | Archived with status header | `git mv` |
| `docs/plan/phase-2g-*.md` | `docs/archive/old-plans/phase-2g-*.md` | Closed Phase 2G planning artifacts | archive | Archived with status header | `git mv` |
| `docs/plan/phase-2h-*.md` | `docs/archive/old-plans/phase-2h-*.md` | Closed Phase 2H planning artifacts | archive | Archived with status header | `git mv` |
| `docs/plan/phase-3-*.md` | `docs/archive/old-plans/phase-3-*.md` | Closed Phase 3 planning artifacts | archive | Archived with status header | `git mv` |
| `docs/plan/backend-runtime-runplan-gap-fix-plan.md` | `docs/archive/old-plans/backend-runtime-runplan-gap-fix-plan.md` | Superseded by acceptance and closeout reports | archive | Archived with status header | `git mv` |
| `docs/RC1_CODEX_REVIEW_TRACKING.md` | `docs/archive/historical/rc1-codex-review-tracking.md` | Historical RC1 review closure record | archive | Archived with status header | `git mv` |
| `docs/RC1_REVIEW_FIX_PLAN.md` | `docs/archive/historical/rc1-review-fix-plan.md` | Historical RC1 review plan | archive | Archived with status header | `git mv` |
| `docs/archive/RC1_PATCH_TEST.md` | `docs/archive/historical/rc1-patch-test.md` | Historical patch test, renamed to kebab-case | archive | Archived with status header | `git mv` |
| `docs/reports/phase-3/` | `docs/reports/archive/phase-3/` | Closed Phase 3 report/evidence set | archive | Archived directory README explains status; Markdown files have archive headers | `git mv` directory |
| `docs/reports/rc2/` | `docs/reports/archive/rc2/` | Closed RC2 report/evidence set | archive | Archived directory README explains status; Markdown files have archive headers | `git mv` directory |
| `docs/reports/full-project-review/` | `docs/reports/archive/full-project-review/` | Historical full-project review set | archive | Archived directory README explains status; Markdown files have archive headers | `git mv` directory |
| `docs/reports/rc2-audit-open-issues-closeout.md` | `docs/reports/archive/rc2-audit-open-issues-closeout.md` | Closed RC2 closeout belongs in report archive | archive | Archived with status header | `git mv` |
| `docs/00-*.md` through `docs/10-*.md` | unchanged | Existing `AGENTS.md` references these exact paths; keeping avoids breaking agent bootstrap instructions | keep | Classified as REFERENCE in `docs-index.md`; not current Phase 4 entrypoint | keep |
| `docs/REVIEW-GPUSTACK-*.md` | unchanged | Existing `AGENTS.md` references these exact paths when GPUStack is mentioned | keep | Classified as REFERENCE; not current implementation guidance | keep |
| E2E evidence directories | unchanged | Timestamped evidence paths are already under topic report directories | keep | Classified as EVIDENCE | keep |

No current design or acceptance report was deleted. Historical evidence was moved, not removed.
