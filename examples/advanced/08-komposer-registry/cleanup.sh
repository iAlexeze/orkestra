#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 08-komposer-registry..."
kubectl delete -f cr.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
helm uninstall orkestra -n orkestra-system
echo "✓ Done."
