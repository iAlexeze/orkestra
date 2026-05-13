#!/usr/bin/env bash
# load.sh — generic CR loader for autoscale testing.
#
# Usage:
#   ./load.sh <crd> <count>
#   ./load.sh <crd> up <count>
#   ./load.sh <crd> down <to>
#
# Defaults:
#   CRD = ingestor
#
# Examples:
#   ./load.sh processor 200
#   ./load.sh loader up 150
#   ./load.sh auditor down 10
#   ./load.sh 100              # defaults to ingestor
#   ./load.sh down 0           # defaults to ingestor

set -euo pipefail

DEFAULT_CRD="ingestor"

CRD="${1:-$DEFAULT_CRD}"
CMD="${2:-up}"
ARG="${3:-}"

# Shorthand: ./load.sh 200
if [[ "$CRD" =~ ^[0-9]+$ ]]; then
  ARG="$CRD"
  CRD="$DEFAULT_CRD"
  CMD="up"
fi

# Shorthand: ./load.sh processor 200
if [[ "$CMD" =~ ^[0-9]+$ ]]; then
  ARG="$CMD"
  CMD="up"
fi

# Valid CRDs
VALID_CRDS=("ingestor" "loader" "processor" "auditor")

is_valid_crd() {
  local x
  for x in "${VALID_CRDS[@]}"; do
    [[ "$x" == "$1" ]] && return 0
  done
  return 1
}

print_help() {
  echo "Usage:"
  echo "  $0 <crd> <count>"
  echo "  $0 <crd> up <count>"
  echo "  $0 <crd> down <to>"
  echo
  echo "Valid CRDs: ${VALID_CRDS[*]}"
  echo "(crd defaults to '${DEFAULT_CRD}')"
}

# Validate CRD
if ! is_valid_crd "$CRD"; then
  echo "❌ Unknown CRD '$CRD'"
  print_help
  exit 1
fi

# Validate command
if [[ "$CMD" != "up" && "$CMD" != "down" ]]; then
  echo "❌ Unknown command '$CMD'"
  print_help
  exit 1
fi

# Emit correct spec for each CRD
emit_spec() {
  case "$1" in
    loader|processor|ingestor)
      cat <<EOF
spec:
  image: nginx
  replicas: 1
EOF
      ;;
    auditor)
      cat <<EOF
spec:
  auditMode: "standard"
EOF
      ;;
  esac
}

case "$CMD" in
up)
  COUNT="${ARG:-200}"
  echo "→ Creating ${CRD}-1 through ${CRD}-${COUNT}"
  for i in $(seq 1 "$COUNT"); do
    kubectl apply -f - <<EOF
apiVersion: autoscale.orkestra.io/v1alpha1
kind: ${CRD^}
metadata:
  name: ${CRD}-$i
  labels:
    orkestra.load: "true"
$(emit_spec "$CRD")
EOF
  done
  echo "✔ Created ${COUNT} ${CRD^} resources"
  ;;

down)
  TO="${ARG:-0}"
  MAX="${4:-500}"
  echo "→ Deleting ${CRD}-$((TO+1)) through ${CRD}-${MAX}"
  for i in $(seq 1 "$MAX"); do
    kubectl delete "$CRD" "${CRD}-$i" --ignore-not-found --wait=false || true
  done
  wait
  echo "✔ Scaled down to ${TO} ${CRD^} resources"
  ;;
esac
