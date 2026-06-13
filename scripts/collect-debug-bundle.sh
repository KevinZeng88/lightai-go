#!/bin/bash
# LightAI Go - Debug Bundle Collection Script
# Collects logs, configs, and system info for troubleshooting.
# Output: dist/debug-bundles/lightai-debug-<timestamp>.tar.gz

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
OUTPUT_DIR="$PROJECT_DIR/dist/debug-bundles"
BUNDLE_NAME="lightai-debug-$TIMESTAMP"
TMP_DIR=$(mktemp -d)

mkdir -p "$OUTPUT_DIR"

echo "=== LightAI Go Debug Bundle ==="
echo "Timestamp: $TIMESTAMP"
echo "Project: $PROJECT_DIR"
echo "Temp dir: $TMP_DIR"
echo ""

# Helper: sanitize config files (remove tokens, passwords).
sanitize_config() {
  local file="$1"
  sed -e 's/agent_token:.*/agent_token: "***REDACTED***"/' \
      -e 's/password:.*/password: "***REDACTED***"/' \
      -e 's/password_env:.*/password_env: "***REDACTED***"/' \
      "$file"
}

echo "[1/10] Collecting Server log..."
if [ -f "$PROJECT_DIR/logs/lightai-server.log" ]; then
  cp "$PROJECT_DIR/logs/lightai-server.log" "$TMP_DIR/lightai-server.log"
  echo "  OK: $(wc -l < "$TMP_DIR/lightai-server.log") lines"
else
  echo "  NOT FOUND: logs/lightai-server.log"
fi

echo "[2/10] Collecting Agent log..."
if [ -f "$PROJECT_DIR/logs/lightai-agent.log" ]; then
  cp "$PROJECT_DIR/logs/lightai-agent.log" "$TMP_DIR/lightai-agent.log"
  echo "  OK: $(wc -l < "$TMP_DIR/lightai-agent.log") lines"
else
  echo "  NOT FOUND: logs/lightai-agent.log"
fi

echo "[3/10] Collecting Server config (sanitized)..."
if [ -f "$PROJECT_DIR/configs/server.dev.yaml" ]; then
  sanitize_config "$PROJECT_DIR/configs/server.dev.yaml" > "$TMP_DIR/server.dev.yaml"
  echo "  OK"
else
  echo "  NOT FOUND"
fi

echo "[4/10] Collecting Agent config (sanitized)..."
if [ -f "$PROJECT_DIR/configs/agent.dev.yaml" ]; then
  sanitize_config "$PROJECT_DIR/configs/agent.dev.yaml" > "$TMP_DIR/agent.dev.yaml"
  echo "  OK"
else
  echo "  NOT FOUND"
fi

echo "[5/10] System info..."
echo "go version: $(go version 2>/dev/null || echo 'not found')" > "$TMP_DIR/system-info.txt"
echo "uname: $(uname -a)" >> "$TMP_DIR/system-info.txt"
echo "hostname: $(hostname)" >> "$TMP_DIR/system-info.txt"
echo "date: $(date -Iseconds)" >> "$TMP_DIR/system-info.txt"
echo "git HEAD: $(cd "$PROJECT_DIR" && git rev-parse HEAD 2>/dev/null || echo 'not a git repo')" >> "$TMP_DIR/system-info.txt"
echo "git status:" >> "$TMP_DIR/system-info.txt"
(cd "$PROJECT_DIR" && git status --short 2>/dev/null || echo 'not a git repo') >> "$TMP_DIR/system-info.txt"
echo "  OK"

echo "[6/10] NVIDIA GPU info..."
if command -v nvidia-smi &>/dev/null; then
  nvidia-smi --query-gpu=index,name,uuid,pci.bus_id,driver_version,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw --format=csv,noheader,nounits > "$TMP_DIR/nvidia-smi-query.txt" 2>/dev/null || true
  nvidia-smi > "$TMP_DIR/nvidia-smi-full.txt" 2>/dev/null || true
  echo "  OK: $(wc -l < "$TMP_DIR/nvidia-smi-query.txt") GPU(s)"
else
  echo "  nvidia-smi not found"
fi

echo "[7/10] MetaX GPU info..."
if command -v mx-smi &>/dev/null; then
  mx-smi > "$TMP_DIR/mx-smi.txt" 2>/dev/null || true
  echo "  OK"
else
  echo "  mx-smi not found (expected if no MetaX GPU)"
fi

echo "[8/10] Health check..."
if curl -sf http://127.0.0.1:18080/healthz > "$TMP_DIR/healthz.json" 2>/dev/null; then
  echo "  OK: $(cat "$TMP_DIR/healthz.json")"
else
  echo "  FAILED: Server not reachable at http://127.0.0.1:18080"
fi

echo "[9/10] Metrics targets..."
if curl -sf http://127.0.0.1:18080/metrics/targets > "$TMP_DIR/metrics-targets.json" 2>/dev/null; then
  echo "  OK: $(python3 -c "import json; d=json.load(open('$TMP_DIR/metrics-targets.json')); print(len(d), 'targets')" 2>/dev/null || echo 'parsed')"
else
  echo "  FAILED"
fi

echo "[10/10] Agent metrics..."
if curl -sf http://127.0.0.1:19091/metrics > "$TMP_DIR/agent-metrics.txt" 2>/dev/null; then
  head -100 "$TMP_DIR/agent-metrics.txt" > "$TMP_DIR/agent-metrics-head100.txt"
  echo "  OK: $(wc -l < "$TMP_DIR/agent-metrics.txt") lines (first 100 saved)"
else
  echo "  FAILED: Agent metrics not reachable at http://127.0.0.1:19091"
fi

echo ""
echo "Creating bundle archive..."
cd "$TMP_DIR"
tar -czf "$OUTPUT_DIR/$BUNDLE_NAME.tar.gz" .
echo "Bundle created: $OUTPUT_DIR/$BUNDLE_NAME.tar.gz"
echo "Size: $(du -h "$OUTPUT_DIR/$BUNDLE_NAME.tar.gz" | cut -f1)"
echo ""

# Cleanup.
rm -rf "$TMP_DIR"
echo "Done. Send the tar.gz file for analysis."
