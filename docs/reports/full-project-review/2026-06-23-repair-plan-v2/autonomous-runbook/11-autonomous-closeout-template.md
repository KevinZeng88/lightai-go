# Autonomous Closeout Template

> Each batch MUST generate a closeout using this template.
> File: `autonomous-runbook/batch-{X}-closeout.md`

---

```markdown
# Batch {X} Closeout: {Name}

> Date: {YYYY-MM-DD}
> Status: PASS / FAIL / STOPPED

---

## Before Baseline

- **Git SHA**: {before_sha}
- **go build**: PASS/FAIL
- **go test ./internal/server/...**: PASS/FAIL ({N} tests)
- **go test ./internal/agent/...**: PASS/FAIL ({N} tests)
- **cd web && npm run build**: PASS/FAIL
- **cd web && npm test**: PASS/FAIL
- **Golden path**: {list per-flow status}

---

## Changes Made

### Files Created
| File | Purpose |
|------|---------|
| {path} | {purpose} |

### Files Modified
| File | Changes |
|------|---------|
| {path} | {summary} |

### Commits
| SHA | Message |
|-----|---------|
| {sha} | {message} |

---

## After Verification

- **Git SHA**: {after_sha}
- **go build**: PASS/FAIL
- **go test ./internal/server/...**: PASS/FAIL ({N} tests)
- **go test ./internal/agent/...**: PASS/FAIL ({N} tests)
- **cd web && npm run build**: PASS/FAIL
- **cd web && npm test**: PASS/FAIL
- **go test -race**: PASS/FAIL
- **Golden path**: {list per-flow status}
- **New tests**: {count}

---

## Non-Regression Results

| Check | Result | Notes |
|-------|--------|-------|
| {check} | PASS/FAIL | {notes} |

---

## Evidence

| Evidence | Path |
|----------|------|
| Test output | {path} |
| Build output | {path} |
| Git log | {path} |

---

## Not Verified

| Item | Reason |
|------|--------|
| {item} | {reason} |

---

## Stop Conditions

- [ ] Golden Path broken
- [ ] NBR params blocked
- [ ] Default tenant blocked
- [ ] Server cannot reach agent
- [ ] Unrelated working tree changes
- [ ] Tests fail unclear reason
- [ ] Environment dependency missing
- [ ] Out-of-scope change needed
- [ ] Contradicts decisions

**Any triggered?**: YES/NO
**If YES**: {description and recovery action}

---

## Git Status

```
{git status --short output}
```

---

## Push Status

- Pushed: YES/NO
- Remote: {remote}
- Branch: {branch}
```
