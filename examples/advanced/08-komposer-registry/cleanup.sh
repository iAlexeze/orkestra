#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 08-komposer-registry..."
kubectl delete -f website-crd.yaml --ignore-not-found
helm uninstall orkestra -n orkestra-system
echo "✓ Done."
