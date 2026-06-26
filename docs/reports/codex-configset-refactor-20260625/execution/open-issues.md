# ConfigSet Refactor — Open Issues

Date: 2026-06-26

## SGLang Backend Capabilities — RESOLVED BY CONFIGSET REFACTOR

Issue: Prior to ConfigSet refactor, `sglang-0.4.6-compatible` had empty
`capabilities_json` in fresh DB. This was caused by the old catalog reload's
`ON CONFLICT DO UPDATE` behavior with the array-format capabilities in the
legacy seed data.

Resolution: ConfigSet refactor replaced `capabilities_json` with `config_set_json`.
Backend versions now materialize capabilities from `configs/backend-catalog/versions/`
and `configs/config-registry/` into proper ConfigItem entries under
`config_set.items["backend.capabilities"]`. Fresh DB verified:
- vLLM v0.23.0: capabilities present ✅
- SGLang 0.4.6-compatible: capabilities present ✅
- SGLang v0.5.13.post1: capabilities present ✅

Impact: None — resolved by schema change.
Not a blocker for ConfigSet closeout.

## Table Naming: inference_backends vs /api/v1/backends

Issue: DB table is `inference_backends`; API route is `/api/v1/backends`.
Status: Pre-existing. ConfigSet refactor did not change table names.
Not a blocker for ConfigSet closeout. Fix would require broader rename.
