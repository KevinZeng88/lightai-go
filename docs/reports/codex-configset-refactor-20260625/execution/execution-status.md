# ConfigSet Refactor Execution Status

## Current Checkpoint

| Field | Value |
| --- | --- |
| Current checkpoint | Checkpoint D |
| Phase report | `phase-02-03-clean-schema-and-configset-copy-on-create.md` |
| Status | Checkpoints B and C implemented in one clean-state commit range; continuing to renderer / RunPlan / AgentRunSpec / DockerSpec |
| Branch | `main` |
| Design document | `docs/design/catalog-configset-and-runtime-snapshot.md` |

## Checkpoint Status

| Checkpoint | Scope | Status | Evidence |
| --- | --- | --- | --- |
| A | design document + inventory + old-structure deletion list | PASS committed/pushed | `phase-01-design-and-inventory.md`, `validation-log.md`, commit `1886f0f` |
| B | config registry / backend catalog loader + db.go seed hardcode removal | PASS in worktree; commit pending | Added `configs/config-registry/items.yaml`; added `internal/server/catalog`; removed active db.go hardcoded catalog seed/migration replay path. |
| C | DB schema rebuild + ConfigSet copy-on-create | PASS in worktree; commit pending | Fresh schema uses ConfigSet/source metadata authority for Backend, BackendVersion, BackendRuntime, NodeBackendRuntime, Deployment, ModelArtifact capability set. API tests verify NBR/deployment copy-on-create boundaries. |
| D | renderer + RunPlan / AgentRunSpec / DockerSpec | IN_PROGRESS | RunPlan currently consumes ConfigSet-derived runtime data; Checkpoint D renderer and Checkpoint E static/API/UI cleanup continue. |
| E | API/UI refactor + stale documentation archive | NOT_STARTED | Not started per instruction |
| F | full validation + fresh DB + three runtime platform-chain smoke + final closeout | NOT_STARTED | Not started per instruction |

## Current Working Tree Notes

- The worktree had pre-existing unrelated modified files: `web/package.json`, `web/package-lock.json`.
- The worktree had substantial pre-existing untracked report/evidence directories.
- All Checkpoint B+ staging must use explicit paths only. Do not use `git add .`.
- Rejected V29 additive compatibility migration and temporary legacy-column transition.
- Expanded DB cleanup scope from V29 additive migration rejection to full V1->V28 historical compatibility migration audit and clean-schema replacement.
- Future checkpoints must not commit or push legacy compatibility paths.

## Next Phase

Commit and push Checkpoints B/C clean-state implementation, then continue Checkpoint D without legacy compatibility paths.
