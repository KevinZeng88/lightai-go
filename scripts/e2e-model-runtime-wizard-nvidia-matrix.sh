#!/usr/bin/env bash
set -euo pipefail
# Matrix wrapper: runs llama.cpp, vLLM, SGLang E2E backends sequentially.
# Each backend runs default + modified params. Individual failures are
# collected but execution continues so all backends get tested.
# Final exit code is non-zero if any backend failed.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

RUN_ID="${RUN_ID:-$(date +%Y%m%d-%H%M%S)}"
MATRIX_DIR="${MATRIX_DIR:-docs/reports/model-runtime-node-wizard/e2e-matrix-${RUN_ID}}"
mkdir -p "$MATRIX_DIR"

export E2E_RUN_ID="$RUN_ID"
export MATRIX_DIR="$MATRIX_DIR"

log() { printf '[%s] [matrix] %s\n' "$(date '+%H:%M:%S')" "$*"; }

RESULTS=""
EXIT_CODE=0

run_one() {
  local label="$1" script="$2"
  log "========== $label start =========="
  if [ -x "$script" ]; then
    if bash "$script" 2>&1 | tee "$MATRIX_DIR/${label}.log"; then
      RESULTS="$RESULTS\n$label: PASS"
      log "$label PASS"
    else
      RESULTS="$RESULTS\n$label: FAIL"
      log "$label FAIL"
      EXIT_CODE=1
    fi
  else
    log "$label: SKIP (script not found: $script)"
    RESULTS="$RESULTS\n$label: SKIP"
  fi
}

# ── Run each backend ──
run_one "llamacpp"        "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-llamacpp.sh"
run_one "vllm"            "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-vllm.sh"
run_one "sglang"          "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-sglang.sh"

# ── Save server/agent logs ──
tail -n 5000 logs/lightai-server.log > "$MATRIX_DIR/server-this-run.log" 2>/dev/null || true
tail -n 5000 logs/lightai-agent.log > "$MATRIX_DIR/agent-this-run.log" 2>/dev/null || true
docker ps -a --format 'table {{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}' > "$MATRIX_DIR/docker-ps-after.txt" 2>/dev/null || true

# ── Summary ──
echo ""
echo "========== E2E Matrix Summary =========="
echo -e "$RESULTS"
echo ""
log "Matrix complete exit=$EXIT_CODE"
exit $EXIT_CODE
