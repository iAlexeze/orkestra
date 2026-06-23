#!/usr/bin/env bash
set -euo pipefail

echo "Cleaning up safe-reconcile example..."

kubectl delete -f cr-app.yaml          --ignore-not-found
kubectl delete -f cr-monitor.yaml      --ignore-not-found
kubectl delete -f cr-queue.yaml        --ignore-not-found
kubectl delete -f crd-typed.yaml       --ignore-not-found
kubectl delete -f crd-declarative.yaml --ignore-not-found

echo "Done. Stop ork run with Ctrl+C if still running."
