#!/bin/sh
# LightAI GPU Collector - MetaX Metrics
# Uses mx-smi default summary + mx-smi -L for real-time metrics.
# Exit codes: 0=success, 10=not_available, 30=command_failed, 40=parse_failed

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/../common.sh"

# Find mx-smi.
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

# 1. Build index -> uuid/raw_model map from mx-smi -L.
LIST_OUTPUT=$("$MX_SMI_CMD" -L 2>/dev/null) || true

uuid_map=""
name_map=""
if [ -n "$LIST_OUTPUT" ]; then
  echo "$LIST_OUTPUT" | while IFS= read -r line; do
    [ -z "$line" ] && continue
    idx=$(echo "$line" | sed -n 's/^GPU#\([0-9]*\).*/\1/p' | collector_trim)
    raw_model=$(echo "$line" | sed -n 's/^GPU#[0-9]*[[:space:]]\+\([^[:space:]]*\).*/\1/p' | collector_trim)
    uuid=$(echo "$line" | sed -n 's/.*UUID: \([^)]*\).*/\1/p' | collector_trim)
    if [ -n "$idx" ] && [ -n "$uuid" ]; then
      name=$(collector_normalize_metax_name "$raw_model")
      echo "$idx $uuid $name"
    fi
  done > /tmp/lightai-metax-map.$$
else
  collector_emit_status metax false "mx-smi -L failed"
  exit 30
fi

# 2. Get real-time metrics from mx-smi default summary.
SUMMARY=$("$MX_SMI_CMD" 2>/dev/null) || {
  echo "mx-smi metrics failed" >&2
  rm -f /tmp/lightai-metax-map.$$
  collector_emit_status metax false "mx-smi command failed"
  exit 30
}

if [ -z "$SUMMARY" ]; then
  rm -f /tmp/lightai-metax-map.$$
  collector_emit_status metax false "no MetaX GPUs found"
  exit 10
fi

collector_emit_status metax true ok

# 3. Parse mx-smi summary table.
# The table is two lines per GPU:
# Line 1: | <idx> <name> | <temp_idx> <ecc> | <pci> | <util>% <persistence> |
# Line 2: | <power>W / <power_limit>W | <temp>C <perf> | <mem_used>/<mem_total> MiB | <state> |
#
# We accumulate line 1, then complete on line 2.

parse_error=0
gpu_line1=""
in_table=0

echo "$SUMMARY" | while IFS= read -r line; do
  # Detect table start: lines with pipe separators that contain GPU data.
  if echo "$line" | grep -qE '^\|.*[0-9]+.*MiB.*\|' || echo "$line" | grep -qE '^\|.*[0-9]+[[:space:]]+[A-Za-z]'; then
    in_table=1
  fi
  [ "$in_table" = "0" ] && continue

  # Check if this is a "line 1" (has PCI address pattern).
  if echo "$line" | grep -qE '[0-9a-fA-F]{4}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-9a-fA-F]'; then
    gpu_line1="$line"
    continue
  fi

  # Check if this is a "line 2" (has MiB pattern).
  if echo "$line" | grep -qE '[0-9]+/[0-9]+[[:space:]]*MiB'; then
    line2="$line"
    line1="$gpu_line1"
    gpu_line1=""

    [ -z "$line1" ] && continue

    # Parse line 1: | <idx> <name> | <ign> | <pci> | <util>% <ign> |
    # Field 1: "<idx> <name>"
    f1_1=$(echo "$line1" | cut -d'|' -f2 | collector_trim)
    idx=$(echo "$f1_1" | awk '{print $1}')
    name_from_summary=$(echo "$f1_1" | cut -d' ' -f2- | collector_trim)

    # Field 3: PCI address
    pci=$(echo "$line1" | cut -d'|' -f4 | collector_trim)

    # Field 4: "<util>% <persistence>"
    f1_4=$(echo "$line1" | cut -d'|' -f5 | collector_trim)
    gpu_util=$(echo "$f1_4" | awk '{print $1}' | sed 's/%//' | collector_trim)

    # Parse line 2: | <power>W / <limit>W | <temp>C <perf> | <used>/<total> MiB | <state> |
    f2_1=$(echo "$line2" | cut -d'|' -f2 | collector_trim)
    power=$(echo "$f2_1" | awk '{print $1}' | sed 's/W//' | collector_trim)

    f2_2=$(echo "$line2" | cut -d'|' -f3 | collector_trim)
    temp=$(echo "$f2_2" | awk '{print $1}' | sed 's/C//' | collector_trim)

    f2_3=$(echo "$line2" | cut -d'|' -f4 | collector_trim)
    mem_used_mib=$(echo "$f2_3" | awk -F'/' '{print $1}' | collector_trim)
    mem_total_mib=$(echo "$f2_3" | awk -F'/' '{print $2}' | awk '{print $1}' | collector_trim)

    f2_4=$(echo "$line2" | cut -d'|' -f5 | collector_trim)
    gpu_state=$(echo "$f2_4" | collector_trim)

    # Look up uuid and name from the map.
    uuid=""
    name_from_map=""
    if [ -f /tmp/lightai-metax-map.$$ ]; then
      while read -r mid muuid mname; do
        if [ "$mid" = "$idx" ]; then
          uuid="$muuid"
          name_from_map="$mname"
          break
        fi
      done < /tmp/lightai-metax-map.$$
    fi

    # Use summary name if available, otherwise map name.
    name="$name_from_summary"
    if [ -z "$name" ] || [ "$name" = "$idx" ]; then
      name="$name_from_map"
    fi
    [ -z "$name" ] && name=$(collector_normalize_metax_name "$name_from_map")

    if [ -z "$idx" ] || [ -z "$uuid" ]; then
      echo "metax metrics: missing index or uuid at idx=$idx" >&2
      parse_error=1
      continue
    fi

    # Convert MiB to bytes.
    mem_total_bytes=$(collector_mib_to_bytes_or_null "$mem_total_mib")
    mem_used_bytes=$(collector_mib_to_bytes_or_null "$mem_used_mib")
    mem_free_bytes=$(collector_calc_free_bytes_or_null "$mem_total_bytes" "$mem_used_bytes")

    # Normalize values.
    gpu_util_val=$(collector_percent_or_null "$gpu_util")
    mem_util_val=$(collector_calc_percent_or_null "$mem_used_bytes" "$mem_total_bytes")
    temp_val=$(collector_number_or_null "$temp")
    power_val=$(collector_number_or_null "$power")

    # Map GPU state.
    health="unknown"
    status="unknown"
    case "$gpu_state" in
      Available) health="healthy"; status="available" ;;
      "In Use")  health="healthy"; status="available" ;;
      Unavailable) health="warning"; status="unavailable" ;;
      Error)      health="unhealthy"; status="unavailable" ;;
    esac

    collector_emit_metric metax "$idx" "$uuid" "$name" \
      "$mem_total_bytes" "$mem_used_bytes" "$mem_free_bytes" \
      "$gpu_util_val" "$mem_util_val" "$temp_val" "$power_val" \
      "$health" "$status"
  fi
done

rm -f /tmp/lightai-metax-map.$$
[ "$parse_error" = "1" ] && exit 40
exit 0
