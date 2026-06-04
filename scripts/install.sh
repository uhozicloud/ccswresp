#!/usr/bin/env bash
# ccswresp Universal Install Script (Go binary)
# Usage: curl -fsSL https://raw.githubusercontent.com/uhozicloud/ccswresp/main/scripts/install.sh | bash

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info()  { echo -e "${CYAN}[INFO]${NC} $*"; }
ok()    { echo -e "${GREEN}[ OK ]${NC} $*"; }
err()   { echo -e "${RED}[ERR ]${NC} $*"; }

PKG_NAME="ccswresp"
VERSION="${CCSWRESP_VERSION:-1.0.0}"
GITHUB_REPO="uhozicloud/ccswresp"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
detect_platform() {
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)

  case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) err "Unsupported architecture: $ARCH"; exit 1 ;;
  esac

  case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *) err "Unsupported OS: $OS"; exit 1 ;;
  esac
}

# Check for existing install
check_existing() {
  if command -v ccswresp &>/dev/null; then
    EXISTING=$(ccswresp --version 2>/dev/null | awk '{print $2}' || echo "unknown")
    info "ccswresp v${EXISTING} already installed"
    if [ "$EXISTING" = "$VERSION" ]; then
      info "Already at latest version. Use CCSWRESP_VERSION=x.y.z to override."
      exit 0
    fi
    info "Upgrading from v${EXISTING} to v${VERSION}..."
  fi
}

# Download and install
install_binary() {
  local ARCHIVE_NAME="${PKG_NAME}_${OS}-${ARCH}"
  if [ "$OS" = "windows" ]; then
    ARCHIVE_NAME="${ARCHIVE_NAME}.zip"
  else
    ARCHIVE_NAME="${ARCHIVE_NAME}.tar.gz"
  fi

  local DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${ARCHIVE_NAME}"

  info "Downloading ${DOWNLOAD_URL}..."
  local TMP_DIR=$(mktemp -d)

  if command -v curl &>/dev/null; then
    curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/${ARCHIVE_NAME}"
  elif command -v wget &>/dev/null; then
    wget -q "$DOWNLOAD_URL" -O "${TMP_DIR}/${ARCHIVE_NAME}"
  else
    err "Neither curl nor wget found. Please install one of them."
    exit 1
  fi

  # Extract
  if [ "$OS" = "windows" ]; then
    unzip -q "${TMP_DIR}/${ARCHIVE_NAME}" -d "$TMP_DIR"
  else
    tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "$TMP_DIR"
  fi

  # Install
  if [ "$OS" = "windows" ]; then
    info "On Windows, move ccswresp.exe to a directory in your PATH"
    cp "${TMP_DIR}/ccswresp.exe" "${HOME}/"
    ok "Binary saved to ${HOME}/ccswresp.exe"
  else
    if [ -w "$INSTALL_DIR" ]; then
      cp "${TMP_DIR}/ccswresp" "$INSTALL_DIR/ccswresp"
      chmod +x "$INSTALL_DIR/ccswresp"
    else
      sudo cp "${TMP_DIR}/ccswresp" "$INSTALL_DIR/ccswresp"
      sudo chmod +x "$INSTALL_DIR/ccswresp"
    fi
    ok "Installed to ${INSTALL_DIR}/ccswresp"
  fi

  rm -rf "$TMP_DIR"
}

# Initialize config
init_config() {
  if [ ! -f "${HOME}/.ccswresp/.env" ]; then
    info "Initializing config..."
    ccswresp --init
    ok "Config created at ~/.ccswresp/.env"
    echo ""
    echo -e "  ${BOLD}Next step:${NC} Edit ${CYAN}~/.ccswresp/.env${NC} and set your API key."
  fi
}

main() {
  echo ""
  echo -e "${BOLD}${CYAN}ccswresp Installer v${VERSION}${NC}"
  echo ""

  detect_platform
  check_existing
  install_binary

  # Verify
  echo ""
  if command -v ccswresp &>/dev/null; then
    ok "ccswresp $(ccswresp --version 2>/dev/null) ready"
    init_config
  fi

  echo ""
  echo -e "  ${BOLD}Quick start:${NC}"
  echo "  1. Edit ~/.ccswresp/.env and set your API key"
  echo "  2. Run: ${CYAN}ccswresp${NC}"
  echo "  3. Point Codex CLI to http://127.0.0.1:11435/v1/responses"
  echo ""
}

main
