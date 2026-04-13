#!/usr/bin/env bash
# Install custos binary from GitHub releases.
#
# Usage:
#   curl -sSfL https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh | bash
#   curl -sSfL https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh | bash -s -- -b /usr/local/bin
#   curl -sSfL https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh | bash -s -- -v v0.1.0

set -euo pipefail

REPO="timkrebs/custos"
BINARY="custos"
INSTALL_DIR="${HOME}/.local/bin"
VERSION="latest"

usage() {
  cat <<EOF
Install custos — the missing terraform plan for Vault policies.

Usage:
  install.sh [flags]

Flags:
  -b DIR      Install directory (default: \$HOME/.local/bin)
  -v VERSION  Version to install (default: latest)
  -h          Show this help

Examples:
  # Install latest to ~/.local/bin
  curl -sSfL https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh | bash

  # Install specific version to /usr/local/bin
  curl -sSfL https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh | bash -s -- -b /usr/local/bin -v v0.1.0
EOF
}

parse_args() {
  while getopts "b:v:h" opt; do
    case "$opt" in
      b) INSTALL_DIR="$OPTARG" ;;
      v) VERSION="$OPTARG" ;;
      h) usage; exit 0 ;;
      *) usage; exit 1 ;;
    esac
  done
}

detect_platform() {
  local os arch

  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    linux*)  os="linux" ;;
    darwin*) os="darwin" ;;
    mingw*|msys*|cygwin*) os="windows" ;;
    *) echo "Error: unsupported OS: $os" >&2; exit 1 ;;
  esac

  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) echo "Error: unsupported architecture: $arch" >&2; exit 1 ;;
  esac

  echo "${os}_${arch}"
}

resolve_version() {
  if [ "$VERSION" = "latest" ]; then
    VERSION="$(curl -sSfL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')"
    if [ -z "$VERSION" ]; then
      echo "Error: could not determine latest version" >&2
      exit 1
    fi
  fi
  # Strip leading 'v' for archive name matching
  VERSION_NUM="${VERSION#v}"
}

download_and_install() {
  local platform="$1"
  local tmpdir
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  local ext="tar.gz"
  if [[ "$platform" == windows_* ]]; then
    ext="zip"
  fi

  local archive_name="${BINARY}_${VERSION_NUM}_${platform}.${ext}"
  local checksums_name="${BINARY}_${VERSION_NUM}_SHA256SUMS"
  local download_url="https://github.com/${REPO}/releases/download/${VERSION}/${archive_name}"
  local checksums_url="https://github.com/${REPO}/releases/download/${VERSION}/${checksums_name}"

  echo "Downloading custos ${VERSION} for ${platform}..."
  curl -sSfL -o "${tmpdir}/${archive_name}" "$download_url"
  curl -sSfL -o "${tmpdir}/${checksums_name}" "$checksums_url"

  # Verify checksum
  echo "Verifying checksum..."
  (cd "$tmpdir" && grep "${archive_name}" "${checksums_name}" | sha256sum -c --quiet 2>/dev/null) || \
  (cd "$tmpdir" && grep "${archive_name}" "${checksums_name}" | shasum -a 256 -c --quiet 2>/dev/null) || {
    echo "Error: checksum verification failed" >&2
    exit 1
  }

  # Extract
  echo "Extracting..."
  if [ "$ext" = "tar.gz" ]; then
    tar -xzf "${tmpdir}/${archive_name}" -C "$tmpdir"
  else
    unzip -q "${tmpdir}/${archive_name}" -d "$tmpdir"
  fi

  # Install
  mkdir -p "$INSTALL_DIR"
  local bin_name="$BINARY"
  if [[ "$platform" == windows_* ]]; then
    bin_name="${BINARY}.exe"
  fi

  cp "${tmpdir}/${bin_name}" "${INSTALL_DIR}/${bin_name}"
  chmod +x "${INSTALL_DIR}/${bin_name}"

  echo ""
  echo "custos ${VERSION} installed to ${INSTALL_DIR}/${bin_name}"
  echo ""

  # Check if install dir is in PATH
  if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
    echo "Add ${INSTALL_DIR} to your PATH:"
    echo ""
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo ""
    echo "Add this line to your ~/.bashrc, ~/.zshrc, or ~/.profile to make it permanent."
  fi
}

main() {
  parse_args "$@"

  local platform
  platform="$(detect_platform)"

  resolve_version
  download_and_install "$platform"
}

main "$@"
