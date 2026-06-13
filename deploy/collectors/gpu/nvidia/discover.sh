#!/bin/sh
# LightAI GPU Collector - NVIDIA Discover
# Converts nvidia-smi output to LightAI GPU Collector Protocol.
# Exit codes: 0=success, 10=not_available, 30=command_failed, 40=parse_failed

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/../common.sh"

# Find nvidia-smi.
NVIDIA_SMI=""
NVIDIA_SMI=$(collector_find_command nvidia-smi /usr/bin/nvidia-smi /usr/lib/wsl/lib/nvidia-smi) || {
  collector_emit_status nvidia false "nvidia-smi not found"
  exit 10
}

# Execute query.
OUTPUT=$("$NVIDIA_SMI" --query-gpu=index,name,uuid,pci.bus_id,driver_version,memory.total --format=csv,noheader,nounits 2>/dev/null) || {
  echo "nvidia-smi discover failed: exit=$?" >&2
  collector_emit_status nvidia false "nvidia-smi command failed"
  exit 30
}

if [ -z "$OUTPUT" ]; then
  collector_emit_status nvidia false "no NVIDIA GPUs found"
  exit 10
fi

collector_emit_status nvidia true ok

# Parse each GPU line.
# Format: index, name, uuid, pci.bus_id, driver_version, memory.total
parse_error=0
echo "$OUTPUT" | while IFS=',' read -r idx name uuid pci driver mem_total; do
  idx=$(echo "$idx" | collector_trim)
  name=$(echo "$name" | collector_trim)
  uuid=$(echo "$uuid" | collector_trim)
  pci=$(echo "$pci" | collector_trim)
  driver=$(echo "$driver" | collector_trim)
  mem_total=$(echo "$mem_total" | collector_trim)

  # Required fields.
  if [ -z "$idx" ] || [ -z "$uuid" ] || [ -z "$name" ] || [ -z "$pci" ]; then
    echo "nvidia discover: missing required field at index=$idx uuid=$uuid" >&2
    continue
  fi

  # Convert memory MB to bytes.
  mem_bytes=$(collector_mib_to_bytes_or_null "$mem_total")

  collector_emit_device nvidia "$idx" "$uuid" "$name" "$pci" "${driver:-unknown}" "$mem_bytes"
done

# Check for parse errors from the pipe.
if [ "${PIPESTATUS[0]}" != "0" ]; then
  exit 40
fi

exit 0
