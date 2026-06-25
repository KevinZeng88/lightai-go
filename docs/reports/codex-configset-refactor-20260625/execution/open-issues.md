# ConfigSet Refactor Open Issues

## Status

No Checkpoint A blockers.

## Tracked Non-Blocking Items

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| --- | --- | --- | --- | --- | --- | --- | --- |
| CS-A-001 | Old ConfigSet predecessor fields remain broadly present. | `validation-log.md` old field counts; 328 files with old fields/routes/smoke phrases. | Expected before implementation; later phases must delete or replace them. | DOCUMENTED_FOR_NEXT_CHECKPOINT | Checkpoints B-E | Static `rg` gates and full tests in later checkpoints | Proceed to Checkpoint B after A commit/push |
| CS-A-002 | Backend-name references are broad and need classification. | `validation-log.md`; 589 files contain backend-name terms. | Some are valid catalog/docs/tests; business logic hardcode must be removed later. | DOCUMENTED_FOR_NEXT_CHECKPOINT | Checkpoints B-D | Classify allowed vs forbidden locations; run static hardcode gate | Proceed to Checkpoint B after A commit/push |
| CS-A-003 | Worktree contains pre-existing unrelated modified/untracked files. | `git status --short` in validation log and phase report. | Commit staging must avoid unrelated files. | DOCUMENTED_FOR_NEXT_CHECKPOINT | Commit policy / explicit `git add` | `git diff --cached --name-only` before commit | Proceed with explicit path staging only |
| CS-B-001 | Rejected V29 additive compatibility migration and temporary legacy-column transition. | User correction during Checkpoint B: additive `config_set_json/source_metadata_json` columns plus retained old authority fields are not acceptable. | Any commit with ConfigSet plus legacy authority fields would create dual authority and violate the no-legacy-compatibility rule. | FIXED_IN_WORKTREE | Design docs, execution prompt, commit policy, `internal/server/db/db.go`, API/runtime paths | Fresh schema must use ConfigSet as the only configuration authority; no V29 additive migration, old-column transition, dual-read/write, or fallback commit is allowed. | Updated docs to require fresh-schema clean-state commits; future checkpoints must not commit or push legacy compatibility paths. |

Allowed states for this file during Checkpoint A:

- `DOCUMENTED_FOR_NEXT_CHECKPOINT`
- `BLOCKED`
- `CLOSED`
