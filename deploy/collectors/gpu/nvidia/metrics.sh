#!/bin/sh
# LightAI GPU Collector - NVIDIA Metrics (awk fast path)
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
  echo "nvidia-smi metrics failed" >&2
  collector_emit_status nvidia false "nvidia-smi command failed"
  exit 30
}

if [ -z "$OUTPUT" ]; then
  collector_emit_status nvidia false "no NVIDIA GPUs found"
  exit 10
fi

collector_emit_status nvidia true ok

# Single awk pass: parse CSV, emit METRIC lines.
echo "$OUTPUT" | awk -F, '
function trim(s) { gsub(/^[[:space:]]+|[[:space:]]+$/, "", s); return s }
function quote(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); return s }
function mib_to_bytes(v) {
  v = trim(v)
  if (v == "" || v == "N/A" || v == "null") return "null"
  return int(v) * 1024 * 1024
}
function num_or_null(v) {
  v = trim(v)
  if (v == "" || v == "N/A" || v == "[N/A]" || v == "null") return "null"
  return v
}
{
  if (NF < 5) { print "nvidia metrics: too few fields: " NF > "/dev/stderr"; next }
  idx     = trim($1)
  name    = trim($2)
  uuid    = trim($3)
  mt_raw  = trim($4)
  mu_raw  = trim($5)
  mf_raw  = trim($6)
  gu_raw  = trim($7)
  mu2_raw = trim($8)
  tmp_raw = trim($9)
  pw_raw  = trim($10)

  if (idx == "" || uuid == "") { print "nvidia metrics: missing required field" > "/dev/stderr"; next }

  mt = mib_to_bytes(mt_raw)
  mu = mib_to_bytes(mu_raw)
  mf = mib_to_bytes(mf_raw)
  # Calculate free if missing or N/A
  if (mf == "null" && mt != "null" && mu != "null") mf = int(mt) - int(mu)

  gu = num_or_null(gu_raw)
  mu2 = num_or_null(mu2_raw)
  # Calculate memory util if missing
  if (mu2 == "null" && mt != "null" && mu != "null") mu2 = int(mu) * 100 / int(mt)

  tmp = num_or_null(tmp_raw)
  pw  = num_or_null(pw_raw)

  printf "METRIC vendor=nvidia index=%s uuid=%s name=\"%s\" memory_total_bytes=%s memory_used_bytes=%s memory_free_bytes=%s gpu_utilization_percent=%s memory_utilization_percent=%s temperature_celsius=%s power_draw_watts=%s health=%s status=%s\n",
    idx, uuid, quote(name), mt, mu, mf, gu, mu2, tmp, pw, "healthy", "available"
}
'
exit 0
