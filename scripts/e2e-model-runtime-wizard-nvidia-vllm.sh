#!/usr/bin/env bash
set -euo pipefail
# E2E: vLLM backend — default + modified params.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/e2e/lib/model-runtime-common.sh"

BACKEND_NAME="vllm"
BACKEND_RUNTIME_ID="vllm-v0.23.0-nvidia-cuda"
IMAGE_REF="vllm/vllm-openai:latest"
MODEL_PATH="/home/kzeng/models/Qwen3-0.6B-Instruct-2512"
HOST_PORT="8004"
E2E_RUN_ID="${E2E_RUN_ID:-$(date +%Y%m%d%H%M%S)}"

MATRIX_DIR="${MATRIX_DIR:-docs/reports/model-runtime-node-wizard/e2e-matrix-${E2E_RUN_ID}}"
ARTIFACT_DIR="$MATRIX_DIR/vllm"

RESULT_DEFAULT="FAIL"
RESULT_MODIFIED="FAIL"

# ── Default params ──
log "===== vLLM default params ====="
DEPLOY_PARAMS=""
if e2e_run_default; then
  RESULT_DEFAULT="PASS"
else
  log "vLLM default FAILED"
fi

# ── Modified params ──
log "===== vLLM modified params ====="
BACKEND_NAME="vllm-modified"
ARTIFACT_DIR="$MATRIX_DIR/vllm-modified"
DEPLOY_PARAMS='"--max-model-len":"2048","--gpu-memory-utilization":"0.80"'
HOST_PORT="8006"
if e2e_run_default; then
  RESULT_MODIFIED="PASS"
else
  log "vLLM modified FAILED"
fi

echo ""
echo "vLLM default: $RESULT_DEFAULT"
echo "vLLM modified params: $RESULT_MODIFIED"

[ "$RESULT_DEFAULT" = "PASS" ] && [ "$RESULT_MODIFIED" = "PASS" ]
