# Progress Reporting and Per-Checkpoint Commit/Push Policy

## 1. Purpose

This file defines how Codex must report progress, write durable evidence, commit work, and push after each checkpoint during the ConfigSet / ConfigItem refactor.

This policy is mandatory. Do not wait until final closeout to commit or report.

## 2. Execution Status Directory

Create and maintain:

```text
docs/reports/codex-configset-refactor-20260625/execution/
```

Required files:

```text
execution-status.md
validation-log.md
open-issues.md
commit-log.md
phase-01-design-and-inventory.md
phase-02-registry-catalog-loader.md
phase-03-db-schema-and-seed-removal.md
phase-04-copy-on-create-renderer.md
phase-05-api-ui-refactor.md
phase-06-validation-and-smoke.md
```

## 3. Checkpoints

Use these checkpoints:

```text
Checkpoint A: design document + inventory + old-structure deletion list
Checkpoint B: config registry / backend catalog loader + db.go seed hardcode removal
Checkpoint C: DB schema rebuild + ConfigSet copy-on-create
Checkpoint D: renderer + RunPlan / AgentRunSpec / DockerSpec
Checkpoint E: API/UI refactor + stale documentation archive
Checkpoint F: full validation + fresh DB + three runtime platform-chain smoke + final closeout
```

## 4. Per-Checkpoint Report Requirements

After each checkpoint, update the corresponding phase report.

Each phase report must include:

```text
1. Phase objective
2. Actual changed files
3. Deleted old structures / APIs / fields
4. New structures / APIs / files added
5. Tests executed with exact commands and exact results
6. Evidence files or logs
7. Remaining issues, if any
8. Whether remaining issues are blockers
9. Next phase
10. git status --short raw output
```

Also update:

```text
execution-status.md
validation-log.md
open-issues.md
commit-log.md
```

## 5. Per-Checkpoint Commit and Push

Each checkpoint must end with a commit and push.

Do not use:

```bash
git add .
```

Use explicit staging only:

```bash
git status --short
git diff --stat
git diff --check
git add <explicit files only>
git commit -m "<checkpoint-specific message>"
git push
```

Suggested commit messages:

```text
docs: add configset refactor baseline
refactor: add config registry catalog loader
refactor: replace catalog seeds with configset snapshots
refactor: render runplans from configsets
refactor: migrate api ui to configsets
test: validate configset runtime smoke
```

## 6. Terminal Output Format

After each checkpoint, print exactly this structure:

```text
PHASE_STATUS: <phase-name>
RESULT: PASS | FAIL | BLOCKED
SUMMARY:
- changed_files:
- deleted_old_structures:
- new_structures:
- tests_run:
- validation_result:
- remaining_issues:
- next_phase:
GIT:
- latest_commit:
- push_result:
- git_status_short:
REPORTS:
- phase_report:
- validation_log:
- open_issues:
- commit_log:
```

## 7. Do Not Stop for Ordinary Failures

Do not stop for ordinary issues such as:

```text
compile failure
test failure
script failure
documentation mismatch
runtime smoke bug
API mismatch
UI build issue
catalog validation failure
fresh DB failure
```

Fix them and rerun validation.

Only stop with `RESULT: BLOCKED` when one of these is true:

```text
1. Docker/GPU/image/model external resource is unavailable and command-level evidence is provided.
2. Git push fails due to credentials or network.
3. A change would delete user data not covered by the agreed fresh-DB policy.
4. The required change is clearly outside the ConfigSet/ConfigItem refactor scope.
```

## 8. Final Output Requirements

Final output must include:

```text
FINAL_STATUS: CONFIGSET_REFACTOR_COMPLETE | CONFIGSET_REFACTOR_BLOCKED

DESIGN_DOC:
EXECUTION_REPORTS:
ARCHIVED_DOCS:

REMOVED_OLD_FIELDS:
REMOVED_OLD_APIS:
REMOVED_DB_SEEDS:
REMOVED_COMPATIBILITY_PATHS:

REGISTRY_CATALOG_LOADER:
CONFIG_REGISTRY:
BACKEND_CATALOG:

COPY_ON_CREATE_TESTS:
RENDERER_TESTS:
EXTRA_ARGS_TESTS:
FRESH_DB_VALIDATION:

RUNTIME_SMOKE:
- vLLM:
- SGLang:
- llama.cpp:

FULL_TESTS:
- go test ./...
- go build ./cmd/server/...
- go build ./cmd/agent/...
- cd web && npm test
- cd web && npm run build

COMMITS:
PUSH_RESULT:
GIT_STATUS_SHORT:
OPEN_TASKS:
```

Do not output:

```text
Complete
FINAL_CLOSEOUT_ACCEPTED
CONFIGSET_REFACTOR_COMPLETE
```

until all required validation, runtime smoke evidence, commits, pushes, and execution reports are complete.
