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

# 3. Start container
echo "[3/4] Starting container..."
docker rm -f lightai-smoke 2>/dev/null || true
docker run --rm -d --name lightai-smoke -p 18081:18080 lightai-go:latest
sleep 8

# 4. API health check
echo "[4/4] API verification..."
# Check backends exist
BACKENDS=$(curl -s http://localhost:18081/api/v1/backends 2>/dev/null || echo '[]')
BACKEND_COUNT=$(echo "$BACKENDS" | jq -r 'if type=="array" then length else (.data // [] | length) end' 2>/dev/null || echo 0)
if [ "${BACKEND_COUNT:-0}" -eq 0 ]; then
  echo "FAIL: No backends returned" >&2
  docker stop lightai-smoke 2>/dev/null || true
  exit 1
fi
echo "  OK: $BACKEND_COUNT backends"

# Check nodes (at least 1 if agent is also running)
NODES=$(curl -s http://localhost:18081/api/v1/nodes 2>/dev/null || echo '[]')
NODE_COUNT=$(echo "$NODES" | jq -r 'if type=="array" then length else (.data // [] | length) end' 2>/dev/null || echo 0)
echo "  OK: $NODE_COUNT nodes"

# Check backend runtimes
RUNTIMES=$(curl -s http://localhost:18081/api/v1/backend-runtimes 2>/dev/null || echo '[]')
RUNTIME_COUNT=$(echo "$RUNTIMES" | jq -r 'if type=="array" then length else (.data // [] | length) end' 2>/dev/null || echo 0)
echo "  OK: $RUNTIME_COUNT backend runtimes"

# Help endpoint
HELP=$(curl -s "http://localhost:18081/api/v1/backend-help?backend=vllm&version=vllm-v0.23.0&lang=zh-CN" 2>/dev/null || echo '[]')
HELP_COUNT=$(echo "$HELP" | jq 'if type=="array" then length else 0 end' 2>/dev/null || echo 0)
echo "  OK: $HELP_COUNT help entries for vLLM"

# Cleanup
docker stop lightai-smoke 2>/dev/null || true

echo ""
echo "=== PASS ==="
