#!/usr/bin/env bash
# cleanup.sh — remove developer pack resources
# Usage: ./cleanup.sh [example-number]
#   ./cleanup.sh      — clean up everything
#   ./cleanup.sh 01   — clean up only example 01 (my-api)
#   ./cleanup.sh 02   — clean up example 02 (+ my-frontend)
set -euo pipefail

EXAMPLE="${1:-all}"

remove_app() {
  local name="$1"
  local ns="${name}-orkestra-ns"
  local cm="${name}-orkestra"

  echo "  Removing ${name}..."

  # Disable deletion protection first (ignore errors — may not be enabled)
  kubectl patch configmap "${cm}" -n "${ns}" \
    --patch '{}' 2>/dev/null || true

  kubectl delete configmap "${cm}" -n "${ns}" --ignore-not-found
  kubectl delete namespace "${ns}" --ignore-not-found
  rm -rf "./${name}/.orkestra"
  echo "  ✓ ${name} removed"
}

remove_state() {
  local name="$1"
  local state_file="${HOME}/.orkestra/deploy/state.json"
  if [[ -f "${state_file}" ]]; then
    # Remove the project entry from state.json using Python (available on most systems)
    python3 -c "
import json, sys
with open('${state_file}') as f:
    s = json.load(f)
s.get('projects', {}).pop('${name}', None)
with open('${state_file}', 'w') as f:
    json.dump(s, f, indent=2)
" 2>/dev/null || true
  fi
}

echo ""
echo "Cleaning up developer pack examples..."
echo ""

case "${EXAMPLE}" in
  01)
    remove_app "my-api"
    remove_state "my-api"
    ;;
  02)
    remove_app "my-frontend"
    remove_state "my-frontend"
    ;;
  03)
    remove_app "my-api"
    remove_state "my-api"
    ;;
  04)
    kubectl delete secret orkestra-notification -n orkestra-system --ignore-not-found
    remove_app "my-api"
    remove_state "my-api"
    echo "  ✓ orkestra-notification Secret removed"
    ;;
  05)
    remove_app "my-api"
    remove_state "my-api"
    ;;
  all)
    kubectl delete secret orkestra-notification -n orkestra-system --ignore-not-found || true
    for app in my-api my-frontend; do
      remove_app "${app}" || true
      remove_state "${app}" || true
    done
    ;;
  *)
    echo "Unknown example: ${EXAMPLE}"
    echo "Usage: ./cleanup.sh [01|02|03|04|05|all]"
    exit 1
    ;;
esac

echo ""
echo "Done. To also remove the kind cluster:"
echo "  kind delete cluster --name orkestra-playground"
echo ""
echo "To remove all Orkestra state:"
echo "  rm -rf ~/.orkestra/deploy/"
echo "  helm uninstall orkestra -n orkestra-system"
echo ""
