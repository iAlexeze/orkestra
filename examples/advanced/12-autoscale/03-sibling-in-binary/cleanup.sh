#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 11-autoscale / 03-sibling-in-Binary..."

# Delete any load CRs first
kubectl delete ingestors -l orkestra.load=true --ignore-not-found 2>/dev/null || true

kubectl delete -f cr-loader.yaml  --ignore-not-found
kubectl delete -f cr-processor.yaml  --ignore-not-found
kubectl delete -f cr-auditor.yaml  --ignore-not-found

kubectl delete -f crd-loader.yaml  --ignore-not-found
kubectl delete -f crd-processor.yaml  --ignore-not-found
kubectl delete -f crd-auditor.yaml  --ignore-not-found

echo "✓ Done. Stop 'ork run' with Ctrl+C if still running."
