#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

BACKEND_NAME="sglang"
BACKEND_RUNTIME_ID="runtime.sglang.nvidia-docker"
IMAGE_REF="${SGLANG_IMAGE_REF:-lmsysorg/sglang:latest}"
MODEL_PATH="${SGLANG_MODEL_PATH:-/home/kzeng/models/Qwen3-0.6B-Instruct-2512}"
HOST_PORT="${SGLANG_HOST_PORT:-8005}"
E2E_RUN_ID="${E2E_RUN_ID:-$(date +%Y%m%d%H%M%S)}"
MATRIX_DIR="${MATRIX_DIR:-docs/reports/model-runtime-node-wizard/e2e-matrix-${E2E_RUN_ID}}"
ARTIFACT_DIR="$MATRIX_DIR/sglang"
DEPLOY_PARAMS="${DEPLOY_PARAMS:-\"--tp\":1}"

source "$SCRIPT_DIR/e2e/lib/model-runtime-common.sh"

log "===== SGLang ConfigSet E2E ====="
e2e_run_default
