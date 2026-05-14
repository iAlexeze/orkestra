#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 07-multi-crd-pipeline..."

kubectl delete -f crd-loader.yaml    --ignore-not-found
kubectl delete -f crd-processor.yaml --ignore-not-found
kubectl delete -f crd-auditor.yaml   --ignore-not-found
kubectl delete -f crd-pipeline.yaml  --ignore-not-found

echo "✓ Done. Stop 'ork run' with Ctrl+C if still running."
