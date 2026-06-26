# Runtime and RunPlan Review

## Current chain

Current create/start path:

1. User selects ready NBR.
2. `HandleCreateDeployment` derives `backend_runtime_id` and node from NBR.
3. Deployment stores `source_node_backend_runtime_id` and `config_snapshot_json`.
4. Dry-run/start call `preflightDeployment`.
5. `preflightDeployment` reads deployment snapshot, NBR, model location, GPU info, BackendVersion capability data.
6. `runplan.Resolve` returns final plan.
7. Start stores `resolved_run_plans` and sends AgentRunSpec to agent task.
8. Agent Docker driver consumes structured spec.

## Good design choices

- Agent does not re-derive business objects from DB.
- RunPlan resolver has typed inputs and tests.
- NBR snapshot and deployment snapshot decouple most parent edits.
- Image resolution has clear intended precedence: NBR image ref over snapshot over catalog.
- Health check and Docker failure diagnostics are part of lifecycle.

## Gaps

- `/deployments/preflight` is not the same final plan boundary.
- Snapshot mutation exists in migrations and legacy branches.
- Deployment template sync is implemented, but running-instance behavior and explicit NBR reapply/change workflow remain product gaps.
- GPU assignment is simplistic; it auto-picks the first available GPU when none is specified.
- Multi-replica fields are not implemented as multi-instance run plan generation.

## Acceptance target

For the next phase, the project should treat the following as the RunPlan contract:

```text
deployment create/update -> deployment snapshot
final preview/preflight -> runplan.Resolve
start -> exact same resolver input class
agent task -> exact same resolved plan translated to AgentRunSpec
docker start -> no business re-resolution
```

Any endpoint that only computes candidate nodes should be named and documented as such.
