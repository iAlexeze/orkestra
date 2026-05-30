#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up normalize/03-defaults-without-webhook..."

kubectl delete -f cr-minimal.yaml --ignore-not-found
kubectl delete -f cr-full.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found

echo "✓ Done."
