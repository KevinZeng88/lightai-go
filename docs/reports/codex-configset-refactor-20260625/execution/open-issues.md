# ConfigSet Refactor — Open Issues

Date: 2026-06-26

## Active Issues

| ID | Issue | Impact | Blocker? |
|----|-------|--------|----------|
| CFG-001 | Table `inference_backends` vs API `/api/v1/backends` naming | Cosmetic — pre-existing naming convention | No |

## Resolved

| ID | Issue | Resolution |
|----|-------|------------|
| CFG-000 | SGLang capabilities empty in fresh DB | ConfigSet materializes capabilities from catalog YAML into config_set.items["backend.capabilities"] — resolved by schema change |
