# Phase 4 - View-Level Semantics

Date: 2026-07-01

## Semantics

The display selector is productized as a hierarchical filter:

- `常用`: normal fields
- `高级`: normal + advanced fields
- `专家`: normal + advanced + developer/diagnostic fields

Backend projection already enforces this through `fieldVisibleAtView`; the UI now shares one option source from `web/src/utils/configEditDisplay.ts`.

## Applied Consumers

Shared view-level options and help are applied to:

- BackendRuntime page
- NodeBackendRuntime page
- Deployment page

Raw JSON and low-level diagnostics remain gated behind expert/developer diagnostics surfaces.

