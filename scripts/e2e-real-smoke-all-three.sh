#!/usr/bin/env bash
# Run the current ConfigSet platform-chain runtime smoke for vLLM, SGLang,
# and llama.cpp through the product API path.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_ID="${LIGHTAI_E2E_RUN_ID:-$(date +%Y%m%d%H%M%S)}"
MATRIX_DIR="${MATRIX_DIR:-docs/reports/model-runtime-node-wizard/e2e-matrix-${RUN_ID}}"
mkdir -p "$MATRIX_DIR"

run_one() {
  local label="$1"
  local script="$2"
  printf '[%s] [smoke] start %s\n' "$(date '+%H:%M:%S')" "$label"
  MATRIX_DIR="$MATRIX_DIR" E2E_RUN_ID="$RUN_ID" "$SCRIPT_DIR/$script"
  printf '[%s] [smoke] pass %s\n' "$(date '+%H:%M:%S')" "$label"
}

run_one "vLLM" "e2e-model-runtime-wizard-nvidia-vllm.sh"
run_one "SGLang" "e2e-model-runtime-wizard-nvidia-sglang.sh"
run_one "llama.cpp" "e2e-model-runtime-wizard-nvidia-llamacpp.sh"

printf '\nConfigSet real smoke PASS\nArtifacts: %s\n' "$MATRIX_DIR"
