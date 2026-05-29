#!/usr/bin/env bash
set -euo pipefail

# ────────────────────────────────────────────────────────────────
# Colors
RED="\033[31m"; GREEN="\033[32m"; YELLOW="\033[33m"
BLUE="\033[34m"; MAGENTA="\033[35m"; CYAN="\033[36m"
RESET="\033[0m"

# ────────────────────────────────────────────────────────────────
# Config
CLUSTERS=(01 02 03 04 05 06)
NS_PREFIX="orkestra-system-"
DEMO_NS_PREFIX="demo-"

# Ports: 8080, skip 8081, then 8082–8086
BASE_PORT=8080

# ────────────────────────────────────────────────────────────────
log() {
    local color="$1"; shift
    echo -e "${color}$*${RESET}"
}

cluster_log() {
    local cluster="$1"; shift
    echo -e "${MAGENTA}[cluster-${cluster}]${RESET} $*"
}

# ────────────────────────────────────────────────────────────────
log "${CYAN}" "═══ Setting up namespaces and resources ═══"

for c in "${CLUSTERS[@]}"; do
    NS="${NS_PREFIX}${c}"
    DEMO_NS="${DEMO_NS_PREFIX}${c}"
    DIR="cluster-${c}"

    cluster_log "$c" "Creating namespaces: ${NS}, ${DEMO_NS}"
    kubectl create ns "${NS}" --dry-run=client -o yaml | kubectl apply -f -
    kubectl create ns "${DEMO_NS}" --dry-run=client -o yaml | kubectl apply -f -

    cluster_log "$c" "Applying CRDs and CRs"
    kubectl apply -f "${DIR}/crd.yaml"
    [[ -f "${DIR}/setup.yaml" ]] && kubectl apply -f "${DIR}/setup.yaml"
    kubectl apply -f "${DIR}/cr.yaml"
done

# ────────────────────────────────────────────────────────────────
log "${CYAN}" "\n═══ Launching ork run (background) ═══"

PIDS=()
PORT=$BASE_PORT

for c in "${CLUSTERS[@]}"; do
    NS="${NS_PREFIX}${c}"
    DIR="cluster-${c}"

    # Skip 8081 (reserved for control center)
    if [[ "$PORT" -eq 8081 ]]; then
        PORT=$((PORT+1))
    fi

    cluster_log "$c" "Starting ork run on port ${PORT} (namespace ${NS})"

    (
        export ORK_NAMESPACE="${NS}"
        export ORK_PORT="${PORT}"
        cd "${DIR}" && ork run
    ) &

    PIDS+=($!)
    PORT=$((PORT+1))
done

sleep 3

# ────────────────────────────────────────────────────────────────
# Build runtime URLs for control center
URLS=()
PORT=$BASE_PORT
for c in "${CLUSTERS[@]}"; do
    [[ "$PORT" -eq 8081 ]] && PORT=$((PORT+1))
    URLS+=("http://localhost:${PORT}")
    PORT=$((PORT+1))
done
URL_LIST=$(IFS=,; echo "${URLS[*]}")

log "${CYAN}" "\n═══ Starting control center (cluster-01) ═══"
cluster_log "01" "Control center URLs: ${URL_LIST}"

(
    export ORK_NAMESPACE="${NS_PREFIX}01"
    export ORK_PORT=8081
    cd cluster-01 && ork control start -u "${URL_LIST}"
) &
PIDS+=($!)

# ────────────────────────────────────────────────────────────────
log "${GREEN}" "\n═══ Dev environment running ═══"
echo -e "${YELLOW}PIDs: ${PIDS[*]}${RESET}"
echo -e "${BLUE}Use Ctrl+C to stop everything.${RESET}"

trap 'echo -e "\n${RED}Stopping all clusters...${RESET}"; kill ${PIDS[*]} 2>/dev/null' INT TERM
wait
