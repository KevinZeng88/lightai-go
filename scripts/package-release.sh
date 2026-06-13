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
mkdir -p "$BUILD_DIR"/{bin,configs/observability,deploy/observability,deploy/collectors,scripts,logs,data/prometheus,data/grafana,run}

# 1. Build Go binaries.
echo "[1/7] Building Go binaries..."
go build -tags web -o "$BUILD_DIR/bin/lightai-server" ./cmd/server
go build -o "$BUILD_DIR/bin/lightai-agent" ./cmd/agent
echo "  OK"

# 2. Build Web.
echo "[2/7] Building Web assets..."
if [ -d web ]; then
  (cd web && npm ci --silent 2>/dev/null || npm install --silent 2>/dev/null)
  (cd web && npm run build 2>/dev/null)
fi
echo "  OK"

# 3. Observability binaries.
if $WITH_OBSERVABILITY; then
  echo "[3/7] Bundling observability binaries..."
  THIRD="$PROJECT_DIR/third_party/observability"
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
  cp "$THIRD/prometheus/prometheus" "$BUILD_DIR/bin/prometheus"
  cp -r "$THIRD/grafana" "$BUILD_DIR/bin/grafana"
  echo "  OK"
else
  echo "[3/7] Skipping observability binaries (--without-observability)"
fi

# 4. Copy configs, collectors, scripts.
echo "[4/7] Copying configs and scripts..."
cp configs/server.release.yaml "$BUILD_DIR/configs/"
cp configs/agent.metax.yaml "$BUILD_DIR/configs/"
cp configs/agent.nvidia.yaml "$BUILD_DIR/configs/"
cp configs/observability/prometheus.yml "$BUILD_DIR/configs/observability/" 2>/dev/null || true
cp configs/observability/grafana.ini "$BUILD_DIR/configs/observability/" 2>/dev/null || true
cp -r deploy/collectors "$BUILD_DIR/deploy/" 2>/dev/null || true
cp -r deploy/observability "$BUILD_DIR/deploy/" 2>/dev/null || true
cp scripts/start-server.sh scripts/start-agent.sh "$BUILD_DIR/scripts/"
cp scripts/stop-server.sh scripts/stop-agent.sh "$BUILD_DIR/scripts/"
cp scripts/start-observability.sh scripts/stop-observability.sh "$BUILD_DIR/scripts/"
cp scripts/status.sh scripts/verify-local.sh "$BUILD_DIR/scripts/"
cp scripts/collect-logs.sh "$BUILD_DIR/scripts/"
chmod +x "$BUILD_DIR"/scripts/*.sh
echo "  OK"

# 5. Copy README and licenses.
echo "[5/7] Copying docs..."
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

# 6. Write VERSION.
echo "[6/7] Writing VERSION..."
cat > "$BUILD_DIR/VERSION" << EOF
lightai_version=$VERSION
git_commit=$COMMIT
build_time=$BUILD_TIME
go_version=$GO_VERSION
node_version=$NODE_VERSION
prometheus_version=$PROMETHEUS_VERSION
grafana_version=$GRAFANA_VERSION
EOF
echo "  OK"

# 7. Build tarball.
echo "[7/7] Creating tarball..."
mkdir -p dist
rm -f "$TARBALL"
tar -czf "$TARBALL" -C dist "$RELEASE_NAME"
echo "  OK"

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
