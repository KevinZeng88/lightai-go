#!/usr/bin/env bash
set -euo pipefail
# E2E: llama.cpp backend — default + modified params.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/e2e/lib/model-runtime-common.sh"

BACKEND_NAME="llamacpp"
BACKEND_RUNTIME_ID="llamacpp-b9700-nvidia-cuda13"
IMAGE_REF="ghcr.io/ggml-org/llama.cpp:server-cuda13"
MODEL_PATH="/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf"
HOST_PORT="8002"
E2E_RUN_ID="${E2E_RUN_ID:-$(date +%Y%m%d%H%M%S)}"

MATRIX_DIR="${MATRIX_DIR:-docs/reports/model-runtime-node-wizard/e2e-matrix-${E2E_RUN_ID}}"
ARTIFACT_DIR="$MATRIX_DIR/llamacpp"

RESULT_DEFAULT="FAIL"
RESULT_MODIFIED="FAIL"

# ── Default params ──
log "===== llama.cpp default params ====="
DEPLOY_PARAMS=""
if e2e_run_default; then
  RESULT_DEFAULT="PASS"
else
  log "llamacpp default FAILED"
fi

# ── Modified params ──
log "===== llama.cpp modified params ====="
BACKEND_NAME="llamacpp-modified"
ARTIFACT_DIR="$MATRIX_DIR/llamacpp-modified"
DEPLOY_PARAMS='"\"--ctx-size\":\"2048\",\"--n-gpu-layers\":\"-1\""'
HOST_PORT="8003"
if e2e_run_default; then
  RESULT_MODIFIED="PASS"
else
  log "llamacpp modified FAILED"
fi

echo ""
echo "llama.cpp default: $RESULT_DEFAULT"
echo "llama.cpp modified params: $RESULT_MODIFIED"

[ "$RESULT_DEFAULT" = "PASS" ] && [ "$RESULT_MODIFIED" = "PASS" ]
