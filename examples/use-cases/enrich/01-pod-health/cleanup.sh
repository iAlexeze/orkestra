#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up enrich/01-pod-health..."

kubectl delete -f cr.yaml --ignore-not-found
kubectl delete -f ../crd.yaml --ignore-not-found

echo "✓ Done."
