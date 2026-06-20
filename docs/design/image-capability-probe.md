# Image Capability Probe — NodeBackendRuntime Check Design

> Status: BLOCKER FIX IMPLEMENTED
> Date: 2026-06-20
> Scope: Fix NBR check-request missing_image false positive + layered probe foundation

## Root Cause

Agent `/docker-images` returns fields `repository`, `tag`, `image_ref`, `image_id`.
Server `HandleRequestNodeBackendRuntimeCheck` was decoding as `repotags` (array) and `id` (string).
Field names mismatch → `RepoTags` always nil → `imagePresent` stays false → `missing_image`.

**Fix**: Replaced struct decode with map-based decode matching agent's actual field names (`image_ref` first, then `repository:tag`).

## Authoritative Existence Check: ImageInspect, NOT docker-images list

### Rule

- `/docker-images` list is used for **UI selection** and as supporting evidence only.
- `/docker-image-inspect` (Docker `docker image inspect`) is the **authoritative** source for image existence.
- **ONLY** ImageInspect returning a clear "not found" error produces `missing_image`.
- All other failures (agent unreachable, docker error, inspect failure, decode error) produce their own distinct status codes.

### Status Mapping

| Condition | Status | Blocks? |
|-----------|--------|---------|
| ImageInspect returns "not found" | `missing_image` | YES |
| ImageInspect succeeds | `ready` or `ready_with_warnings` | No |
| ImageInspect fails (not "not found") | `inspect_failed` | YES |
| Agent unreachable | `agent_unreachable` | YES |
| Agent returns HTTP 5xx | `docker_error` | YES |
| Docker daemon error | `docker_error` | YES |
| No image_ref configured | `evidence_missing` | No |

### Critical Rule: Evidence ≠ Missing Image

The following MUST NOT produce `missing_image`:
- List didn't find image, but ImageInspect did → `ready` / `ready_with_warnings`
- List decode error, but ImageInspect succeeded → `ready` / `ready_with_warnings`
- ImageInspect error (not "not found") → `inspect_failed`
- Agent unreachable → `agent_unreachable`
- Docker error → `docker_error`
- No evidence → `evidence_missing`

### Not-Found Detection

ImageInspect "not found" is detected by matching the error string against patterns:
- `no such image`
- `not found`
- `does not exist`

These are the patterns Docker CLI returns when an image truly does not exist on the node.

## Probe Levels

### Level 1: Docker Image List (Evidence Only)
- Queries agent `/docker-images`
- Source: `docker_images_list`
- Captures: `image_ref`, `image_id`, `digest`, `created_at`, `size`
- Does NOT determine `missing_image`

### Level 2: Docker ImageInspect (Authoritative)
- Queries agent `/docker-image-inspect?ref=<image_ref>`
- Source: `docker_image_inspect`
- **This is the authoritative existence check**
- Captures: `Id`, `RepoTags`, `RepoDigests`, `Architecture`, `Os`, `Size`, `Created`, `Config.Entrypoint`, `Config.Cmd`, `Config.Env`, `Config.ExposedPorts`, `Config.Labels`
- `inspect_not_found: true` → signals the image was explicitly confirmed as not present

### Level 3: Backend Type Matching (Best-Effort, Lenient)
- Matches image `RepoTags` and `Labels` against known patterns per backend
- **Vendor is NOT used to derive backend** (`nvidia` ≠ `vllm`)
- Status fields:
  - `backend_match_status`: `confirmed_match` | `declared_match_unverified` | `ambiguous` | `not_checked`
  - `confirmed_match`: `true` only when pattern/label match is strong
  - `blocking`: `true` only for `confirmed_mismatch`
  - `warning`: `true` for `declared_match_unverified` / `ambiguous`
- Vendor-built images (MetaX, Huawei, etc.) that don't match known patterns → `declared_match_unverified` with `warning: true`, NOT `runtime_image_mismatch`

### Level 4: Version Probe (DEFERRED)
- NOT currently implemented
- Requires security review before deployment:
  - `--pull=never` (never pull)
  - `--network=none` (no network)
  - `--cap-drop=ALL` (drop all capabilities)
  - `--security-opt no-new-privileges`
  - No GPU, no mounts, no privileged
  - 5-10 second timeout
  - stdout/stderr truncated to 4096 bytes
  - Probe command from BackendVersion catalog config only (not user-supplied)

## Probe Results Storage

Stored in `node_backend_runtimes.probe_results_json` (DB migration V24).

**Node scope**: `probe_results_json` is stored on `node_backend_runtimes`, which is a **node-level** table. Each record has `id = node_id:backend_runtime_id` and is unique per `(node_id, backend_runtime_id)`. A single NBR record corresponds to exactly one node. Probe results are **node-scoped** — different nodes have different probe results.

## Status Model (Complete)

| Status | Blocker? | Meaning |
|--------|----------|---------|
| `ready` | No | ImageInspect succeeded, backend matched, no warnings |
| `ready_with_warnings` | No | ImageInspect succeeded, but backend match unverified or version not probed |
| `missing_image` | YES | ImageInspect explicitly returned "not found" |
| `inspect_failed` | YES | ImageInspect failed for reasons other than "not found" |
| `agent_unreachable` | YES | Cannot contact target node agent |
| `docker_error` | YES | Agent's Docker daemon/CLI returned an error |
| `runtime_image_mismatch` | YES | Image clearly mismatches expected backend |
| `evidence_missing` | No | No image_ref configured |

## Backend Type Matching Patterns

| Backend | Patterns |
|---------|----------|
| `vllm` | `vllm`, `vllm-openai` |
| `sglang` | `sglang`, `lmsysorg/sglang` |
| `llamacpp` | `llama.cpp`, `llama-cpp`, `llamacpp`, `ghcr.io/ggml-org/llama.cpp` |
| `ollama` | `ollama` |

**Important**: These patterns are for confirming a match, not for rejecting images. Images that don't match known patterns (e.g., MetaX/Huawei self-built images) are treated as `declared_match_unverified` — the declared backend is accepted but noted as unverified.

## Database Changes

| Migration | Change |
|-----------|--------|
| V24 | Added `probe_results_json TEXT NOT NULL DEFAULT '{}'` to `node_backend_runtimes` |

## Agent Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/docker-images` | GET | Lists Docker images (UI selection) |
| `/docker-image-inspect` | GET | Docker image inspect (authoritative existence) |

## Server Routes

| Route | Method | Description |
|-------|--------|-------------|
| `/api/v1/nodes/{id}/docker-images` | GET | Proxies to agent `/docker-images` |
| `/api/v1/nodes/{id}/docker-image-inspect` | GET | Proxies to agent `/docker-image-inspect` |
| `POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/check-request` | POST | NBR check endpoint (Image Capability Probe) |

## Future Work (NOT implemented)

These are deferred to dedicated design phases:

1. **Full NBR Image Probe page** — dedicated UI with comprehensive probe results display
2. **Version probe execution** — catalog-driven probe commands with strict security boundaries
3. **Probe results in Start Wizard** — pre-flight probe before deployment
4. **MetaX/Huawei vendor adapters** — backend patterns for non-NVIDIA vendor images

## Related Files

- `internal/server/api/runtime_handlers.go` — Probe handler, status evaluation, backend matching
- `cmd/agent/main.go` — Agent endpoints for docker-images, docker-image-inspect
- `internal/server/api/agent_handlers.go` — Server-to-agent proxy handlers
- `internal/server/db/db.go` — Migration V24
- `web/src/pages/RunnerConfigsPage.vue` — Wizard and detail drawer UI
- `web/src/utils/status.ts` — Status type/translation mappings
- `docs/design/image-capability-probe.md` — This document
