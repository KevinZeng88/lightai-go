# Documentation Governance Open Issues

> Status: CURRENT_REPORT
> Last reviewed: 2026-06-18
> Scope: Documentation governance findings and links to formal product closeouts
> Read order: See `docs/CURRENT.md`

| ID | Title | Source | Status | Owner area | Resolution | Linked doc |
| --- | --- | --- | --- | --- | --- | --- |
| DOC-GOV-001 | `docs/README.md` described old Phase 0-2 reading order as current | Documentation inventory | CLOSED | Docs | Replaced with current documentation entrypoint and Phase 4 reading order | `docs/README.md` |
| DOC-GOV-002 | `docs/PHASE-STATUS.md` still described RC2 as current | Documentation inventory | CLOSED | Docs | Rewrote phase status to reflect `89bdf68`, BackendRuntime acceptance, and Phase 4 scheme B | `docs/PHASE-STATUS.md` |
| DOC-GOV-003 | Current design documents were left in docs root with old suggested paths | Documentation inventory | CLOSED | Docs | Moved to `docs/design/backend-runtime-runplan-docker.md` and `docs/design/model-runtime-node-wizard.md` | `docs/reports/documentation-governance/rename-plan.md` |
| DOC-GOV-004 | BackendRuntime reports were split between reports root and topic directory | Documentation inventory | CLOSED | Docs | Moved acceptance/current-state audit into `docs/reports/backend-runtime-runplan/` | `docs/reports/documentation-governance/rename-plan.md` |
| DOC-GOV-005 | Old Phase/RC reports and plans remained in visible paths | Documentation inventory | CLOSED | Docs | Archived old plans and historical reports under `docs/archive/` and `docs/reports/archive/` | `docs/reports/documentation-governance/rename-plan.md` |
| DOC-GOV-006 | Historical archived documents could still display `Status: In progress` | Archive status header pass | CLOSED | Docs | Added archive header to archived Markdown files | `docs/archive/README.md`, `docs/reports/archive/README.md` |
| DOC-GOV-007 | Phase 4 current-state audit referenced stale final commit `50a25a5` | Current report review | CLOSED | Docs | Updated report to `89bdf68` | `docs/reports/model-runtime-node-wizard/current-state-audit.md` |
| DOC-GOV-008 | Model consistency deep comparison remains product-depth work | Phase 4 closeout | DOCUMENTED | Model runtime | Tracked as P2 documented blocker, not a documentation blocker | `docs/reports/model-runtime-node-wizard/open-issues-closeout.md` |
| DOC-GOV-009 | MetaX real hardware validation remains unavailable | BackendRuntime closeout | DOCUMENTED | Runtime vendor validation | Tracked as external validation required; do not mark MetaX ready without hardware evidence | `docs/reports/backend-runtime-runplan/open-issues-closeout.md` |
| DOC-GOV-010 | Huawei vendor adapter is not implemented | BackendRuntime closeout | DOCUMENTED | Runtime vendor adapter | Tracked as template-only/future adapter work; do not mark Huawei ready | `docs/reports/backend-runtime-runplan/open-issues-closeout.md` |
| DOC-GOV-011 | GPU index mapping real multi-GPU validation needs more evidence | Phase 4/Runtime review | DOCUMENTED | GPU placement/runtime validation | Tracked as future validation work, not a current blocker for single-node NVIDIA E2E | `docs/reports/model-runtime-node-wizard/open-issues-closeout.md` |
| DOC-GOV-012 | Runtime edit UX depth can improve | Phase 4 full-chain review | DOCUMENTED | Web Runtime UX | Tracked as P2 product work | `docs/reports/model-runtime-node-wizard/open-issues-closeout.md` |
| DOC-GOV-013 | Backend Catalog productization depth can improve | Documentation governance review | DOCUMENTED | Backend Catalog UX | Tracked as product-depth roadmap item; current catalog API/display remains usable | `docs/plan/future-roadmap.md` |
| DOC-GOV-014 | Node Runtime tab depth can improve | Phase 4 full-chain review | DOCUMENTED | Web Node Runtime UX | Tracked as P2 product work | `docs/reports/model-runtime-node-wizard/open-issues-closeout.md` |

No documentation-governance blocker remains open.
