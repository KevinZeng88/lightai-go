# ConfigSet Refactor Validation Log

All commands were run from `/home/kzeng/projects/ai-platform-study/lightai-go`.

| Command | Result | Summary |
| --- | --- | --- |
| `sed -n ... docs/reports/codex-configset-refactor-20260625/00-index.md` | PASS | Read handoff index and principles. |
| `sed -n ... 01-current-code-findings.md` | PASS | Read current findings: db.go seed literals, V27 repair, YAML mirror drift, legacy fields. |
| `sed -n ... 02-configset-configitem-design.md` | PASS | Read ConfigSet / ConfigItem design. |
| `sed -n ... 03-implementation-plan.md` | PASS | Read phased implementation plan. |
| `sed -n ... 04-validation-and-acceptance.md` | PASS | Read validation matrix and acceptance rules. |
| `sed -n ... 05-codex-execution-prompt.md` | PASS | Read execution prompt and hard constraints. |
| `sed -n ... 06-progress-reporting-and-commit-policy.md` | PASS | Read mandatory progress/commit/push policy. |
| `git status --short` | PASS | Worktree has unrelated modified `web/package*.json`, untracked ConfigSet docs, project-wide reports, and historical E2E evidence. |
| `git log --oneline -20` | PASS | Latest commit before Checkpoint A work: `8c0d31a test: add catalog seed drift detection and capabilities format checks`. |
| `diff -u docs/reports/codex-configset-refactor-20260625/02-configset-configitem-design.md docs/design/catalog-configset-and-runtime-snapshot.md` | PASS | No diff. Design document is synchronized with the handoff design. |
| `wc -l ...02-configset-configitem-design.md ...catalog-configset-and-runtime-snapshot.md` | PASS | Both files have 581 lines. |
| `rg -l <old-structure-patterns> ... \| wc -l` | PASS | 328 files contain old fields/routes/smoke phrases targeted by the refactor. |
| `rg -l <backend-name-patterns> ... \| wc -l` | PASS | 589 files contain backend-name related terms; these require classification in later checkpoints. |
| `rg -n "seedBuiltInBackends|seedTargetBackendCatalog|repairBackendCapabilitiesV27" internal/server/db internal --glob '*.go'` | PASS | Found calls/definitions in `internal/server/db/db.go`: lines 209, 214, 986, 1220, 1229, 1230, 1352, 1597, 1599, 1611, 1613. |
| `rg -n <old-field-patterns> internal/server/db internal/server/api internal/server/runplan web/src scripts configs/backend-catalog docs/api` | PASS with truncation | Confirmed old fields remain in db/api/runplan/web/scripts/catalog/OpenAPI; output truncated after 220 lines. |
| `git diff --stat` | PASS | Only tracked diff before Checkpoint A reports was pre-existing `web/package*.json` changes: 2 files, 56 insertions, 7 deletions. |
| `git diff --check` | PASS | No whitespace errors reported in tracked diffs. |
| `git diff --cached --check` | FIXED | Initial staged check found one trailing whitespace in `01-current-code-findings.md`; removed it, re-staged, and reran successfully. |

## Old Field Hit Counts

| Pattern | Files |
| --- | ---: |
| `capabilities_json` | 69 |
| `parameter_schema_json` | 31 |
| `parameter_values_json` | 63 |
| `env_json` | 74 |
| `ports_json` | 13 |
| `volumes_json` | 10 |
| `devices_json` | 9 |
| `health_check_json` | 24 |
| `resource_controls_json` | 5 |
| `parameters_json` | 180 |
| `default_args_json` | 37 |
| `parameter_defs_json` | 25 |
| `default_backend_params_json` | 20 |
| `default_images_json` | 24 |
| `image_candidates_json` | 15 |
| `docker_options_json` | 16 |
| `model_mount_json` | 38 |
| `seedBuiltInBackends` | 9 |
| `seedTargetBackendCatalog` | 10 |
| `repairBackendCapabilitiesV27` | 5 |
| `/check` | 100 |
| `preflight PASS only` | 3 |
| `task claimed only` | 3 |
| `image/model present only` | 3 |

## Validation Result

PASS for Checkpoint A documentation/inventory scope before commit.
