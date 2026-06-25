# Phase 06: Final Validation And Runtime Smoke

## Scope

Checkpoint F completed the final ConfigSet/ConfigItem validation pass:

- full Go and Web validation;
- fresh DB clean-schema verification;
- active stale-contract/static gates;
- platform-chain runtime smoke for vLLM, SGLang, and llama.cpp;
- final issue closure and evidence capture.

## Changes Made In This Phase

- Applied NBR probe-derived `process_start_detection` to deployment preflight/start so SGLang image-default containers receive the required launcher command.
- Hardened runtime smoke helpers:
  - GGUF model files are created as `format=gguf` and `path_type=file`;
  - directory models use parent model root plus relative model directory;
  - API failures in scan/create/NBR/deployment/start stages are hard failures;
  - inference is retried and must return `ok=true`;
  - post-start failures run stop/cleanup to avoid stale GPU leases.
- Removed remaining active backend catalog old-field naming:
  - `capabilities_json` catalog key replaced by `capabilities_detail`;
  - unused old schema model structs removed.

## Fresh DB Evidence

Fresh DB root:

```text
/tmp/lightai-configset-f-20260626061549
```

Schema probe summary:

| Table | ConfigSet authority | Source metadata | Old authority columns |
| --- | --- | --- | --- |
| `backends` | Not a runtime config table | Not a runtime config table | none |
| `backend_versions` | `config_set_json` | `source_metadata_json` | none |
| `backend_runtimes` | `config_set_json` | `source_metadata_json` | none |
| `node_backend_runtimes` | `config_set_json` | `source_metadata_json` | none |
| `model_deployments` | `config_set_json` | `source_metadata_json` | none |

`schema_version=100` is a clean-schema baseline marker only; it does not imply historical DB upgrade support.

## Runtime Smoke Evidence

Final run ID:

```text
configset-f-20260626061623
```

Artifact directory:

```text
docs/reports/model-runtime-node-wizard/e2e-matrix-configset-f-20260626061623
```

Terminal log:

```text
docs/reports/model-runtime-node-wizard/e2e-matrix-configset-f-20260626061623/real-smoke.log
```

| Backend | Health | Inference | Logs | Stop/Cleanup |
| --- | --- | --- | --- | --- |
| vLLM default | PASS | PASS, `ok=true`, `response_preview=pong` | PASS | PASS |
| vLLM modified | PASS | PASS, `ok=true`, `response_preview=pong` | PASS | PASS |
| SGLang | PASS | PASS, `ok=true`, `response_preview=pong` | PASS | PASS |
| llama.cpp | PASS | PASS, `ok=true`, GGUF path | PASS | PASS |

## Validation Commands

| Command | Result |
| --- | --- |
| `go test ./...` | PASS |
| `go build ./cmd/server/...` | PASS |
| `go build ./cmd/agent/...` | PASS |
| `cd web && npm test` | PASS |
| `cd web && npm run build` | PASS |
| OpenAPI current-path parse check | PASS |
| Active old-field/static stale gate | PASS |
| Fresh DB schema probe | PASS |
| `scripts/e2e-real-smoke-all-three.sh` | PASS |
| `docker ps -a` smoke cleanup check | PASS |
| `git diff --check` | PASS |

## Issue Closure

See `open-issues.md`.

New Checkpoint F issues found during validation:

- `CS-F-001`: SGLang process start detection not applied to preflight/start. Status: `FIXED`.
- `CS-F-002`: llama.cpp GGUF artifact created with HuggingFace directory semantics. Status: `FIXED`.
- `CS-F-003`: smoke inference did not hard-fail or clean up reliably. Status: `FIXED`.
- `CS-F-004`: residual old catalog field naming and unused old schema structs. Status: `FIXED`.

No undocumented Checkpoint F blockers remain.
