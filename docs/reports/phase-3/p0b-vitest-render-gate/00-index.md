# P0-B Vitest Render Gate Docs

Date: 2026-06-28
Project: LightAI Go
Scope: Frontend rendered runtime/config/probe regression tests

## Purpose

This document set defines the controlled design and execution boundary for P0-B rendered UI tests.

The goal is to introduce Vitest + Vue Test Utils as a regular frontend test gate and cover the actual runtime/config/probe UI regressions found during manual validation, without expanding into a full page-by-page testing rewrite.

## Files

| File | Purpose |
| --- | --- |
| `11-p0b-vitest-render-gate-design.md` | Full design, boundaries, test selection, acceptance criteria. |
| `12-p0b-vitest-render-gate-implementation-plan.md` | Step-by-step implementation plan for Claude. |
| `13-p0b-claude-execution-prompt.md` | Short execution prompt pointing Claude to the design and plan. |

## Execution Order

1. Read `docs/reports/phase-3/test-inventory-and-gap-review.md`.
2. Read `docs/reports/phase-3/runtime-config-display-probe-fix/05-closeout.md`.
3. Read this document set.
4. Implement only the P0-B rendered test gate described here.
5. Run required tests.
6. Commit and push.

## Non-goals

- Do not add page tests for every page.
- Do not introduce Playwright for this batch.
- Do not depend on server, agent, Docker, GPU, or model files.
- Do not rewrite the frontend architecture.
- Do not process `VERSION`.
