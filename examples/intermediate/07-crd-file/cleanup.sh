#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 07-crd-file..."

kubectl delete -f cr-app.yaml      --ignore-not-found
kubectl delete -f cr-database.yaml --ignore-not-found
kubectl delete -f crd-app.yaml     --ignore-not-found
kubectl delete -f crd-database.yaml --ignore-not-found

echo "✓ Done. Stop 'ork run' with Ctrl+C if still running."
