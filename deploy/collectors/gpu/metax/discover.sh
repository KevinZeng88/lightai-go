#!/bin/sh
# LightAI GPU Collector - MetaX Discover (POSIX awk)
# Uses mx-smi -L for GPU list + --show-version for driver version.
# Exit codes: 0=success, 10=not_available, 30=command_failed, 40=parse_failed
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/../common.sh"

# P1-003: Use mktemp for error files, clean up on exit.
ERR_FILE=$(mktemp /tmp/lightai-metax-discover-err.XXXXXX)
OUT_FILE=$(mktemp /tmp/lightai-metax-discover-out.XXXXXX)
trap 'rm -f "$ERR_FILE" "$OUT_FILE"' EXIT

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

# Driver version.
DRIVER_VERSION="unknown"
VER_OUTPUT=$("$MX_SMI_CMD" --show-version 2>/dev/null) || true
if [ -n "$VER_OUTPUT" ]; then
  DRIVER_VERSION=$(echo "$VER_OUTPUT" | awk '
    /KMD/ {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "")
      n = split($0, a, ":")
      if (n >= 2) { v = a[2]; gsub(/^[[:space:]]+|[[:space:]]+$/, "", v); print v; exit }
    }
  ')
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

# Single awk pass: parse GPU# lines, emit DEVICE to temp output file.
echo "$LIST_OUTPUT" | awk -v driver="$DRIVER_VERSION" '
function trim(s) { gsub(/^[[:space:]]+|[[:space:]]+$/, "", s); return s }
function quote(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); return s }
function norm_name(raw) {
  if (raw ~ /^MXC[0-9]/) { sub(/^MXC/, "MetaX C", raw); return raw }
  return raw
}
function extract_match(str, pat,    s) {
  if (match(str, pat)) { s = substr(str, RSTART, RLENGTH); return trim(s) }
  return ""
}
function extract_capture(str, pat, keep,    s) {
  if (match(str, pat)) { s = substr(str, RSTART, RLENGTH); sub(keep, "", s); return trim(s) }
  return ""
}
/^[[:space:]]*GPU#[0-9]+[[:space:]]/ {
  gsub(/^[[:space:]]+|[[:space:]]+$/, "")
  idx = extract_capture($0, "GPU#[0-9]+", "GPU#")
  split($0, f, /[[:space:]]+/)
  raw_model = (length(f) >= 2) ? f[2] : ""
  pci = extract_match($0, "[0-9a-fA-F]{4}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}.[0-9a-fA-F]")
  state = extract_match($0, "Available|Unavailable|In Use|Error")
  uuid = extract_capture($0, "UUID: [^)]+", "UUID: ")
  if (idx == "" || uuid == "") { print "metax discover: missing index or uuid: " $0 > "/dev/stderr"; next }
  name = raw_model
  if (name == "") name = "unknown"
  name = norm_name(name)
  printf "DEVICE vendor=metax index=%s uuid=%s name=\"%s\" pci_bus_id=%s driver_version=%s memory_total_bytes=null\n", idx, uuid, quote(name), (pci==""?"unknown":pci), driver
}
' > "$OUT_FILE" 2>"$ERR_FILE"

# P1-002: Validate output before declaring success.
if [ -s "$OUT_FILE" ]; then
  cat "$OUT_FILE"
  collector_emit_status metax true ok
elif [ -s "$ERR_FILE" ]; then
  collector_emit_status metax false "parse failed"
  exit 40
else
  collector_emit_status metax false "no devices parsed"
  exit 10
fi

exit 0
