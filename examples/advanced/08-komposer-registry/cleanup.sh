#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 08-komposer-registry..."
kubectl delete -f ../install.yaml --ignore-not-found
kubectl delete configmap orkestra-katalog -n orkestra-system --ignore-not-found
kubectl delete -f website-crd.yaml --ignore-not-found
echo "✓ Done."
