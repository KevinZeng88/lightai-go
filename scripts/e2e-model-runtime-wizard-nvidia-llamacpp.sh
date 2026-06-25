#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

BACKEND_NAME="llamacpp"
BACKEND_RUNTIME_ID="runtime.llamacpp.nvidia-docker"
IMAGE_REF="${LLAMACPP_IMAGE_REF:-ghcr.io/ggml-org/llama.cpp:server-cuda13}"
MODEL_PATH="${LLAMACPP_MODEL_PATH:-/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf}"
HOST_PORT="${LLAMACPP_HOST_PORT:-8002}"
E2E_RUN_ID="${E2E_RUN_ID:-$(date +%Y%m%d%H%M%S)}"
MATRIX_DIR="${MATRIX_DIR:-docs/reports/model-runtime-node-wizard/e2e-matrix-${E2E_RUN_ID}}"
ARTIFACT_DIR="$MATRIX_DIR/llamacpp"
DEPLOY_PARAMS="${DEPLOY_PARAMS:-\"--ctx-size\":2048,\"--n-gpu-layers\":30}"

source "$SCRIPT_DIR/e2e/lib/model-runtime-common.sh"

log "===== llama.cpp ConfigSet E2E ====="
e2e_run_default
