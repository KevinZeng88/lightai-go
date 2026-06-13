#!/bin/sh
# LightAI Go - Release Package Builder
# Generates dist/lightai-go-<version>-linux-amd64.tar.gz
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

# Determine version.
VERSION="${LIGHTAI_VERSION:-0.1.0}"
COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -Iseconds)
GO_VERSION=$(go version 2>/dev/null | awk '{print $3}' || echo "unknown")
NODE_VERSION=$(node --version 2>/dev/null || echo "unknown")

ARCH="linux-amd64"
RELEASE_NAME="lightai-go-${VERSION}-${ARCH}"
BUILD_DIR="dist/${RELEASE_NAME}"
TARBALL="dist/${RELEASE_NAME}.tar.gz"

echo "=== LightAI Go Release Builder ==="
echo "Version: $VERSION"
echo "Commit:  $COMMIT"
echo "Arch:    $ARCH"
echo ""

# Clean.
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/bin" "$BUILD_DIR/logs" "$BUILD_DIR/data" "$BUILD_DIR/run"
mkdir -p "$BUILD_DIR/configs" "$BUILD_DIR/scripts" "$BUILD_DIR/deploy/collectors/gpu" "$BUILD_DIR/deploy/observability"

# 1. Build Go binaries.
echo "[1/6] Building Go binaries..."
go build -tags web -o "$BUILD_DIR/bin/lightai-server" ./cmd/server
go build -o "$BUILD_DIR/bin/lightai-agent" ./cmd/agent
echo "  OK"

# 2. Build Web.
echo "[2/6] Building Web assets..."
if [ -d web ]; then
  (cd web && npm ci --silent 2>/dev/null || npm install --silent 2>/dev/null)
  (cd web && npm run build --silent 2>/dev/null)
fi
echo "  OK"

# 3. Copy configs, collectors, scripts.
echo "[3/6] Copying configs and scripts..."
cp configs/server.release.yaml "$BUILD_DIR/configs/" 2>/dev/null || true
cp configs/agent.metax.yaml "$BUILD_DIR/configs/" 2>/dev/null || true
cp configs/agent.nvidia.yaml "$BUILD_DIR/configs/" 2>/dev/null || true
cp -r deploy/collectors "$BUILD_DIR/deploy/" 2>/dev/null || true
cp scripts/start-server.sh "$BUILD_DIR/scripts/"
cp scripts/start-agent.sh "$BUILD_DIR/scripts/"
cp scripts/stop-server.sh "$BUILD_DIR/scripts/"
cp scripts/stop-agent.sh "$BUILD_DIR/scripts/"
cp scripts/status.sh "$BUILD_DIR/scripts/"
cp scripts/verify-local.sh "$BUILD_DIR/scripts/"
cp scripts/collect-logs.sh "$BUILD_DIR/scripts/"
cp scripts/observability-up.sh "$BUILD_DIR/scripts/" 2>/dev/null || true
cp scripts/observability-down.sh "$BUILD_DIR/scripts/" 2>/dev/null || true
cp scripts/observability-status.sh "$BUILD_DIR/scripts/" 2>/dev/null || true
cp -r deploy/observability "$BUILD_DIR/deploy/" 2>/dev/null || true
chmod +x "$BUILD_DIR"/scripts/*.sh
echo "  OK"

# 4. Copy README.
echo "[4/6] Copying README..."
cp README-RELEASE.md "$BUILD_DIR/" 2>/dev/null || true
echo "  OK"

# 5. Write VERSION.
echo "[5/6] Writing VERSION..."
cat > "$BUILD_DIR/VERSION" << EOF
version=$VERSION
commit=$COMMIT
build_time=$BUILD_TIME
go_version=$GO_VERSION
node_version=$NODE_VERSION
EOF
echo "  OK"

# 6. Build tarball.
echo "[6/6] Creating tarball..."
mkdir -p dist
rm -f "$TARBALL"
tar -czf "$TARBALL" -C dist "$RELEASE_NAME"
echo "  OK"

echo ""
echo "Release package: $TARBALL"
echo "Size: $(du -h "$TARBALL" | cut -f1)"
echo ""
echo "Deploy:"
echo "  scp $TARBALL user@metax-server:/tmp/"
echo "  ssh user@metax-server"
echo "  tar -xzf /tmp/${RELEASE_NAME}.tar.gz"
echo "  cd ${RELEASE_NAME}"
echo "  export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='...'"
echo "  ./scripts/start-server.sh"
echo "  ./scripts/start-agent.sh metax"
