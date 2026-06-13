#!/bin/sh
# LightAI GPU Collector - MetaX Discover (fast path)
# Uses mx-smi -L for GPU list + --show-version for driver version.
# Exit codes: 0=success, 10=not_available, 30=command_failed, 40=parse_failed
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/../common.sh"

# Find mx-smi.
MX_SMI_CMD=""
if [ -n "${MX_SMI:-}" ] && [ -x "$MX_SMI" ]; then
  MX_SMI_CMD="$MX_SMI"
else
  MX_SMI_CMD=$(collector_find_command mx-smi /usr/bin/mx-smi /usr/local/bin/mx-smi \
    /opt/maca/bin/mx-smi /usr/local/maca/bin/mx-smi /opt/mxdriver/bin/mx-smi) || {
    collector_emit_status metax false "mx-smi not found"
    exit 10
  }
fi

# Driver version: prefer --show-version, fallback to summary header.
DRIVER_VERSION="unknown"
VER_OUTPUT=$("$MX_SMI_CMD" --show-version 2>/dev/null) || true
if [ -n "$VER_OUTPUT" ]; then
  DRIVER_VERSION=$(echo "$VER_OUTPUT" | awk '/KMD/{gsub(/^[[:space:]]+/,""); gsub(/[[:space:]]+$/,""); split($0,a,":"); v=a[2]; gsub(/^[[:space:]]+/,"",v); gsub(/[[:space:]]+$/,"",v); print v; exit}')
  [ -z "$DRIVER_VERSION" ] && DRIVER_VERSION="unknown"
fi
if [ "$DRIVER_VERSION" = "unknown" ]; then
  HEADER=$("$MX_SMI_CMD" 2>/dev/null | head -5) || true
  if [ -n "$HEADER" ]; then
    DRIVER_VERSION=$(echo "$HEADER" | grep -oE 'Kernel Mode Driver Version:\s*[0-9.]+' | sed 's/Kernel Mode Driver Version:\s*//' | collector_trim)
    [ -z "$DRIVER_VERSION" ] && DRIVER_VERSION="unknown"
  fi
fi

# GPU list from mx-smi -L.
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

# Single awk pass: parse only GPU# lines, emit DEVICE.
echo "$LIST_OUTPUT" | awk -v driver="$DRIVER_VERSION" '
function trim(s) { gsub(/^[[:space:]]+|[[:space:]]+$/, "", s); return s }
function quote(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); return s }
function norm_name(raw) {
  if (raw ~ /^MXC[0-9]/) { sub(/^MXC/, "MetaX C", raw); return raw }
  return raw
}
/^[[:space:]]*GPU#[0-9]+[[:space:]]/ {
  idx=""; raw_model=""; pci=""; state=""; uuid=""
  gsub(/^[[:space:]]+|[[:space:]]+$/, "")
  if (match($0, /GPU#([0-9]+)/, a)) idx=a[1]
  # raw_model is the second field after GPU#N
  split($0, f, /[[:space:]]+/)
  if (length(f) >= 2) raw_model=f[2]
  if (match($0, /[0-9a-fA-F]{4}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-9a-fA-F]/, a)) pci=a[0]
  if (match($0, /(Available|Unavailable|In Use|Error)/, a)) state=a[1]
  if (match($0, /UUID: ([^)]+)/, a)) uuid=a[1]
  idx=trim(idx); uuid=trim(uuid); pci=trim(pci)
  if (idx == "" || uuid == "") { print "metax discover: missing index or uuid: " $0 > "/dev/stderr"; next }
  name = raw_model
  if (name == "") name = "unknown"
  name = norm_name(name)
  printf "DEVICE vendor=metax index=%s uuid=%s name=\"%s\" pci_bus_id=%s driver_version=%s memory_total_bytes=null\n", idx, uuid, quote(name), (pci==""?"unknown":pci), driver
}
' 2>/tmp/lightai-metax-discover.err

if [ -s /tmp/lightai-metax-discover.err ]; then
  rm -f /tmp/lightai-metax-discover.err
  exit 40
fi
rm -f /tmp/lightai-metax-discover.err
exit 0
