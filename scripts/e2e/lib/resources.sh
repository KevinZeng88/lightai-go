#!/usr/bin/env bash
# Resource tracking helpers for controlled E2E cleanup.

if [ "${LIGHTAI_E2E_RESOURCES_SH:-}" = "1" ]; then
  return 0 2>/dev/null || exit 0
fi
LIGHTAI_E2E_RESOURCES_SH=1

set -euo pipefail

E2E_RESOURCES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$E2E_RESOURCES_DIR/env.sh"

E2E_RESOURCES_FILE="$LIGHTAI_E2E_ARTIFACT_DIR/resources.tsv"
: > "$E2E_RESOURCES_FILE"

e2e_resource_name() {
  local flow="${1:-flow}"
  printf '%s-%s' "$LIGHTAI_E2E_PREFIX" "$flow"
}

e2e_register_resource() {
  local kind="$1" id="$2" cleanup_path="${3:-}"
  printf '%s\t%s\t%s\n' "$kind" "$id" "$cleanup_path" >> "$E2E_RESOURCES_FILE"
}

e2e_registered_resources() {
  cat "$E2E_RESOURCES_FILE"
}
