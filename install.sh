#!/usr/bin/env bash
# install.sh — Orkestra CLI Installer (Runtime + Control Center)
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/orkspace/orkestra/refs/heads/main/install.sh | bash
#
# Options (via environment variables):
#   ORK_VERSION     — pin a specific version (default: latest release)
#   ORK_INSTALL_DIR — install directory (default: $HOME/.orkestra/bin)
#   ORK_SKIP_CC     — skip Control Center installation (default: false)
#
# Examples:
#   # Install latest (both runtime and control center)
#   curl -sSL https://raw.githubusercontent.com/orkspace/orkestra/refs/heads/main/install.sh | bash
#
#   # Install specific version
#   curl -sSL https://raw.githubusercontent.com/orkspace/orkestra/refs/heads/main/install.sh | ORK_VERSION=v1.0.0 bash
#
#   # Install to custom directory
#   curl -sSL https://raw.githubusercontent.com/orkspace/orkestra/refs/heads/main/install.sh | ORK_INSTALL_DIR=~/.local/bin bash
#
#   # Install runtime only (skip control center)
#   curl -sSL https://raw.githubusercontent.com/orkspace/orkestra/refs/heads/main/install.sh | ORK_SKIP_CC=true bash

set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────

REPO="orkspace/orkestra"
RUNTIME_BINARY="ork"
CONTROL_BINARY="orkcc"
INSTALL_DIR="${ORK_INSTALL_DIR:-/usr/local/bin}"
VERSION="${ORK_VERSION:-}"
ORK_SKIP_COMPLETION="${ORK_SKIP_COMPLETION:-false}"

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

    info "Fetching latest version..." >&2

    curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed -E 's/.*"tag_name": "([^"]+)".*/\1/'
}

# ── Check dependencies ────────────────────────────────────────────────────────

check_deps() {
    for cmd in curl tar; do
        command -v "${cmd}" >/dev/null 2>&1 \
            || fatal "Required command not found: ${cmd}"
    done
}

# ── Install command completion ───────────────────────────────────────────────────
install_completion() {
    if [[ "${ORK_SKIP_COMPLETION}" == "true" ]]; then
        info "Skipping shell completion installation (ORK_SKIP_COMPLETION=true)"
        return
    fi

    # Detect shell
    local shell_name
    shell_name=$(basename "${SHELL:-}")

    case "${shell_name}" in
        bash)
            local dir="${HOME}/.bash_completion.d"
            mkdir -p "${dir}"
            info "Installing bash completion to ${dir}/ork"
            ork completion bash > "${dir}/ork"
            ;;
        zsh)
            local dir="${HOME}/.oh-my-zsh/completions"
            mkdir -p "${dir}"
            info "Installing zsh completion to ${dir}/_ork"
            ork completion zsh > "${dir}/_ork"
            ;;
        fish)
            local dir="${HOME}/.config/fish/completions"
            mkdir -p "${dir}"
            info "Installing fish completion to ${dir}/ork.fish"
            ork completion fish > "${dir}/ork.fish"
            ;;
        *)
            warn "Unknown shell '${shell_name}'. Skipping completion installation."
            return
            ;;
    esac

    success "Shell completion installed for ${shell_name}"
    echo
    echo "Restart your shell or run 'source ~/.bashrc' / 'source ~/.zshrc' to activate."
}

# ── Generic installer (runtime + control center) ──────────────────────────────

install_component() {
    local binary="$1"
    local platform="$2"
    local version="$3"
    local archive="${binary}_${platform}.tar.gz"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${archive}"
    local checksum_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap '[[ -n "${tmp_dir:-}" ]] && rm -rf "${tmp_dir}"' RETURN

    info "Downloading ${archive}..."
    if ! curl -sSfL "${download_url}" -o "${tmp_dir}/${archive}"; then
        warn "${binary} not available for this version — skipping"
        return 1
    fi

    # Verify checksum if available
    if curl -sSfL "${checksum_url}" -o "${tmp_dir}/checksums.txt" 2>/dev/null; then
        info "Verifying checksum..."
        (
            cd "${tmp_dir}"
            if command -v sha256sum >/dev/null 2>&1; then
                grep "${archive}" checksums.txt | sha256sum --check --quiet \
                    || fatal "Checksum verification failed for ${archive}"
            else
                grep "${archive}" checksums.txt | shasum -a 256 --check --quiet \
                    || fatal "Checksum verification failed for ${archive}"
            fi
        )
    fi

    info "Extracting ${archive}..."
    tar -xzf "${tmp_dir}/${archive}" -C "${tmp_dir}"

    info "Installing ${binary} to ${INSTALL_DIR}..."
    if [[ ! -w "${INSTALL_DIR}" ]]; then
        sudo install -m 755 "${tmp_dir}/${binary}" "${INSTALL_DIR}/${binary}"
    else
        install -m 755 "${tmp_dir}/${binary}" "${INSTALL_DIR}/${binary}"
    fi

    success "${binary} ${version} installed"
}

# ── Verify installation ───────────────────────────────────────────────────────

verify_install() {
    echo

    if command -v ork >/dev/null 2>&1; then
        ork version
    else
        warn "ork is not in your PATH."
    fi

    if command -v orkcc >/dev/null 2>&1; then
        orkcc --version || true
    fi

    echo
    success "Installation complete."
    echo
    echo -e "  ${BOLD}Get started:${RESET}"
    echo -e "    ork init my-operator           Scaffold a new operator"
    echo -e "    ork validate --katalog <path>  Validate a Katalog"
    echo -e "    ork run --katalog <path>       Start the operator runtime"
    echo
    echo -e "  ${BOLD}Control Center:${RESET}"
    echo -e "    ork control start              Start the web UI (port 8090)"
    echo -e "    ork control start --port 9090 --urls \"http://...\""
    echo -e "    ork control version            Show Control Center version"
    echo
    echo -e "  ${BOLD}Documentation:${RESET}"
    echo -e "    https://github.com/${REPO}"
    echo
}

# ── Main ──────────────────────────────────────────────────────────────────────

main() {
    check_deps

    local platform version
    platform=$(detect_platform)
    version=$(resolve_version)

    # Install runtime and control center
    install_component "${RUNTIME_BINARY}" "${platform}" "${version}"
    install_component "${CONTROL_BINARY}" "${platform}" "${version}" || true

    # Install shell completion
    install_completion

    # Verify installation
    verify_install
}

main
