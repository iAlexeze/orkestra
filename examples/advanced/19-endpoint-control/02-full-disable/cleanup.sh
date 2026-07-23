#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 19-endpoint-control / 02-full-disable..."

kubectl delete -f cr.yaml  --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found

echo "✓ Done. Stop 'ork run' with Ctrl+C if still running."
