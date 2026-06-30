# 07 — Work Package A: Contract Tests + ConfigEdit Object Foundation

## Objective

Create the contract tests and the foundational ConfigEdit object model. This package establishes the target behavior and prevents future page-level patches from bypassing the architecture.

## Scope

Implement the minimum object contract required for BackendRuntime, NodeBackendRuntime, Deployment, and Deployment override.

Do not implement the full template editor in this package.

## Required contract tests

Add tests before or alongside implementation. Tests must encode the target contract, not current flawed behavior.

### Backend tests

Cover at least:

1. ConfigEdit object shape

- BackendRuntime can be opened as a ConfigEdit object.
- NodeBackendRuntime can be opened as a ConfigEdit object.
- Deployment can be opened as a ConfigEdit object.
- Deployment override can be opened as a ConfigEdit object.

Expected minimum object fields:

```json
{
  "object_kind": "deployment",
  "object_id": "...",
  "template_id": "...",
  "snapshot_id": "...",
  "parent": {
    "object_kind": "node_backend_runtime",
    "object_id": "...",
    "snapshot_id": "..."
  },
  "child_init": {
    "strategy": "copy_effective_snapshot"
  },
  "sections": [],
  "components": [],
  "fields": [],
  "effects_preview": [],
  "diagnostics": {},
  "view_level": "normal"
}
```

2. Snapshot copy

- BackendRuntime creation copies BackendVersion effective ConfigEdit/config snapshot.
- NodeBackendRuntime enable copies BackendRuntime effective ConfigEdit snapshot.
- Deployment create copies NodeBackendRuntime effective ConfigEdit snapshot.
- After child creation, later parent edits do not live-overwrite the child snapshot.
- Child can override its own fields.

3. Deployment ConfigEdit scope

Deployment ConfigEdit must include runtime-affecting configuration currently split across:

- `config_set_json`
- `service_json`
- `placement_json`
- health check
- model mount
- device binding
- args
- env
- Docker options

If storage still keeps `service_json` and `placement_json`, tests must prove they are mirrors/compatibility fields or integrated inputs, not the only source consumed by UI/RunPlan.

4. Reset semantics

Add backend/API-level tests for:

- reset to parent
- reset to default

UI can lag until later packages, but the object model must reserve and test this behavior.

5. Final Docker token mapping baseline

For vLLM, SGLang, and llama.cpp fixtures, assert that final Docker command tokens are expected to map to ConfigEdit component/effect identity.

This test may initially fail. It must pass by the end of the full implementation.

6. UI contract tests

Add tests or source-level assertions that normal view must not display:

- raw source map
- raw Config JSON
- unresolved template command
- internal technical keys
- patch target internals
- system_generated internals

Developer view can display raw/debug content.

## Implementation requirements

### ConfigEdit object model

Introduce or promote a versioned ConfigEdit object type. Avoid creating an empty wrapper that only decorates the old field list.

Minimum concepts:

- object identity
- template identity
- snapshot identity
- parent snapshot reference
- child init/copy contract
- component identity
- field identity
- field value state
- enabled state
- source/provenance
- editability
- reset behavior
- validation
- view level
- effects preview
- diagnostics

### Deployment ConfigEdit

Deployment ConfigEdit must begin unifying:

- service port
- served model name
- placement/device selection
- health
- mount
- args
- env
- Docker options
- backend runtime params

The immediate goal is not to delete all existing storage columns. The immediate goal is to establish a single ConfigEdit object contract that UI and RunPlan can use as the main path.

### API behavior

Add or extend endpoints to retrieve/apply ConfigEdit object data. Reuse existing `/api/v1/config-edit/view` and `/api/v1/config-edit/apply` if possible, but the response must support the object contract.

Prefer clean implementation over compatibility branching if the current DB can be rebuilt. If a migration/backfill is needed, document the strategy.

## Self-audit gate

Before moving to Work Package B, answer these in `execution-log.md`:

- Is ConfigEdit now an object model rather than only a projected field list?
- Can BackendRuntime, NodeBackendRuntime, Deployment, and Deployment override be opened as ConfigEdit objects?
- Does Deployment ConfigEdit include parent snapshot metadata?
- Does Deployment create copy an effective NBR snapshot?
- Are `service_json` and `placement_json` still the only source for service/device behavior? If yes, fix before moving on.
- Are contract tests present and meaningful?
- Which tests pass now?
- Which full-contract tests are intentionally expected to pass only after later packages?
- What files changed?

## Package verification

Run package-relevant tests and, when affordable:

```bash
go test ./internal/server/configedit/... ./internal/server/runplan/... ./internal/server/api/...
cd web
npm run test:unit
```

Commit after the package is stable.
