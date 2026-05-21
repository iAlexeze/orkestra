#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 17-apitype-override..."

kubectl delete -f cr.yaml          --ignore-not-found
kubectl delete -f crd-mine.yaml    --ignore-not-found
kubectl delete -f crd-vendor.yaml  --ignore-not-found

echo "✓ Done. Stop 'ork run' with Ctrl+C if still running."
