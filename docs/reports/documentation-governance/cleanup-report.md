# Documentation Governance Cleanup Report

> Status: CURRENT_REPORT
> Last reviewed: 2026-06-18
> Scope: Documentation governance closeout
> Read order: See `docs/CURRENT.md`

## 1. Goal

This round consolidated scattered, outdated, duplicated, and potentially misleading documentation under `docs/`. The cleanup establishes a current entrypoint, separates current guidance from historical evidence, archives superseded plans/reports, and records all findings in formal documents.

## 2. Branch And Commit

```text
Branch: phase-4-model-runtime-wizards
Baseline commit at start: 89bdf68 fix: add node model root policy and harden wizard flow
Documentation governance commit: this report is included in `docs: consolidate current guidance and archive stale reports`; use `git log -1` for the final hash.
```

The unrelated user-owned `VERSION` modification was present before this task and was not touched, staged, or committed.

## 3. Inventory

```text
Total docs files after cleanup: 194
Markdown files after cleanup: 112
Archived Markdown files: 54
Archived files total: 74
```

Generated inventory:

```text
docs/reports/documentation-governance/docs-file-inventory.txt
docs/reports/documentation-governance/inventory-summary.md
```

## 4. Directory Structure Changes

Current design documents are now under:

```text
docs/design/
```

Current implementation plans are now under:

```text
docs/plan/
```

Current acceptance/closeout reports are now under topic directories:

```text
docs/reports/backend-runtime-runplan/
docs/reports/model-runtime-node-wizard/
docs/reports/documentation-governance/
```

Historical material is now under:

```text
docs/archive/
docs/reports/archive/
```

## 5. Rename And Archive Mapping

Full mapping:

```text
docs/reports/documentation-governance/rename-plan.md
```

Key moves:

| Old path | New path |
| --- | --- |
| `docs/lightai-backend-runtime-runplan-docker-design.md` | `docs/design/backend-runtime-runplan-docker.md` |
| `docs/model-runtime-node-wizard-design.md` | `docs/design/model-runtime-node-wizard.md` |
| `docs/reports/backend-runtime-runplan-acceptance-report.md` | `docs/reports/backend-runtime-runplan/acceptance-report.md` |
| `docs/reports/backend-runtime-runplan-current-state-audit.md` | `docs/reports/backend-runtime-runplan/current-state-audit.md` |
| `docs/plan/model-runtime-node-wizard-implementation-plan.md` | `docs/plan/phase-4-model-runtime-wizard-implementation.md` |
| `docs/design/12-model-runtime-serving-design.md` | `docs/archive/superseded/12-model-runtime-serving-design.md` |
| `docs/design/13-backend-runplan-runtime-design.md` | `docs/archive/superseded/13-backend-runplan-runtime-design.md` |
| `docs/reports/phase-3/` | `docs/reports/archive/phase-3/` |
| `docs/reports/rc2/` | `docs/reports/archive/rc2/` |
| `docs/reports/full-project-review/` | `docs/reports/archive/full-project-review/` |

## 6. Current Documents

Current entrypoints:

```text
docs/CURRENT.md
docs/README.md
docs/PHASE-STATUS.md
```

Current design:

```text
docs/design/backend-runtime-runplan-docker.md
docs/design/model-runtime-node-wizard.md
```

Current reports:

```text
docs/reports/backend-runtime-runplan/acceptance-report.md
docs/reports/backend-runtime-runplan/open-issues-closeout.md
docs/reports/model-runtime-node-wizard/acceptance-report.md
docs/reports/model-runtime-node-wizard/full-run-chain-review.md
docs/reports/model-runtime-node-wizard/open-issues-closeout.md
docs/reports/documentation-governance/open-issues.md
```

Complete index:

```text
docs/reports/documentation-governance/docs-index.md
```

## 7. Archived Documents

Archived directories:

```text
docs/archive/historical/
docs/archive/old-plans/
docs/archive/superseded/
docs/reports/archive/phase-3/
docs/reports/archive/rc2/
docs/reports/archive/full-project-review/
```

Archived documents were not deleted. They were moved and marked with archive headers or covered by archive README rules.

## 8. Superseded Documents

| Superseded document | Replaced by |
| --- | --- |
| `docs/archive/superseded/12-model-runtime-serving-design.md` | `docs/design/model-runtime-node-wizard.md` |
| `docs/archive/superseded/13-backend-runplan-runtime-design.md` | `docs/design/backend-runtime-runplan-docker.md` |
| Old `docs/plan/12-*.md` phase plans | `docs/plan/phase-4-model-runtime-wizard-implementation.md` and topic acceptance reports |
| Old Phase 3 / RC2 reports | Current topic reports under `docs/reports/backend-runtime-runplan/` and `docs/reports/model-runtime-node-wizard/` |

## 9. Fixed Contradictions

| Issue | Resolution |
| --- | --- |
| `docs/README.md` still described Phase 0-2 as the current window | Replaced with current Phase 4 entrypoint and reading order |
| `docs/PHASE-STATUS.md` still described RC2 as current | Rewritten to reflect BackendRuntime acceptance and Phase 4 scheme B |
| Current design documents referenced old suggested paths | Updated to current `docs/design/` paths |
| BackendRuntime current-state audit referenced old design path | Updated to `docs/design/backend-runtime-runplan-docker.md` |
| Phase 4 current-state audit referenced stale final commit `50a25a5` | Updated to `89bdf68` |
| Old plans and reports remained in visible current paths | Moved to archive directories with status rules |
| Archived documents could still display `Status: In progress` | Added archive headers so current entrypoint wins |

## 10. Automatically Fixed Findings

| Finding | Fix |
| --- | --- |
| Missing `docs/CURRENT.md` | Created current state entrypoint |
| Missing directory README files | Added `docs/design/README.md`, `docs/plan/README.md`, `docs/reports/README.md`, `docs/archive/README.md`, `docs/reports/archive/README.md` |
| Missing documentation index | Added `docs/reports/documentation-governance/docs-index.md` |
| Missing formal documentation governance issue register | Added `docs/reports/documentation-governance/open-issues.md` |
| Missing rename/archive mapping | Added `docs/reports/documentation-governance/rename-plan.md` |
| Missing risk/link inventories | Generated `docs-risk-keywords.txt` and `docs-internal-links.txt` |

## 11. Findings Not Fixed In Code

No runtime code issues were found during this documentation cleanup.

Product-depth items were not implemented in this round because they are P2 or external validation work and are formally tracked:

```text
docs/reports/backend-runtime-runplan/open-issues-closeout.md
docs/reports/model-runtime-node-wizard/open-issues-closeout.md
docs/reports/documentation-governance/open-issues.md
```

## 12. Open Issues Closeout

Documentation governance open issues:

```text
docs/reports/documentation-governance/open-issues.md
```

Status:

```text
No documentation-governance blocker remains open.
No chat-only findings remain.
```

## 13. E2E

E2E was not rerun in this round.

Reason:

```text
This task changed documentation only and did not affect model wizard, Runtime wizard, root policy, preflight, RunPlan preview, Docker start, logs, or cleanup code paths.
```

Existing current E2E evidence remains:

```text
docs/reports/model-runtime-node-wizard/e2e-run-20260618-115214/
```

## 14. Verification

Required for documentation-only change:

```text
git diff --check: PASS
git diff --cached --check: PASS
```

No Go/Web/script/config files were modified in this round.

## 15. VERSION

`VERSION` had a pre-existing user-owned modification and was not touched, staged, or committed.

## 16. Git Status

Final git status is recorded in the final response after commit and push.

## 17. Closure

No chat-only findings remain.

All discovered problems are either:

```text
CLOSED in this cleanup,
or DOCUMENTED in formal open issue / closeout files.
```
