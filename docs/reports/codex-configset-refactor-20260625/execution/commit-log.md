# ConfigSet Refactor Commit Log

| Checkpoint | Commit | Message | Push Result | Notes |
| --- | --- | --- | --- | --- |
| A | `1886f0f` | `docs: add configset refactor baseline` | Pushed | Checkpoint A explicit-path documentation/inventory commit. |
| B-policy | `b62adbd` | `docs: require clean configset checkpoint state` | Pushed | Documents the rejection of V29 additive compatibility migration and legacy-column transition before any Checkpoint B implementation commit. |
| B-migration-policy | `52da305` | `docs: require clean db migration baseline` | Pushed | Expanded DB cleanup scope from V29 additive migration rejection to full V1->V28 historical compatibility migration audit and clean-schema replacement. |
| B/C | `dee0dd8` | `refactor: replace catalog seeds with configset snapshots` | Pushed | Clean ConfigSet registry/catalog loader, fresh DB schema, API copy-on-create, RunPlan/API test updates. |
| D | `6935951` | `refactor: render runplans from configsets` | Pushed | ConfigSet parameter renderer styles, repeat-flag preservation, ResolvedRunPlan-to-AgentRunSpec conversion, and deployment start adapter integration. |
| E | `a822ac3` | `refactor: migrate api ui to configsets` | Pushed | Public API/OpenAPI, Web pages/tests, and active scripts migrated to ConfigSet/current deployment contracts; stale legacy-contract scripts archived or removed from active paths. |
| F | `bbe0686` | `test: validate configset runtime smoke` | Pushed | Full validation, fresh DB schema probe, catalog old-field cleanup, and real platform-chain smoke for vLLM/SGLang/llama.cpp completed and pushed. |

## Commit Policy Reminder

Do not use:

```bash
git add .
```

Use explicit staging only.
