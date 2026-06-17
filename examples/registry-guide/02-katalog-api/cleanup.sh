#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 02-katalog-api example..."

kubectl delete -f cr.yaml --ignore-not-found
helm uninstall orkestra -n orkestra-system --ignore-not-found
kubectl delete -f bundle.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found

echo "✓ Done."
