#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up enrich/02-warning-events..."

kubectl delete -f cr-healthy.yaml --ignore-not-found
kubectl delete -f cr-broken.yaml --ignore-not-found
kubectl delete -f ../crd.yaml --ignore-not-found

echo "✓ Done."
