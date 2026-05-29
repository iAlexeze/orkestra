#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up Kubebuilder Conversion Solution..."

kubectl delete -f cr-v1.yaml --ignore-not-found
kubectl delete -f cr-v2.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
kubectl delete -f bundle.yaml --ignore-not-found
helm uninstall orkestra -n orkestra-system

echo "✓ Done."
