#!/usr/bin/env bash
# install.sh — Orkestra CLI installer
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/refs/heads/main/install.sh | bash
#
# Options (via environment variables):
#   ORK_VERSION     — pin a specific version (default: latest)
#   ORK_INSTALL_DIR — install directory (default: /usr/local/bin)
#
# Examples:
#   # Install latest
#   curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/refs/heads/main/install.sh | bash
#
#   # Install specific version
#   curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/refs/heads/main/install.sh | ORK_VERSION=v1.0.0 bash
#
#   # Install to custom directory
#   curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/refs/heads/main/install.sh | ORK_INSTALL_DIR=~/.local/bin bash

set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────

REPO="iAlexeze/orkestra"
BINARY="ork"
INSTALL_DIR="${ORK_INSTALL_DIR:-/usr/local/bin}"
VERSION="${ORK_VERSION:-}"

# ── Colours ───────────────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { echo -e "${BLUE}[orkestra]${RESET} $*"; }
success() { echo -e "${GREEN}[orkestra]${RESET} $*"; }
warn()    { echo -e "${YELLOW}[orkestra]${RESET} $*"; }
error()   { echo -e "${RED}[orkestra]${RESET} $*" >&2; }
fatal()   { error "$*"; exit 1; }

# ── Banner ────────────────────────────────────────────────────────────────────

echo -e "${BOLD}"
cat <<'EOF'
   ___       _              _
  / _ \ _  _| |___ _ _  ___| |_ _ _ __ _
 | (_) | || | / -_) ' \/ -_)  _| '_/ _` |
  \___/ \_,_|_\___|_||_\___|\__|_| \__,_|
          O R K E S T R A
EOF
echo -e "${RESET}"

# ── Detect OS and architecture ────────────────────────────────────────────────

detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux)   os="linux" ;;
        Darwin)  os="darwin" ;;
        *)       fatal "Unsupported OS: $(uname -s). Use WSL on Windows." ;;
    esac

    case "$(uname -m)" in
        x86_64)         arch="amd64" ;;
        aarch64|arm64)  arch="arm64" ;;
        *)              fatal "Unsupported architecture: $(uname -m)" ;;
    esac

    echo "${os}_${arch}"
}

# ── Resolve latest version ────────────────────────────────────────────────────

resolve_version() {
    if [[ -n "${VERSION}" ]]; then
        echo "${VERSION}"
        return
    fi

    info "Fetching latest version..."

    local latest
    latest=$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')

    if [[ -z "${latest}" ]]; then
        fatal "Could not resolve latest version. Set ORK_VERSION manually."
    fi

    echo "${latest}"
}

# ── Check dependencies ────────────────────────────────────────────────────────

check_deps() {
    for cmd in curl tar; do
        if ! command -v "${cmd}" &>/dev/null; then
            fatal "Required command not found: ${cmd}"
        fi
    done
}

# ── Download and install ──────────────────────────────────────────────────────

install_ork() {
    local platform version download_url tmp_dir archive checksum_url

    platform=$(detect_platform)
    version=$(resolve_version)

    info "Installing ${BOLD}ork ${version}${RESET} for ${platform}..."

    # Expected release asset format: ork_linux_amd64.tar.gz
    archive="ork_${platform}.tar.gz"
    download_url="https://github.com/${REPO}/releases/download/${version}/${archive}"
    checksum_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

    # Create temp directory — cleaned up automatically on exit
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "${tmp_dir}"' EXIT

    info "Downloading ${archive}..."
    if ! curl -sSfL "${download_url}" -o "${tmp_dir}/${archive}"; then
        fatal "Download failed. Check that version ${version} exists at:
  https://github.com/${REPO}/releases"
    fi

    if [[ ! -f "${tmp_dir}/${archive}" ]]; then
        fatal "Downloaded file not found — something went wrong"
    fi

    # Verify checksum if available
    if curl -sSfL "${checksum_url}" -o "${tmp_dir}/checksums.txt" 2>/dev/null; then
        info "Verifying checksum..."
        cd "${tmp_dir}"
        if command -v sha256sum &>/dev/null; then
            grep "${archive}" checksums.txt | sha256sum --check --quiet \
                || fatal "Checksum verification failed"
        elif command -v shasum &>/dev/null; then
            grep "${archive}" checksums.txt | shasum -a 256 --check --quiet \
                || fatal "Checksum verification failed"
        else
            warn "sha256sum not available — skipping checksum verification"
        fi
        cd - >/dev/null
    fi

    # Extract
    info "Extracting..."
    tar -xzf "${tmp_dir}/${archive}" -C "${tmp_dir}"

    # Install
    info "Installing to ${INSTALL_DIR}..."
    if [[ ! -w "${INSTALL_DIR}" ]]; then
        warn "${INSTALL_DIR} requires elevated permissions"
        sudo install -m 755 "${tmp_dir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    else
        install -m 755 "${tmp_dir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    fi

    success "ork ${version} installed to ${INSTALL_DIR}/${BINARY}"
}

# ── Verify installation ───────────────────────────────────────────────────────

verify_install() {
    if ! command -v ork &>/dev/null; then
        warn "ork is not in your PATH."
        warn "Add ${INSTALL_DIR} to your PATH:"
        warn "  export PATH=\"\$PATH:${INSTALL_DIR}\""
        return
    fi

    echo
    ork version
    echo
    success "Installation complete."
    echo
    echo -e "  ${BOLD}Get started:${RESET}"
    echo -e "    ork init my-operator           Scaffold a new operator"
    echo -e "    ork validate --katalog <path>  Validate a Katalog"
    echo -e "    ork run --katalog <path>        Start the operator runtime"
    echo
    echo -e "  ${BOLD}Documentation:${RESET}"
    echo -e "    https://github.com/${REPO}"
    echo
}

# ── Main ──────────────────────────────────────────────────────────────────────

main() {
    check_deps
    install_ork
    verify_install
}

main