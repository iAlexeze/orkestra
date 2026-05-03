#!/usr/bin/env bash
# setup-kind.sh — Kind cluster lifecycle for Orkestra examples.
# Usage: ./setup-kind.sh create|delete|list [cluster-name]
set -euo pipefail

ACTION=${1:-create}
CLUSTER=${2:-orkestra-playground}

check_kind() {
    if ! command -v kind &>/dev/null; then
        echo "Error: 'kind' is not installed."
        echo "Install: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
        exit 1
    fi
}

case "$ACTION" in
  create)
    check_kind
    if kind get clusters 2>/dev/null | grep -q "^${CLUSTER}$"; then
        echo "Cluster '${CLUSTER}' already exists — skipping creation."
    else
        echo "Creating Kind cluster: ${CLUSTER}"
        kind create cluster --name "${CLUSTER}" --wait 60s
        echo "Cluster '${CLUSTER}' is ready."
    fi
    ;;
  delete)
    check_kind
    if kind get clusters 2>/dev/null | grep -q "^${CLUSTER}$"; then
        echo "Deleting Kind cluster: ${CLUSTER}"
        kind delete cluster --name "${CLUSTER}"
    else
        echo "Cluster '${CLUSTER}' not found — nothing to delete."
    fi
    ;;
  list)
    check_kind
    kind get clusters
    ;;
  *)
    echo "Usage: $0 create|delete|list [cluster-name]"
    exit 1
    ;;
esac
