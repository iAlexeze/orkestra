#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 18-crd-file-komposer..."

kubectl delete -f cr.yaml       --ignore-not-found
kubectl delete -f crd-mine.yaml --ignore-not-found

echo "✓ Done. Stop 'ork run' with Ctrl+C if still running."
