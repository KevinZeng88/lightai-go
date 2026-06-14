#!/bin/sh
# LightAI Go - Apply Cumulative Patch
# No git dependency. Reads VERSION and patch-manifest.json.
# Usage: sh apply-patch.sh [--root <path>] [--force]
# No set -e: use explicit exit for error handling.

semver_compare() {
  # Returns: 0 if $1 == $2, 1 if $1 > $2, 2 if $1 < $2
  a_major="${1%%.*}"; a_rest="${1#*.}"
  a_minor="${a_rest%%.*}"; a_patch="${a_rest#*.}"
  b_major="${2%%.*}"; b_rest="${2#*.}"
  b_minor="${b_rest%%.*}"; b_patch="${b_rest#*.}"
  for part in "$a_major" "$a_minor" "$a_patch" "$b_major" "$b_minor" "$b_patch"; do
    case "$part" in ''|*[!0-9]*) echo "ERROR: invalid semver: $1 or $2" >&2; exit 1 ;; esac
  done
  if [ "$a_major" -gt "$b_major" ]; then return 1; fi
  if [ "$a_major" -lt "$b_major" ]; then return 2; fi
  if [ "$a_minor" -gt "$b_minor" ]; then return 1; fi
  if [ "$a_minor" -lt "$b_minor" ]; then return 2; fi
  if [ "$a_patch" -gt "$b_patch" ]; then return 1; fi
  if [ "$a_patch" -lt "$b_patch" ]; then return 2; fi
  return 0
}

ROOT="$(pwd)"
FORCE=false

while [ $# -gt 0 ]; do
  case "$1" in
    --root) ROOT="$2"; shift 2 ;;
    --force) FORCE=true; shift ;;
    *) echo "Usage: $0 [--root <path>] [--force]" >&2; exit 1 ;;
  esac
done

if [ ! -f "$ROOT/VERSION" ]; then
  echo "ERROR: VERSION file not found in $ROOT" >&2
  exit 1
fi
CURRENT_VERSION=$(head -1 "$ROOT/VERSION" | tr -d '[:space:]')

PATCH_DIR="$(cd "$(dirname "$0")" && pwd)"
MANIFEST="$PATCH_DIR/patch-manifest.json"
if [ ! -f "$MANIFEST" ]; then
  echo "ERROR: patch-manifest.json not found in $PATCH_DIR" >&2
  exit 1
fi

FROM_VERSION=$(python3 -c "import json;print(json.load(open('$MANIFEST')).get('from_version',''))" 2>/dev/null || echo "")
TO_VERSION=$(python3 -c "import json;print(json.load(open('$MANIFEST')).get('to_version',''))" 2>/dev/null || echo "")
PATCH_MODE=$(python3 -c "import json;print(json.load(open('$MANIFEST')).get('patch_mode',''))" 2>/dev/null || echo "cumulative")

echo "=== LightAI Go Patch ==="
echo "Current: $CURRENT_VERSION"
echo "Patch:   $FROM_VERSION -> $TO_VERSION (mode: $PATCH_MODE)"
echo ""

# Version checks (cumulative mode).
if [ "$PATCH_MODE" = "cumulative" ]; then
  semver_compare "$CURRENT_VERSION" "$FROM_VERSION"
  cmp_from=$?
  if [ $cmp_from -eq 2 ]; then
    echo "ERROR: Current version ($CURRENT_VERSION) is BELOW from_version ($FROM_VERSION)." >&2
    echo "Upgrade to at least $FROM_VERSION first." >&2
    exit 1
  fi

  semver_compare "$CURRENT_VERSION" "$TO_VERSION"
  cmp_to=$?
  if [ $cmp_to -eq 1 ] || [ $cmp_to -eq 0 ]; then
    echo "ERROR: Current version ($CURRENT_VERSION) >= to_version ($TO_VERSION)." >&2
    echo "Already up to date or newer. Patch not needed." >&2
    exit 1
  fi
fi

echo "Applying patch..."

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_DIR="$ROOT/backups/patch-$TIMESTAMP"
mkdir -p "$BACKUP_DIR"
echo "Backup: $BACKUP_DIR"

APPLIED=0
EXCLUDES="data/ logs/ run/ runtime/ data/prometheus/ data/grafana/ backups/ .git/"

for f in "$PATCH_DIR"/*; do
  fname=$(basename "$f")
  case "$fname" in
    apply-patch.sh|patch-manifest.json|PATCH-MANIFEST.txt) continue ;;
  esac

  dst="$ROOT/$fname"
  # Check excludes.
  skip=false
  for ex in $EXCLUDES; do
    case "$fname" in $ex*) skip=true; break ;; esac
  done
  $skip && continue

  if [ -f "$dst" ]; then
    mkdir -p "$(dirname "$BACKUP_DIR/$fname")" 2>/dev/null || true
    cp "$dst" "$BACKUP_DIR/$fname" 2>/dev/null || true
  fi

  if [ -d "$f" ]; then
    mkdir -p "$dst"
    cp -r "$f"/* "$dst"/ 2>/dev/null || true
  else
    dir=$(dirname "$dst")
    [ -d "$dir" ] || mkdir -p "$dir"
    cp "$f" "$dst"
  fi
  APPLIED=$((APPLIED + 1))
done

# Update VERSION.
cp "$ROOT/VERSION" "$BACKUP_DIR/VERSION" 2>/dev/null || true
echo "$TO_VERSION" > "$ROOT/VERSION"

echo ""
echo "=== Patch Applied ==="
echo "Version: $CURRENT_VERSION -> $TO_VERSION"
echo "Applied: $APPLIED items"
echo "Backup:  $BACKUP_DIR"
echo ""
echo "Rollback: cp -r $BACKUP_DIR/* $ROOT/"
echo "Restart:  cd $ROOT && sh scripts/start-server.sh && sh scripts/start-agent.sh && sh scripts/start-observability.sh"
