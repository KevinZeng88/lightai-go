# Execution Decisions — Runtime Parameter Editing Auto-Complete

> Date: 2026-06-24
> Status: Active — auto-complete mode

---

## 1. Push Strategy

- Push after Batch D audit passes
- Push after Batch E completes
- Push after Batch F final closeout

## 2. Isolated Validation Strategy

- Do NOT depend on any old running instance
- Build server/agent from current repo code
- Create independent temp instance directory
- Use separate data/log/runtime directories
- Use separate ports to avoid conflicts
- Trigger migration
- Execute API / preflight / preview / UI smoke
- Write evidence to report directory
- Clean up temp processes after verification
- Do NOT pollute historical running directories

## 3. E2E Reuse Strategy

- Locate and reuse existing E2E common libraries
- Check `scripts/e2e/lib/model-runtime-common.sh`
- Extend existing runtime/deployment/preflight/runplan smoke scripts
- Do NOT write duplicate logic
- If common lib insufficient, add shared functions

## 4. DB Strategy

- No complex compatibility migration
- V28 migration auto-adds columns
- Tests and E2E use temp DB or temp running directory
- No fallback for old data
- If clean state needed, create new temp instance

## 5. Review Strategy

After Batch F, do one final review covering:
- RunPlan source-of-truth
- Parameter merge semantics
- Disabled tombstone
- Env/args pollution
- Web i18n leakage
- API permission / tenant scope
- Migration / seed
- Test coverage
- E2E evidence
- Docs consistency
