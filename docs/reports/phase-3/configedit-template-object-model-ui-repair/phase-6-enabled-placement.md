# Phase 6 - Enabled Placement Stability

Date: 2026-07-01

## Behavior

Field group placement uses the load-time enabled snapshot for non-expert fields:

- initial load: non-expert fields with `original_enabled=true` appear under 已启用参数
- editing: checking or unchecking does not reorder the field immediately
- save/reload: backend returns a fresh ConfigEdit view and `original_enabled` reflects the saved state
- unchecked fields return to their normal/advanced/expert group after reload
- expert/security/raw fields remain in 专家参数 even when enabled, preserving the raw/diagnostic boundary

## Implementation

`displayGroupForField` uses:

`field.original_enabled ?? field.enabled`

This preserves edit-session stability without adding page-local state or hardcoded per-page sorting.

## Evidence

`web/src/utils/__tests__/configEditView.test.ts` verifies enabled-first load ordering, no live reorder on toggle, enabled placement after reload, and return to original tier group after reload.
