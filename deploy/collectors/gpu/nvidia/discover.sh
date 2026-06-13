#!/bin/sh
# LightAI GPU Collector - NVIDIA Discover
# Converts nvidia-smi output to LightAI GPU Collector Protocol.
# Exit codes: 0=success, 10=not_available, 30=command_failed

set -e

NVIDIA_SMI=""

# Find nvidia-smi.
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

# Execute query.
OUTPUT=$("$NVIDIA_SMI" --query-gpu=index,name,uuid,pci.bus_id,driver_version,memory.total --format=csv,noheader,nounits 2>/dev/null) || {
  echo "STATUS vendor=nvidia ok=false message=\"nvidia-smi command failed\""
  exit 30
}

if [ -z "$OUTPUT" ]; then
  echo "STATUS vendor=nvidia ok=true message=\"no NVIDIA GPUs found\""
  exit 0
fi

echo "STATUS vendor=nvidia ok=true message=ok"

# Parse each GPU line.
# Format: index, name, uuid, pci.bus_id, driver_version, memory.total
echo "$OUTPUT" | while IFS=',' read -r idx name uuid pci driver mem_total; do
  # Trim whitespace.
  idx=$(echo "$idx" | xargs)
  name=$(echo "$name" | xargs)
  uuid=$(echo "$uuid" | xargs)
  pci=$(echo "$pci" | xargs)
  driver=$(echo "$driver" | xargs)
  mem_total=$(echo "$mem_total" | xargs)

  # Convert memory from MB to bytes.
  mem_total_bytes=$(($mem_total * 1024 * 1024))

  echo "DEVICE vendor=nvidia index=$idx uuid=$uuid name=\"$name\" pci_bus_id=$pci driver_version=$driver memory_total_bytes=$mem_total_bytes"
done
