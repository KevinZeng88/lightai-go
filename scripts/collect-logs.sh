#!/bin/sh
# LightAI Go - Collect Logs and Diagnostics
# Output: lightai-go-logs-<timestamp>.tar.gz
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RELEASE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
OUTPUT="lightai-go-logs-$TIMESTAMP.tar.gz"
TMPDIR=$(mktemp -d)
cd "$RELEASE_ROOT"

echo "=== LightAI Go Log Collection ==="
echo "Timestamp: $TIMESTAMP"
echo ""

mkdir -p "$TMPDIR/lightai-logs"

# Copy logs.
if [ -d logs ]; then
  cp -r logs "$TMPDIR/lightai-logs/" 2>/dev/null || true
  echo "Logs copied."
else
  echo "No logs/ directory."
fi

# Copy configs (sanitized — remove tokens).
mkdir -p "$TMPDIR/lightai-logs/configs"
for f in configs/*.yaml; do
  [ -f "$f" ] || continue
  sed -e 's/agent_token:.*/agent_token: "***REDACTED***"/' \
      -e 's/password:.*/password: "***REDACTED***"/' \
      "$f" > "$TMPDIR/lightai-logs/configs/$(basename "$f")"
done
echo "Configs copied (sanitized)."

# Copy VERSION.
if [ -f VERSION ]; then
  cp VERSION "$TMPDIR/lightai-logs/"
fi

# Current status.
./scripts/status.sh > "$TMPDIR/lightai-logs/status.txt" 2>&1 || true

# Verification output.
./scripts/verify-local.sh > "$TMPDIR/lightai-logs/verify.txt" 2>&1 || true

# System info.
{
  echo "date: $(date -Iseconds)"
  echo "hostname: $(hostname)"
  echo "uname: $(uname -a)"
  echo "go version: $(go version 2>/dev/null || echo 'not found')"
} > "$TMPDIR/lightai-logs/system-info.txt"

# GPU info.
if command -v nvidia-smi >/dev/null 2>&1; then
  nvidia-smi --query-gpu=index,name,uuid,memory.total,memory.used --format=csv,noheader,nounits \
    > "$TMPDIR/lightai-logs/nvidia-smi.txt" 2>/dev/null || true
fi
if command -v mx-smi >/dev/null 2>&1; then
  mx-smi -L > "$TMPDIR/lightai-logs/mx-smi-list.txt" 2>/dev/null || true
fi

# Package.
cd "$TMPDIR"
tar -czf "$RELEASE_ROOT/$OUTPUT" lightai-logs/
cd "$RELEASE_ROOT"
rm -rf "$TMPDIR"

echo ""
echo "Log bundle created: $OUTPUT"
echo "Size: $(du -h "$OUTPUT" | cut -f1)"
echo "Send this file for analysis."
