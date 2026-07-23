#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up temporal/03-regional-peak..."
kubectl delete cdnedge my-cdn-edge --ignore-not-found
kubectl delete -f ./crd.yaml --ignore-not-found
echo "✓ Done."
