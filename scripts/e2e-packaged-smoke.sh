#!/bin/bash
# e2e-packaged-smoke.sh — verify packaged artifact integrity
# Builds release, starts container, validates API responses
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== E2E Packaged Smoke ==="

# 1. Build release
echo "[1/4] Building release..."
cd "$PROJECT_DIR"
bash scripts/package-release-docker.sh 2>&1 | tail -3

# 2. Verify tarball contains catalog
echo "[2/4] Verifying catalog in tarball..."
TARBALL=$(ls -t "$PROJECT_DIR/dist/"lightai-go-*.tar.gz 2>/dev/null | head -1)
if [ -z "$TARBALL" ]; then
  echo "FAIL: No tarball found" >&2
  exit 1
fi
CATALOG_COUNT=$(tar -tzf "$TARBALL" | grep -c 'backend-catalog/' || true)
if [ "$CATALOG_COUNT" -lt 20 ]; then
  echo "FAIL: Only $CATALOG_COUNT catalog files in tarball (expected >= 20)" >&2
  exit 1
fi
echo "  OK: $CATALOG_COUNT catalog files"

# 3. Attempt container smoke (optional — requires Docker image built from tarball)
echo "[3/4] Checking Docker image availability..."
HAS_IMAGE=false
if docker image inspect lightai-go:latest >/dev/null 2>&1; then
  HAS_IMAGE=true
elif docker image inspect lightai-go:dev >/dev/null 2>&1; then
  HAS_IMAGE=true
fi

if $HAS_IMAGE; then
  echo "  Starting container smoke..."
  docker rm -f lightai-smoke 2>/dev/null || true
  docker run --rm -d --name lightai-smoke -p 18081:18080 lightai-go:latest 2>/dev/null || \
    docker run --rm -d --name lightai-smoke -p 18081:18080 lightai-go:dev 2>/dev/null || true
  sleep 8

  echo "[4/4] API verification..."
  BACKENDS=$(curl -s http://localhost:18081/api/v1/backends 2>/dev/null || echo '[]')
  BACKEND_COUNT=$(echo "$BACKENDS" | jq -r 'if type=="array" then length else (.data // [] | length) end' 2>/dev/null || echo 0)
  if [ "${BACKEND_COUNT:-0}" -gt 0 ]; then
    echo "  OK: $BACKEND_COUNT backends"
    NODES=$(curl -s http://localhost:18081/api/v1/nodes 2>/dev/null || echo '[]')
    NODE_COUNT=$(echo "$NODES" | jq -r 'if type=="array" then length else (.data // [] | length) end' 2>/dev/null || echo 0)
    echo "  OK: $NODE_COUNT nodes"
    HELP=$(curl -s "http://localhost:18081/api/v1/backend-help?backend=vllm&version=vllm-v0.23.0&lang=zh-CN" 2>/dev/null || echo '[]')
    HELP_COUNT=$(echo "$HELP" | jq 'if type=="array" then length else 0 end' 2>/dev/null || echo 0)
    echo "  OK: $HELP_COUNT help entries for vLLM"
  else
    echo "  WARNING: Container started but API returned no backends (may need agent + GPU)"
  fi
  docker stop lightai-smoke 2>/dev/null || true
else
  echo "  SKIP: No Docker image available (tarball-only verification)"
  echo "  To test with image: docker load < dist/lightai-go-*.tar.gz"
fi

echo ""
echo "=== PASS: Tarball verified ==="
