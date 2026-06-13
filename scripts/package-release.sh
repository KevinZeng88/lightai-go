#!/bin/sh
# LightAI Go - Release Package Builder
# Generates dist/lightai-go-<version>-linux-amd64.tar.gz
# Requires: third_party/observability/ (run prepare-observability-binaries.sh first)
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

VERSION="${LIGHTAI_VERSION:-0.1.0}"
COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -Iseconds)
GO_VERSION=$(go version 2>/dev/null | awk '{print $3}' || echo "unknown")
NODE_VERSION=$(node --version 2>/dev/null || echo "unknown")

# Observability versions.
PROMETHEUS_VERSION="${PROMETHEUS_VERSION:-3.12.0}"
GRAFANA_VERSION="${GRAFANA_VERSION:-13.0.2}"
WITH_OBSERVABILITY=true

case "${1:-}" in
  --without-observability) WITH_OBSERVABILITY=false ;;
esac

ARCH="linux-amd64"
RELEASE_NAME="lightai-go-${VERSION}-${ARCH}"
BUILD_DIR="dist/${RELEASE_NAME}"
TARBALL="dist/${RELEASE_NAME}.tar.gz"

echo "=== LightAI Go Release Builder ==="
echo "Version:     $VERSION"
echo "Commit:      $COMMIT"
echo "Prometheus:  $PROMETHEUS_VERSION"
echo "Grafana:     $GRAFANA_VERSION"
echo "Observability: $WITH_OBSERVABILITY"
echo ""

# Clean.
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/bin"
mkdir -p "$BUILD_DIR/configs"
mkdir -p "$BUILD_DIR/configs/observability"
mkdir -p "$BUILD_DIR/deploy"
mkdir -p "$BUILD_DIR/deploy/observability"
mkdir -p "$BUILD_DIR/deploy/collectors"
mkdir -p "$BUILD_DIR/scripts"
mkdir -p "$BUILD_DIR/logs"
mkdir -p "$BUILD_DIR/data/prometheus"
mkdir -p "$BUILD_DIR/data/grafana"
mkdir -p "$BUILD_DIR/run"

# 1. Check observability binaries FIRST (fail fast).
echo "[1/8] Checking observability binaries..."
THIRD="$PROJECT_DIR/third_party/observability"
if $WITH_OBSERVABILITY; then
  if [ ! -x "$THIRD/prometheus/prometheus" ]; then
    echo "  ERROR: Prometheus binary not found at $THIRD/prometheus/prometheus"
    echo "  Run: ./scripts/prepare-observability-binaries.sh --download"
    echo "  Or:  ./scripts/package-release.sh --without-observability"
    exit 1
  fi
  if [ ! -d "$THIRD/grafana/bin" ]; then
    echo "  ERROR: Grafana not found at $THIRD/grafana/bin"
    echo "  Run: ./scripts/prepare-observability-binaries.sh --download"
    exit 1
  fi
  echo "  Prometheus: $THIRD/prometheus/prometheus"
  echo "  Grafana:    $THIRD/grafana/bin/"
  echo "  OK"
else
  echo "  Skipped (--without-observability)"
fi

# 2. Build Go binaries.
echo "[2/8] Building Go binaries..."
go build -tags web -o "$BUILD_DIR/bin/lightai-server" ./cmd/server
go build -o "$BUILD_DIR/bin/lightai-agent" ./cmd/agent
echo "  OK"

# 3. Build Web.
echo "[3/8] Building Web assets..."
if [ -d web ]; then
  (cd web && npm ci --silent 2>/dev/null || npm install --silent 2>/dev/null)
  (cd web && npm run build 2>/dev/null)
fi
echo "  OK"

# 4. Bundled observability binaries.
PROMETHEUS_BIN="$THIRD/prometheus/prometheus"
GRAFANA_DIR="$THIRD/grafana"
if $WITH_OBSERVABILITY; then
  echo "[4/8] Bundling observability binaries..."
  echo "  Observability input: $THIRD"
  if [ ! -x "$PROMETHEUS_BIN" ]; then
    echo "  ERROR: Prometheus binary not found at $PROMETHEUS_BIN"
    exit 1
  fi
  if [ ! -d "$GRAFANA_DIR/bin" ]; then
    echo "  ERROR: Grafana not found at $GRAFANA_DIR/bin"
    exit 1
  fi
  install -m 0755 "$PROMETHEUS_BIN" "$BUILD_DIR/bin/prometheus"
  cp -a "$GRAFANA_DIR" "$BUILD_DIR/bin/grafana"
  echo "  Observability bundled: bin/prometheus bin/grafana/"
  echo "  OK"
else
  echo "[4/8] Skipping observability binaries (--without-observability)"
fi

# 5. Copy configs, collectors, scripts.
echo "[5/8] Copying configs and scripts..."
# Ensure target directories exist.
for d in "$BUILD_DIR/configs" "$BUILD_DIR/configs/observability" "$BUILD_DIR/deploy" "$BUILD_DIR/scripts"; do
  if [ ! -d "$d" ]; then
    echo "  ERROR: directory missing: $d" >&2
    exit 1
  fi
done
cp configs/server.release.yaml "$BUILD_DIR/configs/"
cp configs/agent.metax.yaml "$BUILD_DIR/configs/"
cp configs/agent.nvidia.yaml "$BUILD_DIR/configs/"
cp configs/observability/prometheus.yml "$BUILD_DIR/configs/observability/" 2>/dev/null || true
cp configs/observability/grafana.ini "$BUILD_DIR/configs/observability/" 2>/dev/null || true
cp -r deploy/collectors "$BUILD_DIR/deploy/" 2>/dev/null || true
cp -r deploy/observability "$BUILD_DIR/deploy/" 2>/dev/null || true
cp scripts/start-server.sh scripts/start-agent.sh "$BUILD_DIR/scripts/"
cp scripts/stop-server.sh scripts/stop-agent.sh "$BUILD_DIR/scripts/"
cp scripts/start-observability.sh scripts/stop-observability.sh scripts/stop-all.sh "$BUILD_DIR/scripts/"
cp scripts/status.sh scripts/verify-local.sh "$BUILD_DIR/scripts/"
cp scripts/collect-logs.sh "$BUILD_DIR/scripts/"
chmod +x "$BUILD_DIR"/scripts/*.sh
echo "  OK"

# 6. Copy README and licenses.
echo "[6/8] Copying docs..."
cp README-RELEASE.md "$BUILD_DIR/"
# Prometheus LICENSE.
if [ -f "$THIRD/prometheus/LICENSE" ]; then
  mkdir -p "$BUILD_DIR/LICENSES/prometheus"
  cp "$THIRD/prometheus/LICENSE" "$BUILD_DIR/LICENSES/prometheus/"
fi
# Grafana LICENSE.
if [ -f "$THIRD/grafana/LICENSE" ]; then
  mkdir -p "$BUILD_DIR/LICENSES/grafana"
  cp "$THIRD/grafana/LICENSE" "$BUILD_DIR/LICENSES/grafana/"
fi
echo "  OK"

# 7. Write VERSION.
echo "[7/8] Writing VERSION..."
cat > "$BUILD_DIR/VERSION" << EOF
version=$VERSION
git_commit=$COMMIT
build_time=$BUILD_TIME
build_type=full
go_version=$GO_VERSION
node_version=$NODE_VERSION
prometheus_version=$PROMETHEUS_VERSION
grafana_version=$GRAFANA_VERSION
EOF

# Generate MANIFEST.sha256 (exclude runtime dirs).
# Format: file <sha256> <path> or symlink <target> <path>
MANIFEST="$BUILD_DIR/MANIFEST.sha256"
: > "$MANIFEST"
(cd "$BUILD_DIR" && find . -type f \
  ! -path './data/*' ! -path './logs/*' ! -path './run/*' \
  ! -path './data/prometheus/*' ! -path './data/grafana/*' \
  ! -name MANIFEST.sha256 \
  | sort | while IFS= read -r f; do
    [ -f "$f" ] && sha256sum "$f" 2>/dev/null | awk '{print "file " $1 " " $2}'
  done) >> "$MANIFEST"
# Record symlinks too.
(cd "$BUILD_DIR" && find . -type l \
  ! -path './data/*' ! -path './logs/*' ! -path './run/*' \
  | sort | while IFS= read -r f; do
    target=$(readlink "$f" 2>/dev/null || echo "")
    echo "symlink $target $f"
  done) >> "$MANIFEST"
echo "  OK"

# 8. Build tarball and verify.
echo "[8/8] Creating tarball..."
mkdir -p dist
rm -f "$TARBALL"
tar -czf "$TARBALL" -C dist "$RELEASE_NAME"
echo "  OK"

# Quick verification.
echo ""
echo "--- Package Contents ---"
echo "  bin/lightai-server: $(tar -tzf "$TARBALL" | grep -c 'bin/lightai-server$')"
echo "  bin/lightai-agent:  $(tar -tzf "$TARBALL" | grep -c 'bin/lightai-agent$')"
if $WITH_OBSERVABILITY; then
  echo "  bin/prometheus:     $(tar -tzf "$TARBALL" | grep -c 'bin/prometheus$')"
  echo "  bin/grafana/:       $(tar -tzf "$TARBALL" | grep -c 'bin/grafana/')"
fi
echo "  configs:            $(tar -tzf "$TARBALL" | grep -c 'configs/')"
echo "  scripts:            $(tar -tzf "$TARBALL" | grep -c 'scripts/')"
echo "  VERSION:            $(tar -tzf "$TARBALL" | grep -c 'VERSION$')"

echo ""
echo "Release: $TARBALL"
echo "Size:    $(du -h "$TARBALL" | cut -f1)"
echo ""
echo "Deploy:"
echo "  tar -xzf ${RELEASE_NAME}.tar.gz && cd ${RELEASE_NAME}"
echo "  export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='...'"
echo "  export LIGHTAI_GRAFANA_ADMIN_PASSWORD='...'"
echo "  ./scripts/start-server.sh"
echo "  ./scripts/start-agent.sh metax   # or nvidia"
echo "  ./scripts/start-observability.sh"
