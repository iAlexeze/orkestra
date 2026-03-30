#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 07-validation-mutation..."

kubectl delete -f cr-valid.yaml --ignore-not-found
kubectl delete website bad-site --ignore-not-found
kubectl delete website warn-site --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
kubectl delete -f ../install.yaml --ignore-not-found
kubectl delete -f orkestra-configmap.yaml --ignore-not-found
kubectl delete -f namespace.yaml --ignore-not-found

echo "✓ Done."
