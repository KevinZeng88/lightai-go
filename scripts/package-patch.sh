#!/bin/sh
# LightAI Go - Incremental Patch Builder
# Usage: ./scripts/package-patch.sh --from 0.1.0 --to 0.1.1
set -e

FROM_VERSION=""
TO_VERSION=""
ARCH="linux-amd64"

while [ $# -gt 0 ]; do
  case "$1" in
    --from) FROM_VERSION="$2"; shift 2 ;;
    --to)   TO_VERSION="$2";   shift 2 ;;
    *) echo "Usage: $0 --from <ver> --to <ver>" >&2; exit 1 ;;
  esac
done
[ -z "$FROM_VERSION" ] || [ -z "$TO_VERSION" ] && { echo "Usage: $0 --from <ver> --to <ver>" >&2; exit 1; }

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
  # Compare grafana binary SHA.
  GRAF_BIN_FROM=$(file_sha "$FROM_DIR/bin/grafana/bin/grafana")
  GRAF_BIN_TO=$(file_sha "$TO_DIR/bin/grafana/bin/grafana")
  [ "$GRAF_BIN_FROM" != "$GRAF_BIN_TO" ] && GRAF_CHANGED=true
fi

echo "Comparing files..."
CHANGED=0
REMOVED=0

> "$PATCH_DIR/PATCH-MANIFEST.txt"

# Function: add a file to the patch.
add_patch_file() {
  local file="$1" action="$2" sha="$3"
  dir=$(dirname "$file")
  mkdir -p "$PATCH_DIR/$dir"
  cp "$TO_DIR/$file" "$PATCH_DIR/$file"
  echo "$action $file $sha" >> "$PATCH_DIR/PATCH-MANIFEST.txt"
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
    # For symlinks, compare target.
    from_target=$(readlink "$from_file" 2>/dev/null || echo "")
    if [ "$from_target" != "$target" ]; then
      add_patch_file "$file" "S" "$target"
      CHANGED=$((CHANGED + 1))
    fi
  elif [ -f "$from_file" ]; then
    from_sha=$(sha256sum "$from_file" | awk '{print $1}')
    if [ "$from_sha" != "$sha" ]; then
      add_patch_file "$file" "C" "$sha"
      CHANGED=$((CHANGED + 1))
    fi
  else
    # New entry in TO.
    add_patch_file "$file" "$type" "${sha:-$target}"
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
    echo "R $file" >> "$PATCH_DIR/PATCH-MANIFEST.txt"
    REMOVED=$((REMOVED + 1))
  fi
done < "$FROM_DIR/MANIFEST.sha256"

# Write PATCH-MANIFEST header.
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
TIMESTAMP=$(date -Iseconds)
{
  echo "from_version=$FROM_VERSION"
  echo "to_version=$TO_VERSION"
  echo "created_at=$TIMESTAMP"
  echo "git_commit=$GIT_COMMIT"
  echo "changed_files=$CHANGED"
  echo "removed_files=$REMOVED"
  echo "requires_stop=true"
  echo "prometheus_version_from=$FROM_PROM_VER"
  echo "prometheus_version_to=$TO_PROM_VER"
  echo "prometheus_included=$PROM_CHANGED"
  echo "grafana_version_from=$FROM_GRAF_VER"
  echo "grafana_version_to=$TO_GRAF_VER"
  echo "grafana_included=$GRAF_CHANGED"
  echo "---"
  cat "$PATCH_DIR/PATCH-MANIFEST.txt"
} > "$PATCH_DIR/PATCH-MANIFEST.txt.tmp"
mv "$PATCH_DIR/PATCH-MANIFEST.txt.tmp" "$PATCH_DIR/PATCH-MANIFEST.txt"

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
