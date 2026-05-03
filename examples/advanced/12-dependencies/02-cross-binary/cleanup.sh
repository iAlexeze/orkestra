#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 12-dependencies / 02-cross-binary..."

kubectl delete -f cr-app.yaml      --ignore-not-found -n app-system
kubectl delete -f cr-database.yaml --ignore-not-found -n db-system
kubectl delete -f crd.yaml         --ignore-not-found

kubectl delete namespace db-system  --ignore-not-found
kubectl delete namespace app-system --ignore-not-found

echo "✓ Done."
