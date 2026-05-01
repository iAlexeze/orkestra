#!/usr/bin/env bash
# install.sh — Orkestra CLI installer
#
# Usage:
#   curl -sSL https://get.orkestra.sh | bash
#
# Environment variables:
#   ORK_VERSION         — pin a specific version tag (default: latest release)
#   ORK_INSTALL_DIR     — install directory (default: /usr/local/bin)
#   ORK_SKIP_CC         — skip Control Center binary (default: false)
#   ORK_SKIP_COMPLETION — skip shell completion setup (default: false)
#
# ── Adding a new Orkestra binary ──────────────────────────────────────────────
# 1. Add the binary name to BINARIES.
# 2. If it should be skippable, add a case entry in component_skipped().
# That is the only change needed — download URL, checksum, and extraction
# all follow the same convention automatically.
# ─────────────────────────────────────────────────────────────────────────────

set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────

REPO="orkspace/orkestra"
INSTALL_DIR="${ORK_INSTALL_DIR:-/usr/local/bin}"
VERSION="${ORK_VERSION:-}"
SKIP_CC="${ORK_SKIP_CC:-false}"
SKIP_COMPLETION="${ORK_SKIP_COMPLETION:-false}"

# All Orkestra CLI binaries, in install order.
# To add a new binary, append it here.
BINARIES=("ork" "orkcc")

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

# ── Dependencies ──────────────────────────────────────────────────────────────

check_deps() {
    for cmd in curl tar grep sed; do
        command -v "${cmd}" >/dev/null 2>&1 \
            || fatal "Required command not found: ${cmd}"
    done
}

# ── Platform detection ────────────────────────────────────────────────────────

detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux)   os="linux"  ;;
        Darwin)  os="darwin" ;;
        *)       fatal "Unsupported OS: $(uname -s). Use WSL on Windows." ;;
    esac

    case "$(uname -m)" in
        x86_64)        arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)             fatal "Unsupported architecture: $(uname -m)" ;;
    esac

    echo "${os}_${arch}"
}

# ── Version resolution ────────────────────────────────────────────────────────

resolve_version() {
    if [[ -n "${VERSION}" ]]; then
        echo "${VERSION}"
        return
    fi

    info "Fetching latest version..." >&2

    local tag
    tag=$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')

    if [[ -z "${tag}" ]]; then
        fatal "Could not resolve latest version. Check your network or set ORK_VERSION explicitly."
    fi

    echo "${tag}"
}

# ── Component skip rules ──────────────────────────────────────────────────────
# Returns 0 (true) when the binary should be skipped.
# Add a case entry here when a new binary needs a skip flag.

component_skipped() {
    local binary="$1"
    case "${binary}" in
        orkcc) [[ "${SKIP_CC}" == "true" ]] ;;
        *)     return 1 ;;
    esac
}

# ── Checksum verification ─────────────────────────────────────────────────────
# Uses -c (POSIX short form) which works on GNU coreutils, BusyBox, and macOS.
# --check is GNU-only and fails on BusyBox (Alpine/CI containers).

verify_checksum() {
    local archive="$1"
    local checksums_file="$2"

    if command -v sha256sum >/dev/null 2>&1; then
        grep "${archive}" "${checksums_file}" | sha256sum -c \
            || fatal "Checksum verification failed for ${archive}"
    elif command -v shasum >/dev/null 2>&1; then
        grep "${archive}" "${checksums_file}" | shasum -a 256 -c \
            || fatal "Checksum verification failed for ${archive}"
    else
        warn "No checksum tool found (sha256sum / shasum) — skipping verification"
    fi
}

# ── Single component installer ────────────────────────────────────────────────

install_component() {
    local binary="$1"
    local platform="$2"
    local version="$3"

    if component_skipped "${binary}"; then
        info "Skipping ${binary} (disabled by flag)"
        return 0
    fi

    local archive="${binary}_${platform}.tar.gz"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${archive}"
    local checksum_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

    local tmp_dir
    tmp_dir=$(mktemp -d)
    # Always clean up temp dir, even on error.
    # Double-quoted so ${tmp_dir} is captured at trap-set time, not at RETURN time.
    # Single quotes would cause "unbound variable" under set -u when the trap fires
    # in the calling scope after the local variable goes out of scope.
    # shellcheck disable=SC2064
    trap "rm -rf '${tmp_dir}'" RETURN

    info "Downloading ${archive}..."
    if ! curl -sSfL "${download_url}" -o "${tmp_dir}/${archive}"; then
        warn "${binary} not available for ${version} — skipping"
        return 0
    fi

    if curl -sSfL "${checksum_url}" -o "${tmp_dir}/checksums.txt" 2>/dev/null; then
        info "Verifying checksum..."
        (cd "${tmp_dir}" && verify_checksum "${archive}" "checksums.txt")
    else
        warn "No checksums.txt found for ${version} — skipping verification"
    fi

    info "Extracting ${archive}..."
    tar -xzf "${tmp_dir}/${archive}" -C "${tmp_dir}"

    local src="${tmp_dir}/${binary}"
    if [[ ! -f "${src}" ]]; then
        fatal "Expected binary '${binary}' not found in archive"
    fi

    info "Installing ${binary} to ${INSTALL_DIR}..."
    if [[ ! -w "${INSTALL_DIR}" ]]; then
        sudo install -m 755 "${src}" "${INSTALL_DIR}/${binary}"
    else
        install -m 755 "${src}" "${INSTALL_DIR}/${binary}"
    fi

    success "${binary} ${version} installed to ${INSTALL_DIR}/${binary}"
}

# ── Shell completion ──────────────────────────────────────────────────────────

install_completion() {
    if [[ "${SKIP_COMPLETION}" == "true" ]]; then
        info "Skipping shell completion (ORK_SKIP_COMPLETION=true)"
        return
    fi

    local shell_name
    shell_name=$(basename "${SHELL:-bash}")

    case "${shell_name}" in
        bash)
            # Standard bash-completion drop-in directory (works with and without oh-my-bash)
            local dir="${HOME}/.bash_completion.d"
            mkdir -p "${dir}"
            info "Installing bash completion → ${dir}/ork"
            ork completion bash > "${dir}/ork"
            echo "  Source with: source ${dir}/ork  (or restart your shell)"
            ;;
        zsh)
            # Prefer XDG / user fpath over hard-coding oh-my-zsh paths.
            local dir="${HOME}/.zsh/completions"
            mkdir -p "${dir}"
            info "Installing zsh completion → ${dir}/_ork"
            ork completion zsh > "${dir}/_ork"
            echo "  Add to ~/.zshrc if not present: fpath=(${dir} \$fpath)"
            ;;
        fish)
            local dir="${HOME}/.config/fish/completions"
            mkdir -p "${dir}"
            info "Installing fish completion → ${dir}/ork.fish"
            ork completion fish > "${dir}/ork.fish"
            ;;
        *)
            warn "Shell '${shell_name}' not recognised — skipping completion."
            warn "Run 'ork completion <shell>' to generate it manually."
            return
            ;;
    esac

    success "Shell completion installed for ${shell_name}"
}

# ── Post-install summary ──────────────────────────────────────────────────────

print_summary() {
    local version="$1"
    echo
    success "Installation complete  (${version})"
    echo

    for binary in "${BINARIES[@]}"; do
        if component_skipped "${binary}"; then
            continue
        fi
        if command -v "${binary}" >/dev/null 2>&1; then
            echo -e "  ${GREEN}✓${RESET}  ${binary}  $(${binary} version 2>/dev/null | head -1 || true)"
        else
            warn "${binary} not found in PATH — add ${INSTALL_DIR} to your PATH"
        fi
    done

    echo
    echo -e "  ${BOLD}Get started:${RESET}"
    echo -e "    ork init my-operator             Scaffold a new operator"
    echo -e "    ork validate -k katalog.yaml     Validate a Katalog"
    echo -e "    ork run -k katalog.yaml           Start the operator runtime"
    echo -e "    ork registry push name:v1 ./dir  Push a pattern to the registry"
    echo
    echo -e "  ${BOLD}Control Center:${RESET}"
    echo -e "    ork control start                Start the web UI (port 8090)"
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

    info "Platform: ${platform}  Version: ${version}"
    echo

    for binary in "${BINARIES[@]}"; do
        install_component "${binary}" "${platform}" "${version}"
    done

    install_completion

    print_summary "${version}"
}

main
