#!/usr/bin/env bash
# ccswresp Universal Install Script
# Supports: npm, direct download, and package manager detection
# Usage: curl -fsSL https://raw.githubusercontent.com/hoganyu/ccswresp/main/scripts/install.sh | bash

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
NPM_REGISTRY="https://registry.npmjs.org"

check_node() {
  if command -v node &>/dev/null; then
    NODE_VERSION=$(node -v | sed 's/v//' | cut -d. -f1)
    if [ "$NODE_VERSION" -ge 18 ]; then
      return 0
    fi
  fi
  return 1
}

install_node() {
  info "Node.js >= 18 is required. Installing..."

  OS=$(uname -s)
  if [ "$OS" = "Darwin" ]; then
    if command -v brew &>/dev/null; then
      brew install node
    else
      err "Please install Homebrew first: https://brew.sh"
      err "Or install Node.js manually: https://nodejs.org"
      exit 1
    fi
  elif [ "$OS" = "Linux" ]; then
    if command -v apt-get &>/dev/null; then
      curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
      sudo apt-get install -y nodejs
    elif command -v yum &>/dev/null; then
      curl -fsSL https://rpm.nodesource.com/setup_20.x | sudo -E bash -
      sudo yum install -y nodejs
    elif command -v dnf &>/dev/null; then
      curl -fsSL https://rpm.nodesource.com/setup_20.x | sudo -E bash -
      sudo dnf install -y nodejs
    else
      err "Cannot auto-install Node.js. Please install manually: https://nodejs.org"
      exit 1
    fi
  else
    err "Unsupported OS. Please install Node.js manually: https://nodejs.org"
    exit 1
  fi
}

install_via_npm() {
  info "Installing ${PKG_NAME} via npm..."
  npm install -g "${PKG_NAME}"
  ok "${PKG_NAME} installed successfully!"
}

init_config() {
  if [ ! -f "${HOME}/.ccswresp/.env" ]; then
    info "Initializing config..."
    ccswresp --init
    ok "Config created at ~/.ccswresp/.env"
    echo ""
    echo -e "  ${BOLD}Next step:${NC} Edit ${CYAN}~/.ccswresp/.env${NC} and set your API key."
    echo "  Run ${CYAN}ccswresp --help${NC} for all options."
  fi
}

main() {
  echo ""
  echo -e "${BOLD}${CYAN}ccswresp Installer${NC}"
  echo ""

  # Check Node.js
  if ! check_node; then
    install_node
  fi
  ok "Node.js $(node -v) detected"

  # Check npm
  if ! command -v npm &>/dev/null; then
    err "npm not found. Please install Node.js properly."
    exit 1
  fi

  # Install
  install_via_npm

  # Verify
  if command -v ccswresp &>/dev/null; then
    ok "ccswresp v$(ccswresp --version 2>/dev/null | head -1 | awk '{print $2}') ready"
  fi

  # Init config
  init_config

  echo ""
  echo -e "  ${BOLD}Quick start:${NC}"
  echo "  1. Edit ~/.ccswresp/.env and set your API key"
  echo "  2. Run: ${CYAN}ccswresp${NC}"
  echo "  3. Point Codex CLI to http://127.0.0.1:11435/v1/responses"
  echo ""
}

main
