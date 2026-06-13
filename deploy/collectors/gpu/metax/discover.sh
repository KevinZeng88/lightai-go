#!/bin/sh
# LightAI GPU Collector - MetaX Discover
# Uses mx-smi -L to discover GPU devices.
# Exit codes: 0=success, 10=not_available, 30=command_failed, 40=parse_failed

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/../common.sh"

# Find mx-smi. Respect MX_SMI env var but validate it.
MX_SMI_CMD=""
if [ -n "${MX_SMI:-}" ] && [ -x "$MX_SMI" ]; then
  MX_SMI_CMD="$MX_SMI"
else
  MX_SMI_CMD=$(collector_find_command \
    mx-smi \
    /usr/bin/mx-smi \
    /usr/local/bin/mx-smi \
    /opt/maca/bin/mx-smi \
    /usr/local/maca/bin/mx-smi \
    /opt/mxdriver/bin/mx-smi) || {
    collector_emit_status metax false "mx-smi not found"
    exit 10
  }
fi

# Get driver version from default output header.
DRIVER_VERSION="unknown"
HEADER=$("$MX_SMI_CMD" 2>/dev/null | head -5) || true
if [ -n "$HEADER" ]; then
  DRIVER_VERSION=$(echo "$HEADER" | grep -oE 'Kernel Mode Driver Version:\s*[0-9.]+' | sed 's/Kernel Mode Driver Version:\s*//' | collector_trim)
  [ -z "$DRIVER_VERSION" ] && DRIVER_VERSION="unknown"
fi

# Get GPU list from mx-smi -L.
LIST_OUTPUT=$("$MX_SMI_CMD" -L 2>/dev/null) || {
  echo "mx-smi -L failed" >&2
  collector_emit_status metax false "mx-smi -L command failed"
  exit 30
}

if [ -z "$LIST_OUTPUT" ]; then
  collector_emit_status metax false "no MetaX GPUs found"
  exit 10
fi

collector_emit_status metax true ok

# Parse mx-smi -L format:
# GPU#0    MXC500      0000:0e:00.0   Available (UUID: GPU-xxx)
parse_error=0
echo "$LIST_OUTPUT" | while IFS= read -r line; do
  [ -z "$line" ] && continue

  # Extract fields using pattern matching.
  # GPU#<index> <model> <pci> <state> (UUID: <uuid>)
  idx=$(echo "$line" | sed -n 's/^GPU#\([0-9]*\).*/\1/p')
  raw_model=$(echo "$line" | sed -n 's/^GPU#[0-9]*[[:space:]]\+\([^[:space:]]*\).*/\1/p')
  pci=$(echo "$line" | sed -n 's/.*[[:space:]]\([0-9a-fA-F]\{4\}:[0-9a-fA-F]\{2\}:[0-9a-fA-F]\{2\}\.[0-9a-fA-F]\).*/\1/p')
  state=$(echo "$line" | sed -n 's/.*[[:space:]]\(Available\|Unavailable\|In Use\|Error\).*(UUID:.*/\1/p')
  uuid=$(echo "$line" | sed -n 's/.*UUID: \([^)]*\).*/\1/p')

  idx=$(echo "$idx" | collector_trim)
  raw_model=$(echo "$raw_model" | collector_trim)
  pci=$(echo "$pci" | collector_trim)
  state=$(echo "$state" | collector_trim)
  uuid=$(echo "$uuid" | collector_trim)

  if [ -z "$idx" ] || [ -z "$uuid" ]; then
    echo "metax discover: missing index or uuid in line: $line" >&2
    parse_error=1
    continue
  fi

  # Normalize name: MXC500 -> MetaX C500, etc.
  name=$(collector_normalize_metax_name "$raw_model")

  collector_emit_device metax "$idx" "$uuid" "$name" "$pci" "$DRIVER_VERSION" null
done

[ "$parse_error" = "1" ] && exit 40
exit 0
