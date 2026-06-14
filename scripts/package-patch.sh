#!/bin/sh
# LightAI Go - Incremental Patch Builder
# Usage: ./scripts/package-patch.sh --from 0.1.0 --to 0.1.1
# Generates:
#   apply-patch.sh           – shell-native patch applier (no python3 required)
#   patch-manifest.json      – JSON metadata (audit / optional python3 enhanced display)
#   patch-files.tsv          – tab-separated file manifest (primary, shell-native)
set -e

FROM_VERSION=""
TO_VERSION=""
FROM_MIN_VERSION=""   # P0-005: minimum version this patch can be applied to
ARCH="linux-amd64"

while [ $# -gt 0 ]; do
  case "$1" in
    --from) FROM_VERSION="$2"; shift 2 ;;
    --to)   TO_VERSION="$2";   shift 2 ;;
    --from-min) FROM_MIN_VERSION="$2"; shift 2 ;;
    *) echo "Usage: $0 --from <ver> --to <ver> [--from-min <min-ver>]" >&2; exit 1 ;;
  esac
done
[ -z "$FROM_VERSION" ] || [ -z "$TO_VERSION" ] && { echo "Usage: $0 --from <ver> --to <ver> [--from-min <min-ver>]" >&2; exit 1; }

# P0-005: If --from-min is not set, use --from as the minimum.
if [ -z "$FROM_MIN_VERSION" ]; then
  FROM_MIN_VERSION="$FROM_VERSION"
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

FROM_DIR="dist/lightai-go-${FROM_VERSION}-${ARCH}"
TO_DIR="dist/lightai-go-${TO_VERSION}-${ARCH}"
[ -d "$FROM_DIR" ] || { echo "ERROR: $FROM_DIR not found. Build first." >&2; exit 1; }
[ -d "$TO_DIR" ]   || { echo "ERROR: $TO_DIR not found. Build first." >&2; exit 1; }

PATCH_NAME="lightai-go-patch-${FROM_VERSION}-to-${TO_VERSION}-${ARCH}"
PATCH_DIR="dist/${PATCH_NAME}"
PATCH_TAR="dist/${PATCH_NAME}.tar.gz"

echo "=== LightAI Go Patch Builder ==="
echo "From: $FROM_VERSION  To: $TO_VERSION"
echo ""

rm -rf "$PATCH_DIR"
mkdir -p "$PATCH_DIR"

# Read versions from VERSION files.
from_version_line() { grep "^$1=" "$FROM_DIR/VERSION" 2>/dev/null | cut -d= -f2- || echo ""; }
to_version_line()   { grep "^$1=" "$TO_DIR/VERSION" 2>/dev/null | cut -d= -f2- || echo ""; }

FROM_PROM_VER=$(from_version_line prometheus_version)
TO_PROM_VER=$(to_version_line prometheus_version)
FROM_GRAF_VER=$(from_version_line grafana_version)
TO_GRAF_VER=$(to_version_line grafana_version)

# Helper: get sha256 of a file (empty if file doesn't exist).
file_sha() { [ -f "$1" ] && sha256sum "$1" | awk '{print $1}' || echo ""; }

# Helper: get octal file mode (e.g. 0755, 0644).
file_mode() { [ -e "$1" ] && stat -c '%a' "$1" 2>/dev/null | awk '{printf "%04d", $1}' || echo "0644"; }

# Check if bin/prometheus binary itself changed.
PROM_SHA_FROM=$(file_sha "$FROM_DIR/bin/prometheus")
PROM_SHA_TO=$(file_sha "$TO_DIR/bin/prometheus")
PROM_CHANGED=false
if [ "$FROM_PROM_VER" != "$TO_PROM_VER" ] || [ "$PROM_SHA_FROM" != "$PROM_SHA_TO" ]; then
  PROM_CHANGED=true
fi

# Check if bin/grafana/ changed (version or content).
GRAF_CHANGED=false
if [ "$FROM_GRAF_VER" != "$TO_GRAF_VER" ]; then
  GRAF_CHANGED=true
else
  GRAF_BIN_FROM=$(file_sha "$FROM_DIR/bin/grafana/bin/grafana")
  GRAF_BIN_TO=$(file_sha "$TO_DIR/bin/grafana/bin/grafana")
  [ "$GRAF_BIN_FROM" != "$GRAF_BIN_TO" ] && GRAF_CHANGED=true
fi

echo "Comparing files..."
CHANGED=0
REMOVED=0

# Temporary file for accumulating TSV lines (without header yet).
TSV_TMP="$PATCH_DIR/.tsv-tmp"
> "$TSV_TMP"

# Function: add a file to the patch and record it in the TSV.
add_patch_file() {
  local file="$1" action="$2" sha="$3"
  local dir mode
  dir=$(dirname "$file")
  mkdir -p "$PATCH_DIR/$dir"
  cp "$TO_DIR/$file" "$PATCH_DIR/$file"
  mode=$(file_mode "$PATCH_DIR/$file")
  printf '%s\t%s\t%s\t%s\n' "$action" "$mode" "$sha" "$file" >> "$TSV_TMP"
}

# Read TO manifest line by line, compare with FROM.
# Format: "file <sha256> <path>" or "symlink <target> <path>"
while IFS= read -r line; do
  [ -z "$line" ] && continue
  type="${line%% *}"
  rest="${line#* }"
  case "$type" in
    file)    sha="${rest%% *}"; file="${rest#* }" ;;
    symlink) sha=""; target="${rest%% *}"; file="${rest#* }" ;;
    *)       continue ;;
  esac
  # file starts with "./"
  case "$file" in
    ./data/*|./logs/*|./run/*|./runtime/*|./data/prometheus/*|./data/grafana/*) continue ;;
  esac

  # Skip Prometheus binary if unchanged.
  case "$file" in
    ./bin/prometheus*|./LICENSES/prometheus/*)
      $PROM_CHANGED || continue ;;
    ./bin/grafana/*|./LICENSES/grafana/*)
      $GRAF_CHANGED || continue ;;
  esac

  from_file="$FROM_DIR/$file"
  if [ "$type" = "symlink" ]; then
    from_target=$(readlink "$from_file" 2>/dev/null || echo "")
    if [ "$from_target" != "$target" ]; then
      add_patch_file "$file" "update" "$target"
      CHANGED=$((CHANGED + 1))
    fi
  elif [ -f "$from_file" ]; then
    from_sha=$(sha256sum "$from_file" | awk '{print $1}')
    if [ "$from_sha" != "$sha" ]; then
      add_patch_file "$file" "update" "$sha"
      CHANGED=$((CHANGED + 1))
    fi
  else
    # New entry in TO.
    add_patch_file "$file" "create" "${sha:-$target}"
    CHANGED=$((CHANGED + 1))
  fi
done < "$TO_DIR/MANIFEST.sha256"

# Check for removed files/symlinks.
while IFS= read -r line; do
  [ -z "$line" ] && continue
  type="${line%% *}"
  rest="${line#* }"
  case "$type" in
    file)    file="${rest#* }" ;;
    symlink) file="${rest#* }" ;;
    *)       continue ;;
  esac
  case "$file" in
    ./data/*|./logs/*|./run/*|./runtime/*) continue ;;
  esac
  if [ ! -f "$TO_DIR/$file" ]; then
    printf 'delete\t0000\t-\t%s\n' "$file" >> "$TSV_TMP"
    REMOVED=$((REMOVED + 1))
  fi
done < "$FROM_DIR/MANIFEST.sha256"

	# --- Write patch-files.tsv (shell-native primary manifest) ---
	TIMESTAMP=$(date -Iseconds)
	{
	  echo "# from_version=$FROM_VERSION"
	  echo "# to_version=$TO_VERSION"
	  echo "# from_min_version=$FROM_MIN_VERSION"
	  echo "# from_max_exclusive=$TO_VERSION"
	  echo "# patch_mode=cumulative"
	  echo "# created_at=$TIMESTAMP"
	  echo "# changed_files=$CHANGED"
	  echo "# removed_files=$REMOVED"
	  echo "# action	mode	sha256	path"
	  cat "$TSV_TMP"
	} > "$PATCH_DIR/patch-files.tsv"
	rm -f "$TSV_TMP"
rm -f "$TSV_TMP"

# --- Write patch-manifest.json (audit / optional python3 enhanced display) ---
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
prom_json() { [ "$1" = "true" ] && echo "true" || echo "false"; }
cat > "$PATCH_DIR/patch-manifest.json" << JSONEOF
{
  "from_version": "$FROM_VERSION",
  "to_version": "$TO_VERSION",
  "patch_mode": "cumulative",
  "created_at": "$TIMESTAMP",
  "git_commit": "$GIT_COMMIT",
  "changed_files": $CHANGED,
  "removed_files": $REMOVED,
  "requires_stop": true,
  "prometheus_version_from": "$FROM_PROM_VER",
  "prometheus_version_to": "$TO_PROM_VER",
  "prometheus_included": $(prom_json "$PROM_CHANGED"),
  "grafana_version_from": "$FROM_GRAF_VER",
  "grafana_version_to": "$TO_GRAF_VER",
  "grafana_included": $(prom_json "$GRAF_CHANGED")
}
JSONEOF

# Copy apply-patch.sh into the patch.
cp "$PROJECT_DIR/scripts/apply-patch.sh" "$PATCH_DIR/apply-patch.sh" 2>/dev/null || true

# Build tarball.
rm -f "$PATCH_TAR"
tar -czf "$PATCH_TAR" -C dist "$PATCH_NAME"

echo ""
echo "=== Patch Summary ==="
echo "Changed: $CHANGED files  Removed: $REMOVED files"
echo "Prometheus: $FROM_PROM_VER -> $TO_PROM_VER ($($PROM_CHANGED && echo 'INCLUDED' || echo 'unchanged, skipped'))"
echo "Grafana:    $FROM_GRAF_VER -> $TO_GRAF_VER ($($GRAF_CHANGED && echo 'INCLUDED' || echo 'unchanged, skipped'))"
echo ""
echo "Patch: $PATCH_TAR"
echo "Size:  $(du -h "$PATCH_TAR" | cut -f1)"
echo "Full:  $(du -sh "$TO_DIR" | cut -f1)"
echo ""
echo "Apply: tar -xzf ${PATCH_NAME}.tar.gz && cd ${PATCH_NAME} && sh apply-patch.sh"
