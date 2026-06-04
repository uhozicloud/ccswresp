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
  local BINARY_NAME="${PKG_NAME}_${OS}_${ARCH}"
  if [ "$OS" = "windows" ]; then
    BINARY_NAME="${BINARY_NAME}.exe"
  fi

  local DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${BINARY_NAME}"

  info "Downloading ${DOWNLOAD_URL}..."
  local TMP_DIR=$(mktemp -d)
  local TMP_FILE="${TMP_DIR}/${BINARY_NAME}"

  if command -v curl &>/dev/null; then
    curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE" || {
      # Try .tar.gz if raw binary not found
      DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${BINARY_NAME}.tar.gz"
      info "Trying archive: ${DOWNLOAD_URL}..."
      curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/${BINARY_NAME}.tar.gz"
      tar -xzf "${TMP_DIR}/${BINARY_NAME}.tar.gz" -C "$TMP_DIR"
    }
  elif command -v wget &>/dev/null; then
    wget -q "$DOWNLOAD_URL" -O "$TMP_FILE" || {
      DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${BINARY_NAME}.tar.gz"
      wget -q "$DOWNLOAD_URL" -O "${TMP_DIR}/${BINARY_NAME}.tar.gz"
      tar -xzf "${TMP_DIR}/${BINARY_NAME}.tar.gz" -C "$TMP_DIR"
    }
  else
    err "Neither curl nor wget found. Please install one of them."
    exit 1
  fi

  # Install
  if [ "$OS" = "windows" ]; then
    info "On Windows, move ${BINARY_NAME}.exe to a directory in your PATH"
    cp "${TMP_DIR}/${BINARY_NAME}"* "${HOME}/"
    ok "Binary saved to ${HOME}/${BINARY_NAME}"
  else
    if [ -w "$INSTALL_DIR" ]; then
      cp "${TMP_DIR}/${BINARY_NAME}" "$INSTALL_DIR/ccswresp"
      chmod +x "$INSTALL_DIR/ccswresp"
    else
      sudo cp "${TMP_DIR}/${BINARY_NAME}" "$INSTALL_DIR/ccswresp"
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
  echo "  3. Point Codex CLI to http://127.0.0.1:11435/responses"
  echo ""
}

main
