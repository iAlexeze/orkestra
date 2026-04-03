#!/usr/bin/env bash
# install.sh — Orkestra CLI Installer (Runtime + Control Center)
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/refs/heads/main/install.sh | bash
#
# Options (via environment variables):
#   ORK_VERSION     — pin a specific version (default: latest release)
#   ORK_INSTALL_DIR — install directory (default: /usr/local/bin)
#   ORK_SKIP_CC     — skip Control Center installation (default: false)
#
# Examples:
#   # Install latest (both runtime and control center)
#   curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/refs/heads/main/install.sh | bash
#
#   # Install specific version
#   curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/refs/heads/main/install.sh | ORK_VERSION=v1.0.0 bash
#
#   # Install to custom directory
#   curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/refs/heads/main/install.sh | ORK_INSTALL_DIR=~/.local/bin bash
#
#   # Install runtime only (skip control center)
#   curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/refs/heads/main/install.sh | ORK_SKIP_CC=true bash

set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────

REPO="iAlexeze/orkestra"
RUNTIME_BINARY="ork"
CONTROL_CENTER_BINARY="orkcc"
INSTALL_DIR="${ORK_INSTALL_DIR:-/usr/local/bin}"
VERSION="${ORK_VERSION:-}"
SKIP_CC="${ORK_SKIP_CC:-false}"

# ── Colours ───────────────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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
echo -e "${CYAN}  The Kubernetes operator runtime that needs no programming language${RESET}"
echo

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

    info "Fetching latest version from GitHub..."

    local latest
    latest=$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed -E 's/.*"tag_name": "([^"]+)".*/\1/' \
        | head -1)

    if [[ -z "${latest}" ]]; then
        fatal "Could not resolve latest version. Set ORK_VERSION manually."
    fi

    echo "${latest}"
}

# ── Check dependencies ────────────────────────────────────────────────────────

check_deps() {
    local missing=()
    for cmd in curl tar; do
        if ! command -v "${cmd}" &>/dev/null; then
            missing+=("${cmd}")
        fi
    done

    if [[ ${#missing[@]} -gt 0 ]]; then
        fatal "Required commands not found: ${missing[*]}"
    fi
}

# ── Download and install runtime ──────────────────────────────────────────────

install_runtime() {
    local platform version download_url tmp_dir archive checksum_url

    platform=$(detect_platform)
    version=$(resolve_version)

    info "Installing ${BOLD}ork${RESET} (runtime) version ${version} for ${platform}..."

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
                || fatal "Checksum verification failed for ${archive}"
        elif command -v shasum &>/dev/null; then
            grep "${archive}" checksums.txt | shasum -a 256 --check --quiet \
                || fatal "Checksum verification failed for ${archive}"
        else
            warn "sha256sum not available — skipping checksum verification"
        fi
        cd - >/dev/null
    else
        warn "No checksums.txt found — skipping verification"
    fi

    # Extract
    info "Extracting ${archive}..."
    tar -xzf "${tmp_dir}/${archive}" -C "${tmp_dir}"

    # Install
    info "Installing to ${INSTALL_DIR}..."
    if [[ ! -w "${INSTALL_DIR}" ]]; then
        warn "${INSTALL_DIR} requires elevated permissions"
        sudo install -m 755 "${tmp_dir}/${RUNTIME_BINARY}" "${INSTALL_DIR}/${RUNTIME_BINARY}"
    else
        install -m 755 "${tmp_dir}/${RUNTIME_BINARY}" "${INSTALL_DIR}/${RUNTIME_BINARY}"
    fi

    success "ork ${version} installed to ${INSTALL_DIR}/${RUNTIME_BINARY}"
}

# ── Download and install control center ───────────────────────────────────────

install_control_center() {
    if [[ "${SKIP_CC}" == "true" ]]; then
        warn "Skipping Control Center installation (ORK_SKIP_CC=true)"
        return
    fi

    local platform version download_url tmp_dir archive

    platform=$(detect_platform)
    version=$(resolve_version)

    info "Installing ${BOLD}orkcc${RESET} (Control Center) version ${version} for ${platform}..."

    archive="orkcc_${platform}.tar.gz"
    download_url="https://github.com/${REPO}/releases/download/${version}/${archive}"

    tmp_dir=$(mktemp -d)
    # Don't set trap here — it's already set from runtime installation
    # But we need to ensure cleanup still happens

    info "Downloading ${archive}..."
    if ! curl -sSfL "${download_url}" -o "${tmp_dir}/${archive}"; then
        warn "Control Center download failed — continuing with runtime only"
        rm -rf "${tmp_dir}"
        return
    fi

    # Extract
    info "Extracting ${archive}..."
    tar -xzf "${tmp_dir}/${archive}" -C "${tmp_dir}"

    # Install
    if [[ ! -f "${tmp_dir}/${CONTROL_CENTER_BINARY}" ]]; then
        warn "Control Center binary not found in archive — skipping"
        rm -rf "${tmp_dir}"
        return
    fi

    if [[ ! -w "${INSTALL_DIR}" ]]; then
        sudo install -m 755 "${tmp_dir}/${CONTROL_CENTER_BINARY}" "${INSTALL_DIR}/${CONTROL_CENTER_BINARY}"
    else
        install -m 755 "${tmp_dir}/${CONTROL_CENTER_BINARY}" "${INSTALL_DIR}/${CONTROL_CENTER_BINARY}"
    fi

    success "orkcc ${version} installed to ${INSTALL_DIR}/${CONTROL_CENTER_BINARY}"

    # Clean up temp dir (the trap will handle it, but we can remove early)
    rm -rf "${tmp_dir}"
}

# ── Verify installation ───────────────────────────────────────────────────────

verify_install() {
    local has_ork=false
    local has_orkcc=false

    if command -v ork &>/dev/null; then
        has_ork=true
    fi

    if command -v orkcc &>/dev/null; then
        has_orkcc=true
    fi

    if [[ "${has_ork}" == "false" ]]; then
        warn "ork is not in your PATH."
        warn "Add ${INSTALL_DIR} to your PATH:"
        warn "  export PATH=\"\$PATH:${INSTALL_DIR}\""
        return
    fi

    echo
    echo -e "${BOLD}Installed versions:${RESET}"
    ork version

    if [[ "${has_orkcc}" == "true" ]]; then
        orkcc --version 2>/dev/null || echo "orkcc version ${VERSION:-latest}"
    fi

    echo
    success "Installation complete!"
    echo
    echo -e "${BOLD}🚀 Get started:${RESET}"
    echo -e "  ${CYAN}ork run --katalog my-katalog.yaml${RESET}    Start the operator runtime"
    echo -e "  ${CYAN}ork validate --katalog my-katalog.yaml${RESET} Validate a Katalog"
    echo -e "  ${CYAN}ork kompose --katalog komposer.yaml${RESET}   Compose multiple Katalogs"
    echo
    echo -e "${BOLD}📊 Control Center:${RESET}"
    echo -e "  ${CYAN}orkcc -u http://localhost:8080 -p 8090${RESET}  Start the web UI"
    echo -e "  ${CYAN}ork control start${RESET}                        Launch from ork CLI (coming soon)"
    echo
    echo -e "${BOLD}📚 Documentation:${RESET}"
    echo -e "  https://github.com/${REPO}"
    echo
}

# ── Main ──────────────────────────────────────────────────────────────────────

main() {
    echo
    check_deps
    install_runtime
    install_control_center
    verify_install
}

main "$@"