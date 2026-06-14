#!/bin/sh
# LightAI GPU Collector - MetaX Metrics (CSV fast path, POSIX awk)
# Uses combined CSV + mx-smi -L for uuid map.
# Exit codes: 0=success, 10=not_available, 30=command_failed, 40=parse_failed
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/../common.sh"

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

# P1-003: Use mktemp for ALL temp files, clean up on exit.
TMPDIR="${TMPDIR:-/tmp}"
CSV_FILE=$(mktemp "$TMPDIR/lightai-metax-metrics-csv.XXXXXX")
LIST_FILE=$(mktemp "$TMPDIR/lightai-metax-metrics-list.XXXXXX")
ERR_FILE=$(mktemp "$TMPDIR/lightai-metax-metrics-err.XXXXXX")
OUT_FILE=$(mktemp "$TMPDIR/lightai-metax-metrics-out.XXXXXX")
trap 'rm -f "$CSV_FILE" "$LIST_FILE" "$ERR_FILE" "$OUT_FILE"' EXIT

# 1. Combined CSV.
"$MX_SMI_CMD" --show-memory --show-usage --show-temperature --show-board-power -o "$CSV_FILE" >/dev/null 2>/dev/null || {
  echo "mx-smi combined CSV failed" >&2
  collector_emit_status metax false "mx-smi combined CSV command failed"
  exit 30
}
if [ ! -s "$CSV_FILE" ]; then
  collector_emit_status metax false "empty combined CSV"
  exit 30
fi

# 2. GPU list.
"$MX_SMI_CMD" -L > "$LIST_FILE" 2>/dev/null || {
  collector_emit_status metax false "mx-smi -L failed"
  exit 30
}

# 3. Single awk pass — output to temp file for validation.
awk '
function trim(s) { gsub(/^[[:space:]]+|[[:space:]]+$/, "", s); return s }
function quote(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); return s }
function norm_name(raw) {
  if (raw ~ /^MXC[0-9]/) { sub(/^MXC/, "MetaX C", raw); return raw }
  return raw
}
function kb_to_bytes(kb) {
  kb = trim(kb)
  if (kb == "" || kb == "N/A" || kb == "null") return "null"
  return int(kb) * 1024
}
function num_or_null(v) {
  v = trim(v)
  if (v == "" || v == "N/A" || v == "[N/A]" || v == "null") return "null"
  return v
}
function extract_match(str, pat,    s) {
  if (match(str, pat)) { s = substr(str, RSTART, RLENGTH); return trim(s) }
  return ""
}
function extract_capture(str, pat, keep,    s) {
  if (match(str, pat)) { s = substr(str, RSTART, RLENGTH); sub(keep, "", s); return trim(s) }
  return ""
}

# File 1: mx-smi -L output.
FILENAME == LISTFILE && /^[[:space:]]*GPU#[0-9]+/ {
  line = $0; gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
  idx = extract_capture(line, "GPU#[0-9]+", "GPU#")
  split(line, f, /[[:space:]]+/)
  raw = (length(f) >= 2) ? f[2] : ""
  st = extract_match(line, "Available|Unavailable|In Use|Error")
  uuid = extract_capture(line, "UUID: [^)]+", "UUID: ")
  if (idx != "" && uuid != "") { uuid_map[idx]=uuid; raw_map[idx]=raw; state_map[idx]=st }
  next
}
FILENAME == LISTFILE { next }

# File 2: CSV header.
FNR == 1 {
  FS=","; $0=$0
  for (i=1; i<=NF; i++) {
    hdr = trim($i)
    if (hdr == "deviceId") devId_col = i
    if (hdr ~ /deviceName/) devName_col = i
    if (hdr ~ /utilization\.vram\.total/) vramTotal_col = i
    if (hdr ~ /utilization\.vram\.used/)  vramUsed_col  = i
    if (hdr ~ /utilization\.vram\.usage/) vramUsage_col = i
    if (hdr ~ /utilization\.GPU/)          gpuUtil_col   = i
    if (hdr ~ /temperature\.hotspot/)      temp_col      = i
    if (hdr ~ /power.*\[W\]/)             power_col     = i
  }
  next
}

# File 2: CSV data rows.
{
  FS=","; $0=$0
  if (NF < 5) next

  did = trim($(devId_col))
  gidx = extract_capture(did, "GPU#[0-9]+", "GPU#")
  if (gidx == "") next

  uuid = uuid_map[gidx]
  if (uuid == "") { print "metax metrics: missing uuid for index " gidx > "/dev/stderr"; next }

  name = trim($(devName_col))
  if (name == "" || name == "null") name = raw_map[gidx]
  if (name == "") name = "unknown"
  name = norm_name(name)

  mem_total = kb_to_bytes(trim($(vramTotal_col)))
  mem_used  = kb_to_bytes(trim($(vramUsed_col)))
  mem_free  = "null"
  if (mem_total != "null" && mem_used != "null") mem_free = int(mem_total) - int(mem_used)

  mem_util = num_or_null($(vramUsage_col))
  gpu_util = num_or_null($(gpuUtil_col))
  temp_val = num_or_null($(temp_col))
  power_val = num_or_null($(power_col))

  # P1-002: Health from GPU state, but also check metric quality.
  st = state_map[gidx]
  health = "unknown"; status = "unknown"
  if (st == "Available")   { health = "healthy";   status = "available" }
  else if (st == "In Use") { health = "healthy";   status = "available" }
  else if (st == "Unavailable") { health = "warning"; status = "unavailable" }
  else if (st == "Error")  { health = "unhealthy"; status = "unavailable" }

  # Override: if core metrics failed to parse, downgrade health.
  if (mem_total == "null" || mem_used == "null") {
    health = "error"
    status = "unavailable"
  } else if (gpu_util == "null" || temp_val == "null") {
    if (health == "healthy") health = "degraded"
  }

  printf "METRIC vendor=metax index=%s uuid=%s name=\"%s\" memory_total_bytes=%s memory_used_bytes=%s memory_free_bytes=%s gpu_utilization_percent=%s memory_utilization_percent=%s temperature_celsius=%s power_draw_watts=%s health=%s status=%s\n",
    gidx, uuid, quote(name), mem_total, mem_used, mem_free, gpu_util, mem_util, temp_val, power_val, health, status
}
' LISTFILE="$LIST_FILE" "$LIST_FILE" "$CSV_FILE" > "$OUT_FILE" 2>"$ERR_FILE"

# P1-002: Validate output before declaring success.
if [ -s "$OUT_FILE" ]; then
  cat "$OUT_FILE"
  collector_emit_status metax true ok
elif [ -s "$ERR_FILE" ]; then
  collector_emit_status metax false "parse failed"
  exit 40
else
  collector_emit_status metax false "no metrics produced"
  exit 40
fi

exit 0
