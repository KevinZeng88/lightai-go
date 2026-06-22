# Web AI Config Review Open Issues Closeout

> Status: CURRENT
> Scope: Formal closeout register for presentation-only Web AI workflow work
> Date: 2026-06-21

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| WEB-AI-FU-001 | Model capabilities cannot be manually persisted for ModelArtifact. | `model_artifacts` and ModelArtifact API expose task/metadata fields but no dedicated persisted capability override contract. | UI can show inferred Chat/Completion/Embedding/Rerank/Vision capability badges, but admin checkbox persistence is not available without a new model/API contract. | DOCUMENTED_BLOCKER | Future ModelArtifact capability contract/API; no current schema change in this round. | `npm --prefix web test` covers inferred Qwen3 Chat behavior; no persistence test is expected in this round. | Keep inferred read-only display now; implement persisted override only with an approved data contract. |
| WEB-AI-FU-002 | Deployment extra volume override is not first-class. | Deployment API supports `placement_json`, `service_json`, `parameters_json`, `env_overrides_json`, and `config_snapshot_json`, but no typed deployment extra volume list. | UI cannot safely claim a stable extra-volume editor. | DOCUMENTED_BLOCKER | Future deployment override API/schema or documented typed JSON contract. | `npm --prefix web test` checks UI states this as an existing-field-only override area. | Do not add schema in this round; expose existing env/parameters/service fields only. |
| WEB-AI-FU-003 | Deployment port override is limited to existing service fields. | Current UI/API use host/container/app/health/test style service fields, not a typed multi-port list. | UI cannot present full arbitrary port mapping without changing API semantics. | DOCUMENTED_BLOCKER | Future deployment service override contract. | `npm --prefix web test` checks host/container/app port labels remain distinct. | Use existing service fields in this round. |
| WEB-AI-FU-004 | Endpoint alias / served model alias is not a dedicated deployment field. | `parameters_json` can hold backend parameters, but no explicit endpoint alias schema/API exists. | UI cannot promise a portable alias editor across backends. | DOCUMENTED_BLOCKER | Future deployment parameter contract for served model name/endpoint alias. | Dry-run/build tests verify existing presentation only. | Keep this out of first-class UI until contract exists. |
| WEB-AI-FU-005 | Deployment list summary joins are frontend-enriched rather than server summarized. | `GET /api/v1/deployments` returns IDs and JSON fields; frontend enriches with refs from models/runtimes/NBRs/instances. | Large deployments may benefit from a future summary DTO, but current display works with existing APIs. | DOCUMENTED_BLOCKER | Future read-only deployment summary API/DTO. | `npm --prefix web run build` and `npm --prefix web test` verify current frontend enrichment compiles and passes tests. | Keep frontend enrichment in this presentation-only round. |

---

## Batch 4 VLM Blocker (2026-06-23)

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| VLM-RUNTIME-001 | InternVL2_5-1B blocked on vLLM/SGLang backend architecture support. | vLLM v0.20.1 fails to load InternVLChatModel tokenizer even with sentencepiece installed and --trust-remote-code. Same image loads Qwen3/bge-small/bge-reranker. | VLM models cannot be deployed with current built-in backends; preflight now blocks via `blocked_architectures`. | BACKEND_CAPABILITY_BLOCKED | `internal/server/runplan/compat.go` (architecture check); `internal/server/db/db.go` (V27 repair + seed); `internal/server/api/deployment_lifecycle_handlers.go` (preflight passes architecture) | `go test lightai-go/internal/server/runplan/ -run TestCompatInternVLWithVLLMBlocked` verifies block; `TestCompatHFWithVLLMNoBlock` verifies other architectures unaffected | Preflight blocks InternVLChatModel. Future unlock: validated backend runtime/image that supports InternVL2.5. Not a missing dependency — sentencepiece is present. |

No unresolved problem from this round exists only in chat.
