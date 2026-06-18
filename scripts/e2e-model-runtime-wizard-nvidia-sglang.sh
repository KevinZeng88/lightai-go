#!/usr/bin/env bash
set -euo pipefail
# E2E: SGLang backend — default + modified params.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/e2e/lib/model-runtime-common.sh"

BACKEND_NAME="sglang"
BACKEND_RUNTIME_ID="sglang-v0.5.12-nvidia-cuda"
IMAGE_REF="lmsysorg/sglang:latest"
MODEL_PATH="/home/kzeng/models/Qwen3-0.6B-Instruct-2512"
HOST_PORT="8005"
E2E_RUN_ID="${E2E_RUN_ID:-$(date +%Y%m%d%H%M%S)}"

MATRIX_DIR="${MATRIX_DIR:-docs/reports/model-runtime-node-wizard/e2e-matrix-${E2E_RUN_ID}}"
ARTIFACT_DIR="$MATRIX_DIR/sglang"

RESULT_DEFAULT="FAIL"
RESULT_MODIFIED="FAIL"

# ── Default params ──
log "===== SGLang default params ====="
DEPLOY_PARAMS=""
if e2e_run_default; then
  RESULT_DEFAULT="PASS"
else
  log "SGLang default FAILED"
fi

# ── Modified params ──
log "===== SGLang modified params ====="
BACKEND_NAME="sglang-modified"
ARTIFACT_DIR="$MATRIX_DIR/sglang-modified"
DEPLOY_PARAMS='"\"--tp\":\"1\""'
HOST_PORT="8007"
if e2e_run_default; then
  RESULT_MODIFIED="PASS"
else
  log "SGLang modified FAILED"
fi

echo ""
echo "SGLang default: $RESULT_DEFAULT"
echo "SGLang modified params: $RESULT_MODIFIED"

[ "$RESULT_DEFAULT" = "PASS" ] && [ "$RESULT_MODIFIED" = "PASS" ]
