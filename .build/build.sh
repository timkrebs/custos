#!/usr/bin/env bash
# Build custos binaries for all supported platforms.
# Usage:
#   .build/build.sh              # build for current platform
#   .build/build.sh all          # build for all platforms
#   .build/build.sh linux/amd64  # build for specific platform

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

BINARY_NAME="custos"
OUTPUT_DIR="dist"

# Version metadata
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
GIT_COMMIT="${GIT_COMMIT:-$(git rev-parse HEAD 2>/dev/null || echo "unknown")}"
GIT_TREE_STATE="${GIT_TREE_STATE:-$(test -z "$(git status --porcelain 2>/dev/null)" && echo "clean" || echo "dirty")}"
BUILD_DATE="${BUILD_DATE:-$(date -u '+%Y-%m-%dT%H:%M:%SZ')}"

LDFLAGS="-s -w \
  -X github.com/timkrebs/custos/version.Version=${VERSION} \
  -X github.com/timkrebs/custos/version.GitCommit=${GIT_COMMIT} \
  -X github.com/timkrebs/custos/version.GitTreeState=${GIT_TREE_STATE} \
  -X github.com/timkrebs/custos/version.BuildDate=${BUILD_DATE}"

# All supported platforms
PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
  "windows/arm64"
)

build_platform() {
  local platform="$1"
  local os="${platform%/*}"
  local arch="${platform#*/}"
  local output="${OUTPUT_DIR}/${BINARY_NAME}_${VERSION}_${os}_${arch}"

  if [ "$os" = "windows" ]; then
    output="${output}.exe"
  fi

  echo "Building ${os}/${arch}..."
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -ldflags="${LDFLAGS}" -o "$output" .

  echo "  → $output"
}

main() {
  local target="${1:-current}"

  mkdir -p "$OUTPUT_DIR"

  if [ "$target" = "all" ]; then
    echo "Building custos ${VERSION} for all platforms..."
    echo ""
    for platform in "${PLATFORMS[@]}"; do
      build_platform "$platform"
    done
    echo ""
    echo "Done. Binaries in ${OUTPUT_DIR}/"
  elif [ "$target" = "current" ]; then
    local os
    local arch
    os="$(go env GOOS)"
    arch="$(go env GOARCH)"
    echo "Building custos ${VERSION} for ${os}/${arch}..."
    build_platform "${os}/${arch}"
  else
    echo "Building custos ${VERSION} for ${target}..."
    build_platform "$target"
  fi
}

main "$@"
