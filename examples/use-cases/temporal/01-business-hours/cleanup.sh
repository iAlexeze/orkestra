#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up temporal/01-business-hours..."
kubectl delete devenvironment my-env --ignore-not-found
kubectl delete -f ./crd.yaml --ignore-not-found
echo "✓ Done."
