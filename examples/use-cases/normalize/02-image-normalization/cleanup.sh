#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up normalize/02-image-normalization..."

kubectl delete -f cr-bare.yaml --ignore-not-found
kubectl delete -f cr-tagged.yaml --ignore-not-found
kubectl delete -f cr-full.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found

echo "✓ Done."
