#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up without-webhooks example..."

kubectl delete -f cr-port-string.yaml --ignore-not-found
kubectl delete -f cr-port-structured.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
kubectl delete -f bundle.yaml --ignore-not-found
helm uninstall orkestra -n orkestra-system --ignore-not-found

echo "✓ Done."
