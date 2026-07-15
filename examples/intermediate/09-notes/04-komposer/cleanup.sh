#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 09-notes/03-komposer..."
kubectl delete -f cr-website.yaml  --ignore-not-found
kubectl delete -f cr-workload.yaml --ignore-not-found
kubectl delete -f ../01-built-in/crd.yaml   --ignore-not-found
kubectl delete -f ../02-user-defined/crd.yaml --ignore-not-found
echo "✓ Done. Stop 'ork run' with Ctrl+C if still running."
