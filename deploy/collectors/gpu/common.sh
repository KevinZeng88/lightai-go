#!/bin/sh
# LightAI GPU Collector - Common Helpers
# Source this file from vendor scripts. Do NOT execute directly.
# All functions use collector_ prefix to avoid polluting vendor scripts.

if [ "${COLLECTOR_COMMON_SOURCED:-}" = "1" ]; then
  return 0
fi
COLLECTOR_COMMON_SOURCED=1

# ---- String helpers ----

# collector_trim: remove leading and trailing whitespace.
collector_trim() {
  sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//'
}

# collector_quote: escape backslash and double-quote for protocol output.
collector_quote() {
  sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

# ---- Numeric helpers ----

# collector_is_int: true if arg is a non-negative integer.
collector_is_int() {
  case "$1" in
    ''|*[!0-9]*) return 1 ;;
    *) return 0 ;;
  esac
}

# collector_is_number: true if arg is an integer or decimal.
collector_is_number() {
  case "$1" in
    ''|*[!0-9.]*|.*|*.*.*) return 1 ;;
    *) return 0 ;;
  esac
}

# ---- Value normalization ----

# collector_null_if_na: output null if value is N/A-like, else output value.
collector_null_if_na() {
  case "$1" in
    ""|"N/A"|"[N/A]"|"Unknown"|"unknown"|"None"|"null") echo "null" ;;
    *) echo "$1" ;;
  esac
}

# collector_mib_to_bytes_or_null: convert MiB integer to bytes, or null.
collector_mib_to_bytes_or_null() {
  case "$1" in
    ""|"N/A"|"[N/A]"|"Unknown"|"unknown"|"None"|"null")
      echo "null" ;;
    *)
      if collector_is_int "$1"; then
        echo $(($1 * 1024 * 1024))
      else
        echo "null"
      fi
      ;;
  esac
}

# collector_percent_or_null: strip trailing %, output number or null.
collector_percent_or_null() {
  local val
  val=$(echo "$1" | sed 's/%$//' | collector_trim)
  collector_null_if_na "$val"
}

# collector_number_or_null: output number or null (temp, power, etc.).
collector_number_or_null() {
  local val
  val=$(echo "$1" | collector_trim)
  collector_null_if_na "$val"
}

# collector_calc_free_bytes_or_null: total - used, or null.
collector_calc_free_bytes_or_null() {
  if collector_is_int "$1" && collector_is_int "$2" && [ "$1" -gt 0 ] 2>/dev/null; then
    echo $(($1 - $2))
  else
    echo "null"
  fi
}

# collector_calc_percent_or_null: used / total * 100, 1 decimal, or null.
collector_calc_percent_or_null() {
  if collector_is_int "$1" && collector_is_int "$2" && [ "$2" -gt 0 ] 2>/dev/null; then
    # Integer arithmetic with one decimal: (used * 1000 / total + 5) / 10
    local p
    p=$(( $1 * 1000 / $2 ))
    echo "$(( p / 10 )).$(( p % 10 ))"
  else
    echo "null"
  fi
}

# ---- MetaX name normalization ----

# collector_normalize_metax_name: convert MXCxxx -> MetaX Cxxx, or pass-through.
collector_normalize_metax_name() {
  local raw
  raw="$1"
  case "$raw" in
    MXC[0-9]*)
      # Extract the Cxxx part: MXC500 -> C500, MXC550 -> C550, etc.
      echo "MetaX C${raw#MXC}" ;;
    *)
      echo "$raw" ;;
  esac
}

# ---- Protocol emitters ----

# collector_emit_status: output STATUS line.
collector_emit_status() {
  local vendor="$1" ok="$2" message="$3"
  local escaped
  escaped=$(echo "$message" | collector_quote)
  echo "STATUS vendor=${vendor} ok=${ok} message=\"${escaped}\""
}

# collector_emit_device: output DEVICE line.
collector_emit_device() {
  local vendor="$1" index="$2" uuid="$3" name="$4" pci_bus_id="$5" driver_version="$6" memory_total_bytes="$7"
  local escaped_name
  escaped_name=$(echo "$name" | collector_quote)
  [ -z "$pci_bus_id" ] && pci_bus_id="unknown"
  [ -z "$driver_version" ] && driver_version="unknown"
  [ -z "$memory_total_bytes" ] || [ "$memory_total_bytes" = "null" ] || true
  echo "DEVICE vendor=${vendor} index=${index} uuid=${uuid} name=\"${escaped_name}\" pci_bus_id=${pci_bus_id} driver_version=${driver_version} memory_total_bytes=${memory_total_bytes:-null}"
}

# collector_emit_metric: output METRIC line.
collector_emit_metric() {
  local vendor="$1" index="$2" uuid="$3" name="$4"
  local mem_total="$5" mem_used="$6" mem_free="$7"
  local gpu_util="$8" mem_util="$9" temp="${10}" power="${11}" health="${12}" status="${13}"
  local escaped_name
  escaped_name=$(echo "$name" | collector_quote)

  [ -z "$mem_total" ] && mem_total="null"
  [ -z "$mem_used" ] && mem_used="null"
  [ -z "$mem_free" ] && mem_free="null"
  [ -z "$gpu_util" ] && gpu_util="null"
  [ -z "$mem_util" ] && mem_util="null"
  [ -z "$temp" ] && temp="null"
  [ -z "$power" ] && power="null"
  [ -z "$health" ] && health="unknown"
  [ -z "$status" ] && status="unknown"

  echo "METRIC vendor=${vendor} index=${index} uuid=${uuid} name=\"${escaped_name}\" memory_total_bytes=${mem_total} memory_used_bytes=${mem_used} memory_free_bytes=${mem_free} gpu_utilization_percent=${gpu_util} memory_utilization_percent=${mem_util} temperature_celsius=${temp} power_draw_watts=${power} health=${health} status=${status}"
}

# ---- Command discovery ----

# collector_find_command: find first executable from candidate list.
# Usage: collector_find_command cmd1 /path/cmd1 /other/cmd1 ...
# Returns the first found path, or exits non-zero.
collector_find_command() {
  for candidate in "$@"; do
    if command -v "$candidate" >/dev/null 2>&1; then
      command -v "$candidate"
      return 0
    fi
    if [ -x "$candidate" ]; then
      echo "$candidate"
      return 0
    fi
  done
  return 1
}
