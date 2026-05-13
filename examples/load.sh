#!/usr/bin/env bash
# load.sh — flood the reconcile queue with synthetic CRs to trigger autoscaling.
#
# Usage:
#   ./load.sh create <plural-crd> <count> [namespace]
#   ./load.sh delete <plural-crd> [namespace]
#
# Examples:
#   ./load.sh create ingestors 50 default
#   ./load.sh delete ingestors default
#
#   make load LOAD_CRD=loaders LOAD_COUNT=100
#
# The CRs are minimal — they have generated names (load-<n>) and carry only
# the fields required by the CRD's spec.  Edit the TEMPLATE below to match
# the specific CRD spec shape in your example.

set -euo pipefail

COMMAND="${1:-create}"
CRD="${2:-ingestors}"       # plural resource name
COUNT="${3:-50}"
NAMESPACE="${4:-default}"

# ── Derive singular Kind from plural ─────────────────────────────────────────
# Strips a trailing 's' — sufficient for the generated example CRDs.
# Override KIND= if your CRD uses an irregular plural (e.g. "statuses").
KIND="${KIND:-}"
if [[ -z "$KIND" ]]; then
  # kubectl api-resources can resolve this accurately when a cluster is running
  KIND=$(kubectl api-resources --no-headers 2>/dev/null \
    | awk -v p="$CRD" '$NF==p || $1==p {print $NF; exit}' \
    || true)
  if [[ -z "$KIND" ]]; then
    # Fallback: capitalise and drop trailing 's'
    base="${CRD%s}"
    KIND="${base^}"
  fi
fi

# ── Detect group/version from cluster ────────────────────────────────────────
GV=$(kubectl api-resources --no-headers 2>/dev/null \
  | awk -v p="$CRD" '$1==p {print $(NF-2); exit}' || echo "example.orkestra.io/v1alpha1")
GROUP="${GV%/*}"
VERSION="${GV#*/}"
[[ "$GV" == */* ]] || { GROUP="$GV"; VERSION="v1alpha1"; }

# ── Helpers ───────────────────────────────────────────────────────────────────

cr_name() { echo "load-$1"; }

generate_cr() {
  local n="$1"
  cat <<EOF
apiVersion: ${GROUP}/${VERSION}
kind: ${KIND}
metadata:
  name: $(cr_name "$n")
  namespace: ${NAMESPACE}
  labels:
    orkestra.load: "true"
spec:
  image: "nginx:stable-alpine"
  replicas: "1"
EOF
}

# ── Commands ──────────────────────────────────────────────────────────────────

do_create() {
  echo "Creating $COUNT ${CRD} CRs in namespace ${NAMESPACE}..."
  local created=0
  for i in $(seq 1 "$COUNT"); do
    generate_cr "$i" | kubectl apply -f - -n "$NAMESPACE" --quiet 2>/dev/null && ((created++)) || true
  done
  echo "Done — $created CRs created. Watch autoscaling:"
  echo "  kubectl get ${CRD} -n ${NAMESPACE}"
  echo "  kubectl describe crd ${CRD} | grep -A5 autoscale"
}

do_delete() {
  echo "Deleting load CRs (label: orkestra.load=true) from ${NAMESPACE}..."
  kubectl delete "${CRD}" -n "${NAMESPACE}" -l "orkestra.load=true" --ignore-not-found
  echo "Done."
}

case "$COMMAND" in
  create) do_create ;;
  delete) do_delete ;;
  *)
    echo "Usage: $0 {create|delete} <plural-crd> [count] [namespace]"
    exit 1
    ;;
esac
