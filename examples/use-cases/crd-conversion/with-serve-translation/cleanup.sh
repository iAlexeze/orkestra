#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up CronJob serve-translation example..."

kubectl delete -f cr-default.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found

echo "✓ Done."
