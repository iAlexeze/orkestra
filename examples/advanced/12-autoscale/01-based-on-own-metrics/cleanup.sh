#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 11-autoscale / 01-based-on-own-metrics..."

# Delete any load CRs first
kubectl delete ingestors -l orkestra.load=true --ignore-not-found 2>/dev/null || true

kubectl delete -f cr.yaml  --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found

echo "✓ Done. Stop 'ork run' with Ctrl+C if still running."
