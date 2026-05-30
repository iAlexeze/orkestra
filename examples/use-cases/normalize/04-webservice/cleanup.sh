#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up normalize/04-webservice..."

kubectl delete -f cr-simple.yaml --ignore-not-found
kubectl delete -f cr-full.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found

echo "✓ Done."
