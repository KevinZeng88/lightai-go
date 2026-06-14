#!/bin/sh
# LightAI GPU Collector - NVIDIA Discover (awk fast path)
# Exit codes: 0=success, 10=not_available, 30=command_failed, 40=parse_failed
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/../common.sh"

NVIDIA_SMI=""
NVIDIA_SMI=$(collector_find_command nvidia-smi /usr/bin/nvidia-smi /usr/lib/wsl/lib/nvidia-smi) || {
  collector_emit_status nvidia false "nvidia-smi not found"
  exit 10
}

OUTPUT=$("$NVIDIA_SMI" --query-gpu=index,name,uuid,pci.bus_id,driver_version,memory.total --format=csv,noheader,nounits 2>/dev/null) || {
  echo "nvidia-smi discover failed" >&2
  collector_emit_status nvidia false "nvidia-smi command failed"
  exit 30
}

if [ -z "$OUTPUT" ]; then
  collector_emit_status nvidia false "no NVIDIA GPUs found"
  exit 10
fi

# P1-009: Use mktemp for temp files, trap to clean up.
OUT_FILE=$(mktemp /tmp/lightai-nvidia-discover-out.XXXXXX)
ERR_FILE=$(mktemp /tmp/lightai-nvidia-discover-err.XXXXXX)
trap 'rm -f "$OUT_FILE" "$ERR_FILE"' EXIT

# Single awk pass: parse CSV, emit DEVICE lines.
echo "$OUTPUT" | awk -F, '
function trim(s) { gsub(/^[[:space:]]+|[[:space:]]+$/, "", s); return s }
function quote(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); return s }
function mib_to_bytes(v) {
  v = trim(v)
  if (v == "" || v == "N/A" || v == "null") return "null"
  return int(v) * 1024 * 1024
}
{
  if (NF < 5) { print "nvidia discover: too few fields: " NF > "/dev/stderr"; next }
  idx   = trim($1)
  name  = trim($2)
  uuid  = trim($3)
  pci   = trim($4)
  driver = trim($5)
  mem   = trim($6)
  if (idx == "" || uuid == "" || name == "") { print "nvidia discover: missing required field" > "/dev/stderr"; next }
  mem_bytes = mib_to_bytes(mem)
  printf "DEVICE vendor=nvidia index=%s uuid=%s name=\"%s\" pci_bus_id=%s driver_version=%s memory_total_bytes=%s\n", idx, uuid, quote(name), (pci==""?"unknown":pci), (driver==""?"unknown":driver), mem_bytes
  fflush()
}
' > $OUT_FILE 2>$ERR_FILE

# P1-002: Check if any DEVICE was actually emitted.
if [ -s $OUT_FILE ]; then
  cat $OUT_FILE
  collector_emit_status nvidia true ok
elif [ -s $ERR_FILE ]; then
  collector_emit_status nvidia false "parse failed"
  rm -f $OUT_FILE $ERR_FILE
  exit 40
else
  collector_emit_status nvidia false "no devices parsed"
  rm -f $OUT_FILE $ERR_FILE
  exit 10
fi

rm -f $OUT_FILE $ERR_FILE
exit 0
