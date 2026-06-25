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
| `rg -l <backend-name-patterns> ... \| wc -l` | PASS | 589 files contain backend-name related terms; these are classified by checkpoint-specific static gates. |
| `rg -n "seedBuiltInBackends|seedTargetBackendCatalog|repairBackendCapabilitiesV27" internal/server/db internal --glob '*.go'` | PASS | Found calls/definitions in `internal/server/db/db.go`: lines 209, 214, 986, 1220, 1229, 1230, 1352, 1597, 1599, 1611, 1613. |
| `rg -n <old-field-patterns> internal/server/db internal/server/api internal/server/runplan web/src scripts configs/backend-catalog docs/api` | PASS with truncation | Confirmed old fields remain in db/api/runplan/web/scripts/catalog/OpenAPI; output truncated after 220 lines. |
| `git diff --stat` | PASS | Only tracked diff before Checkpoint A reports was pre-existing `web/package*.json` changes: 2 files, 56 insertions, 7 deletions. |
| `git diff --check` | PASS | No whitespace errors reported in tracked diffs. |
| `git diff --cached --check` | FIXED | Initial staged check found one trailing whitespace in `01-current-code-findings.md`; removed it, re-staged, and reran successfully. |
| Checkpoint B correction | FIXED_IN_WORKTREE | Rejected V29 additive compatibility migration. Rejected temporary legacy-column transition. Updated docs to require fresh-schema clean-state commits. Future checkpoints must not commit or push legacy compatibility paths. |
| Checkpoint B migration-stack correction | FIXED_IN_WORKTREE | Expanded DB cleanup scope from V29 additive migration rejection to full V1->V28 historical compatibility migration audit and clean-schema replacement. Added `db-migration-compatibility-audit.md` and static validation gates for active DB initialization. |
| `git show HEAD:internal/server/db/db.go \| rg -n "func \\(db \\*DB\\) migrateV[0-9]+|..."` | PASS | Used the committed pre-refactor `db.go` as audit input. Confirmed historical `migrateV1` through `migrateV28`, `seedBuiltInBackends`, `seedTargetBackendCatalog`, `repairBackendCapabilitiesV27`, and `normalizeLegacyBackendCatalogIDs` existed and required classification/removal. |
| `rg -n "Fresh DB clean schema|V1->V28|migrateV1|migration compatibility audit|Clean-State Commit Policy|schema_version may remain" ...` | PASS | Confirmed the clean-schema/V1->V28 migration-stack rules were written into design, implementation, validation, execution prompt, commit policy, and migration audit documents. |
| `rg -n "func \\(db \\*DB\\) migrateV[0-9]+|migrateV[0-9]+\\(" internal/server/db \|\| true` | PASS | Current worktree active DB package has no `migrateVx` functions or calls. |
| `rg -n "seedBuiltInBackends|seedTargetBackendCatalog|repairBackendCapabilitiesV27|normalizeLegacyBackendCatalogIDs" internal/server/db internal/server \|\| true` | PASS | Current worktree active code has no old catalog seed/repair/normalize functions. |
| `rg -n "ALTER TABLE .* ADD COLUMN|Backfill|backfill|repair|normalizeLegacy|compat|legacy" internal/server/db internal/server \|\| true` | ATTENTION | No active old authority-field ADD COLUMN chain was found in `internal/server/db`. Output still includes clean-schema legacy-DB rejection text in `db.go` and unrelated existing compatibility references in other modules/tests; these remain implementation-phase cleanup/classification targets. |
| `git diff --check -- <documentation files>` | PASS | No whitespace errors in the documentation-only migration-stack correction. |
| `go test ./internal/server/api -count=1` | PASS | API package passes after ConfigSet clean schema, catalog loader, NBR/deployment copy-on-create, and stale test fixture updates. |
| `go test ./internal/server/api ./internal/server/runplan -count=1` | PASS | API and RunPlan packages pass after ConfigSet-derived runtime data and RunPlan JSON tag cleanup. |
| `go test ./internal/server/catalog ./internal/server/db -count=1` | PASS | Config registry/catalog loader tests pass; DB package builds with fresh-schema initializer. |
| `go build ./cmd/server/... && go build ./cmd/agent/...` | PASS | Server and Agent binaries build. |
| `go test ./...` | PASS | All Go packages pass under the fresh ConfigSet schema. |
| `rg -n "config_snapshot_json|parameter_schema_json|parameter_values_json|image_name|docker_json|default_env_json|capabilities_json|capability_sources_json|parameter_defaults_json|default_args_json|parameter_defs_json|default_backend_params_json|default_images_json|image_candidates_json|docker_options_json|model_mount_json|seedBuiltInBackends|seedTargetBackendCatalog|repairBackendCapabilitiesV27|normalizeLegacyBackendCatalogIDs|migrateV[0-9]+" internal/server/api internal/server/runplan internal/server/db` | PASS | No exact old authority field, old catalog seed/repair, or `migrateVx` hits remain in active API/RunPlan/DB scope. |
| `git diff --check` | PASS | No whitespace errors in current tracked diff. |

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

PASS for combined Checkpoint B/C clean-state implementation scope before commit. The implementation uses a fresh DB schema, ConfigSet catalog loader, and ConfigSet copy-on-create paths without active V1->V28 migration replay or old authority-field fallback in API/RunPlan/DB.
