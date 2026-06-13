#!/bin/sh
# LightAI GPU Collector - NVIDIA Metrics
# Exit codes: 0=success, 10=not_available, 30=command_failed, 40=parse_failed

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/../common.sh"

NVIDIA_SMI=""
NVIDIA_SMI=$(collector_find_command nvidia-smi /usr/bin/nvidia-smi /usr/lib/wsl/lib/nvidia-smi) || {
  collector_emit_status nvidia false "nvidia-smi not found"
  exit 10
}

QUERY="index,name,uuid,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw"
OUTPUT=$("$NVIDIA_SMI" --query-gpu="$QUERY" --format=csv,noheader,nounits 2>/dev/null) || {
  echo "nvidia-smi metrics failed: exit=$?" >&2
  collector_emit_status nvidia false "nvidia-smi command failed"
  exit 30
}

if [ -z "$OUTPUT" ]; then
  collector_emit_status nvidia false "no NVIDIA GPUs found"
  exit 10
fi

collector_emit_status nvidia true ok

parse_error=0
echo "$OUTPUT" | while IFS=',' read -r idx name uuid mem_total mem_used mem_free gpu_util mem_util temp power; do
  idx=$(echo "$idx" | collector_trim)
  name=$(echo "$name" | collector_trim)
  uuid=$(echo "$uuid" | collector_trim)
  mem_total=$(echo "$mem_total" | collector_trim)
  mem_used=$(echo "$mem_used" | collector_trim)
  mem_free=$(echo "$mem_free" | collector_trim)
  gpu_util=$(echo "$gpu_util" | collector_trim)
  mem_util=$(echo "$mem_util" | collector_trim)
  temp=$(echo "$temp" | collector_trim)
  power=$(echo "$power" | collector_trim)

  if [ -z "$idx" ] || [ -z "$uuid" ]; then
    echo "nvidia metrics: missing index or uuid at idx=$idx" >&2
    parse_error=1
    continue
  fi

  mem_total_bytes=$(collector_mib_to_bytes_or_null "$mem_total")
  mem_used_bytes=$(collector_mib_to_bytes_or_null "$mem_used")

  if [ "$mem_free" = "" ] || [ "$mem_free" = "N/A" ] || [ "$mem_free" = "[N/A]" ]; then
    mem_free_bytes=$(collector_calc_free_bytes_or_null "$mem_total_bytes" "$mem_used_bytes")
  else
    mem_free_bytes=$(collector_mib_to_bytes_or_null "$mem_free")
  fi

  gpu_util_val=$(collector_percent_or_null "$gpu_util")

  mem_util_val=$(collector_percent_or_null "$mem_util")
  if [ "$mem_util_val" = "null" ] && [ "$mem_total_bytes" != "null" ] && [ "$mem_used_bytes" != "null" ]; then
    mem_util_val=$(collector_calc_percent_or_null "$mem_used_bytes" "$mem_total_bytes")
  fi

  temp_val=$(collector_number_or_null "$temp")
  power_val=$(collector_number_or_null "$power")

  collector_emit_metric nvidia "$idx" "$uuid" "$name" \
    "$mem_total_bytes" "$mem_used_bytes" "$mem_free_bytes" \
    "$gpu_util_val" "$mem_util_val" "$temp_val" "$power_val" \
    healthy available
done

[ "$parse_error" = "1" ] && exit 40
exit 0
