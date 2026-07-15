#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up temporal/02-maintenance-window..."
kubectl delete database my-db --ignore-not-found
kubectl delete -f ./crd.yaml --ignore-not-found
echo "✓ Done."
