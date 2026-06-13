#!/bin/sh
# LightAI GPU Collector - NVIDIA Metrics
# Converts nvidia-smi output to LightAI GPU Collector Protocol.
# Exit codes: 0=success, 10=not_available, 30=command_failed

set -e

NVIDIA_SMI=""

if command -v nvidia-smi >/dev/null 2>&1; then
  NVIDIA_SMI="nvidia-smi"
elif [ -x "/usr/bin/nvidia-smi" ]; then
  NVIDIA_SMI="/usr/bin/nvidia-smi"
elif [ -x "/usr/lib/wsl/lib/nvidia-smi" ]; then
  NVIDIA_SMI="/usr/lib/wsl/lib/nvidia-smi"
else
  echo "STATUS vendor=nvidia ok=false message=\"nvidia-smi not found\""
  exit 10
fi

# Execute query with all fields.
QUERY="index,name,uuid,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw"
OUTPUT=$("$NVIDIA_SMI" --query-gpu="$QUERY" --format=csv,noheader,nounits 2>/dev/null) || {
  echo "STATUS vendor=nvidia ok=false message=\"nvidia-smi command failed\""
  exit 30
}

if [ -z "$OUTPUT" ]; then
  echo "STATUS vendor=nvidia ok=true message=\"no NVIDIA GPUs found\""
  exit 0
fi

echo "STATUS vendor=nvidia ok=true message=ok"

# Parse each GPU line.
# Format: index, name, uuid, memory.total, memory.used, memory.free, utilization.gpu, utilization.memory, temperature.gpu, power.draw
echo "$OUTPUT" | while IFS=',' read -r idx name uuid mem_total mem_used mem_free gpu_util mem_util temp power; do
  # Trim whitespace.
  idx=$(echo "$idx" | xargs)
  name=$(echo "$name" | xargs)
  uuid=$(echo "$uuid" | xargs)
  mem_total=$(echo "$mem_total" | xargs)
  mem_used=$(echo "$mem_used" | xargs)
  mem_free=$(echo "$mem_free" | xargs)
  gpu_util=$(echo "$gpu_util" | xargs)
  mem_util=$(echo "$mem_util" | xargs)
  temp=$(echo "$temp" | xargs)
  power=$(echo "$power" | xargs)

  # Convert memory from MB to bytes.
  mem_total_bytes=$(($mem_total * 1024 * 1024))
  mem_used_bytes=$(($mem_used * 1024 * 1024))
  mem_free_bytes=$(($mem_free * 1024 * 1024))

  # Handle null/N/A values for optional fields.
  gpu_util_val="$gpu_util"
  mem_util_val="$mem_util"
  temp_val="$temp"
  power_val="$power"

  case "$gpu_util" in
    ""|"N/A"|"[N/A]"|"Unknown") gpu_util_val="null" ;;
  esac
  case "$mem_util" in
    ""|"N/A"|"[N/A]"|"Unknown") mem_util_val="null" ;;
  esac
  case "$temp" in
    ""|"N/A"|"[N/A]"|"Unknown") temp_val="null" ;;
  esac
  case "$power" in
    ""|"N/A"|"[N/A]"|"Unknown") power_val="null" ;;
  esac

  echo "METRIC vendor=nvidia index=$idx uuid=$uuid name=\"$name\" memory_total_bytes=$mem_total_bytes memory_used_bytes=$mem_used_bytes memory_free_bytes=$mem_free_bytes gpu_utilization_percent=$gpu_util_val memory_utilization_percent=$mem_util_val temperature_celsius=$temp_val power_draw_watts=$power_val health=healthy status=available"
done
