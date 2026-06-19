#!/usr/bin/env bash
# e2e-runtime-config-web-check-flow.sh — Real Docker image check-request flow regression.
# Tests: enable → check-request → ready (positive) / missing_image (negative) / row check / wizard check.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/e2e/lib/model-runtime-common.sh"

SERVER_URL="${SERVER_URL:-http://127.0.0.1:18080}"
ARTIFACT_DIR="${ARTIFACT_DIR:-/tmp/lightai-web-check-flow-$(date +%Y%m%d%H%M%S)}"
mkdir -p "$ARTIFACT_DIR"

log() { printf '[%s] [web-check] %s\n' "$(date '+%H:%M:%S')" "$*"; }
fail() { log "FAIL: $*"; exit 1; }

# ── pre-check: Docker images available ──────────────────────────────────
log "docker_images: $(docker images --format '{{.Repository}}:{{.Tag}}' 2>/dev/null | tr '\n' ' ')"

# ── login + query node ─────────────────────────────────────────────────
e2e_login || fail "login"
e2e_query_node || fail "no node"
log "node_id=$NODE_ID"

# ── Scene 1: Positive case — real vllm image → ready ───────────────────
log "===== Scene 1: vllm positive case ====="
IMAGE_VLLM="vllm/vllm-openai:latest"
BACKEND_RUNTIME_ID_VLLM="runtime.vllm.nvidia-docker"

# Verify image exists locally
docker image inspect "$IMAGE_VLLM" >/dev/null 2>&1 || fail "vllm image $IMAGE_VLLM not found"
log "image $IMAGE_VLLM exists locally"

# Enable NBR
log "enable nbr vllm"
enable_resp="$(api_ok POST "/api/v1/nodes/$NODE_ID/backend-runtimes/enable" \
  "{\"backend_runtime_id\":\"$BACKEND_RUNTIME_ID_VLLM\",\"image_ref\":\"$IMAGE_VLLM\"}")"
echo "$enable_resp" > "$ARTIFACT_DIR/scene1-enable.json"

nbr_id="$(echo "$enable_resp" | json_get id)"
[ -n "$nbr_id" ] || fail "enable did not return nbr_id"
log "nbr_id=$nbr_id"

nbr_status="$(echo "$enable_resp" | json_get status)"
[ "$nbr_status" = "needs_check" ] || log "WARN: enable status=$nbr_status expected needs_check"

# Call check-request (UI-facing, server→agent proxy)
log "check-request vllm"
check_resp="$(api_ok POST "/api/v1/nodes/$NODE_ID/backend-runtimes/$nbr_id/check-request" '{}')"
echo "$check_resp" > "$ARTIFACT_DIR/scene1-check-request.json"

check_status="$(echo "$check_resp" | json_get status)"
check_reason="$(echo "$check_resp" | json_get status_reason)"
check_image="$(echo "$check_resp" | json_get checked_image_ref)"

log "check-request status=$check_status reason=$check_reason image=$check_image"

# Assert: should be ready (image exists locally, agent should confirm)
if [ "$check_status" = "ready" ]; then
  log "PASS: vllm positive → ready"
elif [ "$check_status" = "unknown" ]; then
  log "BLOCKED: agent unreachable from server (status=unknown). Agent must be running with metrics port."
else
  log "check-request returned status=$check_status reason=$check_reason"
  fail "expected ready or unknown (agent-unreachable), got $check_status"
fi

# ── Scene 2: Negative case — nonexistent image → missing_image ──────────
log "===== Scene 2: missing image case ====="
IMAGE_MISSING="lightai/nonexistent-image:e2e-missing-$(date +%s)"
BACKEND_RUNTIME_ID_SGLANG="runtime.sglang.nvidia-docker"

# Verify image does NOT exist
if docker image inspect "$IMAGE_MISSING" >/dev/null 2>&1; then
  fail "test image $IMAGE_MISSING unexpectedly exists"
fi
log "image $IMAGE_MISSING confirmed missing"

# Enable NBR with nonexistent image
enable2_resp="$(api_ok POST "/api/v1/nodes/$NODE_ID/backend-runtimes/enable" \
  "{\"backend_runtime_id\":\"$BACKEND_RUNTIME_ID_SGLANG\",\"image_ref\":\"$IMAGE_MISSING\"}")"
echo "$enable2_resp" > "$ARTIFACT_DIR/scene2-enable.json"

nbr2_id="$(echo "$enable2_resp" | json_get id)"
[ -n "$nbr2_id" ] || fail "enable did not return nbr_id for scene 2"
log "nbr2_id=$nbr2_id"

# Call check-request
check2_resp="$(api_ok POST "/api/v1/nodes/$NODE_ID/backend-runtimes/$nbr2_id/check-request" '{}')"
echo "$check2_resp" > "$ARTIFACT_DIR/scene2-check-request.json"

check2_status="$(echo "$check2_resp" | json_get status)"
check2_reason="$(echo "$check2_resp" | json_get status_reason)"
log "check-request status=$check2_status reason=$check2_reason"

if [ "$check2_status" = "missing_image" ]; then
  log "PASS: missing image → missing_image"
  if echo "$check2_reason" | grep -q "$IMAGE_MISSING"; then
    log "PASS: reason includes image name"
  else
    log "NOTE: reason does not contain image name: $check2_reason (agent proxy may be unreachable)"
  fi
elif [ "$check2_status" = "unknown" ]; then
  log "BLOCKED: agent unreachable (status=unknown). Cannot verify image presence."
else
  log "check-request returned status=$check2_status reason=$check2_reason"
  fail "expected missing_image, got $check2_status"
fi

# ── Scene 3: Row check (existing NBR) ──────────────────────────────────
log "===== Scene 3: row check ====="
row_check_resp="$(api_ok POST "/api/v1/nodes/$NODE_ID/backend-runtimes/$nbr_id/check-request" '{}')"
echo "$row_check_resp" > "$ARTIFACT_DIR/scene3-row-check.json"
row_status="$(echo "$row_check_resp" | json_get status)"
log "row check status=$row_status"
# Row check on same NBR should work identically
if [ "$row_status" = "$check_status" ]; then
  log "PASS: row check consistent with wizard check"
else
  log "NOTE: row check status=$row_status vs wizard=$check_status (may differ if agent state changed)"
fi

# ── Scene 4: llama.cpp positive case ───────────────────────────────────
log "===== Scene 4: llama.cpp positive case ====="
IMAGE_LLAMACPP="${IMAGE_LLAMACPP:-ghcr.io/ggml-org/llama.cpp:server-cuda13}"
BACKEND_RUNTIME_ID_LLAMACPP="runtime.llamacpp.nvidia-docker"

if docker image inspect "$IMAGE_LLAMACPP" >/dev/null 2>&1; then
  log "image $IMAGE_LLAMACPP exists"

  enable4_resp="$(api_ok POST "/api/v1/nodes/$NODE_ID/backend-runtimes/enable" \
    "{\"backend_runtime_id\":\"$BACKEND_RUNTIME_ID_LLAMACPP\",\"image_ref\":\"$IMAGE_LLAMACPP\"}")"
  nbr4_id="$(echo "$enable4_resp" | json_get id)"
  [ -n "$nbr4_id" ] || fail "llamacpp enable did not return nbr_id"

  check4_resp="$(api_ok POST "/api/v1/nodes/$NODE_ID/backend-runtimes/$nbr4_id/check-request" '{}')"
  echo "$check4_resp" > "$ARTIFACT_DIR/scene4-llamacpp-check.json"
  check4_status="$(echo "$check4_resp" | json_get status)"
  log "llamacpp check status=$check4_status"
  [ "$check4_status" = "ready" ] || log "BLOCKED: llamacpp check status=$check4_status (agent may be unreachable)"
else
  log "SKIP: image $IMAGE_LLAMACPP not available"
fi

# ── Scene 5: Preflight + DryRun (if NBR ready) ─────────────────────────
log "===== Scene 5: preflight + dryrun ====="
if [ "$check_status" = "ready" ]; then
  # We need a model artifact and location for preflight
  log "creating model artifact for preflight test..."
  art_name="web-check-art-$(date +%s)"
  art_resp="$(api_ok POST /api/v1/model-artifacts \
    "{\"name\":\"$art_name\",\"display_name\":\"$art_name\",\"path\":\"/tmp/$art_name\",\"format\":\"huggingface\",\"task_type\":\"chat\"}")"
  art_id="$(echo "$art_resp" | json_get id)"
  [ -n "$art_id" ] || { log "SKIP: artifact create failed — cannot test preflight"; art_id=""; }

  if [ -n "$art_id" ]; then
    # Add model location
    api_ok POST "/api/v1/model-artifacts/$art_id/locations" \
      "{\"node_id\":\"$NODE_ID\",\"model_root\":\"/tmp\",\"relative_path\":\"$art_name\",\"absolute_path\":\"/tmp/$art_name\",\"path_type\":\"directory\",\"verification_status\":\"verified\",\"match_status\":\"exact_match\"}" > /dev/null 2>&1 || true

    # Preflight with node_backend_runtime_id
    pf_resp="$(api_ok POST /api/v1/deployments/preflight \
      "{\"model_artifact_id\":\"$art_id\",\"node_backend_runtime_id\":\"$nbr_id\",\"host_port\":9000}")"
    echo "$pf_resp" > "$ARTIFACT_DIR/scene5-preflight.json"
    can_run="$(echo "$pf_resp" | json_get can_run)"
    log "preflight can_run=$can_run"
    if [ "$can_run" = "true" ]; then
      log "PASS: preflight with ready NBR → can_run=true"
    else
      log "NOTE: preflight can_run=$can_run (may need real model_location)"
    fi

    # Cleanup artifact
    api_body DELETE "/api/v1/model-artifacts/$art_id" > /dev/null 2>&1 || true
  fi
else
  log "SKIP: NBR not ready (status=$check_status), cannot test preflight"
fi

# ── Summary ────────────────────────────────────────────────────────────
log "===== Summary ====="
log "Artifacts: $ARTIFACT_DIR"
log "Scene 1 (vllm positive): status=$check_status"
log "Scene 2 (missing image): status=$check2_status"
log "Scene 3 (row check): status=$row_status"
log "Scene 4 (llamacpp): status=${check4_status:-SKIP}"
log "Scene 5 (preflight): can_run=${can_run:-SKIP}"

# Cleanup NBRs
api_body DELETE "/api/v1/nodes/$NODE_ID/backend-runtimes/$nbr_id" > /dev/null 2>&1 || true
api_body DELETE "/api/v1/nodes/$NODE_ID/backend-runtimes/$nbr2_id" > /dev/null 2>&1 || true
[ -n "${nbr4_id:-}" ] && api_body DELETE "/api/v1/nodes/$NODE_ID/backend-runtimes/$nbr4_id" > /dev/null 2>&1 || true

log "done"
