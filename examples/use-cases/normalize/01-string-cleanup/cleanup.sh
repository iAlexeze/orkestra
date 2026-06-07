#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up normalize/01-string-cleanup..."

kubectl delete -f cr-messy.yaml --ignore-not-found
kubectl delete -f cr-clean.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found

echo "✓ Done."
