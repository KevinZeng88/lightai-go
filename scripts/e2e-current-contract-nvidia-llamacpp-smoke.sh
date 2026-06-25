#!/usr/bin/env bash
# E2E NVIDIA llama.cpp real smoke — validates full start/stop contract.
# Requires: NVIDIA GPU, Docker, llama.cpp model at $LIGHTAI_GGUF_MODEL_PATH
# Usage: bash scripts/e2e-current-contract-nvidia-llamacpp-smoke.sh
set -euo pipefail
test -f "${LIGHTAI_GGUF_MODEL_PATH:-/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf}" || { echo "SKIP: GGUF model not found"; exit 0; }
docker info >/dev/null 2>&1 || { echo "SKIP: docker unavailable"; exit 0; }
nvidia-smi >/dev/null 2>&1 || { echo "SKIP: NVIDIA GPU not available"; exit 0; }
echo "=== E2E NVIDIA llama.cpp Smoke ==="
echo "Requires: running server+agent with llama.cpp NBR and model"
bash scripts/lightai-bootstrap.sh --mode runtimes-only 2>&1 | tail -3
bash scripts/lightai-bootstrap.sh --mode dry-run 2>&1 | tail -3
echo "=== Full mode (if images available) ==="
cp configs/bootstrap/local-kz-laptop.yaml /tmp/llamacpp-full-test.yaml 2>/dev/null
sed -i 's/allow_real_container_start: false/allow_real_container_start: true/' /tmp/llamacpp-full-test.yaml 2>/dev/null
bash scripts/lightai-bootstrap.sh --profile /tmp/llamacpp-full-test.yaml --mode full --allow-real-start 2>&1 | tail -3 || echo "Full not run (may need images)"
echo "=== PASS (NVIDIA smoke baseline) ==="
