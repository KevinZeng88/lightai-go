# Runtime Config Field Display Fix — Complete Index

This document set covers the runtime template detail / config field display fix batches.

## Files

| File | Purpose |
|------|---------|
| `00-index.md` | This index |
| `01-fix-boundary-and-acceptance.md` | P0-1 / P0-2 / P0-3 fix boundary and acceptance (original probe fix) |
| `02-codex-review-prompt.md` | Codex light audit prompt |
| `03-claude-execution-prompt.md` | Initial Claude execution prompt (P0-1/2/3) |
| `04-codex-review-acceptance.md` | Codex audit conclusions adopted |
| `05-closeout.md` | **Closeout document** — all three batches: root causes, changes, tests, commits, DB rebuild |
| `06-mhtml-config-field-review.md` | MHTML snapshot review — object child field parent-value leak |
| `07-config-field-display-design.md` | Batch 3 implementation design — docker sub-field value resolution, widget overrides, structured display |
| `08-claude-execution-prompt.md` | Batch 3 execution prompt — config field display follow-up |

## Execution History

| Batch | Commit | Scope |
|-------|--------|-------|
| 1 | `ee35b5b` | P0-1: config edit envelope unwrap; P0-2: clone naming/version; P0-3: probe evidence |
| 2 | `7671a3e` | Closeout: deployment NBR display_name, level4 i18n, DB seed test |
| 3 | `d6e0523` | Config field display: docker sub-field value leak fix + widget overrides fix |

See `05-closeout.md` for full root causes, all changes, test results, and acceptance verification per batch.
