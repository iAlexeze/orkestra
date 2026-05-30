#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up CronJob without-webhooks example..."

kubectl delete -f cr-string-schedule.yaml --ignore-not-found
kubectl delete -f cr-structured-schedule.yaml --ignore-not-found
kubectl delete -f crd-single.yaml --ignore-not-found

echo "✓ Done."
