#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 12-dependencies / 01-in-binary..."

kubectl delete -f cr-app.yaml      --ignore-not-found
kubectl delete -f cr-database.yaml --ignore-not-found
kubectl delete -f crd.yaml         --ignore-not-found

echo "✓ Done. Stop 'ork run' with Ctrl+C if still running."
