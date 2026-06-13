#!/bin/sh
# LightAI Go - Apply Patch
# Must run from a LightAI installation root, or use --root.
# Usage: sh apply-patch.sh [--root /path/to/lightai] [--force]
set -e

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
  echo "Run from a LightAI installation root or use --root." >&2
  exit 1
fi

CURRENT_VERSION=$(grep '^version=' "$ROOT/VERSION" 2>/dev/null | cut -d= -f2 || echo "unknown")
PATCH_DIR="$(cd "$(dirname "$0")" && pwd)"

# Read patch manifest.
MANIFEST="$PATCH_DIR/PATCH-MANIFEST.txt"
if [ ! -f "$MANIFEST" ]; then
  echo "ERROR: PATCH-MANIFEST.txt not found in $PATCH_DIR" >&2
  exit 1
fi

FROM_VERSION=$(grep '^from_version=' "$MANIFEST" | cut -d= -f2)
TO_VERSION=$(grep '^to_version=' "$MANIFEST" | cut -d= -f2)
CHANGED=$(grep '^changed_files=' "$MANIFEST" | cut -d= -f2)
REMOVED=$(grep '^removed_files=' "$MANIFEST" | cut -d= -f2)
REQUIRES_STOP=$(grep '^requires_stop=' "$MANIFEST" | cut -d= -f2)

echo "=== LightAI Go Patch ==="
echo "Current: $CURRENT_VERSION"
echo "Patch:   $FROM_VERSION -> $TO_VERSION"
echo "Changed: $CHANGED files"
echo "Removed: $REMOVED files"
echo ""

# Version check.
if [ "$CURRENT_VERSION" != "$FROM_VERSION" ]; then
  if $FORCE; then
    echo "WARNING: Current version ($CURRENT_VERSION) does not match expected ($FROM_VERSION)."
    echo "Applying with --force. This may cause issues."
    echo ""
  else
    echo "ERROR: Current version ($CURRENT_VERSION) != patch from-version ($FROM_VERSION)." >&2
    echo "Use --force to override." >&2
    exit 1
  fi
fi

# Check running services.
RUNNING=false
for pidfile in run/server.pid run/agent.pid run/prometheus.pid run/grafana.pid; do
  if [ -f "$ROOT/$pidfile" ]; then
    PID=$(cat "$ROOT/$pidfile")
    if kill -0 "$PID" 2>/dev/null; then
      echo "WARNING: Service running: $pidfile (PID $PID)" >&2
      RUNNING=true
    fi
  fi
done

if $RUNNING; then
  echo ""
  echo "ERROR: Services are still running. Stop them first:" >&2
  echo "  cd $ROOT && ./scripts/stop-all.sh" >&2
  exit 1
fi

# Create backup.
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_DIR="$ROOT/backups/patch-$TIMESTAMP"
mkdir -p "$BACKUP_DIR"
echo "Backup: $BACKUP_DIR"

# Apply changes.
echo ""
echo "Applying patch..."
APPLIED=0
SKIPPED_CONFIGS=0

# Process changed/added files.
grep -E '^[AC] ' "$MANIFEST" | while read -r type file rest; do
  src="$PATCH_DIR/$file"
  dst="$ROOT/$file"

  if [ ! -f "$src" ]; then
    echo "  WARNING: file in manifest not found in patch: $file"
    continue
  fi

  # Config protection: if dst exists and is a config, write to .new instead.
  case "$file" in
    ./configs/*|./configs/observability/*)
      if [ -f "$dst" ]; then
        # Compare; if different, write .new.
        if ! cmp -s "$src" "$dst" 2>/dev/null; then
          cp "$src" "${dst}.new"
          echo "  CONFIG: ${file}.new (your existing config preserved)"
          SKIPPED_CONFIGS=$((SKIPPED_CONFIGS + 1))
        fi
        continue
      fi
      ;;
  esac

  # Backup original if it exists.
  if [ -f "$dst" ]; then
    mkdir -p "$(dirname "$BACKUP_DIR/$file")"
    cp "$dst" "$BACKUP_DIR/$file" 2>/dev/null || true
  fi

  # Install.
  mkdir -p "$(dirname "$dst")"
  cp "$src" "$dst"
  APPLIED=$((APPLIED + 1))
done

# Remove files marked for deletion.
grep '^R ' "$MANIFEST" | while read -r type file; do
  dst="$ROOT/$file"
  if [ -f "$dst" ]; then
    mkdir -p "$(dirname "$BACKUP_DIR/$file")"
    cp "$dst" "$BACKUP_DIR/$file" 2>/dev/null || true
    rm -f "$dst"
    echo "  REMOVED: $file"
  fi
done

# Update VERSION.
cp "$ROOT/VERSION" "$BACKUP_DIR/VERSION" 2>/dev/null || true
if [ -f "$PATCH_DIR/VERSION" ]; then
  cp "$PATCH_DIR/VERSION" "$ROOT/VERSION"
else
  # Generate minimal VERSION update.
  sed -i "s/^version=.*/version=$TO_VERSION/" "$ROOT/VERSION" 2>/dev/null || \
    echo "version=$TO_VERSION" > "$ROOT/VERSION"
fi

echo ""
echo "=== Patch Applied ==="
echo "From: $FROM_VERSION"
echo "To:   $TO_VERSION"
echo "Applied: $APPLIED files"
if [ "$SKIPPED_CONFIGS" != "0" ]; then
  echo "Config templates (.new): $SKIPPED_CONFIGS"
  echo "Review and merge .new config files into your existing configs."
fi
echo "Backup: $BACKUP_DIR"
echo ""
echo "Rollback: cp -r $BACKUP_DIR/* $ROOT/"
echo ""
echo "Restart services:"
echo "  ./scripts/start-server.sh"
echo "  ./scripts/start-agent.sh metax   # or nvidia"
echo "  ./scripts/start-observability.sh"
