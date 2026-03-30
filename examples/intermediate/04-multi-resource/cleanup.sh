#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 04-multi-resource..."
kubectl delete -f cr.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
echo "✓ Done. Stop 'ork run' with Ctrl+C if still running."
